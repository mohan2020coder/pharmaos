package main

import (
	"context"
	"errors"
	"flag"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go.uber.org/zap"
	gormlogger "gorm.io/gorm/logger"

	"github.com/pharmaos/pharmaos/backend/internal/config"
	"github.com/pharmaos/pharmaos/backend/internal/database"
	"github.com/pharmaos/pharmaos/backend/internal/platform/logging"
	"github.com/pharmaos/pharmaos/backend/internal/server"
)

func main() {
	migrateDown := flag.Int("migrate-down", 0, "roll back N migrations and exit")
	flag.Parse()

	cfg, err := config.Load()
	if err != nil {
		panic(err)
	}
	logger, err := logging.New(cfg.LogLevel, cfg.Environment == "production")
	if err != nil {
		panic(err)
	}
	defer logger.Sync()

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	if *migrateDown > 0 {
		version, err := database.MigrateDown(ctx, cfg.DatabaseURL, cfg.MigrationsDir, *migrateDown)
		if err != nil {
			logger.Fatal("migration rollback failed", zap.Error(err))
		}
		logger.Info("migration rollback complete", zap.Int("next_version", version))
		return
	}
	version, err := database.MigrateUp(ctx, cfg.DatabaseURL, cfg.MigrationsDir, 0)
	if err != nil {
		logger.Fatal("migration failed", zap.Error(err))
	}
	logger.Info("migrations applied", zap.Int("version", version))

	logMode := gormlogger.Warn
	if cfg.Environment == "development" && cfg.LogLevel == "debug" {
		logMode = gormlogger.Info
	}
	db, err := database.Connect(cfg.DatabaseURL, 40, 10, logMode)
	if err != nil {
		logger.Fatal("database connection failed", zap.Error(err))
	}
	defer db.Close()

	app := server.New(cfg, db, logger)
	workerCtx, workerCancel := context.WithCancel(context.Background())
	defer workerCancel()
	if worker := app.IntegrationWorker(); worker != nil {
		go worker.Run(workerCtx)
	}
	httpServer := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           app.Engine,
		ReadHeaderTimeout: 10 * time.Second,
		IdleTimeout:       60 * time.Second,
		WriteTimeout:      2 * time.Minute,
	}

	go func() {
		logger.Info("starting PharmaOS API", zap.String("addr", cfg.HTTPAddr), zap.String("env", cfg.Environment))
		if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Fatal("http server failed", zap.Error(err))
		}
	}()

	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)
	<-signals
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer shutdownCancel()
	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		logger.Error("graceful shutdown failed", zap.Error(err))
	}
	workerCancel()
	logger.Info("server stopped")
}

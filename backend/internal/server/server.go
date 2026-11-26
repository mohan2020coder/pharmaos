package server

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/pharmaos/pharmaos/backend/internal/audit"
	"github.com/pharmaos/pharmaos/backend/internal/auth"
	"github.com/pharmaos/pharmaos/backend/internal/config"
	"github.com/pharmaos/pharmaos/backend/internal/crm"
	"github.com/pharmaos/pharmaos/backend/internal/database"
	"github.com/pharmaos/pharmaos/backend/internal/dcr"
	"github.com/pharmaos/pharmaos/backend/internal/distribution"
	"github.com/pharmaos/pharmaos/backend/internal/divisions"
	"github.com/pharmaos/pharmaos/backend/internal/employees"
	"github.com/pharmaos/pharmaos/backend/internal/health"
	"github.com/pharmaos/pharmaos/backend/internal/integrations"
	"github.com/pharmaos/pharmaos/backend/internal/masterdata"
	"github.com/pharmaos/pharmaos/backend/internal/metrics"
	"github.com/pharmaos/pharmaos/backend/internal/middleware"
	"github.com/pharmaos/pharmaos/backend/internal/organization"
	"github.com/pharmaos/pharmaos/backend/internal/permissions"
	"github.com/pharmaos/pharmaos/backend/internal/platform/apidocs"
	"github.com/pharmaos/pharmaos/backend/internal/platform/crud"
	"github.com/pharmaos/pharmaos/backend/internal/reporting"
	"github.com/pharmaos/pharmaos/backend/internal/roles"
	"github.com/pharmaos/pharmaos/backend/internal/syncapi"
	"github.com/pharmaos/pharmaos/backend/internal/tenants"
	"github.com/pharmaos/pharmaos/backend/internal/territories"
	"github.com/pharmaos/pharmaos/backend/internal/tours"
	"github.com/pharmaos/pharmaos/backend/internal/users"
	"github.com/pharmaos/pharmaos/backend/internal/visits"
)

type Server struct {
	Engine *gin.Engine
	worker *integrations.Worker
}

func (s *Server) IntegrationWorker() *integrations.Worker { return s.worker }

func New(cfg config.Config, db *database.DB, logger *zap.Logger) *Server {
	if cfg.Environment == "production" {
		gin.SetMode(gin.ReleaseMode)
	}
	router := gin.New()
	router.SetTrustedProxies(cfg.TrustedProxies)
	router.Use(
		middleware.Recovery(logger),
		middleware.RequestID(),
		middleware.Logging(logger),
		middleware.SecurityHeaders(),
		middleware.CORS(cfg.CORSOrigins),
		middleware.ClientHints(),
		metrics.Default().Middleware(),
	)
	if db == nil || cfg.RateLimitBackend == "" || cfg.RateLimitBackend == "memory" {
		router.Use(middleware.RateLimit(cfg.RateLimitPerMin))
	} else {
		router.Use(middleware.DistributedRateLimit(db.DB, cfg.RateLimitPerMin))
	}

	migrations, err := database.LoadMigrations(cfg.MigrationsDir)
	if err != nil {
		logger.Fatal("load migrations", zap.Error(err))
	}
	latest := 0
	for _, migration := range migrations {
		latest = migration.Version
	}

	if db != nil {
		health.NewHandler(db, cfg.DatabaseURL, latest).Register(router.Group("/"))
	}

	router.GET("/docs/openapi.yaml", func(c *gin.Context) {
		c.Data(http.StatusOK, "application/yaml", apidocs.Spec())
	})
	router.GET("/docs", func(c *gin.Context) {
		c.Data(http.StatusOK, "text/html; charset=utf-8", apidocs.UI())
	})
	router.GET("/metrics", metrics.Default().Handler())

	if db == nil {
		return &Server{Engine: router}
	}

	base := db.DB
	tokens := auth.NewManager(cfg.JWTSecret, cfg.AccessTokenTTL, cfg.RefreshTokenTTL)
	auditService := audit.NewService(base)
	authRepo := auth.NewRepository()
	authService := auth.NewService(base, authRepo, tokens, auditService, cfg.PasswordHashCost, nil)
	authHandler := auth.NewHandler(authService)

	tenantRepo := tenants.NewRepository(base)
	tenantService := tenants.NewService(tenantRepo, auditService)
	tenantHandler := tenants.NewHandler(tenantService)

	userRepo := users.NewRepository(base)
	userService := users.NewService(userRepo, auditService, cfg.PasswordHashCost)
	userHandler := users.NewHandler(userService)

	permissionRepo := permissions.NewRepository(base)
	permissionHandler := permissions.NewHandler(permissionRepo)

	roleRepo := roles.NewRepository(base)
	roleService := roles.NewService(roleRepo, auditService)
	roleHandler := roles.NewHandler(roleService)

	divisionRepo := crud.New[divisions.Division](base)
	divisionService := divisions.NewService(divisionRepo)
	divisionHandler := divisions.NewHandler(divisionService)

	regionRepo := crud.New[territories.Region](base)
	territoryRepo := crud.New[territories.Territory](base)
	territoryService := territories.NewService(base, regionRepo, territoryRepo)
	territoryHandler := territories.NewHandler(territoryService)

	branchRepo := crud.New[organization.Branch](base)
	organizationService := organization.NewService(branchRepo, base)
	organizationHandler := organization.NewHandler(organizationService)

	employeeRepo := crud.New[employees.Employee](base)
	positionRepo := crud.New[employees.Position](base)
	employeeService := employees.NewService(base, employeeRepo, positionRepo)
	employeeHandler := employees.NewHandler(employeeService)

	masterDataHandler := masterdata.NewHandler(base)
	crmService := crm.NewService(base)
	crmHandler := crm.NewHandler(crmService)
	tourService := tours.NewService(base, auditService)
	tourHandler := tours.NewHandler(tourService)
	visitService := visits.NewService(base, auditService)
	visitHandler := visits.NewHandler(visitService)
	dcrService := dcr.NewService(base, auditService)
	dcrHandler := dcr.NewHandler(dcrService)
	distributionService := distribution.NewService(base, auditService)
	distributionHandler := distribution.NewHandler(distributionService)
	integrationService := integrations.NewService(base, auditService, integrations.Options{
		Timeout: cfg.WebhookTimeout, MaxAttempts: cfg.WebhookMaxAttempts,
		AllowInsecureURL: cfg.AllowInsecureWebhooks,
	})
	integrationHandler := integrations.NewHandler(integrationService)
	worker := integrations.NewWorker(integrationService, cfg.WebhookWorkerInterval, logger)
	reportingService := reporting.NewService(base, auditService)
	reportingHandler := reporting.NewHandler(reportingService)
	syncService := syncapi.NewService(base, visitService, dcrService, tourService, distributionService)
	syncHandler := syncapi.NewHandler(syncService)

	api := router.Group("/api/v1")
	authHandler.RegisterPublic(api)

	protected := api.Group("")
	protected.Use(middleware.Authenticate(tokens), middleware.TenantTransaction(base))
	authHandler.RegisterProtected(protected)
	tenantHandler.Register(protected)
	userHandler.Register(protected)
	permissionHandler.Register(protected)
	roleHandler.Register(protected)
	divisionHandler.Register(protected)
	territoryHandler.Register(protected)
	organizationHandler.Register(protected)
	employeeHandler.Register(protected)
	masterDataHandler.Register(protected)
	crmHandler.Register(protected)
	tourHandler.Register(protected)
	visitHandler.Register(protected)
	dcrHandler.Register(protected)
	distributionHandler.Register(protected)
	reportingHandler.Register(protected)
	integrationHandler.Register(protected)
	syncHandler.Register(protected)

	return &Server{Engine: router, worker: worker}
}

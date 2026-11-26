package health

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/pharmaos/pharmaos/backend/internal/database"
	"github.com/pharmaos/pharmaos/backend/internal/platform/response"
)

type Handler struct {
	db              *database.DB
	dsn             string
	expectedVersion int
	startedAt       time.Time
}

func NewHandler(db *database.DB, dsn string, expectedVersion int) *Handler {
	return &Handler{db: db, dsn: dsn, expectedVersion: expectedVersion, startedAt: time.Now().UTC()}
}

func (h *Handler) Register(router *gin.RouterGroup) {
	router.GET("/healthz", h.liveness)
	router.GET("/readyz", h.readiness)
}

func (h *Handler) liveness(c *gin.Context) {
	response.OK(c, gin.H{"status": "alive", "uptime_seconds": time.Since(h.startedAt).Seconds()})
}

func (h *Handler) readiness(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
	defer cancel()
	if err := h.db.Ping(ctx); err != nil {
		response.Fail(c, http.StatusServiceUnavailable, "Database unavailable", gin.H{"database": err.Error()})
		return
	}
	applied, err := database.AppliedVersions(ctx, h.dsn)
	if err != nil {
		response.Fail(c, http.StatusServiceUnavailable, "Migration state unavailable", gin.H{"migrations": err.Error()})
		return
	}
	latest := 0
	if len(applied) > 0 {
		latest = applied[len(applied)-1]
	}
	if latest < h.expectedVersion {
		response.Fail(c, http.StatusServiceUnavailable, "Migrations pending", gin.H{"applied": latest, "expected": h.expectedVersion})
		return
	}
	response.OK(c, gin.H{"status": "ready", "database": "ok", "migration_version": latest})
}

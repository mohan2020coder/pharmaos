package permissions

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/pharmaos/pharmaos/backend/internal/middleware"
	"github.com/pharmaos/pharmaos/backend/internal/platform/response"
)

type Handler struct{ repo *Repository }

func NewHandler(repo *Repository) *Handler { return &Handler{repo: repo} }

func (h *Handler) Register(router *gin.RouterGroup) {
	router.GET("/permissions", middleware.RequirePermission("role.view"), h.list)
}

func (h *Handler) list(c *gin.Context) {
	permissions, err := h.repo.List(c.Request.Context())
	if err != nil {
		response.Fail(c, http.StatusInternalServerError, "Internal server error", nil)
		return
	}
	response.OK(c, permissions)
}

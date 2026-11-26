package syncapi

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/pharmaos/pharmaos/backend/internal/auth"
	"github.com/pharmaos/pharmaos/backend/internal/platform/httputil"
	"github.com/pharmaos/pharmaos/backend/internal/platform/response"
)

type Handler struct{ service *Service }

func NewHandler(service *Service) *Handler { return &Handler{service: service} }

func (h *Handler) Register(router *gin.RouterGroup) {
	group := router.Group("/sync")
	group.GET("/changes", h.changes)
	group.POST("/operations", h.operations)
	group.GET("/status", h.status)
}

func (h *Handler) changes(c *gin.Context) {
	limit := intParam(c.Query("limit"), 100)
	changes, err := h.service.Changes(c.Request.Context(), auth.MustPrincipal(c.Request.Context()),
		c.Query("since"), c.Query("entities"), limit)
	if err != nil {
		httputil.FailError(c, err)
		return
	}
	response.JSON(c, http.StatusOK, "Changes retrieved", changes, nil)
}

func (h *Handler) operations(c *gin.Context) {
	var request PushRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		httputil.BadJSON(c, err)
		return
	}
	result, err := h.service.Push(c.Request.Context(), auth.MustPrincipal(c.Request.Context()), request)
	if err != nil {
		httputil.FailError(c, err)
		return
	}
	response.JSON(c, http.StatusOK, "Operations processed", result, nil)
}

func (h *Handler) status(c *gin.Context) {
	status, err := h.service.Status(c.Request.Context(), auth.MustPrincipal(c.Request.Context()))
	if err != nil {
		httputil.FailError(c, err)
		return
	}
	response.OK(c, status)
}

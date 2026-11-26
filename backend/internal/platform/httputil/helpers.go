package httputil

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	apperrors "github.com/pharmaos/pharmaos/backend/internal/platform/errors"
	"github.com/pharmaos/pharmaos/backend/internal/platform/response"
)

func UUID(c *gin.Context, param string) (uuid.UUID, bool) {
	id, err := uuid.Parse(c.Param(param))
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "Invalid identifier", map[string]string{param: "must be a UUID"})
		return uuid.Nil, false
	}
	return id, true
}

func FailError(c *gin.Context, err error) {
	if appErr, ok := err.(*apperrors.AppError); ok {
		response.Fail(c, appErr.Status, appErr.Message, appErr.Details)
		return
	}
	response.Fail(c, http.StatusInternalServerError, "Internal server error", nil)
}

func BadJSON(c *gin.Context, err error) {
	response.Fail(c, http.StatusBadRequest, "Invalid JSON payload", gin.H{"body": err.Error()})
}

func Invalid(c *gin.Context, errors map[string]string) {
	response.Fail(c, http.StatusUnprocessableEntity, "Validation failed", errors)
}

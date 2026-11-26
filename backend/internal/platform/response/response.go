package response

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type Envelope struct {
	Success bool        `json:"success"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
	Meta    interface{} `json:"meta,omitempty"`
	Errors  interface{} `json:"errors,omitempty"`
}

type PageMeta struct {
	Page       int   `json:"page"`
	PageSize   int   `json:"page_size"`
	Total      int64 `json:"total"`
	TotalPages int   `json:"total_pages"`
}

func JSON(c *gin.Context, status int, message string, data, meta interface{}) {
	c.JSON(status, Envelope{Success: true, Message: message, Data: data, Meta: meta})
}

func Created(c *gin.Context, message string, data interface{}) {
	JSON(c, http.StatusCreated, message, data, nil)
}

func OK(c *gin.Context, data interface{}) {
	JSON(c, http.StatusOK, "OK", data, nil)
}

func Fail(c *gin.Context, status int, message string, errors interface{}) {
	c.AbortWithStatusJSON(status, Envelope{Success: false, Message: message, Errors: errors})
}

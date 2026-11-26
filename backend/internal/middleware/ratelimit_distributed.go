package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/pharmaos/pharmaos/backend/internal/metrics"
	"github.com/pharmaos/pharmaos/backend/internal/platform/response"
)

func DistributedRateLimit(db *gorm.DB, perMinute int) gin.HandlerFunc {
	return func(c *gin.Context) {
		var allowed bool
		err := db.WithContext(c.Request.Context()).
			Raw(`SELECT pharmaos_rate_limit(?, ?, 60)`, "ip:"+c.ClientIP(), perMinute).
			Scan(&allowed).Error
		if err != nil {
			response.Fail(c, http.StatusServiceUnavailable, "Rate limiter unavailable", nil)
			metrics.Default().SecurityEvent("rate_limiter_errors")
			return
		}
		if !allowed {
			c.Writer.Header().Set("Retry-After", "60")
			response.Fail(c, http.StatusTooManyRequests, "Too many requests", nil)
			return
		}
		c.Next()
	}
}

package middleware

import (
	"bytes"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/pharmaos/pharmaos/backend/internal/platform/response"
)

const RequestIDHeader = "X-Request-ID"

func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.GetHeader(RequestIDHeader)
		if id == "" {
			id = uuid.NewString()
		}
		c.Set(RequestIDHeader, id)
		c.Writer.Header().Set(RequestIDHeader, id)
		c.Next()
	}
}

func Logging(logger *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		logger.Info("http request",
			zap.String("method", c.Request.Method),
			zap.String("path", c.FullPath()),
			zap.String("query", c.Request.URL.RawQuery),
			zap.Int("status", c.Writer.Status()),
			zap.Duration("latency", time.Since(start)),
			zap.String("request_id", c.GetString(RequestIDHeader)),
			zap.String("ip", c.ClientIP()),
		)
	}
}

func Recovery(logger *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if recovered := recover(); recovered != nil {
				logger.Error("panic recovered", zap.Any("panic", recovered), zap.String("request_id", c.GetString(RequestIDHeader)))
				response.Fail(c, http.StatusInternalServerError, "Internal server error", nil)
			}
		}()
		c.Next()
	}
}

func CORS(allowedOrigins []string) gin.HandlerFunc {
	originSet := make(map[string]bool, len(allowedOrigins))
	for _, origin := range allowedOrigins {
		originSet[origin] = true
	}
	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		if originSet[origin] {
			c.Writer.Header().Set("Access-Control-Allow-Origin", origin)
			c.Writer.Header().Set("Vary", "Origin")
			c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
			c.Writer.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type, X-Request-Id, X-Idempotency-Key, Idempotency-Key")
			c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		}
		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}

type rateBucket struct {
	tokens float64
	last   time.Time
}

func RateLimit(perMinute int) gin.HandlerFunc {
	var mu sync.Mutex
	buckets := map[string]*rateBucket{}
	rate := float64(perMinute) / 60
	capacity := float64(perMinute)
	return func(c *gin.Context) {
		key := c.ClientIP()
		now := time.Now()
		mu.Lock()
		bucket := buckets[key]
		if bucket == nil {
			bucket = &rateBucket{tokens: capacity, last: now}
			buckets[key] = bucket
		}
		bucket.tokens = min(capacity, bucket.tokens+now.Sub(bucket.last).Seconds()*rate)
		bucket.last = now
		allowed := bucket.tokens >= 1
		if allowed {
			bucket.tokens--
		}
		mu.Unlock()
		if !allowed {
			c.Writer.Header().Set("Retry-After", "1")
			response.Fail(c, http.StatusTooManyRequests, "Too many requests", nil)
			return
		}
		c.Next()
	}
}

func SecurityHeaders() gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.Writer.Header()
		header.Set("X-Content-Type-Options", "nosniff")
		header.Set("X-Frame-Options", "DENY")
		header.Set("Referrer-Policy", "no-referrer")
		header.Set("Permissions-Policy", "geolocation=(self), camera=()")
		c.Next()
	}
}

func ClientHints() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set("user_agent", c.GetHeader("User-Agent"))
		c.Set("ip", clientIP(c))
		c.Next()
	}
}

func clientIP(c *gin.Context) string {
	forwarded := c.GetHeader("X-Forwarded-For")
	if forwarded != "" {
		if host, _, err := net.SplitHostPort(strings.TrimSpace(strings.Split(forwarded, ",")[0])); err == nil {
			return host
		}
		return strings.TrimSpace(strings.Split(forwarded, ",")[0])
	}
	return c.ClientIP()
}

func BodyPreview(body []byte) string {
	if len(body) > 2048 {
		body = body[:2048]
	}
	return string(bytes.TrimSpace(body))
}

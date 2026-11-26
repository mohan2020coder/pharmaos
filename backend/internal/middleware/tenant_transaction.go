package middleware

import (
	"context"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/pharmaos/pharmaos/backend/internal/auth"
	"github.com/pharmaos/pharmaos/backend/internal/platform/response"
)

type dbContextKey struct{}

func TenantTransaction(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		tx := db.WithContext(c.Request.Context()).Begin()
		if tx.Error != nil {
			c.Abort()
			return
		}
		if err := tx.Exec(`SET LOCAL ROLE pharmaos_app`).Error; err != nil {
			tx.Rollback()
			c.Abort()
			return
		}
		if principal, ok := auth.PrincipalFrom(c.Request.Context()); ok {
			if principal.TenantID != uuid.Nil && !principal.IsPrivileged {
				if err := tx.Exec(`SELECT set_config('app.current_tenant_id', ?, true)`, principal.TenantID.String()).Error; err != nil {
					tx.Rollback()
					c.Abort()
					return
				}
			}
			if err := tx.Exec(`SELECT set_config('app.is_privileged', ?, true)`, strconv.FormatBool(principal.IsPrivileged)).Error; err != nil {
				tx.Rollback()
				c.Abort()
				return
			}
			var active int
			if err := tx.Raw(`SELECT 1 FROM auth_sessions WHERE id = ? AND revoked_at IS NULL AND expires_at > now()`, principal.SessionID).Scan(&active).Error; err != nil || active != 1 {
				tx.Rollback()
				response.Fail(c, http.StatusUnauthorized, "Session expired or revoked", nil)
				return
			}
		}
		ctx := context.WithValue(c.Request.Context(), dbContextKey{}, tx)
		c.Request = c.Request.WithContext(ctx)
		c.Next()
		if c.Writer.Status() >= 400 || tx.Error != nil {
			tx.Rollback()
			return
		}
		if err := tx.Commit().Error; err != nil {
			c.Abort()
			return
		}
	}
}

func DBFromContext(c *gin.Context) *gorm.DB {
	if tx, ok := c.Request.Context().Value(dbContextKey{}).(*gorm.DB); ok {
		return tx
	}
	return nil
}

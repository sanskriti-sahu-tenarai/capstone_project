package middleware

import (
	"log/slog"
	"net/http"

	"order-management/observability"

	"github.com/gin-gonic/gin"
)

func Recovery() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if r := recover(); r != nil {
				observability.ErrorCount.WithLabelValues("panic_recovery").Inc()
				slog.Error("Panic recovered", "error", r)
				c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
					"error": "Internal server error occurred",
				})
			}
		}()
		c.Next()
	}
}

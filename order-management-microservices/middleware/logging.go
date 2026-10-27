package middleware

import (
	"log/slog"
	"strconv"
	"time"

	"order-management/observability"

	"github.com/gin-gonic/gin"
)

func StructuredLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		duration := time.Since(start)

		reqID, _ := c.Get("request_id")
		status := c.Writer.Status()
		path := c.FullPath()
		if path == "" {
			path = c.Request.URL.Path
		}

		statusStr := strconv.Itoa(status)
		observability.HttpRequestsTotal.WithLabelValues(c.Request.Method, path, statusStr).Inc()
		observability.HttpRequestDuration.WithLabelValues(c.Request.Method, path).Observe(duration.Seconds())

		slog.Info("Request handled",
			"request_id", reqID,
			"method", c.Request.Method,
			"path", c.Request.URL.Path,
			"status", status,
			"duration", duration.String(),
			"error", c.Errors.ByType(gin.ErrorTypePrivate).String(),
		)
	}
}

func SecurityHeaders() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("X-Frame-Options", "DENY")
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("X-XSS-Protection", "1; mode=block")
		c.Header("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		c.Next()
	}
}

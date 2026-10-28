package middleware

import (
	"context"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const HeaderRequestID = "X-Request-ID"

func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		reqID := c.GetHeader(HeaderRequestID)
		if reqID == "" {
			reqID = uuid.New().String()
		}
		c.Header(HeaderRequestID, reqID)
		c.Set("request_id", reqID)

		ctx := context.WithValue(c.Request.Context(), "request_id", reqID)
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	}
}

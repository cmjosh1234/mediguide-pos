package middleware

import (
	"strings"
	"time"

	"mediguide/internal/requestctx"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
)

const RequestIDHeader = "X-Request-ID"

func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := strings.TrimSpace(c.GetHeader(RequestIDHeader))
		if !validRequestID(requestID) {
			requestID = uuid.NewString()
		}
		c.Header(RequestIDHeader, requestID)
		c.Request = c.Request.WithContext(requestctx.WithCorrelationID(c.Request.Context(), requestID))
		c.Next()
	}
}

func validRequestID(value string) bool {
	if value == "" || len(value) > 128 {
		return false
	}
	for _, char := range value {
		if (char >= 'a' && char <= 'z') || (char >= 'A' && char <= 'Z') ||
			(char >= '0' && char <= '9') || char == '-' || char == '_' || char == '.' || char == ':' {
			continue
		}
		return false
	}
	return true
}

func RequestLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		route := c.FullPath()
		if route == "" {
			route = "unmatched"
		}
		log.Info().
			Str("request_id", requestctx.CorrelationID(c.Request.Context())).
			Str("method", c.Request.Method).
			Str("route", route).
			Int("status", c.Writer.Status()).
			Dur("latency", time.Since(start)).
			Msg("request")
	}
}

func PrivateNoStore() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Cache-Control", "private, no-store")
		c.Header("Pragma", "no-cache")
		c.Next()
	}
}

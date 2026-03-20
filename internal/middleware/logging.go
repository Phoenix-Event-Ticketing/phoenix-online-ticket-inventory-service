package middleware

import (
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// AccessLog emits a structured JSON line per request with an optional http block (platform format).
func AccessLog(log *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()

		route := c.FullPath()
		if route == "" {
			route = c.Request.URL.Path
		}

		httpFields := map[string]interface{}{
			"method":           c.Request.Method,
			"route":            route,
			"path":             c.Request.URL.Path,
			"status_code":      c.Writer.Status(),
			"response_time_ms": time.Since(start).Milliseconds(),
		}

		fields := []zap.Field{
			zap.String("event", "request_completed"),
			zap.String("request_id", GetRequestID(c)),
			zap.String("trace_id", GetTraceID(c)),
			zap.String("operation", "http_request"),
			zap.Any("http", httpFields),
			zap.String("message", "request completed"),
		}

		status := c.Writer.Status()
		switch {
		case status >= 500:
			log.Error("request completed", fields...)
		case status >= 400:
			log.Warn("request completed", fields...)
		default:
			log.Info("request completed", fields...)
		}
	}
}

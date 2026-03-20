package middleware

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

const (
	headerRequestID = "X-Request-ID"
	headerTraceID   = "X-Trace-ID"

	ctxRequestID = "ctx_request_id"
	ctxTraceID   = "ctx_trace_id"
)

// RequestID ensures each request has request_id and trace_id (incoming headers or generated).
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		reqID := strings.TrimSpace(c.GetHeader(headerRequestID))
		if reqID == "" {
			reqID = "req-" + randomID()
		}
		traceID := strings.TrimSpace(c.GetHeader(headerTraceID))
		if traceID == "" {
			traceID = reqID
		}

		c.Writer.Header().Set(headerRequestID, reqID)
		c.Writer.Header().Set(headerTraceID, traceID)

		c.Set(ctxRequestID, reqID)
		c.Set(ctxTraceID, traceID)

		c.Next()
	}
}

func randomID() string {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		return fmt.Sprintf("nrand-%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(b)
}

// GetRequestID returns the request id from context, or empty if missing.
func GetRequestID(c *gin.Context) string {
	v, ok := c.Get(ctxRequestID)
	if !ok {
		return ""
	}
	s, _ := v.(string)
	return s
}

// GetTraceID returns the trace id from context, or empty if missing.
func GetTraceID(c *gin.Context) string {
	v, ok := c.Get(ctxTraceID)
	if !ok {
		return ""
	}
	s, _ := v.(string)
	return s
}

package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestRequestID_GeneratesHeaders(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(RequestID())
	r.GET("/x", func(c *gin.Context) {
		if GetRequestID(c) == "" {
			t.Fatal("missing request id")
		}
		if GetTraceID(c) == "" {
			t.Fatal("missing trace id")
		}
		c.Status(http.StatusOK)
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	r.ServeHTTP(w, req)
	if w.Header().Get("X-Request-ID") == "" {
		t.Fatal("response missing X-Request-ID")
	}
}

func TestRequestID_IncomingHeaders(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(RequestID())
	r.GET("/x", func(c *gin.Context) {
		if GetRequestID(c) != "rid-1" {
			t.Fatal(GetRequestID(c))
		}
		c.Status(http.StatusOK)
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	req.Header.Set("X-Request-ID", "rid-1")
	req.Header.Set("X-Trace-ID", "tid-1")
	r.ServeHTTP(w, req)
	if w.Header().Get("X-Request-ID") != "rid-1" {
		t.Fatal(w.Header().Get("X-Request-ID"))
	}
}

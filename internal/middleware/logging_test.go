package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func TestAccessLog(t *testing.T) {
	gin.SetMode(gin.TestMode)
	log := zap.NewNop()
	r := gin.New()
	r.Use(AccessLog(log))
	r.GET("/x", func(c *gin.Context) { c.Status(http.StatusOK) })

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status %d", w.Code)
	}
}

func TestAccessLog_ErrorStatus(t *testing.T) {
	gin.SetMode(gin.TestMode)
	log := zap.NewNop()
	r := gin.New()
	r.Use(AccessLog(log))
	r.GET("/x", func(c *gin.Context) { c.Status(http.StatusInternalServerError) })

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("status %d", w.Code)
	}
}

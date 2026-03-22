package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Phoenix-Event-Ticketing/phoenix-online-ticket-inventory-service/internal/config"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

func TestRequirePermission_UserAllowed(t *testing.T) {
	gin.SetMode(gin.TestMode)
	cfg := &config.Config{
		Environment: "test",
		JWTSecret:   "test-secret",
	}
	mw := NewMiddleware(cfg)

	r := gin.New()
	r.GET("/x", mw.Authenticate(), mw.RequirePermission(ViewTicketInventory), func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	token := mustSign(t, cfg.JWTSecret, &Claims{
		Sub:         "u1",
		Permissions: []string{ViewTicketInventory},
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
		},
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestRequirePermission_UserForbidden(t *testing.T) {
	gin.SetMode(gin.TestMode)
	cfg := &config.Config{Environment: "test", JWTSecret: "test-secret"}
	mw := NewMiddleware(cfg)
	r := gin.New()
	r.GET("/x", mw.Authenticate(), mw.RequirePermission(CreateTicketType), func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	token := mustSign(t, cfg.JWTSecret, &Claims{
		Sub:         "u1",
		Permissions: []string{ViewTicketInventory},
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
		},
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", w.Code)
	}
}

func TestRequirePermission_ServiceRegistry(t *testing.T) {
	gin.SetMode(gin.TestMode)
	cfg := &config.Config{
		Environment: "test",
		JWTSecret:   "test-secret",
		ServiceRegistry: map[string][]string{
			"booking-service": {ReserveTicket},
		},
	}
	mw := NewMiddleware(cfg)
	r := gin.New()
	r.GET("/x", mw.Authenticate(), mw.RequirePermission(ReserveTicket), func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	token := mustSign(t, cfg.JWTSecret, &Claims{
		Sub: "booking-service",
		Typ: "service",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
		},
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestRequirePermission_ServiceNotInRegistry(t *testing.T) {
	gin.SetMode(gin.TestMode)
	cfg := &config.Config{
		Environment:     "test",
		JWTSecret:       "test-secret",
		ServiceRegistry: map[string][]string{"other": {ReserveTicket}},
	}
	mw := NewMiddleware(cfg)
	r := gin.New()
	r.GET("/x", mw.Authenticate(), mw.RequirePermission(ReserveTicket), func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	token := mustSign(t, cfg.JWTSecret, &Claims{
		Sub: "booking-service",
		Typ: "service",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
		},
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", w.Code)
	}
}

func TestAuthenticate_MissingToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	cfg := &config.Config{Environment: "test", JWTSecret: "test-secret"}
	mw := NewMiddleware(cfg)
	r := gin.New()
	r.GET("/x", mw.Authenticate(), func(c *gin.Context) { c.Status(http.StatusOK) })

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestAuthDisabled_SkipsAuth(t *testing.T) {
	gin.SetMode(gin.TestMode)
	cfg := &config.Config{
		Environment:  "test",
		JWTSecret:    "",
		AuthDisabled: "true",
	}
	mw := NewMiddleware(cfg)
	r := gin.New()
	r.GET("/x", mw.Authenticate(), mw.RequirePermission(ViewTicketInventory), func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func mustSign(t *testing.T, secret string, claims *Claims) string {
	t.Helper()
	s, err := SignHS256(secret, claims)
	if err != nil {
		t.Fatal(err)
	}
	return s
}

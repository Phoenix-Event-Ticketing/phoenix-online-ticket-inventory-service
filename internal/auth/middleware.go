package auth

import (
	"net/http"
	"strings"

	"github.com/Phoenix-Event-Ticketing/phoenix-online-ticket-inventory-service/internal/config"
	"github.com/gin-gonic/gin"
)

const (
	ctxClaims = "auth_claims"
)

// Middleware provides JWT authentication and permission checks.
type Middleware struct {
	cfg *config.Config
}

// NewMiddleware builds auth middleware from application config.
func NewMiddleware(cfg *config.Config) *Middleware {
	if cfg == nil {
		cfg = &config.Config{}
	}
	return &Middleware{cfg: cfg}
}

// ClaimsFromContext returns validated claims, or nil if auth was skipped or missing.
func ClaimsFromContext(c *gin.Context) *Claims {
	v, ok := c.Get(ctxClaims)
	if !ok {
		return nil
	}
	cl, _ := v.(*Claims)
	return cl
}

// Authenticate validates Authorization: Bearer JWT and stores claims on the context.
func (m *Middleware) Authenticate() gin.HandlerFunc {
	return func(c *gin.Context) {
		if m.cfg.AuthDisabledEffective() {
			c.Next()
			return
		}
		claims, err := ParseAndValidateBearer(c.GetHeader("Authorization"), m.cfg.JWTSecret)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}
		c.Set(ctxClaims, claims)
		c.Next()
	}
}

// RequirePermission enforces RBAC: users must list the permission; services must be allowed by SERVICE_REGISTRY.
func (m *Middleware) RequirePermission(permission string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if m.cfg.AuthDisabledEffective() {
			c.Next()
			return
		}
		claims := ClaimsFromContext(c)
		if claims == nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}
		if claims.IsService() {
			if !serviceAllowed(m.cfg.ServiceRegistry, claims.Sub, permission) {
				c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "forbidden"})
				return
			}
			c.Next()
			return
		}
		if !userHasPermission(claims.Permissions, permission) {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "forbidden"})
			return
		}
		c.Next()
	}
}

func userHasPermission(perms []string, need string) bool {
	for _, p := range perms {
		if strings.TrimSpace(p) == need {
			return true
		}
	}
	return false
}

func serviceAllowed(registry map[string][]string, serviceID, permission string) bool {
	if registry == nil {
		return false
	}
	allowed, ok := registry[strings.TrimSpace(serviceID)]
	if !ok {
		return false
	}
	for _, p := range allowed {
		if strings.TrimSpace(p) == permission {
			return true
		}
	}
	return false
}

package auth

import (
	"github.com/golang-jwt/jwt/v5"
)

// Claims matches the User Service JWT payload (spec §6) plus a service principal marker.
type Claims struct {
	Sub         string   `json:"sub"`
	Email       string   `json:"email,omitempty"`
	Role        string   `json:"role,omitempty"`
	Permissions []string `json:"permissions,omitempty"`
	// Typ is "service" for service-to-service tokens; empty or "user" for end-user tokens.
	Typ string `json:"typ,omitempty"`
	jwt.RegisteredClaims
}

// IsService returns true when the token represents an internal service caller.
func (c *Claims) IsService() bool {
	return c.Typ == "service"
}

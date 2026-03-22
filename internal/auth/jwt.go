package auth

import (
	"fmt"
	"strings"

	"github.com/golang-jwt/jwt/v5"
)

// ParseAndValidateBearer extracts the Bearer token and verifies HS256 signature and claims.
func ParseAndValidateBearer(authHeader, secret string) (*Claims, error) {
	if strings.TrimSpace(secret) == "" {
		return nil, fmt.Errorf("jwt secret not configured")
	}
	raw := strings.TrimSpace(authHeader)
	if raw == "" {
		return nil, fmt.Errorf("missing authorization")
	}
	const prefix = "Bearer "
	if !strings.HasPrefix(raw, prefix) {
		return nil, fmt.Errorf("invalid authorization scheme")
	}
	tokenStr := strings.TrimSpace(strings.TrimPrefix(raw, prefix))
	if tokenStr == "" {
		return nil, fmt.Errorf("empty bearer token")
	}

	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenStr, claims, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method %v", token.Header["alg"])
		}
		return []byte(secret), nil
	})
	if err != nil {
		return nil, err
	}
	if !token.Valid {
		return nil, fmt.Errorf("invalid token")
	}
	return claims, nil
}

// SignHS256 creates an HS256-signed JWT (tests and tooling; User Service issues production user tokens).
func SignHS256(secret string, claims *Claims) (string, error) {
	if strings.TrimSpace(secret) == "" {
		return "", fmt.Errorf("empty secret")
	}
	t := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return t.SignedString([]byte(secret))
}

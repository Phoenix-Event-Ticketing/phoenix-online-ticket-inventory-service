package auth

import (
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func TestParseAndValidateBearer_Errors(t *testing.T) {
	_, err := ParseAndValidateBearer("", "secret")
	if err == nil {
		t.Fatal("expected error")
	}
	_, err = ParseAndValidateBearer("Basic x", "secret")
	if err == nil || !strings.Contains(err.Error(), "scheme") {
		t.Fatalf("got %v", err)
	}
	_, err = ParseAndValidateBearer("Bearer ", "secret")
	if err == nil {
		t.Fatal("expected error")
	}
	_, err = ParseAndValidateBearer("Bearer x", "")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestSignHS256_EmptySecret(t *testing.T) {
	_, err := SignHS256("", &Claims{RegisteredClaims: jwt.RegisteredClaims{
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
	}})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestClaims_IsService(t *testing.T) {
	var c Claims
	if c.IsService() {
		t.Fatal()
	}
	c.Typ = "service"
	if !c.IsService() {
		t.Fatal()
	}
}

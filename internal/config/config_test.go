package config

import (
	"strings"
	"testing"
	"time"
)

func TestParseServiceRegistry_ValidJSON(t *testing.T) {
	t.Setenv("MONGODB_URI", "mongodb://localhost")
	t.Setenv("SERVICE_REGISTRY", `{"svc":["VIEW_TICKET_INVENTORY"]}`)
	t.Setenv("JWT_SECRET", "x")
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if len(cfg.ServiceRegistry["svc"]) != 1 || cfg.ServiceRegistry["svc"][0] != "VIEW_TICKET_INVENTORY" {
		t.Fatalf("unexpected registry: %#v", cfg.ServiceRegistry)
	}
}

func TestAuthDisabledEffective_OnlyInDevOrTest(t *testing.T) {
	c := Config{Environment: "production", AuthDisabled: "true"}
	if c.AuthDisabledEffective() {
		t.Fatal("auth must not disable in production")
	}
	c.Environment = "prod"
	if c.AuthDisabledEffective() {
		t.Fatal("auth must not disable in prod alias")
	}
	c.Environment = "development"
	if !c.AuthDisabledEffective() {
		t.Fatal("expected disabled in development when AUTH_DISABLED=true")
	}
	c.Environment = "dev"
	if !c.AuthDisabledEffective() {
		t.Fatal("expected disabled in dev alias when AUTH_DISABLED=true")
	}
}

func TestLoad_RequiresJWTWhenAuthEnabled(t *testing.T) {
	t.Setenv("MONGODB_URI", "mongodb://localhost")
	t.Setenv("JWT_SECRET", "")
	t.Setenv("ENVIRONMENT", "production")
	t.Setenv("AUTH_DISABLED", "")
	_, err := Load()
	if err == nil {
		t.Fatal("expected error without JWT_SECRET in production")
	}
}

func TestLoad_AllowsEmptyJWTWhenAuthDisabledInTest(t *testing.T) {
	t.Setenv("MONGODB_URI", "mongodb://localhost")
	t.Setenv("JWT_SECRET", "")
	t.Setenv("ENVIRONMENT", "test")
	t.Setenv("AUTH_DISABLED", "true")
	_, err := Load()
	if err != nil {
		t.Fatal(err)
	}
}

func TestLoad_AllowsEmptyJWTWhenAuthDisabledInDevAlias(t *testing.T) {
	t.Setenv("MONGODB_URI", "mongodb://localhost")
	t.Setenv("JWT_SECRET", "")
	t.Setenv("ENVIRONMENT", "dev")
	t.Setenv("AUTH_DISABLED", "true")
	_, err := Load()
	if err != nil {
		t.Fatal(err)
	}
}

func TestLoad_RejectsAuthDisabledOutsideAllowedEnvironments(t *testing.T) {
	t.Setenv("MONGODB_URI", "mongodb://localhost")
	t.Setenv("JWT_SECRET", "")
	t.Setenv("ENVIRONMENT", "staging")
	t.Setenv("AUTH_DISABLED", "true")
	_, err := Load()
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "AUTH_DISABLED=true is only allowed") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestLoad_InvalidServiceRegistryJSON(t *testing.T) {
	t.Setenv("MONGODB_URI", "mongodb://localhost")
	t.Setenv("JWT_SECRET", "x")
	t.Setenv("SERVICE_REGISTRY", `{`)
	_, err := Load()
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestLoad_InvalidPort(t *testing.T) {
	t.Setenv("PORT", "not-a-number")
	_, err := Load()
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestConfig_HoldTTL(t *testing.T) {
	c := Config{HoldTTLMinutes: 0}
	if c.HoldTTL() != 15*time.Minute {
		t.Fatal(c.HoldTTL())
	}
	c.HoldTTLMinutes = 30
	if c.HoldTTL() != 30*time.Minute {
		t.Fatal(c.HoldTTL())
	}
}

func TestConfig_EventServiceTimeout(t *testing.T) {
	c := Config{EventServiceTimeoutMs: 0}
	if c.EventServiceTimeout() != 3*time.Second {
		t.Fatal(c.EventServiceTimeout())
	}
	c.EventServiceTimeoutMs = 1250
	if c.EventServiceTimeout() != 1250*time.Millisecond {
		t.Fatal(c.EventServiceTimeout())
	}
}

func TestLoad_EventServiceConfigFields(t *testing.T) {
	t.Setenv("MONGODB_URI", "mongodb://localhost")
	t.Setenv("JWT_SECRET", "x")
	t.Setenv("EVENT_SERVICE_URL", " http://event-service:3000/ ")
	t.Setenv("EVENT_SERVICE_TIMEOUT_MS", "4500")
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.EventServiceURL != "http://event-service:3000/" {
		t.Fatalf("unexpected event service URL: %q", cfg.EventServiceURL)
	}
	if cfg.EventServiceTimeoutMs != 4500 {
		t.Fatalf("unexpected timeout ms: %d", cfg.EventServiceTimeoutMs)
	}
}

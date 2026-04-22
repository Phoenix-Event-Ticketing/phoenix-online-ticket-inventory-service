package config

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

// Config holds runtime configuration loaded from the environment.
type Config struct {
	Port                  string
	MongoURI              string
	MongoDatabase         string
	EventServiceURL       string
	Environment           string
	ServiceName           string
	LogLevel              string
	HoldTTLMinutes        int
	EventServiceTimeoutMs int
	MetricsEnabled        bool
	JaegerEndpoint        string
	OtelTracesSampler     string
	// JWTSecret is the HMAC key shared with the User Service for validating Bearer tokens.
	JWTSecret string
	// AuthDisabled skips JWT checks when true; only allowed in development or test environments.
	AuthDisabled string
	// ServiceRegistry maps service IDs (JWT "sub" for typ=service) to allowed permission names.
	ServiceRegistry map[string][]string
}

// Load reads configuration from environment variables. If a .env file exists, it is loaded first (best-effort).
func Load() (Config, error) {
	_ = godotenv.Load()

	port := getEnv("PORT", "8080")
	if _, err := strconv.Atoi(port); err != nil {
		return Config{}, fmt.Errorf("PORT must be numeric: %w", err)
	}

	cfg := Config{
		Port:                  port,
		MongoURI:              os.Getenv("MONGODB_URI"),
		MongoDatabase:         getEnv("MONGODB_DATABASE", "phoenix_inventory"),
		EventServiceURL:       strings.TrimSpace(getEnv("EVENT_SERVICE_URL", "")),
		Environment:           getEnv("ENVIRONMENT", "development"),
		ServiceName:           getEnv("SERVICE_NAME", "ticket-inventory-service"),
		LogLevel:              getEnv("LOG_LEVEL", "info"),
		HoldTTLMinutes:        getEnvInt("HOLD_TTL_MINUTES", 15),
		EventServiceTimeoutMs: getEnvInt("EVENT_SERVICE_TIMEOUT_MS", 3000),
		MetricsEnabled:        getEnvBool("METRICS_ENABLED", true),
		JaegerEndpoint:        strings.TrimSpace(getEnv("JAEGER_ENDPOINT", "")),
		OtelTracesSampler:     strings.TrimSpace(getEnv("OTEL_TRACES_SAMPLER", "1.0")),
		JWTSecret:             strings.TrimSpace(os.Getenv("JWT_SECRET")),
		AuthDisabled:          strings.TrimSpace(os.Getenv("AUTH_DISABLED")),
		ServiceRegistry:       map[string][]string{},
	}

	if cfg.MongoURI == "" {
		return Config{}, fmt.Errorf("MONGODB_URI is required")
	}

	if err := parseServiceRegistry(getEnv("SERVICE_REGISTRY", ""), &cfg); err != nil {
		return Config{}, err
	}

	if strings.EqualFold(strings.TrimSpace(cfg.AuthDisabled), "true") && !isAuthDisableAllowedEnvironment(cfg.Environment) {
		return Config{}, fmt.Errorf("AUTH_DISABLED=true is only allowed when ENVIRONMENT is development/dev or test; got %q. Set JWT_SECRET to enable authentication", strings.TrimSpace(cfg.Environment))
	}

	if !cfg.AuthDisabledEffective() && cfg.JWTSecret == "" {
		return Config{}, fmt.Errorf("JWT_SECRET is required when authentication is enabled (set AUTH_DISABLED=true only in development/test to skip)")
	}

	return cfg, nil
}

func parseServiceRegistry(raw string, cfg *Config) error {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	var m map[string][]string
	if err := json.Unmarshal([]byte(raw), &m); err != nil {
		return fmt.Errorf("SERVICE_REGISTRY must be valid JSON object mapping service id to permission lists: %w", err)
	}
	cfg.ServiceRegistry = m
	return nil
}

// AuthDisabledEffective is true only when AUTH_DISABLED is set and the environment is development or test.
func (c Config) AuthDisabledEffective() bool {
	if !isAuthDisableAllowedEnvironment(c.Environment) {
		return false
	}
	return strings.EqualFold(strings.TrimSpace(c.AuthDisabled), "true")
}

func isAuthDisableAllowedEnvironment(environment string) bool {
	env := normalizeEnvironment(environment)
	return env == "development" || env == "test"
}

func normalizeEnvironment(environment string) string {
	env := strings.ToLower(strings.TrimSpace(environment))
	switch env {
	case "dev":
		return "development"
	case "prod":
		return "production"
	default:
		return env
	}
}

func getEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func getEnvInt(key string, def int) int {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return def
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return def
	}
	return n
}

func getEnvBool(key string, def bool) bool {
	v := strings.TrimSpace(strings.ToLower(os.Getenv(key)))
	if v == "" {
		return def
	}
	return v == "1" || v == "true" || v == "yes" || v == "on"
}

// HoldTTL returns the duration tickets stay on hold before expiring.
func (c Config) HoldTTL() time.Duration {
	if c.HoldTTLMinutes <= 0 {
		return 15 * time.Minute
	}
	return time.Duration(c.HoldTTLMinutes) * time.Minute
}

// EventServiceTimeout returns the timeout used for outbound event service checks.
func (c Config) EventServiceTimeout() time.Duration {
	if c.EventServiceTimeoutMs <= 0 {
		return 3 * time.Second
	}
	return time.Duration(c.EventServiceTimeoutMs) * time.Millisecond
}

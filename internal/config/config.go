package config

import (
	"fmt"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

// Config holds runtime configuration loaded from the environment.
type Config struct {
	Port          string
	MongoURI      string
	MongoDatabase string
	Environment   string
	ServiceName   string
	LogLevel      string
}

// Load reads configuration from environment variables. If a .env file exists, it is loaded first (best-effort).
func Load() (Config, error) {
	_ = godotenv.Load()

	port := getEnv("PORT", "8080")
	if _, err := strconv.Atoi(port); err != nil {
		return Config{}, fmt.Errorf("PORT must be numeric: %w", err)
	}

	cfg := Config{
		Port:          port,
		MongoURI:      os.Getenv("MONGODB_URI"),
		MongoDatabase: getEnv("MONGODB_DATABASE", "phoenix_inventory"),
		Environment:   getEnv("ENVIRONMENT", "development"),
		ServiceName:   getEnv("SERVICE_NAME", "ticket-inventory-service"),
		LogLevel:      getEnv("LOG_LEVEL", "info"),
	}

	if cfg.MongoURI == "" {
		return Config{}, fmt.Errorf("MONGODB_URI is required")
	}

	return cfg, nil
}

func getEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

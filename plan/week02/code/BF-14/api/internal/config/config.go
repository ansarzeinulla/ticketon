// Package config loads the API's runtime settings from the environment.
package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// Config holds everything the API needs to start.
type Config struct {
	Env         string
	Host        string
	Port        int
	DatabaseURL string
	JWTSecret   string
	JWTIssuer   string
	// WebBaseURL is where the web app lives. Account emails link into it.
	WebBaseURL     string
	AccessTokenTTL time.Duration
	BcryptCost     int
}

// devSecret is used only outside production. Production refuses to start
// without an explicit secret, so a deployment can never sign tokens with a
// value that is published in this repository.
const devSecret = "dev-only-insecure-secret-change-me"

// Load reads the configuration from the environment, applying defaults that
// match the docker-compose database.
func Load() (Config, error) {
	cfg := Config{
		Env:       envString("APP_ENV", "development"),
		Host:      envString("API_HOST", "0.0.0.0"),
		JWTIssuer: envString("JWT_ISSUER", "biletflow"),
		DatabaseURL: envString("DATABASE_URL",
			"postgres://biletflow:biletflow_dev_password@localhost:5433/biletflow?sslmode=disable"),
		WebBaseURL: strings.TrimRight(envString("WEB_BASE_URL", "http://localhost:3000"), "/"),
	}

	var err error
	if cfg.Port, err = envInt("APP_PORT", 8080); err != nil {
		return Config{}, err
	}
	if cfg.Port < 1 || cfg.Port > 65535 {
		return Config{}, fmt.Errorf("APP_PORT out of range: %d", cfg.Port)
	}
	if cfg.BcryptCost, err = envInt("BCRYPT_COST", 12); err != nil {
		return Config{}, err
	}
	if cfg.BcryptCost < 4 || cfg.BcryptCost > 31 {
		return Config{}, fmt.Errorf("BCRYPT_COST must be between 4 and 31, got %d", cfg.BcryptCost)
	}
	if cfg.AccessTokenTTL, err = envDuration("ACCESS_TOKEN_TTL", 24*time.Hour); err != nil {
		return Config{}, err
	}
	if cfg.AccessTokenTTL <= 0 {
		return Config{}, fmt.Errorf("ACCESS_TOKEN_TTL must be positive, got %s", cfg.AccessTokenTTL)
	}

	cfg.JWTSecret = os.Getenv("JWT_SECRET")
	if cfg.JWTSecret == "" {
		if cfg.IsProduction() {
			return Config{}, errors.New("JWT_SECRET is required when APP_ENV=production")
		}
		cfg.JWTSecret = devSecret
	}
	return cfg, nil
}

// Addr is the host:port the server listens on.
func (c Config) Addr() string { return fmt.Sprintf("%s:%d", c.Host, c.Port) }

// IsProduction reports whether the API runs in production.
func (c Config) IsProduction() bool { return c.Env == "production" }

func envString(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func envInt(key string, fallback int) (int, error) {
	raw := os.Getenv(key)
	if raw == "" {
		return fallback, nil
	}
	v, err := strconv.Atoi(raw)
	if err != nil {
		return 0, fmt.Errorf("%s must be an integer: %w", key, err)
	}
	return v, nil
}

func envDuration(key string, fallback time.Duration) (time.Duration, error) {
	raw := os.Getenv(key)
	if raw == "" {
		return fallback, nil
	}
	v, err := time.ParseDuration(raw)
	if err != nil {
		return 0, fmt.Errorf("%s must be a duration such as 24h: %w", key, err)
	}
	return v, nil
}

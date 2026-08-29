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
	// WebBaseURL is where the attendee-facing site lives. Campaign QR codes
	// encode a link into it, so it has to be an address a phone can reach -
	// not localhost, when the poster is scanned from a real device.
	WebBaseURL string
	// APIBaseURL is this API's own public address. Notification emails carry
	// ticket download links, and a relative path is no use in an inbox.
	APIBaseURL string
	// UploadDir is where event banners are written. Local disk stands in for
	// object storage in this MVP; the handler and the static route are the
	// only two places that would change.
	UploadDir      string
	AccessTokenTTL time.Duration
	BcryptCost     int
	ReadTimeout    time.Duration
	WriteTimeout   time.Duration
	IdleTimeout    time.Duration
	ShutdownGrace  time.Duration
}

// devSecret is used only when BILETFLOW_ENV is not "production". Production
// refuses to start without an explicit secret, so a deployment can never
// accidentally sign tokens with a value that is published in this repository.
const devSecret = "dev-only-insecure-secret-change-me"

// Load reads the configuration from the environment, applying defaults that
// match the Phase 1 docker-compose setup.
func Load() (Config, error) {
	cfg := Config{
		Env:           envString("BILETFLOW_ENV", "development"),
		Host:          envString("API_HOST", "0.0.0.0"),
		JWTIssuer:     envString("JWT_ISSUER", "biletflow"),
		ReadTimeout:   15 * time.Second,
		WriteTimeout:  30 * time.Second,
		IdleTimeout:   60 * time.Second,
		ShutdownGrace: 10 * time.Second,
	}

	var err error
	if cfg.Port, err = envInt("API_PORT", 8080); err != nil {
		return Config{}, err
	}
	if cfg.BcryptCost, err = envInt("BCRYPT_COST", 12); err != nil {
		return Config{}, err
	}
	if cfg.AccessTokenTTL, err = envDuration("ACCESS_TOKEN_TTL", 24*time.Hour); err != nil {
		return Config{}, err
	}

	cfg.WebBaseURL = strings.TrimRight(
		envString("WEB_BASE_URL", "http://localhost:3000"), "/")

	cfg.APIBaseURL = strings.TrimRight(
		envString("API_BASE_URL", "http://localhost:8080"), "/")

	cfg.UploadDir = envString("UPLOAD_DIR", "./data/uploads")

	cfg.DatabaseURL = envString("DATABASE_URL",
		"postgres://biletflow:biletflow_dev_password@localhost:5433/biletflow?sslmode=disable")

	cfg.JWTSecret = os.Getenv("JWT_SECRET")
	if cfg.JWTSecret == "" {
		if cfg.Env == "production" {
			return Config{}, errors.New("JWT_SECRET is required when BILETFLOW_ENV=production")
		}
		cfg.JWTSecret = devSecret
	}

	if cfg.BcryptCost < 4 || cfg.BcryptCost > 31 {
		return Config{}, fmt.Errorf("BCRYPT_COST must be between 4 and 31, got %d", cfg.BcryptCost)
	}
	if cfg.AccessTokenTTL <= 0 {
		return Config{}, fmt.Errorf("ACCESS_TOKEN_TTL must be positive, got %s", cfg.AccessTokenTTL)
	}

	return cfg, nil
}

// Addr is the listen address for the HTTP server.
func (c Config) Addr() string { return fmt.Sprintf("%s:%d", c.Host, c.Port) }

// IsProduction reports whether the API is running with production guard rails.
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

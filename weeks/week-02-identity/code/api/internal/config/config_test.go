package config

import (
	"strings"
	"testing"
	"time"
)

func TestLoadDefaults(t *testing.T) {
	// Ensure nothing leaks in from the developer's shell.
	for _, key := range []string{
		"BILETFLOW_ENV", "API_HOST", "API_PORT", "DATABASE_URL",
		"JWT_SECRET", "JWT_ISSUER", "ACCESS_TOKEN_TTL", "BCRYPT_COST",
	} {
		t.Setenv(key, "")
	}

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.Env != "development" {
		t.Errorf("Env = %q, want development", cfg.Env)
	}
	if cfg.Port != 8080 {
		t.Errorf("Port = %d, want 8080", cfg.Port)
	}
	if cfg.Addr() != "0.0.0.0:8080" {
		t.Errorf("Addr() = %q, want 0.0.0.0:8080", cfg.Addr())
	}
	if cfg.AccessTokenTTL != 24*time.Hour {
		t.Errorf("AccessTokenTTL = %s, want 24h", cfg.AccessTokenTTL)
	}
	if cfg.BcryptCost != 12 {
		t.Errorf("BcryptCost = %d, want 12", cfg.BcryptCost)
	}
	// The default must point at the Phase 1 container, which listens on 5433.
	if !strings.Contains(cfg.DatabaseURL, ":5433/biletflow") {
		t.Errorf("DatabaseURL = %q, want the docker-compose database on port 5433", cfg.DatabaseURL)
	}
	if cfg.JWTSecret != devSecret {
		t.Error("development should fall back to the built-in secret")
	}
	if cfg.IsProduction() {
		t.Error("IsProduction() = true for the development default")
	}
}

// The dev fallback secret is published in this repository, so production must
// refuse to start without a real one rather than signing tokens with it.
func TestLoadRequiresSecretInProduction(t *testing.T) {
	t.Setenv("BILETFLOW_ENV", "production")
	t.Setenv("JWT_SECRET", "")

	if _, err := Load(); err == nil {
		t.Fatal("Load() succeeded in production without JWT_SECRET")
	}

	t.Setenv("JWT_SECRET", "a-real-production-secret")
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if !cfg.IsProduction() {
		t.Error("IsProduction() = false when BILETFLOW_ENV=production")
	}
	if cfg.JWTSecret == devSecret {
		t.Error("production picked up the development secret")
	}
}

func TestLoadReadsOverrides(t *testing.T) {
	t.Setenv("API_HOST", "127.0.0.1")
	t.Setenv("API_PORT", "9090")
	t.Setenv("JWT_SECRET", "override-secret")
	t.Setenv("JWT_ISSUER", "biletflow-staging")
	t.Setenv("ACCESS_TOKEN_TTL", "15m")
	t.Setenv("BCRYPT_COST", "6")
	t.Setenv("DATABASE_URL", "postgres://user:pass@db:5432/other")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.Addr() != "127.0.0.1:9090" {
		t.Errorf("Addr() = %q, want 127.0.0.1:9090", cfg.Addr())
	}
	if cfg.JWTIssuer != "biletflow-staging" {
		t.Errorf("JWTIssuer = %q, want biletflow-staging", cfg.JWTIssuer)
	}
	if cfg.AccessTokenTTL != 15*time.Minute {
		t.Errorf("AccessTokenTTL = %s, want 15m", cfg.AccessTokenTTL)
	}
	if cfg.BcryptCost != 6 {
		t.Errorf("BcryptCost = %d, want 6", cfg.BcryptCost)
	}
	if cfg.DatabaseURL != "postgres://user:pass@db:5432/other" {
		t.Errorf("DatabaseURL = %q, want the override", cfg.DatabaseURL)
	}
}

func TestLoadRejectsInvalidValues(t *testing.T) {
	tests := []struct {
		name  string
		key   string
		value string
	}{
		{"non-numeric port", "API_PORT", "http"},
		{"non-numeric bcrypt cost", "BCRYPT_COST", "strong"},
		{"bcrypt cost too low", "BCRYPT_COST", "3"},
		{"bcrypt cost too high", "BCRYPT_COST", "32"},
		{"unparseable ttl", "ACCESS_TOKEN_TTL", "a while"},
		{"negative ttl", "ACCESS_TOKEN_TTL", "-1h"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("JWT_SECRET", "test-secret")
			t.Setenv(tt.key, tt.value)

			if _, err := Load(); err == nil {
				t.Errorf("Load() accepted %s=%q", tt.key, tt.value)
			}
		})
	}
}

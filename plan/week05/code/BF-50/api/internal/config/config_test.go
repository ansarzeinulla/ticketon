package config

import "testing"

func TestLoadDefaults(t *testing.T) {
	t.Setenv("APP_ENV", "")
	t.Setenv("APP_PORT", "")
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Port != 8080 || cfg.Env != "development" {
		t.Errorf("defaults = %+v", cfg)
	}
	if cfg.IsProduction() {
		t.Error("development config reports production")
	}
}

func TestLoadRejectsBadPort(t *testing.T) {
	for _, v := range []string{"abc", "0", "70000"} {
		t.Setenv("APP_PORT", v)
		if _, err := Load(); err == nil {
			t.Errorf("APP_PORT=%q accepted", v)
		}
	}
}

func TestLoadReadsTheProcessingFee(t *testing.T) {
	t.Setenv("PROCESSING_FEE_PERCENT", "")
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.ProcessingFeePercent != "3.5" || cfg.ProcessingFeeFixedKZT != "0" {
		t.Errorf("default fee = %q + %q, want 3.5%% + 0", cfg.ProcessingFeePercent, cfg.ProcessingFeeFixedKZT)
	}

	t.Setenv("PROCESSING_FEE_PERCENT", "0")
	if cfg, _ := Load(); cfg.ProcessingFeePercent != "0" {
		t.Errorf("PROCESSING_FEE_PERCENT=0 read as %q", cfg.ProcessingFeePercent)
	}
}

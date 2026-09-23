package config

import (
	"testing"
	"time"
)

func TestDefaults(t *testing.T) {
	t.Setenv("EXTRABADGES_ADDR", "")
	t.Setenv("EXTRABADGES_DB", "")
	t.Setenv("EXTRABADGES_CACHE_TTL", "")
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Addr != ":8080" || cfg.DBPath != "extrabadges.db" || cfg.CacheTTL != 5*time.Minute {
		t.Fatalf("defaults = %+v", cfg)
	}
}

func TestOverrides(t *testing.T) {
	t.Setenv("EXTRABADGES_ADDR", ":9000")
	t.Setenv("EXTRABADGES_DB", "/tmp/x.db")
	t.Setenv("EXTRABADGES_CACHE_TTL", "0s")
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Addr != ":9000" || cfg.DBPath != "/tmp/x.db" || cfg.CacheTTL != 0 {
		t.Fatalf("overrides = %+v", cfg)
	}
}

func TestBadTTL(t *testing.T) {
	t.Setenv("EXTRABADGES_CACHE_TTL", "soon")
	if _, err := Load(); err == nil {
		t.Fatal("expected error")
	}
}

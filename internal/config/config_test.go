package config

import (
	"testing"
	"time"
)

func TestDefaults(t *testing.T) {
	setRequired(t)
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
	setRequired(t)
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
	setRequired(t)
	t.Setenv("EXTRABADGES_CACHE_TTL", "soon")
	if _, err := Load(); err == nil {
		t.Fatal("expected error")
	}
}

func setRequired(t *testing.T) {
	t.Helper()
	t.Setenv("EXTRABADGES_ADMIN_TOKEN", "secret")
	t.Setenv("EXTRABADGES_BOT_TOKEN", "123:bot")
}

func TestRequired(t *testing.T) {
	for _, key := range []string{"EXTRABADGES_ADMIN_TOKEN", "EXTRABADGES_BOT_TOKEN"} {
		t.Run(key, func(t *testing.T) {
			setRequired(t)
			t.Setenv(key, "")
			if _, err := Load(); err == nil {
				t.Fatalf("expected error without %s", key)
			}
		})
	}
}

func TestClientDurations(t *testing.T) {
	setRequired(t)
	t.Setenv("EXTRABADGES_CLIENT_TTL", "")
	t.Setenv("EXTRABADGES_INITDATA_MAX_AGE", "")
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.ClientTTL != 720*time.Hour || cfg.InitDataMaxAge != 24*time.Hour || cfg.BotToken != "123:bot" {
		t.Fatalf("cfg = %+v", cfg)
	}
	for _, bad := range []string{"soon", "0s", "-1h"} {
		t.Setenv("EXTRABADGES_CLIENT_TTL", bad)
		if _, err := Load(); err == nil {
			t.Errorf("CLIENT_TTL=%q: expected error", bad)
		}
	}
}

func TestClientMark(t *testing.T) {
	setRequired(t)
	t.Setenv("EXTRABADGES_MARK_EMOJI_ID", "")
	t.Setenv("EXTRABADGES_MARK_TEXT", "")
	cfg, err := Load()
	if err != nil || cfg.MarkEmojiID != 0 || cfg.MarkText != "exteraless user" {
		t.Fatalf("defaults = %+v, %v", cfg, err)
	}
	t.Setenv("EXTRABADGES_MARK_EMOJI_ID", "5276360730361229322")
	t.Setenv("EXTRABADGES_MARK_TEXT", "hi")
	cfg, err = Load()
	if err != nil || cfg.MarkEmojiID != 5276360730361229322 || cfg.MarkText != "hi" {
		t.Fatalf("overrides = %+v, %v", cfg, err)
	}
	for _, bad := range []string{"x", "0", "-5"} {
		t.Setenv("EXTRABADGES_MARK_EMOJI_ID", bad)
		if _, err := Load(); err == nil {
			t.Errorf("MARK_EMOJI_ID=%q: expected error", bad)
		}
	}
}

func TestTrustedProxies(t *testing.T) {
	setRequired(t)
	t.Setenv("EXTRABADGES_TRUSTED_PROXIES", "172.16.0.0/12, 127.0.0.1/32")
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if len(cfg.TrustedProxies) != 2 || cfg.TrustedProxies[0].String() != "172.16.0.0/12" {
		t.Fatalf("trusted = %v", cfg.TrustedProxies)
	}
	t.Setenv("EXTRABADGES_TRUSTED_PROXIES", "nope")
	if _, err := Load(); err == nil {
		t.Fatal("want error for invalid prefix")
	}
}

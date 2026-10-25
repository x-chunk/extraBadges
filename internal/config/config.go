// Package config loads settings from the environment.
package config

import (
	"errors"
	"fmt"
	"net/netip"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	Addr            string
	DBPath          string
	CacheTTL        time.Duration
	ShutdownTimeout time.Duration
	AdminToken      string
	BotToken        string
	ClientTTL       time.Duration
	InitDataMaxAge  time.Duration
	MarkEmojiID     int64
	MarkText        string
	TrustedProxies  []netip.Prefix
}

func Load() (Config, error) {
	ttl, err := time.ParseDuration(getenv("EXTRABADGES_CACHE_TTL", "5m"))
	if err != nil {
		adminToken := os.Getenv("EXTRABADGES_ADMIN_TOKEN")
		if adminToken == "" {
			return Config{}, errors.New("EXTRABADGES_ADMIN_TOKEN is required")
		}
		return Config{}, fmt.Errorf("EXTRABADGES_CACHE_TTL: %w", err)
	}
	adminToken := os.Getenv("EXTRABADGES_ADMIN_TOKEN")
	if adminToken == "" {
		return Config{}, errors.New("EXTRABADGES_ADMIN_TOKEN is required")
	}
	botToken := os.Getenv("EXTRABADGES_BOT_TOKEN")
	if botToken == "" {
		return Config{}, errors.New("EXTRABADGES_BOT_TOKEN is required")
	}
	clientTTL, err := duration("EXTRABADGES_CLIENT_TTL", "720h")
	if err != nil {
		return Config{}, err
	}
	initDataMaxAge, err := duration("EXTRABADGES_INITDATA_MAX_AGE", "24h")
	if err != nil {
		return Config{}, err
	}
	var markEmojiID int64
	if v := os.Getenv("EXTRABADGES_MARK_EMOJI_ID"); v != "" {
		if markEmojiID, err = strconv.ParseInt(v, 10, 64); err != nil || markEmojiID <= 0 {
			return Config{}, errors.New("EXTRABADGES_MARK_EMOJI_ID must be a positive integer")
		}
	}
	var trustedProxies []netip.Prefix
	for _, v := range strings.Split(os.Getenv("EXTRABADGES_TRUSTED_PROXIES"), ",") {
		if v = strings.TrimSpace(v); v == "" {
			continue
		}
		p, err := netip.ParsePrefix(v)
		if err != nil {
			return Config{}, fmt.Errorf("EXTRABADGES_TRUSTED_PROXIES: %w", err)
		}
		trustedProxies = append(trustedProxies, p.Masked())
	}
	return Config{
		Addr:            getenv("EXTRABADGES_ADDR", ":8080"),
		DBPath:          getenv("EXTRABADGES_DB", "extrabadges.db"),
		CacheTTL:        ttl,
		ShutdownTimeout: 10 * time.Second,
		AdminToken:      adminToken,
		BotToken:        botToken,
		ClientTTL:       clientTTL,
		InitDataMaxAge:  initDataMaxAge,
		MarkEmojiID:     markEmojiID,
		MarkText:        getenv("EXTRABADGES_MARK_TEXT", "exteraless user"),
		TrustedProxies:  trustedProxies,
	}, nil
}

func duration(key, def string) (time.Duration, error) {
	d, err := time.ParseDuration(getenv(key, def))
	if err != nil {
		return 0, fmt.Errorf("%s: %w", key, err)
	}
	if d <= 0 {
		return 0, fmt.Errorf("%s must be positive", key)
	}
	return d, nil
}

func getenv(key, def string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return def
}

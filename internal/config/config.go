// Package config loads settings from the environment.
package config

import (
	"errors"
	"fmt"
	"os"
	"time"
)

type Config struct {
	Addr            string
	DBPath          string
	CacheTTL        time.Duration
	ShutdownTimeout time.Duration
	AdminToken      string
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
	return Config{
		Addr:            getenv("EXTRABADGES_ADDR", ":8080"),
		DBPath:          getenv("EXTRABADGES_DB", "extrabadges.db"),
		CacheTTL:        ttl,
		ShutdownTimeout: 10 * time.Second,
		AdminToken:      adminToken,
	}, nil
}

func getenv(key, def string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return def
}

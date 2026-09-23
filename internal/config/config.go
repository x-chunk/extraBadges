// Package config loads settings from the environment.
package config

import (
	"fmt"
	"os"
	"time"
)

type Config struct {
	Addr            string
	DBPath          string
	CacheTTL        time.Duration
	ShutdownTimeout time.Duration
}

func Load() (Config, error) {
	ttl, err := time.ParseDuration(getenv("EXTRABADGES_CACHE_TTL", "5m"))
	if err != nil {
		return Config{}, fmt.Errorf("EXTRABADGES_CACHE_TTL: %w", err)
	}
	return Config{
		Addr:            getenv("EXTRABADGES_ADDR", ":8080"),
		DBPath:          getenv("EXTRABADGES_DB", "extrabadges.db"),
		CacheTTL:        ttl,
		ShutdownTimeout: 10 * time.Second,
	}, nil
}

func getenv(key, def string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return def
}

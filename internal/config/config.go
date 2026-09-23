// Package config loads settings from the environment.
package config

import (
	"os"
	"time"
)

type Config struct {
	Addr            string
	ShutdownTimeout time.Duration
}

func Load() Config {
	return Config{
		Addr:            getenv("EXTRABADGES_ADDR", ":8080"),
		ShutdownTimeout: 10 * time.Second,
	}
}

func getenv(key, def string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return def
}

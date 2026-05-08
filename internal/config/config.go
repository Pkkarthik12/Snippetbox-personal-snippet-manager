package config

import (
	"errors"
	"os"
	"strconv"
)

type Config struct {
	DatabaseURL  string
	Port         string
	AuthEnabled  bool
	AuthUsername string
	AuthPassword string
	MaxBodySize  int64
}

func Load() (Config, error) {
	authEnabled, err := strconv.ParseBool(getenv("AUTH_ENABLED", "false"))
	if err != nil {
		return Config{}, err
	}

	maxBodySize, err := parseSize(getenv("MAX_BODY_SIZE", "1MB"))
	if err != nil {
		return Config{}, err
	}

	cfg := Config{
		DatabaseURL:  os.Getenv("DATABASE_URL"),
		Port:         getenv("PORT", "8080"),
		AuthEnabled:  authEnabled,
		AuthUsername: getenv("AUTH_USERNAME", "admin"),
		AuthPassword: os.Getenv("AUTH_PASSWORD"),
		MaxBodySize:  maxBodySize,
	}

	if cfg.DatabaseURL == "" {
		return Config{}, errors.New("DATABASE_URL is required")
	}
	if cfg.AuthEnabled && cfg.AuthPassword == "" {
		return Config{}, errors.New("AUTH_PASSWORD is required when AUTH_ENABLED=true")
	}

	return cfg, nil
}

func getenv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func parseSize(value string) (int64, error) {
	units := []struct {
		suffix     string
		multiplier int64
	}{
		{"MB", 1024 * 1024},
		{"KB", 1024},
		{"B", 1},
	}

	for _, unit := range units {
		suffix := unit.suffix
		if len(value) > len(suffix) && value[len(value)-len(suffix):] == suffix {
			n, err := strconv.ParseInt(value[:len(value)-len(suffix)], 10, 64)
			return n * unit.multiplier, err
		}
	}

	return strconv.ParseInt(value, 10, 64)
}

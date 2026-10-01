package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

type Config struct {
	ListenAddr    string
	DatabaseURL   string
	DBMaxConns    int32
	DBMinConns    int32
	RedisAddr     string
	RedisPassword string
	RedisDB       int
	CacheTTL      time.Duration
}

func Load() (Config, error) {
	cfg := Config{
		ListenAddr:    getenv("LISTEN_ADDR", ":8080"),
		DatabaseURL:   os.Getenv("DATABASE_URL"),
		DBMaxConns:    int32(getenvInt("DB_MAX_CONNS", 10)),
		DBMinConns:    int32(getenvInt("DB_MIN_CONNS", 2)),
		RedisAddr:     getenv("REDIS_ADDR", "127.0.0.1:6379"),
		RedisPassword: os.Getenv("REDIS_PASSWORD"),
		RedisDB:       getenvInt("REDIS_DB", 0),
		CacheTTL:      time.Duration(getenvInt("CACHE_TTL_SECONDS", 30)) * time.Second,
	}
	if cfg.DatabaseURL == "" {
		return Config{}, fmt.Errorf("DATABASE_URL is required")
	}
	return cfg, nil
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getenvInt(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return fallback
}

package config

import (
	"errors"
	"os"
)

type Config struct {
	Addr  string
	DBURL string
}

func Load() (*Config, error) {
	addr := os.Getenv("HTTP_ADDR")
	if addr == "" {
		addr = ":8080"
	}
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		return nil, errors.New("DATABASE_URL environment variable not set")
	}

	return &Config{Addr: addr, DBURL: dbURL}, nil
}

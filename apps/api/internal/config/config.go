package config

import (
	"log/slog"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	Env         string
	Port        string
	DatabaseURL string
	JwtSecret   string
}

func Load() *Config {
	if err := godotenv.Load(); err != nil {
		slog.Info(".env file not found, using environment variables")
	}

	return &Config{
		Env:  os.Getenv("ENV"),
		Port: os.Getenv("PORT"),

		DatabaseURL: os.Getenv("DATABASE_URL"),
		JwtSecret:   os.Getenv("JWT_SECRET"),
	}
}

package config

import (
	"log"
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
	err := godotenv.Load()

	if err != nil {
		log.Fatal("error loading .env file", "error", err)
	}

	return &Config{
		Env:  os.Getenv("ENV"),
		Port: os.Getenv("PORT"),

		DatabaseURL: os.Getenv("DATABASE_URL"),
		JwtSecret:   os.Getenv("JWT_SECRET"),
	}
}

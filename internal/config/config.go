package config

import (
	"fmt"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	AppEnv     string
	AppPort    string
	DBHost     string
	DBPort     string
	DBUser     string
	DBPassword string
	DBName     string
	DBParams   string
}

func Load() (Config, error) {
	if err := godotenv.Load(); err != nil {
		if !os.IsNotExist(err) {
			return Config{}, fmt.Errorf("load .env: %w", err)
		}
	}

	cfg := Config{
		AppEnv:     getEnv("App_Env", "development"),
		AppPort:    getEnv("APP_PORT", "8080"),
		DBHost:     os.Getenv("DB_HOST"),
		DBPort:     getEnv("DB_PORT", "3306"),
		DBUser:     os.Getenv("DB_USER"),
		DBPassword: os.Getenv("DB_PASSWORD"),
		DBName:     os.Getenv("DB_NAME"),
		DBParams:   getEnv("DB_PARAMS", "parseTime=true&charset=utf8mb4"),
	}

	if cfg.DBHost == "" || cfg.DBUser == "" || cfg.DBName == "" {
		return Config{}, fmt.Errorf("DB_host, DB_USER and DB_NAME are required")
	}

	if _, err := strconv.Atoi(cfg.AppPort); err != nil {
		return Config{}, fmt.Errorf("APP_PORT must be a number.")
	}

	if _, err := strconv.Atoi(cfg.DBPort); err != nil {
		return Config{}, fmt.Errorf("DB_PORT must be a number")
	}

	return cfg, nil
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}

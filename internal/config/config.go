package config

import "os"

type Config struct {
	Addr        string
	DatabaseURL string
	LogLevel    string
}

func Load() Config {
	return Config{
		Addr:        getEnv("APP_ADDR", ":8080"),
		DatabaseURL: getEnv("DATABASE_URL", "postgres://subscriptions:subscriptions@localhost:5432/subscriptions?sslmode=disable"),
		LogLevel:    getEnv("APP_LOG_LEVEL", "info"),
	}
}

func getEnv(key, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}

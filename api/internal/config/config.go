package config

import (
	"os"
)

// Config holds API server and dependency configuration.
type Config struct {
	HTTPPort string
	DB       DBConfig
	Kafka    KafkaConfig
	JWT      JWTConfig
}

type DBConfig struct {
	DSN string
}

type KafkaConfig struct {
	Brokers []string
}

type JWTConfig struct {
	Secret     string
	ExpireMins int
}

// Load reads configuration from environment.
func Load() *Config {
	dsn := os.Getenv("DB_DSN")
	if dsn == "" {
		dsn = "host=localhost user=fluxmesh password=fluxmesh_secret dbname=fluxmesh port=5432 sslmode=disable"
	}
	brokers := os.Getenv("KAFKA_BROKERS")
	if brokers == "" {
		brokers = "localhost:9092"
	}
	return &Config{
		HTTPPort: getEnv("HTTP_PORT", "8080"),
		DB:       DBConfig{DSN: dsn},
		Kafka:    KafkaConfig{Brokers: []string{brokers}},
		JWT: JWTConfig{
			Secret:     getEnv("JWT_SECRET", "change-me-in-production"),
			ExpireMins: 60,
		},
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

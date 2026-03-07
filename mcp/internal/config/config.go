package config

import "os"

// Config holds control-plane HTTP server configuration.
type Config struct {
	HTTPPort string
	DB       string
	Kafka    KafkaConfig
}

type KafkaConfig struct {
	Brokers []string
}

// Load reads from environment.
func Load() *Config {
	brokers := os.Getenv("KAFKA_BROKERS")
	if brokers == "" {
		brokers = "localhost:9092"
	}
	dsn := os.Getenv("DB_DSN")
	if dsn == "" {
		dsn = "host=localhost user=fluxmesh password=fluxmesh_secret dbname=fluxmesh port=5432 sslmode=disable"
	}
	return &Config{
		HTTPPort: getEnv("HTTP_PORT", "8081"),
		DB:       dsn,
		Kafka:    KafkaConfig{Brokers: []string{brokers}},
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

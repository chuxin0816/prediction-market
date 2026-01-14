package config

import (
	"os"
)

type Config struct {
	Port            string
	DatabaseURL     string
	JWTSecret       string
	EthRPCURL       string
	ContractAddress string
	OperatorKey     string
}

func Load() *Config {
	return &Config{
		Port:            getEnv("PORT", "8080"),
		DatabaseURL:     getEnv("DATABASE_URL", "postgres://localhost:5432/prediction_market?sslmode=disable"),
		JWTSecret:       getEnv("JWT_SECRET", "dev-secret-change-in-production"),
		EthRPCURL:       getEnv("ETH_RPC_URL", ""),
		ContractAddress: getEnv("CONTRACT_ADDRESS", ""),
		OperatorKey:     getEnv("OPERATOR_KEY", ""),
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

package config

import (
	"fmt"
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
	// Support both DATABASE_URL and individual DB_* env vars
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbHost := getEnv("DB_HOST", "localhost")
		dbPort := getEnv("DB_PORT", "5432")
		dbUser := getEnv("DB_USER", "prediction")
		dbPassword := getEnv("DB_PASSWORD", "prediction123")
		dbName := getEnv("DB_NAME", "prediction_market")
		dbURL = fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
			dbHost, dbPort, dbUser, dbPassword, dbName)
	}

	return &Config{
		Port:            getEnv("SERVER_PORT", getEnv("PORT", "8080")),
		DatabaseURL:     dbURL,
		JWTSecret:       getEnv("JWT_SECRET", "dev-secret-change-in-production"),
		EthRPCURL:       getEnv("ETH_RPC_URL", ""),
		ContractAddress: getEnv("CONTRACT_ADDRESS", ""),
		OperatorKey:     getEnv("OPERATOR_PRIVATE_KEY", getEnv("OPERATOR_KEY", "")),
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

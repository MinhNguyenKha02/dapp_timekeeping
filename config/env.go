package config

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	Port                string
	JWTSecret           string
	DBPath              string
	CanisterID          string
	ICPHost             string
	TokenExpiryDuration string
}

var (
	AppConfig Config
)

func LoadConfig() {
	err := godotenv.Load()
	if err != nil {
		log.Printf("Warning: .env file not found, using environment variables")
	}

	AppConfig = Config{
		Port:                getEnvOrDefault("PORT", "3000"),
		JWTSecret:           mustGetEnv("JWT_SECRET"),
		DBPath:              getEnvOrDefault("DB_PATH", "company.db"),
		CanisterID:          mustGetEnv("COMPANY_REGISTRY_CANISTER_ID"),
		ICPHost:             getEnvOrDefault("ICP_HOST", "https://ic0.app"),
		TokenExpiryDuration: getEnvOrDefault("TOKEN_EXPIRY", "24h"),
	}
}

func mustGetEnv(key string) string {
	value := os.Getenv(key)
	if value == "" {
		log.Fatalf("Environment variable %s is required", key)
	}
	return value
}

func getEnvOrDefault(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}

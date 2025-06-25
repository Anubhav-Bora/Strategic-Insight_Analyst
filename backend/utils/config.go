package utils

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

// Automatically load environment variables when this package is imported
func init() {
	err := godotenv.Load(".env")
	if err != nil {
		log.Println("⚠️ No .env file found, using system environment variables")
	}
}

// Safe getter for env variables with fallback default
func GetEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}

func LoadConfig() {
	// ...
}

package utils

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

func LoadConfig() {
	err := godotenv.Load(".env")
	if err != nil {
		log.Println("No .env file found, using environment variables")
	}
}

func GetEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}
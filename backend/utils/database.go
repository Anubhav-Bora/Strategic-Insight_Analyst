package utils

import (
	"context"
	"database/sql"
	"fmt"
	"log"

	"cloud.google.com/go/storage"
	_ "github.com/lib/pq"
	"google.golang.org/api/option"
)

var dbConnection *sql.DB

func InitDB() (*sql.DB, error) {
	connStr := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		GetEnv("DB_HOST", "localhost"),
		GetEnv("DB_PORT", "5432"),
		GetEnv("DB_USER", "postgres"),
		GetEnv("DB_PASSWORD", "postgres"),
		GetEnv("DB_NAME", "insight_analyst"),
	)

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, fmt.Errorf("failed to open database connection: %v", err)
	}

	err = db.Ping()
	if err != nil {
		return nil, fmt.Errorf("failed to ping database: %v", err)
	}

	dbConnection = db
	log.Println("Successfully connected to database")
	return db, nil
}

func GetDBConnection() (*sql.DB, error) {
	if dbConnection == nil {
		return nil, fmt.Errorf("database connection not initialized")
	}
	return dbConnection, nil
}

func InitStorage() (*storage.Client, error) {
	ctx := context.Background()
	client, err := storage.NewClient(ctx, option.WithCredentialsJSON([]byte(GetEnv("GOOGLE_APPLICATION_CREDENTIALS", ""))))
	if err != nil {
		return nil, fmt.Errorf("failed to create storage client: %v", err)
	}
	return client, nil
}
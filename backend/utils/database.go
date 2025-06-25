package utils

import (
	"database/sql"
	"fmt"
	"log"
	"os"

	_ "github.com/lib/pq"
)

var dbConnection *sql.DB // Changed from sql.DB to *sql.DB

func InitDB() (*sql.DB, error) { // Changed return type to *sql.DB
	connStr := os.Getenv("DATABASE_URL")
	if connStr == "" {
		return nil, fmt.Errorf("DATABASE_URL not set in .env")
	}

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, fmt.Errorf("failed to open database connection: %v", err)
	}

	err = db.Ping()
	if err != nil {
		return nil, fmt.Errorf("failed to ping database: %v", err)
	}

	dbConnection = db
	log.Println("✅ Connected to Neon PostgreSQL database")
	return db, nil
}

func GetDBConnection() (*sql.DB, error) {
	if dbConnection == nil {
		return nil, fmt.Errorf("database connection not initialized")
	}
	return dbConnection, nil
}

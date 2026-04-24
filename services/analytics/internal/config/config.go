package config

import (
	"database/sql"
	"log"
	"os"

	_ "github.com/lib/pq"
)

// Config holds application configuration and database connection.
type Config struct {
	ServerAddr string
	DB         *sql.DB
}

// Load reads environment variables and initializes the PostgreSQL connection.
func Load() *Config {
	serverAddr := os.Getenv("SERVER_ADDR")
	if serverAddr == "" {
		serverAddr = ":8081"
	}

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		log.Fatal("DATABASE_URL environment variable is required")
	}

	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
	if err := db.Ping(); err != nil {
		log.Fatalf("Failed to ping database: %v", err)
	}

	return &Config{
		ServerAddr: serverAddr,
		DB:         db,
	}
}

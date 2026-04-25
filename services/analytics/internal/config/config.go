package config

import (
	"database/sql"
	"log"
	"os"
	"strconv"
	"time"

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

	maxOpenConns, _ := strconv.Atoi(os.Getenv("DB_MAX_OPEN_CONNS"))
	if maxOpenConns == 0 {
		maxOpenConns = 25
	}
	maxIdleConns, _ := strconv.Atoi(os.Getenv("DB_MAX_IDLE_CONNS"))
	if maxIdleConns == 0 {
		maxIdleConns = 5
	}
	connMaxLifetime, _ := strconv.Atoi(os.Getenv("DB_CONN_MAX_LIFETIME_MINUTES"))
	if connMaxLifetime == 0 {
		connMaxLifetime = 30
	}
	db.SetMaxOpenConns(maxOpenConns)
	db.SetMaxIdleConns(maxIdleConns)
	db.SetConnMaxLifetime(time.Duration(connMaxLifetime) * time.Minute)

	return &Config{
		ServerAddr: serverAddr,
		DB:         db,
	}
}

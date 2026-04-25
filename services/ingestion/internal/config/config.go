package config

import (
	"context"
	"database/sql"
	"log"
	"os"
	"strconv"
	"time"

	_ "github.com/lib/pq"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/redis/go-redis/v9"
)

// Config holds application configuration and clients.
type Config struct {
	ServerAddr          string
	DB                  *sql.DB
	Minio               *minio.Client
	BucketName          string
	ProcessedBucketName string
	Redis               *redis.Client
}

// Load reads environment variables, initializes the database and MinIO clients,
// and ensures the required MinIO bucket exists.
func Load() *Config {
	serverAddr := os.Getenv("SERVER_ADDR")
	if serverAddr == "" {
		serverAddr = ":8080"
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

	minioEndpoint := os.Getenv("MINIO_ENDPOINT")
	minioAccessKey := os.Getenv("MINIO_ACCESS_KEY")
	minioSecretKey := os.Getenv("MINIO_SECRET_KEY")

	if minioEndpoint == "" || minioAccessKey == "" || minioSecretKey == "" {
		log.Fatal("MINIO_ENDPOINT, MINIO_ACCESS_KEY, and MINIO_SECRET_KEY are required")
	}

	secure := os.Getenv("MINIO_SECURE") == "true"

	minioClient, err := minio.New(minioEndpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(minioAccessKey, minioSecretKey, ""),
		Secure: secure,
	})
	if err != nil {
		log.Fatalf("Failed to create MinIO client: %v", err)
	}

	bucketName := os.Getenv("S3_BUCKET")
	if bucketName == "" {
		bucketName = "chronoscope-sessions"
	}
	exists, err := minioClient.BucketExists(context.Background(), bucketName)
	if err != nil {
		log.Fatalf("Failed to check bucket existence: %v", err)
	}
	if !exists {
		err = minioClient.MakeBucket(context.Background(), bucketName, minio.MakeBucketOptions{})
		if err != nil {
			log.Fatalf("Failed to create bucket: %v", err)
		}
	}

	processedBucketName := os.Getenv("S3_PROCESSED_BUCKET")
	if processedBucketName == "" {
		processedBucketName = "chronoscope-processed"
	}
	exists, err = minioClient.BucketExists(context.Background(), processedBucketName)
	if err != nil {
		log.Fatalf("Failed to check processed bucket existence: %v", err)
	}
	if !exists {
		err = minioClient.MakeBucket(context.Background(), processedBucketName, minio.MakeBucketOptions{})
		if err != nil {
			log.Fatalf("Failed to create processed bucket: %v", err)
		}
	}

	redisURL := os.Getenv("REDIS_URL")
	if redisURL == "" {
		redisURL = "redis://localhost:6379"
	}

	redisOpts, err := redis.ParseURL(redisURL)
	if err != nil {
		log.Fatalf("Failed to parse REDIS_URL: %v", err)
	}
	redisClient := redis.NewClient(redisOpts)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := redisClient.Ping(ctx).Err(); err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}

	return &Config{
		ServerAddr:          serverAddr,
		DB:                  db,
		Minio:               minioClient,
		BucketName:          bucketName,
		ProcessedBucketName: processedBucketName,
		Redis:               redisClient,
	}
}

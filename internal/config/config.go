package config

import (
	"fmt"
	"log"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	Port    string
	GinMode string

	// PostgreSQL Config
	DBHost     string
	DBPort     string
	DBUser     string
	DBPassword string
	DBName     string
	DBSSLMode  string

	// MongoDB Config
	MongoURI      string
	MongoDatabase string

	// JWT Config
	JWTSecret        string
	JWTExpiry        string
	JWTRefreshExpiry string
}

func Load() *Config {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using system environment variables")
	}

	// Build MongoDB URI from individual components
	mongoHost := getEnv("MONGO_HOST", "localhost")
	mongoPort := getEnv("MONGO_PORT", "27017")
	mongoUser := getEnv("MONGO_USER", "admin")
	mongoPass := getEnv("MONGO_PASSWORD", "password")

	mongoURI := fmt.Sprintf("mongodb://%s:%s@%s:%s",
		mongoUser, mongoPass, mongoHost, mongoPort)

	return &Config{
		Port:    getEnv("PORT", "8080"),
		GinMode: getEnv("GIN_MODE", "debug"),

		// PostgreSQL
		DBHost:     getEnv("DB_HOST", "localhost"),
		DBPort:     getEnv("DB_PORT", "5432"),
		DBUser:     getEnv("DB_USER", "postgres"),
		DBPassword: getEnv("DB_PASSWORD", "password"),
		DBName:     getEnv("DB_NAME", "golang_api"),
		DBSSLMode:  getEnv("DB_SSLMODE", "disable"),

		// MongoDB
		MongoURI:      mongoURI,
		MongoDatabase: getEnv("MONGO_DATABASE", "life_tracker"),

		// JWT
		JWTSecret:        getEnv("JWT_SECRET", "default-secret-change-this"),
		JWTExpiry:        getEnv("JWT_EXPIRY", "24h"),
		JWTRefreshExpiry: getEnv("JWT_REFRESH_EXPIRY", "168h"),
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

package config

import (
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
	// Load .env file
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using system environment variables")
	}

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
		MongoURI:      getEnv("MONGO_URI", "mongodb://admin:password@localhost:27017"),
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

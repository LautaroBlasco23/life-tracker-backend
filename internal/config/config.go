package config

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

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

	ImageStoreAddress string

	CORSAllowedOrigins string
}

func Load() *Config {
	env := os.Getenv("ENVIRONMENT")
	if env == "" {
		env = "dev"
	}

	file := ".env"
	if env == "test" {
		file = ".env.test"
	}

	envPath := findEnvFile(file)
	if envPath != "" {
		if err := godotenv.Load(envPath); err != nil {
			log.Printf("Error loading %s: %v\n", envPath, err)
		}
	} else {
		log.Printf("No %s file found, using system environment variables\n", file)
	}

	mongoURI := os.Getenv("MONGO_URI")
	if mongoURI == "" {
		mongoHost := getEnv("MONGO_HOST", "localhost")
		mongoPort := getEnv("MONGO_PORT", "27017")
		mongoUser := getEnv("MONGO_USER", "admin")
		mongoPass := getEnv("MONGO_PASSWORD", "password")
		mongoURI = fmt.Sprintf(
			"mongodb://%s:%s@%s:%s",
			mongoUser, mongoPass, mongoHost, mongoPort,
		)
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
		MongoURI:      mongoURI,
		MongoDatabase: getEnv("MONGO_DATABASE", "life_tracker"),

		// JWT
		JWTSecret:        getEnv("JWT_SECRET", "default-secret-change-this"),
		JWTExpiry:        getEnv("JWT_EXPIRY", "24h"),
		JWTRefreshExpiry: getEnv("JWT_REFRESH_EXPIRY", "168h"),

		ImageStoreAddress: getEnv("IMAGESTORE_ADDRESS", "localhost:50051"),

		CORSAllowedOrigins: getEnv("CORS_ALLOWED_ORIGINS", "*"),
	}
}

func findEnvFile(filename string) string {
	dir, err := os.Getwd()
	if err != nil {
		return ""
	}

	for {
		path := filepath.Join(dir, filename)
		if _, err := os.Stat(path); err == nil {
			return path
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}

	return ""
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

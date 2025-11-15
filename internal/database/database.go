package database

import (
	"fmt"
	"log"
	"time"

	"life-tracker-backend/internal/config"

	"go.mongodb.org/mongo-driver/mongo"
	"gorm.io/gorm"
)

type Connections struct {
	PostgreSQL *gorm.DB
	MongoDB    *mongo.Database
}

func Initialize(cfg *config.Config) (*Connections, error) {
	return initializeWithRetry(cfg, 10, 5*time.Second)
}

func initializeWithRetry(cfg *config.Config, maxRetries int, delay time.Duration) (*Connections, error) {
	var err error
	for i := 0; i < maxRetries; i++ {
		postgresDB, err := InitializePostgreSQL(cfg)
		if err != nil {
			log.Printf("Attempt %d/%d: Failed to connect to PostgreSQL: %v", i+1, maxRetries, err)
			time.Sleep(delay)
			continue
		}

		mongoDB, err := InitializeMongoDB(cfg)
		if err != nil {
			log.Printf("Attempt %d/%d: Failed to connect to MongoDB: %v", i+1, maxRetries, err)
			time.Sleep(delay)
			continue
		}

		if err := CreateMongoIndexes(mongoDB); err != nil {
			log.Println("Failed to create MongoDB indexes:", err)
		}

		return &Connections{
			PostgreSQL: postgresDB,
			MongoDB:    mongoDB,
		}, nil
	}
	return nil, fmt.Errorf("failed to connect after %d attempts: %w", maxRetries, err)
}

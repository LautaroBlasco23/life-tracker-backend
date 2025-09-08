package database

import (
	"life-tracker-backend/internal/config"
	"log"

	"go.mongodb.org/mongo-driver/mongo"
	"gorm.io/gorm"
)

// DatabaseConnections holds both database connections
type DatabaseConnections struct {
	PostgreSQL *gorm.DB
	MongoDB    *mongo.Database
}

// Initialize sets up both PostgreSQL and MongoDB connections
func Initialize(cfg *config.Config) (*DatabaseConnections, error) {
	// Initialize PostgreSQL
	postgresDB, err := InitializePostgreSQL(cfg)
	if err != nil {
		return nil, err
	}

	// Initialize MongoDB
	mongoDB, err := InitializeMongoDB(cfg)
	if err != nil {
		return nil, err
	}

	// Create MongoDB indexes for better performance
	if err := CreateMongoIndexes(mongoDB); err != nil {
		log.Println("Failed to create MongoDB indexes:", err)
	}

	return &DatabaseConnections{
		PostgreSQL: postgresDB,
		MongoDB:    mongoDB,
	}, nil
}

package database

import (
	"life-tracker-backend/internal/config"
	"log"

	"go.mongodb.org/mongo-driver/mongo"
	"gorm.io/gorm"
)

type Connections struct {
	PostgreSQL *gorm.DB
	MongoDB    *mongo.Database
}

func Initialize(cfg *config.Config) (*Connections, error) {
	postgresDB, err := InitializePostgreSQL(cfg)
	if err != nil {
		return nil, err
	}

	mongoDB, err := InitializeMongoDB(cfg)
	if err != nil {
		return nil, err
	}

	if err := CreateMongoIndexes(mongoDB); err != nil {
		log.Println("Failed to create MongoDB indexes:", err)
	}

	return &Connections{
		PostgreSQL: postgresDB,
		MongoDB:    mongoDB,
	}, nil
}

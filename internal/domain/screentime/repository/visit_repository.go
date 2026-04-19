package repository

import (
	"context"
	"fmt"
	"time"

	"life-tracker-backend/internal/domain/screentime/dto"
	"life-tracker-backend/internal/domain/screentime/model"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type VisitRepository interface {
	Create(ctx context.Context, visit *model.WebVisit) error
	FindByUser(ctx context.Context, userID uint, domain string, from, to *time.Time, limit int) ([]model.WebVisit, error)
	StatsByUser(ctx context.Context, userID uint, from, to *time.Time) ([]dto.DomainStat, error)
}

type MongoVisitRepository struct {
	collection *mongo.Collection
}

func NewVisitRepository(db *mongo.Database) VisitRepository {
	return &MongoVisitRepository{collection: db.Collection("web_visits")}
}

func (r *MongoVisitRepository) Create(ctx context.Context, visit *model.WebVisit) error {
	_, err := r.collection.InsertOne(ctx, visit)
	if err != nil {
		return fmt.Errorf("creating web visit: %w", err)
	}
	return nil
}

func (r *MongoVisitRepository) FindByUser(ctx context.Context, userID uint, domain string, from, to *time.Time, limit int) ([]model.WebVisit, error) {
	filter := bson.M{"userId": userID}

	if domain != "" {
		filter["domain"] = domain
	}

	dateFilter := bson.M{}
	if from != nil {
		dateFilter["$gte"] = from
	}
	if to != nil {
		dateFilter["$lte"] = to
	}
	if len(dateFilter) > 0 {
		filter["visitedAt"] = dateFilter
	}

	opts := options.Find().
		SetSort(bson.D{{Key: "visitedAt", Value: -1}}).
		SetLimit(int64(limit))

	cursor, err := r.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, fmt.Errorf("finding web visits: %w", err)
	}
	defer cursor.Close(ctx)

	var visits []model.WebVisit
	if err := cursor.All(ctx, &visits); err != nil {
		return nil, fmt.Errorf("decoding web visits: %w", err)
	}
	return visits, nil
}

func (r *MongoVisitRepository) StatsByUser(ctx context.Context, userID uint, from, to *time.Time) ([]dto.DomainStat, error) {
	matchStage := bson.M{"userId": userID}

	dateFilter := bson.M{}
	if from != nil {
		dateFilter["$gte"] = from
	}
	if to != nil {
		dateFilter["$lte"] = to
	}
	if len(dateFilter) > 0 {
		matchStage["visitedAt"] = dateFilter
	}

	pipeline := mongo.Pipeline{
		{{Key: "$match", Value: matchStage}},
		{{Key: "$group", Value: bson.D{
			{Key: "_id", Value: "$domain"},
			{Key: "count", Value: bson.D{{Key: "$sum", Value: 1}}},
		}}},
		{{Key: "$sort", Value: bson.D{{Key: "count", Value: -1}}}},
	}

	cursor, err := r.collection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, fmt.Errorf("aggregating domain stats: %w", err)
	}
	defer cursor.Close(ctx)

	var stats []dto.DomainStat
	if err := cursor.All(ctx, &stats); err != nil {
		return nil, fmt.Errorf("decoding domain stats: %w", err)
	}
	return stats, nil
}

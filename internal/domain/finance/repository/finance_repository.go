package repository

import (
	"context"
	"errors"
	"log"
	"time"

	"life-tracker-backend/internal/domain/finance/model"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"gorm.io/gorm"
)

var (
	ErrCategoryNotFound     = errors.New("category not found")
	ErrSubcategoryNotFound  = errors.New("subcategory not found or doesn't belong to category")
	ErrTransactionNotFound  = errors.New("transaction not found")
	ErrInvalidTransactionID = errors.New("invalid transaction ID")
)

type TransactionFilter struct {
	UserID          uint
	TransactionType *string
	CategoryID      *uint
	StartDate       *time.Time
	EndDate         *time.Time
}

type AggregationResult struct {
	Type       string
	CategoryID uint
	Total      float64
	Count      int64
}

type MonthlyAggregationResult struct {
	Month int
	Type  string
	Total float64
	Count int64
}

type CategoryRepository interface {
	FindAll(transactionType *string) ([]model.Category, error)
	FindByID(id uint) (*model.Category, error)
	FindByName(name string) (*model.Category, error)
	Create(category *model.Category) error
}

type SubcategoryRepository interface {
	FindAll() ([]model.Subcategory, error)
	FindByID(id uint) (*model.Subcategory, error)
	FindByCategoryID(categoryID uint) ([]model.Subcategory, error)
	FindByIDAndCategoryID(id, categoryID uint) (*model.Subcategory, error)
	Create(subcategory *model.Subcategory) error
}

type TransactionRepository interface {
	Create(ctx context.Context, transaction *model.Transaction) error
	FindByID(ctx context.Context, id primitive.ObjectID, userID uint) (*model.Transaction, error)
	FindByFilter(ctx context.Context, filter TransactionFilter, limit int) ([]model.Transaction, error)
	Update(ctx context.Context, id primitive.ObjectID, updates bson.M) error
	Delete(ctx context.Context, id primitive.ObjectID, userID uint) error
	Aggregate(ctx context.Context, userID uint, startDate, endDate time.Time) ([]AggregationResult, error)
	AggregateMonthly(ctx context.Context, userID uint, year int, loc *time.Location) ([]MonthlyAggregationResult, error)
}

type GormCategoryRepository struct {
	db *gorm.DB
}

func NewCategoryRepository(db *gorm.DB) CategoryRepository {
	return &GormCategoryRepository{db: db}
}

func (r *GormCategoryRepository) FindAll(transactionType *string) ([]model.Category, error) {
	var categories []model.Category
	query := r.db
	if transactionType != nil {
		query = query.Where("type = ?", *transactionType)
	}
	if err := query.Order("name ASC").Find(&categories).Error; err != nil {
		return nil, err
	}
	return categories, nil
}

func (r *GormCategoryRepository) FindByID(id uint) (*model.Category, error) {
	var category model.Category
	if err := r.db.Where("id = ?", id).First(&category).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrCategoryNotFound
		}
		return nil, err
	}
	return &category, nil
}

func (r *GormCategoryRepository) FindByName(name string) (*model.Category, error) {
	var category model.Category
	if err := r.db.Where("name = ?", name).First(&category).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrCategoryNotFound
		}
		return nil, err
	}
	return &category, nil
}

func (r *GormCategoryRepository) Create(category *model.Category) error {
	return r.db.Create(category).Error
}

type GormSubcategoryRepository struct {
	db *gorm.DB
}

func NewSubcategoryRepository(db *gorm.DB) SubcategoryRepository {
	return &GormSubcategoryRepository{db: db}
}

func (r *GormSubcategoryRepository) FindAll() ([]model.Subcategory, error) {
	var subcategories []model.Subcategory
	if err := r.db.Find(&subcategories).Error; err != nil {
		return nil, err
	}
	return subcategories, nil
}

func (r *GormSubcategoryRepository) FindByID(id uint) (*model.Subcategory, error) {
	var subcategory model.Subcategory
	if err := r.db.Where("id = ?", id).First(&subcategory).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrSubcategoryNotFound
		}
		return nil, err
	}
	return &subcategory, nil
}

func (r *GormSubcategoryRepository) FindByCategoryID(categoryID uint) ([]model.Subcategory, error) {
	var subcategories []model.Subcategory
	if err := r.db.Where("category_id = ?", categoryID).Find(&subcategories).Error; err != nil {
		return nil, err
	}
	return subcategories, nil
}

func (r *GormSubcategoryRepository) FindByIDAndCategoryID(id, categoryID uint) (*model.Subcategory, error) {
	var subcategory model.Subcategory
	err := r.db.Where("id = ? AND category_id = ?", id, categoryID).First(&subcategory).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrSubcategoryNotFound
		}
		return nil, err
	}
	return &subcategory, nil
}

func (r *GormSubcategoryRepository) Create(subcategory *model.Subcategory) error {
	return r.db.Create(subcategory).Error
}

type MongoTransactionRepository struct {
	collection *mongo.Collection
}

func NewTransactionRepository(db *mongo.Database) TransactionRepository {
	return &MongoTransactionRepository{
		collection: db.Collection("transactions"),
	}
}

func (r *MongoTransactionRepository) Create(ctx context.Context, transaction *model.Transaction) error {
	_, err := r.collection.InsertOne(ctx, transaction)
	return err
}

func (r *MongoTransactionRepository) FindByID(ctx context.Context, id primitive.ObjectID, userID uint) (*model.Transaction, error) {
	var transaction model.Transaction
	err := r.collection.FindOne(ctx, bson.M{"_id": id, "userId": userID}).Decode(&transaction)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, ErrTransactionNotFound
		}
		return nil, err
	}
	return &transaction, nil
}

func (r *MongoTransactionRepository) FindByFilter(ctx context.Context, filter TransactionFilter, limit int) ([]model.Transaction, error) {
	bsonFilter := bson.M{"userId": filter.UserID}

	if filter.TransactionType != nil {
		bsonFilter["type"] = *filter.TransactionType
	}
	if filter.CategoryID != nil {
		bsonFilter["categoryId"] = *filter.CategoryID
	}

	if filter.StartDate != nil || filter.EndDate != nil {
		dateFilter := bson.M{}
		if filter.StartDate != nil {
			dateFilter["$gte"] = filter.StartDate.UTC()
		}
		if filter.EndDate != nil {
			dateFilter["$lte"] = filter.EndDate.UTC()
		}
		bsonFilter["date"] = dateFilter
	}

	opts := options.Find().SetSort(bson.D{{Key: "date", Value: -1}})
	if limit > 0 {
		opts.SetLimit(int64(limit))
	}

	cursor, err := r.collection.Find(ctx, bsonFilter, opts)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := cursor.Close(ctx); err != nil {
			log.Printf("failed to close cursor: %v", err)
		}
	}()

	var transactions []model.Transaction
	if err := cursor.All(ctx, &transactions); err != nil {
		return nil, err
	}
	return transactions, nil
}

func (r *MongoTransactionRepository) Update(ctx context.Context, id primitive.ObjectID, updates bson.M) error {
	_, err := r.collection.UpdateOne(ctx, bson.M{"_id": id}, bson.M{"$set": updates})
	return err
}

func (r *MongoTransactionRepository) Delete(ctx context.Context, id primitive.ObjectID, userID uint) error {
	result, err := r.collection.DeleteOne(ctx, bson.M{"_id": id, "userId": userID})
	if err != nil {
		return err
	}
	if result.DeletedCount == 0 {
		return ErrTransactionNotFound
	}
	return nil
}

func (r *MongoTransactionRepository) Aggregate(ctx context.Context, userID uint, startDate, endDate time.Time) ([]AggregationResult, error) {
	pipeline := []bson.M{
		{
			"$match": bson.M{
				"userId": userID,
				"date": bson.M{
					"$gte": startDate.UTC(),
					"$lte": endDate.UTC(),
				},
			},
		},
		{
			"$group": bson.M{
				"_id": bson.M{
					"type":       "$type",
					"categoryId": "$categoryId",
				},
				"total": bson.M{"$sum": "$amount"},
				"count": bson.M{"$sum": 1},
			},
		},
	}

	cursor, err := r.collection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := cursor.Close(ctx); err != nil {
			log.Printf("failed to close cursor: %v", err)
		}
	}()

	var results []AggregationResult
	for cursor.Next(ctx) {
		var raw struct {
			ID struct {
				Type       string `bson:"type"`
				CategoryID uint   `bson:"categoryId"`
			} `bson:"_id"`
			Total float64 `bson:"total"`
			Count int64   `bson:"count"`
		}
		if err := cursor.Decode(&raw); err != nil {
			continue
		}
		results = append(results, AggregationResult{
			Type:       raw.ID.Type,
			CategoryID: raw.ID.CategoryID,
			Total:      raw.Total,
			Count:      raw.Count,
		})
	}
	return results, nil
}

func (r *MongoTransactionRepository) AggregateMonthly(ctx context.Context, userID uint, year int, loc *time.Location) ([]MonthlyAggregationResult, error) {
	startDate := time.Date(year, 1, 1, 0, 0, 0, 0, loc)
	endDate := time.Date(year, 12, 31, 23, 59, 59, 999999999, loc)

	pipeline := []bson.M{
		{
			"$match": bson.M{
				"userId": userID,
				"date": bson.M{
					"$gte": startDate.UTC(),
					"$lte": endDate.UTC(),
				},
			},
		},
		{
			"$addFields": bson.M{
				"localDate": bson.M{
					"$dateToParts": bson.M{
						"date":     "$date",
						"timezone": loc.String(),
					},
				},
			},
		},
		{
			"$group": bson.M{
				"_id": bson.M{
					"month": "$localDate.month",
					"type":  "$type",
				},
				"total": bson.M{"$sum": "$amount"},
				"count": bson.M{"$sum": 1},
			},
		},
	}

	cursor, err := r.collection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := cursor.Close(ctx); err != nil {
			log.Printf("failed to close cursor: %v", err)
		}
	}()

	var results []MonthlyAggregationResult
	for cursor.Next(ctx) {
		var raw struct {
			ID struct {
				Type  string `bson:"type"`
				Month int    `bson:"month"`
			} `bson:"_id"`
			Total float64 `bson:"total"`
			Count int64   `bson:"count"`
		}
		if err := cursor.Decode(&raw); err != nil {
			continue
		}
		results = append(results, MonthlyAggregationResult{
			Month: raw.ID.Month,
			Type:  raw.ID.Type,
			Total: raw.Total,
			Count: raw.Count,
		})
	}
	return results, nil
}

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
)

var (
	ErrTransactionNotFound  = errors.New("transaction not found")
	ErrPaymentNotFound      = errors.New("payment not found")
	ErrInvalidTransactionID = errors.New("invalid transaction ID")
	ErrInvalidPaymentID     = errors.New("invalid payment ID")
	ErrPaymentAlreadyExists = errors.New("payment already exists for this period")
)

type TransactionFilter struct {
	TransactionType *string
	Frequency       *string
	Category        *string
	StartDate       *time.Time
	EndDate         *time.Time
	UserID          uint
}

type PaymentFilter struct {
	TransactionID *primitive.ObjectID
	StartDate     *time.Time
	EndDate       *time.Time
	UserID        uint
}

type AggregationResult struct {
	Type     string
	Category string
	Total    float64
	Count    int64
}

type MonthlyAggregationResult struct {
	Type  string
	Month int
	Total float64
	Count int64
}

type PaymentAggregationResult struct {
	TransactionID primitive.ObjectID
	Total         float64
	Count         int64
}

type TransactionRepository interface {
	Create(ctx context.Context, transaction *model.Transaction) error
	FindByID(ctx context.Context, id primitive.ObjectID, userID uint) (*model.Transaction, error)
	FindByFilter(ctx context.Context, filter TransactionFilter, limit int) ([]model.Transaction, error)
	FindFixedTransactions(ctx context.Context, userID uint) ([]model.Transaction, error)
	Update(ctx context.Context, id primitive.ObjectID, updates bson.M) error
	Delete(ctx context.Context, id primitive.ObjectID, userID uint) error
	Aggregate(ctx context.Context, userID uint, startDate, endDate time.Time) ([]AggregationResult, error)
	AggregateMonthly(ctx context.Context, userID uint, year int, loc *time.Location) ([]MonthlyAggregationResult, error)
}

type PaymentRepository interface {
	Create(ctx context.Context, payment *model.Payment) error
	FindByID(ctx context.Context, id primitive.ObjectID, userID uint) (*model.Payment, error)
	FindByTransactionID(ctx context.Context, transactionID primitive.ObjectID, userID uint) ([]model.Payment, error)
	FindByFilter(ctx context.Context, filter PaymentFilter, limit int) ([]model.Payment, error)
	FindCurrentPeriodPayment(ctx context.Context, transactionID primitive.ObjectID, userID uint, frequency model.PaymentFrequency) (*model.Payment, error)
	Delete(ctx context.Context, id primitive.ObjectID, userID uint) error
	DeleteByTransactionID(ctx context.Context, transactionID primitive.ObjectID, userID uint) error
	AggregateByTransaction(ctx context.Context, transactionIDs []primitive.ObjectID, userID uint) ([]PaymentAggregationResult, error)
	AggregateByDateRange(ctx context.Context, userID uint, startDate, endDate time.Time) ([]PaymentAggregationResult, error)
}

// Transaction Repository Implementation

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
	if filter.Frequency != nil {
		bsonFilter["frequency"] = *filter.Frequency
	}
	if filter.Category != nil {
		bsonFilter["category"] = *filter.Category
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

func (r *MongoTransactionRepository) FindFixedTransactions(ctx context.Context, userID uint) ([]model.Transaction, error) {
	filter := bson.M{
		"userId":    userID,
		"frequency": model.TransactionFrequencyFixed,
	}

	opts := options.Find().SetSort(bson.D{{Key: "createdAt", Value: -1}})

	cursor, err := r.collection.Find(ctx, filter, opts)
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
					"type":     "$type",
					"category": "$category",
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
				Type     string `bson:"type"`
				Category string `bson:"category"`
			} `bson:"_id"`
			Total float64 `bson:"total"`
			Count int64   `bson:"count"`
		}
		if err := cursor.Decode(&raw); err != nil {
			continue
		}
		results = append(results, AggregationResult{
			Type:     raw.ID.Type,
			Category: raw.ID.Category,
			Total:    raw.Total,
			Count:    raw.Count,
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

// Payment Repository Implementation

type MongoPaymentRepository struct {
	collection *mongo.Collection
}

func NewPaymentRepository(db *mongo.Database) PaymentRepository {
	return &MongoPaymentRepository{
		collection: db.Collection("payments"),
	}
}

func (r *MongoPaymentRepository) Create(ctx context.Context, payment *model.Payment) error {
	_, err := r.collection.InsertOne(ctx, payment)
	return err
}

func (r *MongoPaymentRepository) FindByID(ctx context.Context, id primitive.ObjectID, userID uint) (*model.Payment, error) {
	var payment model.Payment
	err := r.collection.FindOne(ctx, bson.M{"_id": id, "userId": userID}).Decode(&payment)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, ErrPaymentNotFound
		}
		return nil, err
	}
	return &payment, nil
}

func (r *MongoPaymentRepository) FindByTransactionID(ctx context.Context, transactionID primitive.ObjectID, userID uint) ([]model.Payment, error) {
	filter := bson.M{
		"transactionId": transactionID,
		"userId":        userID,
	}

	opts := options.Find().SetSort(bson.D{{Key: "date", Value: -1}})

	cursor, err := r.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := cursor.Close(ctx); err != nil {
			log.Printf("failed to close cursor: %v", err)
		}
	}()

	var payments []model.Payment
	if err := cursor.All(ctx, &payments); err != nil {
		return nil, err
	}
	return payments, nil
}

func (r *MongoPaymentRepository) FindByFilter(ctx context.Context, filter PaymentFilter, limit int) ([]model.Payment, error) {
	bsonFilter := bson.M{"userId": filter.UserID}

	if filter.TransactionID != nil {
		bsonFilter["transactionId"] = *filter.TransactionID
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

	var payments []model.Payment
	if err := cursor.All(ctx, &payments); err != nil {
		return nil, err
	}
	return payments, nil
}

func (r *MongoPaymentRepository) FindCurrentPeriodPayment(ctx context.Context, transactionID primitive.ObjectID, userID uint, frequency model.PaymentFrequency) (*model.Payment, error) {
	now := time.Now()
	year, month, _ := now.Date()

	var startDate, endDate time.Time

	switch frequency {
	case model.PaymentFrequencyMonthly:
		startDate = time.Date(year, month, 1, 0, 0, 0, 0, time.UTC)
		endDate = startDate.AddDate(0, 1, 0).Add(-time.Nanosecond)
	case model.PaymentFrequencyBimonthly:
		bimonthlyPeriod := (int(month) - 1) / 2
		startMonth := time.Month(bimonthlyPeriod*2 + 1)
		startDate = time.Date(year, startMonth, 1, 0, 0, 0, 0, time.UTC)
		endDate = startDate.AddDate(0, 2, 0).Add(-time.Nanosecond)
	case model.PaymentFrequencyYearly:
		startDate = time.Date(year, 1, 1, 0, 0, 0, 0, time.UTC)
		endDate = time.Date(year, 12, 31, 23, 59, 59, 999999999, time.UTC)
	default:
		startDate = time.Date(year, month, 1, 0, 0, 0, 0, time.UTC)
		endDate = startDate.AddDate(0, 1, 0).Add(-time.Nanosecond)
	}

	filter := bson.M{
		"transactionId": transactionID,
		"userId":        userID,
		"date": bson.M{
			"$gte": startDate,
			"$lte": endDate,
		},
	}

	var payment model.Payment
	err := r.collection.FindOne(ctx, filter).Decode(&payment)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, nil
		}
		return nil, err
	}
	return &payment, nil
}

func (r *MongoPaymentRepository) Delete(ctx context.Context, id primitive.ObjectID, userID uint) error {
	result, err := r.collection.DeleteOne(ctx, bson.M{"_id": id, "userId": userID})
	if err != nil {
		return err
	}
	if result.DeletedCount == 0 {
		return ErrPaymentNotFound
	}
	return nil
}

func (r *MongoPaymentRepository) DeleteByTransactionID(ctx context.Context, transactionID primitive.ObjectID, userID uint) error {
	_, err := r.collection.DeleteMany(ctx, bson.M{"transactionId": transactionID, "userId": userID})
	return err
}

func (r *MongoPaymentRepository) AggregateByTransaction(ctx context.Context, transactionIDs []primitive.ObjectID, userID uint) ([]PaymentAggregationResult, error) {
	if len(transactionIDs) == 0 {
		return nil, nil
	}

	pipeline := []bson.M{
		{
			"$match": bson.M{
				"userId":        userID,
				"transactionId": bson.M{"$in": transactionIDs},
			},
		},
		{
			"$group": bson.M{
				"_id":   "$transactionId",
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

	var results []PaymentAggregationResult
	for cursor.Next(ctx) {
		var raw struct {
			ID    primitive.ObjectID `bson:"_id"`
			Total float64            `bson:"total"`
			Count int64              `bson:"count"`
		}
		if err := cursor.Decode(&raw); err != nil {
			continue
		}
		results = append(results, PaymentAggregationResult{
			TransactionID: raw.ID,
			Total:         raw.Total,
			Count:         raw.Count,
		})
	}
	return results, nil
}

func (r *MongoPaymentRepository) AggregateByDateRange(ctx context.Context, userID uint, startDate, endDate time.Time) ([]PaymentAggregationResult, error) {
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
				"_id":   "$transactionId",
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

	var results []PaymentAggregationResult
	for cursor.Next(ctx) {
		var raw struct {
			ID    primitive.ObjectID `bson:"_id"`
			Total float64            `bson:"total"`
			Count int64              `bson:"count"`
		}
		if err := cursor.Decode(&raw); err != nil {
			continue
		}
		results = append(results, PaymentAggregationResult{
			TransactionID: raw.ID,
			Total:         raw.Total,
			Count:         raw.Count,
		})
	}
	return results, nil
}

package repository

import (
	"context"
	"errors"
	"log"
	"time"

	"life-tracker-backend/internal/domain/activity/dto"
	"life-tracker-backend/internal/domain/activity/model"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"gorm.io/gorm"
)

var (
	ErrActivityNotFound       = errors.New("activity not found")
	ErrActivityRecordNotFound = errors.New("activity record not found")
)

type CompletionMetadata struct {
	MonthlyCompletions map[uint]time.Time
	OneTimeCompletions map[uint]time.Time
	TodayCompletions   map[uint]int
}

type ActivityRepository interface {
	Create(activity *model.Activity) error
	FindByID(id, userID uint) (*model.Activity, error)
	FindByUserID(userID uint, includeInactive bool) ([]model.Activity, error)
	FindActiveByUserID(userID uint) ([]model.Activity, error)
	FindFiltered(userID uint, filter *dto.ActivityFilter) ([]model.Activity, error)
	Update(activity *model.Activity, updates map[string]interface{}) error
	Delete(id, userID uint) error
}

type ActivityRecordRepository interface {
	Create(ctx context.Context, record *model.ActivityRecord) error
	FindByActivityID(ctx context.Context, activityID, userID uint, limit int) ([]model.ActivityRecord, error)
	FindAll(ctx context.Context, activityID, userID uint) ([]model.ActivityRecord, error)
	CountByActivityID(ctx context.Context, activityID, userID uint) (int64, error)
	DeleteLatestForDate(ctx context.Context, activityID, userID uint, startOfDay, endOfDay time.Time) error
	GetCompletionMetadata(ctx context.Context, userID uint, activityIDs []uint, loc *time.Location) (*CompletionMetadata, error)
	GetCompletionsForDate(ctx context.Context, userID uint, date time.Time, loc *time.Location) (map[uint]int, error)
}

type GormActivityRepository struct {
	db *gorm.DB
}

func NewActivityRepository(db *gorm.DB) ActivityRepository {
	return &GormActivityRepository{db: db}
}

func (r *GormActivityRepository) Create(activity *model.Activity) error {
	return r.db.Create(activity).Error
}

func (r *GormActivityRepository) FindByID(id, userID uint) (*model.Activity, error) {
	var activity model.Activity
	err := r.db.Where("id = ? AND user_id = ?", id, userID).First(&activity).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrActivityNotFound
		}
		return nil, err
	}
	return &activity, nil
}

func (r *GormActivityRepository) FindByUserID(userID uint, includeInactive bool) ([]model.Activity, error) {
	var activities []model.Activity
	query := r.db.Where("user_id = ?", userID)
	if !includeInactive {
		query = query.Where("is_active = ?", true)
	}
	if err := query.Find(&activities).Error; err != nil {
		return nil, err
	}
	return activities, nil
}

func (r *GormActivityRepository) FindActiveByUserID(userID uint) ([]model.Activity, error) {
	var activities []model.Activity
	err := r.db.Where("user_id = ? AND is_active = ?", userID, true).Find(&activities).Error
	if err != nil {
		return nil, err
	}
	return activities, nil
}

func (r *GormActivityRepository) FindFiltered(userID uint, filter *dto.ActivityFilter) ([]model.Activity, error) {
	query := r.db.Where("user_id = ? AND is_active = ?", userID, true)

	if filter.Frequency != "" {
		query = query.Where("frequency = ?", filter.Frequency)
	}
	if filter.DayTime != "" {
		query = query.Where("day_time = ?", filter.DayTime)
	}

	var activities []model.Activity
	if err := query.Find(&activities).Error; err != nil {
		return nil, err
	}
	return activities, nil
}

func (r *GormActivityRepository) Update(activity *model.Activity, updates map[string]interface{}) error {
	return r.db.Model(activity).Updates(updates).Error
}

func (r *GormActivityRepository) Delete(id, userID uint) error {
	result := r.db.Where("user_id = ?", userID).Delete(&model.Activity{}, id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrActivityNotFound
	}
	return nil
}

type MongoActivityRecordRepository struct {
	collection *mongo.Collection
}

func NewActivityRecordRepository(db *mongo.Database) ActivityRecordRepository {
	return &MongoActivityRecordRepository{
		collection: db.Collection("activity_records"),
	}
}

func (r *MongoActivityRecordRepository) Create(ctx context.Context, record *model.ActivityRecord) error {
	result, err := r.collection.InsertOne(ctx, record)
	if err != nil {
		return err
	}
	if oid, ok := result.InsertedID.(primitive.ObjectID); ok {
		record.ID = oid
	}
	return nil
}

func (r *MongoActivityRecordRepository) FindByActivityID(ctx context.Context, activityID, userID uint, limit int) ([]model.ActivityRecord, error) {
	filter := bson.M{"activityId": activityID, "userId": userID}
	opts := options.Find().SetSort(bson.D{{Key: "completionDate", Value: -1}})
	if limit > 0 {
		opts.SetLimit(int64(limit))
	}

	cursor, err := r.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := cursor.Close(ctx); err != nil {
			log.Printf("failed to close cursor: %v", err)
		}
	}()

	var records []model.ActivityRecord
	if err := cursor.All(ctx, &records); err != nil {
		return nil, err
	}
	return records, nil
}

func (r *MongoActivityRecordRepository) FindAll(ctx context.Context, activityID, userID uint) ([]model.ActivityRecord, error) {
	return r.FindByActivityID(ctx, activityID, userID, 0)
}

func (r *MongoActivityRecordRepository) CountByActivityID(ctx context.Context, activityID, userID uint) (int64, error) {
	filter := bson.M{"activityId": activityID, "userId": userID}
	return r.collection.CountDocuments(ctx, filter)
}

func (r *MongoActivityRecordRepository) DeleteLatestForDate(ctx context.Context, activityID, userID uint, startOfDay, endOfDay time.Time) error {
	filter := bson.M{
		"activityId": activityID,
		"userId":     userID,
		"completionDate": bson.M{
			"$gte": startOfDay.UTC(),
			"$lt":  endOfDay.UTC(),
		},
	}

	opts := options.FindOne().SetSort(bson.D{{Key: "completionDate", Value: -1}})
	var record model.ActivityRecord
	err := r.collection.FindOne(ctx, filter, opts).Decode(&record)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return ErrActivityRecordNotFound
		}
		return err
	}

	_, err = r.collection.DeleteOne(ctx, bson.M{"_id": record.ID})
	return err
}

func (r *MongoActivityRecordRepository) GetCompletionMetadata(ctx context.Context, userID uint, activityIDs []uint, loc *time.Location) (*CompletionMetadata, error) {
	now := time.Now().In(loc)
	startOfDay := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc)
	endOfDay := startOfDay.Add(24 * time.Hour)
	startOfMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, loc)

	pipeline := []bson.M{
		{
			"$match": bson.M{
				"userId":     userID,
				"activityId": bson.M{"$in": activityIDs},
			},
		},
		{"$sort": bson.M{"completionDate": -1}},
		{
			"$group": bson.M{
				"_id":              "$activityId",
				"latestCompletion": bson.M{"$first": "$completionDate"},
				"monthlyCompletions": bson.M{
					"$push": bson.M{
						"$cond": bson.A{
							bson.M{"$gte": bson.A{"$completionDate", startOfMonth.UTC()}},
							"$completionDate",
							nil,
						},
					},
				},
				"todayCount": bson.M{
					"$sum": bson.M{
						"$cond": bson.A{
							bson.M{
								"$and": bson.A{
									bson.M{"$gte": bson.A{"$completionDate", startOfDay.UTC()}},
									bson.M{"$lt": bson.A{"$completionDate", endOfDay.UTC()}},
								},
							},
							1, 0,
						},
					},
				},
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

	metadata := &CompletionMetadata{
		MonthlyCompletions: make(map[uint]time.Time),
		OneTimeCompletions: make(map[uint]time.Time),
		TodayCompletions:   make(map[uint]int),
	}

	for cursor.Next(ctx) {
		var result struct {
			LatestCompletion     time.Time   `bson:"latestCompletion"`
			MonthlyCompletions   []time.Time `bson:"monthlyCompletions"`
			ID                   uint        `bson:"_id"`
			TodayCount           int         `bson:"todayCount"`
		}
		if err := cursor.Decode(&result); err != nil {
			continue
		}

		metadata.OneTimeCompletions[result.ID] = result.LatestCompletion
		// Check if any monthly completion is valid (non-zero time)
		for _, mc := range result.MonthlyCompletions {
			if !mc.IsZero() {
				metadata.MonthlyCompletions[result.ID] = mc
				break
			}
		}
		metadata.TodayCompletions[result.ID] = result.TodayCount
	}

	return metadata, nil
}

func (r *MongoActivityRecordRepository) GetCompletionsForDate(ctx context.Context, userID uint, date time.Time, loc *time.Location) (map[uint]int, error) {
	dateInLoc := date.In(loc)
	startOfDay := time.Date(dateInLoc.Year(), dateInLoc.Month(), dateInLoc.Day(), 0, 0, 0, 0, loc)
	endOfDay := startOfDay.Add(24 * time.Hour)

	pipeline := []bson.M{
		{
			"$match": bson.M{
				"userId": userID,
				"completionDate": bson.M{
					"$gte": startOfDay.UTC(),
					"$lt":  endOfDay.UTC(),
				},
			},
		},
		{
			"$group": bson.M{
				"_id":   "$activityId",
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

	completions := make(map[uint]int)
	for cursor.Next(ctx) {
		var result struct {
			ID    uint `bson:"_id"`
			Count int  `bson:"count"`
		}
		if err := cursor.Decode(&result); err != nil {
			continue
		}
		completions[result.ID] = result.Count
	}

	return completions, nil
}

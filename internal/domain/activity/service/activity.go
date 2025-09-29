package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"life-tracker-backend/internal/domain/activity/dto"
	"life-tracker-backend/internal/domain/activity/model"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"gorm.io/gorm"
)

type ActivityService struct {
	db      *gorm.DB
	mongoDB *mongo.Database
}

func NewActivityService(db *gorm.DB, mongoDB *mongo.Database) *ActivityService {
	return &ActivityService{
		db:      db,
		mongoDB: mongoDB,
	}
}

func (s *ActivityService) CreateActivity(userID uint, req *dto.CreateActivityRequest) (*dto.ActivityResponse, error) {
	// Validate day frequency for weekly activities
	if req.Frequency == "weekly" && req.DayFrequency != "" {
		if err := s.validateDayFrequency(req.DayFrequency); err != nil {
			return nil, err
		}
	}

	activity := model.Activity{
		UserID:           userID,
		Title:            req.Title,
		Description:      req.Description,
		CompletionAmount: req.CompletionAmount,
		Frequency:        model.Frequency(req.Frequency),
		DayFrequency:     req.DayFrequency,
		DayTime:          model.DayTime(req.DayTime),
		IsActive:         true,
	}

	if err := s.db.Create(&activity).Error; err != nil {
		return nil, errors.New("failed to create activity")
	}

	return activity.ToResponse(), nil
}

func (s *ActivityService) GetUserActivities(userID uint, includeInactive bool) ([]*dto.ActivityResponse, error) {
	var activities []model.Activity

	query := s.db.Where("user_id = ?", userID)
	if !includeInactive {
		query = query.Where("is_active = ?", true)
	}

	if err := query.Find(&activities).Error; err != nil {
		return nil, errors.New("failed to fetch activities")
	}

	// Get today's completion counts for all activities
	todayCompletions, err := s.getTodayCompletions(userID)
	if err != nil {
		// Log error but don't fail the request
		fmt.Printf("Warning: failed to fetch today's completions: %v\n", err)
		todayCompletions = make(map[uint]int)
	}

	var responses []*dto.ActivityResponse
	for _, activity := range activities {
		completions := todayCompletions[activity.ID]
		responses = append(responses, activity.ToResponseWithCompletions(completions))
	}

	return responses, nil
}

func (s *ActivityService) GetActivity(userID, activityID uint) (*dto.ActivityResponse, error) {
	var activity model.Activity
	if err := s.db.Where("id = ? AND user_id = ?", activityID, userID).First(&activity).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("activity not found")
		}
		return nil, errors.New("failed to fetch activity")
	}

	return activity.ToResponse(), nil
}

func (s *ActivityService) UpdateActivity(userID, activityID uint, req *dto.UpdateActivityRequest) (*dto.ActivityResponse, error) {
	var activity model.Activity
	if err := s.db.Where("id = ? AND user_id = ?", activityID, userID).First(&activity).Error; err != nil {
		return nil, errors.New("activity not found")
	}

	// Build updates map
	updates := make(map[string]interface{})
	if req.Title != nil {
		updates["title"] = *req.Title
	}
	if req.Description != nil {
		updates["description"] = *req.Description
	}
	if req.CompletionAmount != nil {
		updates["completion_amount"] = *req.CompletionAmount
	}
	if req.Frequency != nil {
		// Validate day frequency for weekly activities
		if *req.Frequency == "weekly" && req.DayFrequency != nil && *req.DayFrequency != "" {
			if err := s.validateDayFrequency(*req.DayFrequency); err != nil {
				return nil, err
			}
		}
		updates["frequency"] = *req.Frequency
	}
	if req.DayFrequency != nil {
		updates["day_frequency"] = *req.DayFrequency
	}
	if req.DayTime != nil {
		updates["day_time"] = *req.DayTime // ADD THIS LINE
	}
	if req.IsActive != nil {
		updates["is_active"] = *req.IsActive
	}

	if len(updates) > 0 {
		if err := s.db.Model(&activity).Updates(updates).Error; err != nil {
			return nil, errors.New("failed to update activity")
		}
	}

	// Fetch updated activity
	if err := s.db.Where("id = ? AND user_id = ?", activityID, userID).First(&activity).Error; err != nil {
		return nil, errors.New("failed to fetch updated activity")
	}

	return activity.ToResponse(), nil
}

func (s *ActivityService) DeleteActivity(userID, activityID uint) error {
	result := s.db.Where("user_id = ?", userID).Delete(&model.Activity{}, activityID)
	if result.Error != nil {
		return errors.New("failed to delete activity")
	}
	if result.RowsAffected == 0 {
		return errors.New("activity not found")
	}
	return nil
}

// Activity Records operations (MongoDB)
func (s *ActivityService) RecordActivity(userID, activityID uint, req *dto.RecordActivityRequest) (*dto.ActivityRecordResponse, error) {
	// Verify activity exists and belongs to user
	var activity model.Activity
	if err := s.db.Where("id = ? AND user_id = ?", activityID, userID).First(&activity).Error; err != nil {
		return nil, errors.New("activity not found")
	}

	completionDate := time.Now()
	if !req.CompletionDate.IsZero() {
		completionDate = req.CompletionDate
	}

	record := model.ActivityRecord{
		ID:             primitive.NewObjectID(),
		ActivityID:     activityID,
		UserID:         userID,
		CompletionDate: completionDate,
		Notes:          req.Notes,
		CreatedAt:      time.Now(),
	}

	collection := s.mongoDB.Collection("activity_records")
	_, err := collection.InsertOne(context.Background(), record)
	if err != nil {
		return nil, errors.New("failed to record activity completion")
	}

	return record.ToResponse(), nil
}

func (s *ActivityService) GetActivityRecords(userID, activityID uint, limit int) ([]*dto.ActivityRecordResponse, error) {
	// Verify activity belongs to user
	var activity model.Activity
	if err := s.db.Where("id = ? AND user_id = ?", activityID, userID).First(&activity).Error; err != nil {
		return nil, errors.New("activity not found")
	}

	collection := s.mongoDB.Collection("activity_records")
	filter := bson.M{"activityId": activityID, "userId": userID}

	opts := options.Find().SetSort(bson.D{{Key: "completionDate", Value: -1}})
	if limit > 0 {
		opts.SetLimit(int64(limit))
	}

	cursor, err := collection.Find(context.Background(), filter, opts)
	if err != nil {
		return nil, errors.New("failed to fetch activity records")
	}
	defer cursor.Close(context.Background())

	var records []model.ActivityRecord
	if err := cursor.All(context.Background(), &records); err != nil {
		return nil, errors.New("failed to decode activity records")
	}

	var responses []*dto.ActivityRecordResponse
	for _, record := range records {
		responses = append(responses, record.ToResponse())
	}

	return responses, nil
}

func (s *ActivityService) GetActivityStats(userID, activityID uint) (*dto.ActivityStatsResponse, error) {
	// Verify activity belongs to user
	var activity model.Activity
	if err := s.db.Where("id = ? AND user_id = ?", activityID, userID).First(&activity).Error; err != nil {
		return nil, errors.New("activity not found")
	}

	collection := s.mongoDB.Collection("activity_records")
	filter := bson.M{"activityId": activityID, "userId": userID}

	// Get total completions
	totalCompletions, err := collection.CountDocuments(context.Background(), filter)
	if err != nil {
		return nil, errors.New("failed to count completions")
	}

	// Get recent records
	recentRecords, err := s.GetActivityRecords(userID, activityID, 10)
	if err != nil {
		recentRecords = []*dto.ActivityRecordResponse{}
	}

	// TODO: Implement streak calculations and completion rate
	// This would require more complex MongoDB aggregation queries

	stats := &dto.ActivityStatsResponse{
		ActivityID:       activityID,
		Title:            activity.Title,
		TotalCompletions: totalCompletions,
		CurrentStreak:    0, // TODO: Calculate
		LongestStreak:    0, // TODO: Calculate
		CompletionRate:   0, // TODO: Calculate
		RecentRecords:    recentRecords,
	}

	return stats, nil
}

func (s *ActivityService) RevertLastCompletion(userID, activityID uint) error {
	// Verify activity exists and belongs to user
	var activity model.Activity
	if err := s.db.Where("id = ? AND user_id = ?", activityID, userID).First(&activity).Error; err != nil {
		return errors.New("activity not found")
	}

	collection := s.mongoDB.Collection("activity_records")

	// Get today's date range
	now := time.Now()
	startOfDay := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	endOfDay := startOfDay.Add(24 * time.Hour)

	// Find the most recent completion record for today
	filter := bson.M{
		"activityId": activityID,
		"userId":     userID,
		"completionDate": bson.M{
			"$gte": startOfDay,
			"$lt":  endOfDay,
		},
	}

	// Sort by completion date descending to get the most recent
	opts := options.FindOne().SetSort(bson.D{{Key: "completionDate", Value: -1}})

	var record model.ActivityRecord
	err := collection.FindOne(context.Background(), filter, opts).Decode(&record)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return errors.New("no completion found to revert")
		}
		return errors.New("failed to find completion record")
	}

	// Delete the most recent completion record
	_, err = collection.DeleteOne(context.Background(), bson.M{"_id": record.ID})
	if err != nil {
		return errors.New("failed to revert completion")
	}

	return nil
}

// Helper methods
func (s *ActivityService) validateDayFrequency(dayFrequency string) error {
	var days []string
	if err := json.Unmarshal([]byte(dayFrequency), &days); err != nil {
		return errors.New("invalid day frequency format")
	}

	validDays := map[string]bool{
		"monday": true, "tuesday": true, "wednesday": true, "thursday": true,
		"friday": true, "saturday": true, "sunday": true,
	}

	for _, day := range days {
		if !validDays[day] {
			return fmt.Errorf("invalid day: %s", day)
		}
	}

	return nil
}

// Helper method to get today's completion counts for all user activities
func (s *ActivityService) getTodayCompletions(userID uint) (map[uint]int, error) {
	collection := s.mongoDB.Collection("activity_records")

	// Get start and end of today in local time
	now := time.Now()
	startOfDay := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	endOfDay := startOfDay.Add(24 * time.Hour)

	// MongoDB aggregation pipeline to count completions per activity for today
	pipeline := []bson.M{
		{
			"$match": bson.M{
				"userId": userID,
				"completionDate": bson.M{
					"$gte": startOfDay,
					"$lt":  endOfDay,
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

	cursor, err := collection.Aggregate(context.Background(), pipeline)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(context.Background())

	completions := make(map[uint]int)
	for cursor.Next(context.Background()) {
		var result struct {
			ID    uint `bson:"_id"`
			Count int  `bson:"count"`
		}
		if err := cursor.Decode(&result); err != nil {
			continue // Skip invalid results
		}
		completions[result.ID] = result.Count
	}

	return completions, nil
}

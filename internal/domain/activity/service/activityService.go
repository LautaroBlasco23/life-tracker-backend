package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"life-tracker-backend/internal/domain/activity/dto"
	"life-tracker-backend/internal/domain/activity/model"
	"life-tracker-backend/internal/infrastructure/monitoring"

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

type CompletionMetadata struct {
	MonthlyCompletions map[uint]time.Time
	OneTimeCompletions map[uint]time.Time
	TodayCompletions   map[uint]int
}

func (s *ActivityService) CreateActivity(userID uint, req *dto.CreateActivityRequest) (*dto.ActivityResponse, error) {
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

	monitoring.ActivitiesCreated.WithLabelValues(string(activity.Frequency)).Inc()

	return activity.ToResponse(), nil
}

func (s *ActivityService) GetUserActivities(userID uint, includeInactive bool, loc *time.Location) ([]*dto.ActivityResponse, error) {
	var activities []model.Activity

	query := s.db.Where("user_id = ?", userID)
	if !includeInactive {
		query = query.Where("is_active = ?", true)
	}

	if err := query.Find(&activities).Error; err != nil {
		return nil, errors.New("failed to fetch activities")
	}

	todayCompletions, err := s.getTodayCompletions(userID, loc)
	if err != nil {
		fmt.Printf("Warning: failed to fetch today's completions: %v\n", err)
		todayCompletions = make(map[uint]int)
	}

	responses := make([]*dto.ActivityResponse, len(activities))
	for i := range activities {
		completions := todayCompletions[activities[i].ID]
		responses[i] = activities[i].ToResponseWithCompletions(completions)
	}

	return responses, nil
}

func (s *ActivityService) GetTodayActivities(userID uint, loc *time.Location) ([]*dto.ActivityResponse, error) {
	var activities []model.Activity

	if err := s.db.Where("user_id = ? AND is_active = ?", userID, true).Find(&activities).Error; err != nil {
		return nil, errors.New("failed to fetch activities")
	}

	if len(activities) == 0 {
		return []*dto.ActivityResponse{}, nil
	}

	activityIDs := make([]uint, len(activities))
	for i := range activities {
		activityIDs[i] = activities[i].ID
	}

	metadata, err := s.getCompletionMetadata(userID, activityIDs, loc)
	if err != nil {
		fmt.Printf("Warning: failed to fetch completion metadata: %v\n", err)
		metadata = &CompletionMetadata{
			MonthlyCompletions: make(map[uint]time.Time),
			OneTimeCompletions: make(map[uint]time.Time),
			TodayCompletions:   make(map[uint]int),
		}
	}

	now := time.Now().In(loc)
	var responses []*dto.ActivityResponse

	for i := range activities {
		if s.shouldShowToday(&activities[i], metadata, now) {
			completions := metadata.TodayCompletions[activities[i].ID]
			responses = append(responses, activities[i].ToResponseWithCompletions(completions))
		}
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

	updates := s.buildUpdateMap(req)
	if len(updates) == 0 {
		return activity.ToResponse(), nil
	}

	if err := s.db.Model(&activity).Updates(updates).Error; err != nil {
		return nil, errors.New("failed to update activity")
	}

	if err := s.db.Where("id = ? AND user_id = ?", activityID, userID).First(&activity).Error; err != nil {
		return nil, errors.New("failed to fetch updated activity")
	}

	return activity.ToResponse(), nil
}

func (s *ActivityService) buildUpdateMap(req *dto.UpdateActivityRequest) map[string]interface{} {
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
		if *req.Frequency == "weekly" && req.DayFrequency != nil && *req.DayFrequency != "" {
			if err := s.validateDayFrequency(*req.DayFrequency); err != nil {
				return nil
			}
		}
		updates["frequency"] = *req.Frequency
	}
	if req.DayFrequency != nil {
		updates["day_frequency"] = *req.DayFrequency
	}
	if req.DayTime != nil {
		updates["day_time"] = *req.DayTime
	}
	if req.IsActive != nil {
		updates["is_active"] = *req.IsActive
	}

	return updates
}

func (s *ActivityService) DeleteActivity(userID, activityID uint) error {
	result := s.db.Where("user_id = ?", userID).Delete(&model.Activity{}, activityID)
	if result.Error != nil {
		return errors.New("failed to delete activity")
	}
	if result.RowsAffected == 0 {
		return errors.New("activity not found")
	}

	monitoring.ActivitiesDeleted.Inc()

	return nil
}

func (s *ActivityService) RecordActivity(userID, activityID uint, req *dto.RecordActivityRequest) (*dto.ActivityRecordResponse, error) {
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

	monitoring.ActivityCompletions.WithLabelValues(string(activity.Frequency)).Inc()

	return record.ToResponse(), nil
}

func (s *ActivityService) GetActivityRecords(userID, activityID uint, limit int) ([]*dto.ActivityRecordResponse, error) {
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
	defer func() {
		if err := cursor.Close(context.Background()); err != nil {
			log.Printf("failed to close cursor: %v", err)
		}
	}()

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
	var activity model.Activity
	if err := s.db.Where("id = ? AND user_id = ?", activityID, userID).First(&activity).Error; err != nil {
		return nil, errors.New("activity not found")
	}

	collection := s.mongoDB.Collection("activity_records")
	filter := bson.M{"activityId": activityID, "userId": userID}

	totalCompletions, err := collection.CountDocuments(context.Background(), filter)
	if err != nil {
		return nil, errors.New("failed to count completions")
	}

	recentRecords, err := s.GetActivityRecords(userID, activityID, 10)
	if err != nil {
		recentRecords = []*dto.ActivityRecordResponse{}
	}

	stats := &dto.ActivityStatsResponse{
		ActivityID:       activityID,
		Title:            activity.Title,
		TotalCompletions: totalCompletions,
		CurrentStreak:    0,
		LongestStreak:    0,
		CompletionRate:   0,
		RecentRecords:    recentRecords,
	}

	if stats.CurrentStreak > 0 {
		monitoring.ActivityStreakDays.Observe(float64(stats.CurrentStreak))
	}

	return stats, nil
}

func (s *ActivityService) RevertLastCompletion(userID, activityID uint, targetDate *time.Time, loc *time.Location) error {
	var activity model.Activity
	if err := s.db.Where("id = ? AND user_id = ?", activityID, userID).First(&activity).Error; err != nil {
		return errors.New("activity not found")
	}

	collection := s.mongoDB.Collection("activity_records")

	var startOfDay, endOfDay time.Time
	if targetDate != nil {
		dateInLoc := targetDate.In(loc)
		startOfDay = time.Date(dateInLoc.Year(), dateInLoc.Month(), dateInLoc.Day(), 0, 0, 0, 0, loc)
	} else {
		now := time.Now().In(loc)
		startOfDay = time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc)
	}
	endOfDay = startOfDay.Add(24 * time.Hour)

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
	err := collection.FindOne(context.Background(), filter, opts).Decode(&record)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return errors.New("no completion found to revert")
		}
		return errors.New("failed to find completion record")
	}

	_, err = collection.DeleteOne(context.Background(), bson.M{"_id": record.ID})
	if err != nil {
		return errors.New("failed to revert completion")
	}

	return nil
}

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

func (s *ActivityService) getTodayCompletions(userID uint, loc *time.Location) (map[uint]int, error) {
	return s.getCompletionsForDate(userID, time.Now().In(loc), loc)
}

func (s *ActivityService) getCompletionMetadata(userID uint, activityIDs []uint, loc *time.Location) (*CompletionMetadata, error) {
	collection := s.mongoDB.Collection("activity_records")

	now := time.Now().In(loc)
	startOfDay := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc)
	endOfDay := startOfDay.Add(24 * time.Hour)
	startOfMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, loc)

	startOfDayUTC := startOfDay.UTC()
	endOfDayUTC := endOfDay.UTC()
	startOfMonthUTC := startOfMonth.UTC()

	pipeline := []bson.M{
		{
			"$match": bson.M{
				"userId":     userID,
				"activityId": bson.M{"$in": activityIDs},
			},
		},
		{
			"$sort": bson.M{"completionDate": -1},
		},
		{
			"$group": bson.M{
				"_id":              "$activityId",
				"latestCompletion": bson.M{"$first": "$completionDate"},
				"monthlyCompletion": bson.M{"$first": bson.M{
					"$cond": bson.A{
						bson.M{"$gte": bson.A{"$completionDate", startOfMonthUTC}},
						"$completionDate",
						nil,
					},
				}},
				"todayCount": bson.M{
					"$sum": bson.M{
						"$cond": bson.A{
							bson.M{
								"$and": bson.A{
									bson.M{"$gte": bson.A{"$completionDate", startOfDayUTC}},
									bson.M{"$lt": bson.A{"$completionDate", endOfDayUTC}},
								},
							},
							1,
							0,
						},
					},
				},
			},
		},
	}

	cursor, err := collection.Aggregate(context.Background(), pipeline)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := cursor.Close(context.Background()); err != nil {
			log.Printf("failed to close cursor: %v", err)
		}
	}()

	metadata := &CompletionMetadata{
		MonthlyCompletions: make(map[uint]time.Time),
		OneTimeCompletions: make(map[uint]time.Time),
		TodayCompletions:   make(map[uint]int),
	}

	for cursor.Next(context.Background()) {
		var result struct {
			LatestCompletion  time.Time  `bson:"latestCompletion"`
			MonthlyCompletion *time.Time `bson:"monthlyCompletion"`
			ID                uint       `bson:"_id"`
			TodayCount        int        `bson:"todayCount"`
		}
		if err := cursor.Decode(&result); err != nil {
			continue
		}

		metadata.OneTimeCompletions[result.ID] = result.LatestCompletion
		if result.MonthlyCompletion != nil {
			metadata.MonthlyCompletions[result.ID] = *result.MonthlyCompletion
		}
		metadata.TodayCompletions[result.ID] = result.TodayCount
	}

	return metadata, nil
}

func (s *ActivityService) shouldShowToday(activity *model.Activity, metadata *CompletionMetadata, now time.Time) bool {
	switch activity.Frequency {
	case model.FrequencyDaily:
		return true

	case model.FrequencyWeekly:
		if activity.DayFrequency == "" {
			return false
		}

		var days []string
		if err := json.Unmarshal([]byte(activity.DayFrequency), &days); err != nil {
			return false
		}

		currentWeekday := strings.ToLower(now.Weekday().String())
		for _, day := range days {
			if strings.EqualFold(day, currentWeekday) {
				return true
			}
		}
		return false

	case model.FrequencyMonthly:
		_, completedThisMonth := metadata.MonthlyCompletions[activity.ID]
		return !completedThisMonth

	case model.FrequencyOneTime:
		lastCompletion, hasCompletion := metadata.OneTimeCompletions[activity.ID]
		if !hasCompletion {
			return true
		}

		startOfToday := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
		endOfToday := startOfToday.Add(24 * time.Hour)

		return lastCompletion.After(startOfToday) && lastCompletion.Before(endOfToday)

	default:
		return false
	}
}

func (s *ActivityService) GetUserActivitiesFiltered(userID uint, filter *dto.ActivityFilter, loc *time.Location) ([]*dto.ActivityResponse, error) {
	query := s.db.Where("user_id = ? AND is_active = ?", userID, true)

	if filter.Frequency != "" {
		query = query.Where("frequency = ?", filter.Frequency)
	}

	if filter.DayTime != "" {
		query = query.Where("day_time = ?", filter.DayTime)
	}

	var activities []model.Activity
	if err := query.Find(&activities).Error; err != nil {
		return nil, errors.New("failed to fetch activities")
	}

	var targetDate time.Time
	if filter.ScheduledFor != "" {
		parsed, err := time.ParseInLocation("2006-01-02", filter.ScheduledFor, loc)
		if err != nil {
			return nil, errors.New("invalid date format, use YYYY-MM-DD")
		}
		targetDate = parsed
		activities = s.filterByScheduledDate(activities, targetDate)
	} else {
		targetDate = time.Now().In(loc)
	}

	activityIDs := make([]uint, len(activities))
	for i := range activities {
		activityIDs[i] = activities[i].ID
	}

	completions, err := s.getCompletionsForDate(userID, targetDate, loc)
	if err != nil {
		return nil, err
	}
	if completions == nil {
		completions = make(map[uint]int)
	}

	responses := make([]*dto.ActivityResponse, len(activities))
	for i := range activities {
		responses[i] = activities[i].ToResponseWithCompletions(completions[activities[i].ID])
	}

	return responses, nil
}

func (s *ActivityService) getCompletionsForDate(userID uint, date time.Time, loc *time.Location) (map[uint]int, error) {
	collection := s.mongoDB.Collection("activity_records")

	dateInLoc := date.In(loc)
	startOfDay := time.Date(dateInLoc.Year(), dateInLoc.Month(), dateInLoc.Day(), 0, 0, 0, 0, loc)
	endOfDay := startOfDay.Add(24 * time.Hour)

	startOfDayUTC := startOfDay.UTC()
	endOfDayUTC := endOfDay.UTC()

	pipeline := []bson.M{
		{
			"$match": bson.M{
				"userId": userID,
				"completionDate": bson.M{
					"$gte": startOfDayUTC,
					"$lt":  endOfDayUTC,
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
	defer func() {
		if err := cursor.Close(context.Background()); err != nil {
			log.Printf("failed to close cursor: %v", err)
		}
	}()

	completions := make(map[uint]int)
	for cursor.Next(context.Background()) {
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

func (s *ActivityService) filterByScheduledDate(activities []model.Activity, targetDate time.Time) []model.Activity {
	var filtered []model.Activity
	weekday := strings.ToLower(targetDate.Weekday().String())

	for i := range activities {
		activity := &activities[i]
		switch activity.Frequency {
		case model.FrequencyDaily:
			filtered = append(filtered, *activity)

		case model.FrequencyWeekly:
			if activity.DayFrequency == "" {
				continue
			}
			var days []string
			if err := json.Unmarshal([]byte(activity.DayFrequency), &days); err != nil {
				continue
			}
			for _, day := range days {
				if strings.EqualFold(day, weekday) {
					filtered = append(filtered, *activity)
					break
				}
			}

		case model.FrequencyMonthly, model.FrequencyOneTime:
			filtered = append(filtered, *activity)
		}
	}

	return filtered
}

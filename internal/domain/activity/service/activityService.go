package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"life-tracker-backend/internal/domain/activity/dto"
	"life-tracker-backend/internal/domain/activity/model"
	"life-tracker-backend/internal/domain/activity/repository"
	"life-tracker-backend/internal/infrastructure/monitoring"

	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"gorm.io/gorm"
)

type ActivityService struct {
	activityRepo repository.ActivityRepository
	recordRepo   repository.ActivityRecordRepository
}

func NewActivityService(db *gorm.DB, mongoDB *mongo.Database) *ActivityService {
	return &ActivityService{
		activityRepo: repository.NewActivityRepository(db),
		recordRepo:   repository.NewActivityRecordRepository(mongoDB),
	}
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

	if err := s.activityRepo.Create(&activity); err != nil {
		return nil, errors.New("failed to create activity")
	}

	monitoring.ActivitiesCreated.WithLabelValues(string(activity.Frequency)).Inc()

	return activity.ToResponse(), nil
}

func (s *ActivityService) GetUserActivities(userID uint, includeInactive bool, loc *time.Location) ([]*dto.ActivityResponse, error) {
	activities, err := s.activityRepo.FindByUserID(userID, includeInactive)
	if err != nil {
		return nil, errors.New("failed to fetch activities")
	}

	todayCompletions, err := s.recordRepo.GetCompletionsForDate(context.Background(), userID, time.Now().In(loc), loc)
	if err != nil {
		fmt.Printf("Warning: failed to fetch today's completions: %v\n", err)
		todayCompletions = make(map[uint]int)
	}

	responses := make([]*dto.ActivityResponse, len(activities))
	for i := range activities {
		streak, err := s.CalculateStreak(userID, activities[i].ID, &activities[i], loc)
		if err != nil {
			streak = &dto.StreakInfo{Current: 0, Longest: 0}
		}
		completions := todayCompletions[activities[i].ID]
		responses[i] = activities[i].ToResponseWithCompletions(completions, streak)
	}

	return responses, nil
}

func (s *ActivityService) GetTodayActivities(userID uint, loc *time.Location) ([]*dto.ActivityResponse, error) {
	activities, err := s.activityRepo.FindActiveByUserID(userID)
	if err != nil {
		return nil, errors.New("failed to fetch activities")
	}

	if len(activities) == 0 {
		return []*dto.ActivityResponse{}, nil
	}

	activityIDs := make([]uint, len(activities))
	for i := range activities {
		activityIDs[i] = activities[i].ID
	}

	metadata, err := s.recordRepo.GetCompletionMetadata(context.Background(), userID, activityIDs, loc)
	if err != nil {
		fmt.Printf("Warning: failed to fetch completion metadata: %v\n", err)
		metadata = &repository.CompletionMetadata{
			MonthlyCompletions: make(map[uint]time.Time),
			OneTimeCompletions: make(map[uint]time.Time),
			TodayCompletions:   make(map[uint]int),
		}
	}

	now := time.Now().In(loc)
	var responses []*dto.ActivityResponse

	for i := range activities {
		if s.shouldShowToday(&activities[i], metadata, now) {
			streak, err := s.CalculateStreak(userID, activities[i].ID, &activities[i], loc)
			if err != nil {
				streak = &dto.StreakInfo{Current: 0, Longest: 0}
			}
			completions := metadata.TodayCompletions[activities[i].ID]
			responses = append(responses, activities[i].ToResponseWithCompletions(completions, streak))
		}
	}

	return responses, nil
}

func (s *ActivityService) GetActivity(userID, activityID uint) (*dto.ActivityResponse, error) {
	activity, err := s.activityRepo.FindByID(activityID, userID)
	if err != nil {
		if errors.Is(err, repository.ErrActivityNotFound) {
			return nil, errors.New("activity not found")
		}
		return nil, errors.New("failed to fetch activity")
	}

	return activity.ToResponse(), nil
}

func (s *ActivityService) UpdateActivity(userID, activityID uint, req *dto.UpdateActivityRequest) (*dto.ActivityResponse, error) {
	activity, err := s.activityRepo.FindByID(activityID, userID)
	if err != nil {
		return nil, errors.New("activity not found")
	}

	updates := s.buildUpdateMap(req)
	if len(updates) == 0 {
		return activity.ToResponse(), nil
	}

	if err := s.activityRepo.Update(activity, updates); err != nil {
		return nil, errors.New("failed to update activity")
	}

	updatedActivity, err := s.activityRepo.FindByID(activityID, userID)
	if err != nil {
		return nil, errors.New("failed to fetch updated activity")
	}

	return updatedActivity.ToResponse(), nil
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
	if err := s.activityRepo.Delete(activityID, userID); err != nil {
		if errors.Is(err, repository.ErrActivityNotFound) {
			return errors.New("activity not found")
		}
		return errors.New("failed to delete activity")
	}

	monitoring.ActivitiesDeleted.Inc()
	return nil
}

func (s *ActivityService) RecordActivity(userID, activityID uint, req *dto.RecordActivityRequest) (*dto.ActivityRecordResponse, error) {
	activity, err := s.activityRepo.FindByID(activityID, userID)
	if err != nil {
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

	if err := s.recordRepo.Create(context.Background(), &record); err != nil {
		return nil, errors.New("failed to record activity completion")
	}

	monitoring.ActivityCompletions.WithLabelValues(string(activity.Frequency)).Inc()

	return record.ToResponse(), nil
}

func (s *ActivityService) GetActivityRecords(userID, activityID uint, limit int) ([]*dto.ActivityRecordResponse, error) {
	if _, err := s.activityRepo.FindByID(activityID, userID); err != nil {
		return nil, errors.New("activity not found")
	}

	records, err := s.recordRepo.FindByActivityID(context.Background(), activityID, userID, limit)
	if err != nil {
		return nil, errors.New("failed to fetch activity records")
	}

	responses := make([]*dto.ActivityRecordResponse, len(records))
	for i := range records {
		responses[i] = records[i].ToResponse()
	}

	return responses, nil
}

func (s *ActivityService) GetActivityStats(userID, activityID uint, loc *time.Location) (*dto.ActivityStatsResponse, error) {
	activity, err := s.activityRepo.FindByID(activityID, userID)
	if err != nil {
		return nil, errors.New("activity not found")
	}

	totalCompletions, err := s.recordRepo.CountByActivityID(context.Background(), activityID, userID)
	if err != nil {
		return nil, errors.New("failed to count completions")
	}

	recentRecords, err := s.GetActivityRecords(userID, activityID, 10)
	if err != nil {
		recentRecords = []*dto.ActivityRecordResponse{}
	}

	streak, err := s.CalculateStreak(userID, activityID, activity, loc)
	if err != nil {
		streak = &dto.StreakInfo{Current: 0, Longest: 0}
	}

	stats := &dto.ActivityStatsResponse{
		ActivityID:       activityID,
		Title:            activity.Title,
		TotalCompletions: totalCompletions,
		CurrentStreak:    streak.Current,
		LongestStreak:    streak.Longest,
		CompletionRate:   0,
		RecentRecords:    recentRecords,
	}

	if stats.CurrentStreak > 0 {
		monitoring.ActivityStreakDays.Observe(float64(stats.CurrentStreak))
	}

	return stats, nil
}

func (s *ActivityService) RevertLastCompletion(userID, activityID uint, targetDate *time.Time, loc *time.Location) error {
	if _, err := s.activityRepo.FindByID(activityID, userID); err != nil {
		return errors.New("activity not found")
	}

	var startOfDay, endOfDay time.Time
	if targetDate != nil {
		dateInLoc := targetDate.In(loc)
		startOfDay = time.Date(dateInLoc.Year(), dateInLoc.Month(), dateInLoc.Day(), 0, 0, 0, 0, loc)
	} else {
		now := time.Now().In(loc)
		startOfDay = time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc)
	}
	endOfDay = startOfDay.Add(24 * time.Hour)

	err := s.recordRepo.DeleteLatestForDate(context.Background(), activityID, userID, startOfDay, endOfDay)
	if err != nil {
		if errors.Is(err, repository.ErrActivityRecordNotFound) {
			return errors.New("no completion found to revert")
		}
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

func (s *ActivityService) shouldShowToday(activity *model.Activity, metadata *repository.CompletionMetadata, now time.Time) bool {
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
	activities, err := s.activityRepo.FindFiltered(userID, filter)
	if err != nil {
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

	completions, err := s.recordRepo.GetCompletionsForDate(context.Background(), userID, targetDate, loc)
	if err != nil {
		return nil, err
	}
	if completions == nil {
		completions = make(map[uint]int)
	}

	responses := make([]*dto.ActivityResponse, len(activities))
	for i := range activities {
		streak, err := s.CalculateStreak(userID, activities[i].ID, &activities[i], loc)
		if err != nil {
			streak = &dto.StreakInfo{Current: 0, Longest: 0}
		}
		responses[i] = activities[i].ToResponseWithCompletions(completions[activities[i].ID], streak)
	}

	return responses, nil
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

func (s *ActivityService) CalculateStreak(userID, activityID uint, activity *model.Activity, loc *time.Location) (*dto.StreakInfo, error) {
	records, err := s.recordRepo.FindAll(context.Background(), activityID, userID)
	if err != nil {
		return nil, err
	}

	if len(records) == 0 {
		return &dto.StreakInfo{Current: 0, Longest: 0}, nil
	}

	switch activity.Frequency {
	case model.FrequencyDaily:
		return s.calculateDailyStreak(records, activity.CompletionAmount, loc), nil
	case model.FrequencyWeekly:
		return s.calculateWeeklyStreak(records, activity, loc), nil
	case model.FrequencyMonthly:
		return s.calculateMonthlyStreak(records, loc), nil
	default:
		return &dto.StreakInfo{Current: 0, Longest: 0}, nil
	}
}

func (s *ActivityService) calculateDailyStreak(records []model.ActivityRecord, requiredAmount int, loc *time.Location) *dto.StreakInfo {
	completionsByDay := make(map[string]int)
	for _, r := range records {
		day := r.CompletionDate.In(loc).Format("2006-01-02")
		completionsByDay[day]++
	}

	today := time.Now().In(loc)
	current := 0
	streakActive := true

	for i := 0; i < 365; i++ {
		checkDate := today.AddDate(0, 0, -i)
		dayKey := checkDate.Format("2006-01-02")
		count := completionsByDay[dayKey]

		if count >= requiredAmount {
			if streakActive {
				current++
			}
		} else {
			if i == 0 {
				continue
			}
			streakActive = false
			break
		}
	}

	longest := s.calculateLongestDailyStreak(completionsByDay, requiredAmount)
	if current > longest {
		longest = current
	}

	return &dto.StreakInfo{Current: current, Longest: longest}
}

func (s *ActivityService) calculateLongestDailyStreak(completionsByDay map[string]int, requiredAmount int) int {
	if len(completionsByDay) == 0 {
		return 0
	}

	var dates []time.Time
	for dayStr := range completionsByDay {
		if t, err := time.Parse("2006-01-02", dayStr); err == nil {
			dates = append(dates, t)
		}
	}

	sort.Slice(dates, func(i, j int) bool { return dates[i].Before(dates[j]) })

	longest, streak := 0, 0
	var prevDate time.Time

	for _, d := range dates {
		dayKey := d.Format("2006-01-02")
		if completionsByDay[dayKey] < requiredAmount {
			streak = 0
			continue
		}

		if prevDate.IsZero() {
			streak = 1
		} else {
			daysDiff := int(d.Sub(prevDate).Hours() / 24)
			if daysDiff == 1 {
				streak++
			} else {
				streak = 1
			}
		}

		if streak > longest {
			longest = streak
		}
		prevDate = d
	}

	return longest
}

func (s *ActivityService) calculateWeeklyStreak(records []model.ActivityRecord, activity *model.Activity, loc *time.Location) *dto.StreakInfo {
	if activity.DayFrequency == "" {
		return &dto.StreakInfo{Current: 0, Longest: 0}
	}

	var scheduledDays []string
	if err := json.Unmarshal([]byte(activity.DayFrequency), &scheduledDays); err != nil {
		return &dto.StreakInfo{Current: 0, Longest: 0}
	}

	completionsByDay := make(map[string]int)
	for _, r := range records {
		day := r.CompletionDate.In(loc).Format("2006-01-02")
		completionsByDay[day]++
	}

	today := time.Now().In(loc)
	current, longest := 0, 0

	for weekOffset := 0; weekOffset < 52; weekOffset++ {
		weekStart := today.AddDate(0, 0, -int(today.Weekday())-7*weekOffset)
		weekComplete := true

		for _, dayName := range scheduledDays {
			dayOffset := dayNameToOffset(dayName)
			targetDate := weekStart.AddDate(0, 0, dayOffset)

			if targetDate.After(today) {
				continue
			}

			dayKey := targetDate.Format("2006-01-02")
			if completionsByDay[dayKey] < activity.CompletionAmount {
				weekComplete = false
				break
			}
		}

		if weekComplete {
			current++
			if current > longest {
				longest = current
			}
		} else {
			if weekOffset > 0 {
				break
			}
		}
	}

	return &dto.StreakInfo{Current: current, Longest: longest}
}

func dayNameToOffset(day string) int {
	offsets := map[string]int{
		"sunday": 0, "monday": 1, "tuesday": 2, "wednesday": 3,
		"thursday": 4, "friday": 5, "saturday": 6,
	}
	return offsets[strings.ToLower(day)]
}

func (s *ActivityService) calculateMonthlyStreak(records []model.ActivityRecord, loc *time.Location) *dto.StreakInfo {
	months := make(map[string]bool)
	for _, r := range records {
		monthKey := r.CompletionDate.In(loc).Format("2006-01")
		months[monthKey] = true
	}

	today := time.Now().In(loc)
	current, longest := 0, 0

	for i := 0; i < 24; i++ {
		checkMonth := today.AddDate(0, -i, 0).Format("2006-01")
		if months[checkMonth] {
			current++
			if current > longest {
				longest = current
			}
		} else {
			if i > 0 {
				break
			}
		}
	}

	return &dto.StreakInfo{Current: current, Longest: longest}
}

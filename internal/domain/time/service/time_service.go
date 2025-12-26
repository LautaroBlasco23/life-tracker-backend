package service

import (
	"errors"
	"time"

	"life-tracker-backend/internal/domain/time/dto"
	"life-tracker-backend/internal/domain/time/model"

	"gorm.io/gorm"
)

type TimeService struct {
	db *gorm.DB
}

func NewTimeService(db *gorm.DB) *TimeService {
	return &TimeService{db: db}
}

func (s *TimeService) CreateRecord(userID uint, req *dto.CreateTimeRecordRequest) (*dto.TimeRecordResponse, error) {
	record := model.TimeRecord{
		UserID:          userID,
		Category:        req.Category,
		Description:     req.Description,
		DurationMinutes: req.DurationMinutes,
	}

	if err := s.db.Create(&record).Error; err != nil {
		return nil, errors.New("failed to create time record")
	}

	return record.ToResponse(), nil
}

func (s *TimeService) GetRecords(userID uint, filter *dto.TimeRecordFilter, loc *time.Location) ([]*dto.TimeRecordResponse, error) {
	var records []model.TimeRecord

	query := s.db.Where("user_id = ?", userID).Order("created_at DESC")

	if filter != nil {
		if filter.Category != "" {
			query = query.Where("category = ?", filter.Category)
		}

		effectiveStartDate, effectiveEndDate := s.calculateDateRange(filter, loc)

		if effectiveStartDate != nil {
			query = query.Where("created_at >= ?", effectiveStartDate.UTC())
		}
		if effectiveEndDate != nil {
			query = query.Where("created_at <= ?", effectiveEndDate.UTC())
		}
	}

	if err := query.Find(&records).Error; err != nil {
		return nil, errors.New("failed to fetch time records")
	}

	responses := make([]*dto.TimeRecordResponse, len(records))
	for i := range records {
		responses[i] = records[i].ToResponse()
	}

	return responses, nil
}

func (s *TimeService) GetRecord(userID, recordID uint) (*dto.TimeRecordResponse, error) {
	var record model.TimeRecord

	if err := s.db.Where("id = ? AND user_id = ?", recordID, userID).First(&record).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("time record not found")
		}
		return nil, errors.New("failed to fetch time record")
	}

	return record.ToResponse(), nil
}

func (s *TimeService) UpdateRecord(userID, recordID uint, req *dto.UpdateTimeRecordRequest) (*dto.TimeRecordResponse, error) {
	var record model.TimeRecord

	if err := s.db.Where("id = ? AND user_id = ?", recordID, userID).First(&record).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("time record not found")
		}
		return nil, errors.New("failed to fetch time record")
	}

	updates := s.buildUpdateMap(req)
	if len(updates) == 0 {
		return record.ToResponse(), nil
	}

	if err := s.db.Model(&record).Updates(updates).Error; err != nil {
		return nil, errors.New("failed to update time record")
	}

	if err := s.db.Where("id = ? AND user_id = ?", recordID, userID).First(&record).Error; err != nil {
		return nil, errors.New("failed to fetch updated time record")
	}

	return record.ToResponse(), nil
}

func (s *TimeService) buildUpdateMap(req *dto.UpdateTimeRecordRequest) map[string]interface{} {
	updates := make(map[string]interface{})

	if req.Category != nil {
		updates["category"] = *req.Category
	}
	if req.Description != nil {
		updates["description"] = *req.Description
	}
	if req.DurationMinutes != nil {
		updates["duration_minutes"] = *req.DurationMinutes
	}

	return updates
}

func (s *TimeService) DeleteRecord(userID, recordID uint) error {
	result := s.db.Where("user_id = ?", userID).Delete(&model.TimeRecord{}, recordID)

	if result.Error != nil {
		return errors.New("failed to delete time record")
	}
	if result.RowsAffected == 0 {
		return errors.New("time record not found")
	}

	return nil
}

func (s *TimeService) GetStats(userID uint) (*dto.TimeStatsResponse, error) {
	var records []model.TimeRecord

	if err := s.db.Where("user_id = ?", userID).Find(&records).Error; err != nil {
		return nil, errors.New("failed to fetch time records for stats")
	}

	totalMinutes := 0
	categoryTotals := make(map[string]int)

	for i := range records {
		totalMinutes += records[i].DurationMinutes
		categoryTotals[records[i].Category] += records[i].DurationMinutes
	}

	var topCategory string
	var topCategoryMins int
	for cat, mins := range categoryTotals {
		if mins > topCategoryMins {
			topCategory = cat
			topCategoryMins = mins
		}
	}

	return &dto.TimeStatsResponse{
		TotalMinutes:    totalMinutes,
		RecordCount:     len(records),
		CategoryTotals:  categoryTotals,
		TopCategory:     topCategory,
		TopCategoryMins: topCategoryMins,
	}, nil
}

func (s *TimeService) calculateDateRange(filter *dto.TimeRecordFilter, loc *time.Location) (*time.Time, *time.Time) {
	if filter.Month != nil && filter.Year != nil {
		start := time.Date(*filter.Year, time.Month(*filter.Month), 1, 0, 0, 0, 0, loc)
		end := start.AddDate(0, 1, 0).Add(-time.Nanosecond)
		return &start, &end
	}

	if filter.Month != nil {
		currentYear := time.Now().In(loc).Year()
		start := time.Date(currentYear, time.Month(*filter.Month), 1, 0, 0, 0, 0, loc)
		end := start.AddDate(0, 1, 0).Add(-time.Nanosecond)
		return &start, &end
	}

	if filter.Year != nil {
		start := time.Date(*filter.Year, 1, 1, 0, 0, 0, 0, loc)
		end := time.Date(*filter.Year, 12, 31, 23, 59, 59, 999999999, loc)
		return &start, &end
	}

	if filter.StartDate != nil || filter.EndDate != nil {
		var effectiveStart, effectiveEnd *time.Time
		if filter.StartDate != nil {
			start := time.Date(filter.StartDate.Year(), filter.StartDate.Month(), filter.StartDate.Day(), 0, 0, 0, 0, loc)
			effectiveStart = &start
		}
		if filter.EndDate != nil {
			end := time.Date(filter.EndDate.Year(), filter.EndDate.Month(), filter.EndDate.Day(), 23, 59, 59, 999999999, loc)
			effectiveEnd = &end
		}
		return effectiveStart, effectiveEnd
	}

	return nil, nil
}

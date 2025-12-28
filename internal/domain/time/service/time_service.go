package service

import (
	"errors"
	"time"

	"life-tracker-backend/internal/domain/time/dto"
	"life-tracker-backend/internal/domain/time/model"
	"life-tracker-backend/internal/domain/time/repository"

	"gorm.io/gorm"
)

type TimeService struct {
	repo repository.TimeRecordRepository
}

func NewTimeService(db *gorm.DB) *TimeService {
	return &TimeService{
		repo: repository.NewTimeRecordRepository(db),
	}
}

func (s *TimeService) CreateRecord(userID uint, req *dto.CreateTimeRecordRequest) (*dto.TimeRecordResponse, error) {
	record := model.TimeRecord{
		UserID:          userID,
		Category:        req.Category,
		Description:     req.Description,
		DurationMinutes: req.DurationMinutes,
	}

	if err := s.repo.Create(&record); err != nil {
		return nil, errors.New("failed to create time record")
	}

	return record.ToResponse(), nil
}

func (s *TimeService) GetRecords(userID uint, filter *dto.TimeRecordFilter, loc *time.Location) ([]*dto.TimeRecordResponse, error) {
	repoFilter := repository.BuildTimeRecordFilter(filter, loc)

	records, err := s.repo.FindByUserID(userID, repoFilter)
	if err != nil {
		return nil, errors.New("failed to fetch time records")
	}

	responses := make([]*dto.TimeRecordResponse, len(records))
	for i := range records {
		responses[i] = records[i].ToResponse()
	}

	return responses, nil
}

func (s *TimeService) GetRecord(userID, recordID uint) (*dto.TimeRecordResponse, error) {
	record, err := s.repo.FindByID(recordID, userID)
	if err != nil {
		if errors.Is(err, repository.ErrTimeRecordNotFound) {
			return nil, errors.New("time record not found")
		}
		return nil, errors.New("failed to fetch time record")
	}

	return record.ToResponse(), nil
}

func (s *TimeService) UpdateRecord(userID, recordID uint, req *dto.UpdateTimeRecordRequest) (*dto.TimeRecordResponse, error) {
	record, err := s.repo.FindByID(recordID, userID)
	if err != nil {
		if errors.Is(err, repository.ErrTimeRecordNotFound) {
			return nil, errors.New("time record not found")
		}
		return nil, errors.New("failed to fetch time record")
	}

	updates := s.buildUpdateMap(req)
	if len(updates) == 0 {
		return record.ToResponse(), nil
	}

	if err := s.repo.Update(record, updates); err != nil {
		return nil, errors.New("failed to update time record")
	}

	updatedRecord, err := s.repo.FindByID(recordID, userID)
	if err != nil {
		return nil, errors.New("failed to fetch updated time record")
	}

	return updatedRecord.ToResponse(), nil
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
	if err := s.repo.Delete(recordID, userID); err != nil {
		if errors.Is(err, repository.ErrTimeRecordNotFound) {
			return errors.New("time record not found")
		}
		return errors.New("failed to delete time record")
	}

	return nil
}

func (s *TimeService) GetStats(userID uint) (*dto.TimeStatsResponse, error) {
	records, err := s.repo.FindAllByUserID(userID)
	if err != nil {
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

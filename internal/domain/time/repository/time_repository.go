package repository

import (
	"errors"
	"time"

	"life-tracker-backend/internal/domain/time/dto"
	"life-tracker-backend/internal/domain/time/model"

	"gorm.io/gorm"
)

var ErrTimeRecordNotFound = errors.New("time record not found")

type TimeRecordFilter struct {
	Category  string
	StartDate *time.Time
	EndDate   *time.Time
}

type TimeRecordRepository interface {
	Create(record *model.TimeRecord) error
	FindByID(id, userID uint) (*model.TimeRecord, error)
	FindByUserID(userID uint, filter *TimeRecordFilter) ([]model.TimeRecord, error)
	Update(record *model.TimeRecord, updates map[string]interface{}) error
	Delete(id, userID uint) error
	FindAllByUserID(userID uint) ([]model.TimeRecord, error)
}

type GormTimeRecordRepository struct {
	db *gorm.DB
}

func NewTimeRecordRepository(db *gorm.DB) TimeRecordRepository {
	return &GormTimeRecordRepository{db: db}
}

func (r *GormTimeRecordRepository) Create(record *model.TimeRecord) error {
	return r.db.Create(record).Error
}

func (r *GormTimeRecordRepository) FindByID(id, userID uint) (*model.TimeRecord, error) {
	var record model.TimeRecord
	err := r.db.Where("id = ? AND user_id = ?", id, userID).First(&record).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrTimeRecordNotFound
		}
		return nil, err
	}
	return &record, nil
}

func (r *GormTimeRecordRepository) FindByUserID(userID uint, filter *TimeRecordFilter) ([]model.TimeRecord, error) {
	var records []model.TimeRecord
	query := r.db.Where("user_id = ?", userID).Order("created_at DESC")

	if filter != nil {
		if filter.Category != "" {
			query = query.Where("category = ?", filter.Category)
		}
		if filter.StartDate != nil {
			query = query.Where("created_at >= ?", filter.StartDate.UTC())
		}
		if filter.EndDate != nil {
			query = query.Where("created_at <= ?", filter.EndDate.UTC())
		}
	}

	if err := query.Find(&records).Error; err != nil {
		return nil, err
	}
	return records, nil
}

func (r *GormTimeRecordRepository) Update(record *model.TimeRecord, updates map[string]interface{}) error {
	return r.db.Model(record).Updates(updates).Error
}

func (r *GormTimeRecordRepository) Delete(id, userID uint) error {
	result := r.db.Where("user_id = ?", userID).Delete(&model.TimeRecord{}, id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrTimeRecordNotFound
	}
	return nil
}

func (r *GormTimeRecordRepository) FindAllByUserID(userID uint) ([]model.TimeRecord, error) {
	var records []model.TimeRecord
	if err := r.db.Where("user_id = ?", userID).Find(&records).Error; err != nil {
		return nil, err
	}
	return records, nil
}

func BuildTimeRecordFilter(dtoFilter *dto.TimeRecordFilter, loc *time.Location) *TimeRecordFilter {
	if dtoFilter == nil {
		return nil
	}

	filter := &TimeRecordFilter{
		Category: dtoFilter.Category,
	}

	startDate, endDate := calculateDateRange(dtoFilter, loc)
	filter.StartDate = startDate
	filter.EndDate = endDate

	return filter
}

func calculateDateRange(filter *dto.TimeRecordFilter, loc *time.Location) (*time.Time, *time.Time) {
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

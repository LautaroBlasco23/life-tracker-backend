package repository

import (
	"errors"

	"life-tracker-backend/internal/domain/routine/model"

	"gorm.io/gorm"
)

var ErrRoutineNotFound = errors.New("routine not found")

type RoutineRepository interface {
	Create(routine *model.Routine) error
	FindByID(id, userID uint) (*model.Routine, error)
	FindByIDPublic(id uint) (*model.Routine, error)
	FindByUserID(userID uint) ([]model.Routine, error)
	FindPublicByUserID(userID uint) ([]model.Routine, error)
	Update(routine *model.Routine, updates map[string]interface{}) error
	Delete(id, userID uint) error
	SetActivities(routineID uint, items []model.RoutineActivity) error
	GetActivities(routineID uint) ([]model.RoutineActivity, error)
}

type GormRoutineRepository struct {
	db *gorm.DB
}

func NewRoutineRepository(db *gorm.DB) RoutineRepository {
	return &GormRoutineRepository{db: db}
}

func (r *GormRoutineRepository) Create(routine *model.Routine) error {
	return r.db.Create(routine).Error
}

func (r *GormRoutineRepository) FindByID(id, userID uint) (*model.Routine, error) {
	var routine model.Routine
	err := r.db.Where("id = ? AND user_id = ?", id, userID).First(&routine).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrRoutineNotFound
		}
		return nil, err
	}
	return &routine, nil
}

func (r *GormRoutineRepository) FindByIDPublic(id uint) (*model.Routine, error) {
	var routine model.Routine
	err := r.db.First(&routine, id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrRoutineNotFound
		}
		return nil, err
	}
	return &routine, nil
}

func (r *GormRoutineRepository) FindByUserID(userID uint) ([]model.Routine, error) {
	var routines []model.Routine
	err := r.db.Where("user_id = ?", userID).Find(&routines).Error
	return routines, err
}

func (r *GormRoutineRepository) FindPublicByUserID(userID uint) ([]model.Routine, error) {
	var routines []model.Routine
	err := r.db.Where("user_id = ? AND privacy_status = ?", userID, "public").Find(&routines).Error
	return routines, err
}

func (r *GormRoutineRepository) Update(routine *model.Routine, updates map[string]interface{}) error {
	return r.db.Model(routine).Updates(updates).Error
}

func (r *GormRoutineRepository) Delete(id, userID uint) error {
	result := r.db.Where("user_id = ?", userID).Delete(&model.Routine{}, id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrRoutineNotFound
	}
	return nil
}

func (r *GormRoutineRepository) SetActivities(routineID uint, items []model.RoutineActivity) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("routine_id = ?", routineID).Delete(&model.RoutineActivity{}).Error; err != nil {
			return err
		}
		if len(items) == 0 {
			return nil
		}
		return tx.Create(&items).Error
	})
}

func (r *GormRoutineRepository) GetActivities(routineID uint) ([]model.RoutineActivity, error) {
	var items []model.RoutineActivity
	err := r.db.Where("routine_id = ?", routineID).Order("position ASC").Find(&items).Error
	return items, err
}

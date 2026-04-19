package repository

import (
	"errors"

	"life-tracker-backend/internal/domain/social/model"

	"gorm.io/gorm"
)

var ErrSubscriptionNotFound = errors.New("subscription not found")

type SubscriptionRepository interface {
	Create(sub *model.Subscription) error
	FindExisting(subscriberUserID uint, sourceType model.SourceType, sourceID uint) (*model.Subscription, error)
	FindBySubscriber(subscriberUserID uint) ([]model.Subscription, error)
	Delete(id, subscriberUserID uint) error
}

type GormSubscriptionRepository struct {
	db *gorm.DB
}

func NewSubscriptionRepository(db *gorm.DB) SubscriptionRepository {
	return &GormSubscriptionRepository{db: db}
}

func (r *GormSubscriptionRepository) Create(sub *model.Subscription) error {
	return r.db.Create(sub).Error
}

func (r *GormSubscriptionRepository) FindExisting(subscriberUserID uint, sourceType model.SourceType, sourceID uint) (*model.Subscription, error) {
	var sub model.Subscription
	err := r.db.Where(
		"subscriber_user_id = ? AND source_type = ? AND source_id = ? AND deleted_at IS NULL",
		subscriberUserID, sourceType, sourceID,
	).First(&sub).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &sub, nil
}

func (r *GormSubscriptionRepository) FindBySubscriber(subscriberUserID uint) ([]model.Subscription, error) {
	var subs []model.Subscription
	err := r.db.Where("subscriber_user_id = ?", subscriberUserID).Find(&subs).Error
	return subs, err
}

func (r *GormSubscriptionRepository) Delete(id, subscriberUserID uint) error {
	result := r.db.Where("subscriber_user_id = ?", subscriberUserID).Delete(&model.Subscription{}, id)
	if result.Error != nil {
		return result.Error
	}
	return nil
}

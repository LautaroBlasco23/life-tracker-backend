package repository

import (
	"errors"
	"time"

	"life-tracker-backend/internal/domain/social/model"

	"gorm.io/gorm"
)

var ErrFollowNotFound = errors.New("follow not found")

type FollowRepository interface {
	Create(follow *model.Follow) error
	FindFollow(followerID, followeeID uint) (*model.Follow, error)
	UpdateStatus(followerID, followeeID uint, status model.FollowStatus, acceptedAt *time.Time) error
	Delete(followerID, followeeID uint) error
	FindPendingForFollowee(followeeID uint) ([]model.Follow, error)
	FindFollowers(followeeID uint, limit, offset int) ([]model.Follow, int64, error)
	FindFollowing(followerID uint, limit, offset int) ([]model.Follow, int64, error)
	CountFollowers(followeeID uint) (int64, error)
	CountFollowing(followerID uint) (int64, error)
}

type GormFollowRepository struct {
	db *gorm.DB
}

func NewFollowRepository(db *gorm.DB) FollowRepository {
	return &GormFollowRepository{db: db}
}

func (r *GormFollowRepository) Create(follow *model.Follow) error {
	return r.db.Create(follow).Error
}

func (r *GormFollowRepository) FindFollow(followerID, followeeID uint) (*model.Follow, error) {
	var follow model.Follow
	err := r.db.Where("follower_id = ? AND followee_id = ?", followerID, followeeID).First(&follow).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrFollowNotFound
		}
		return nil, err
	}
	return &follow, nil
}

func (r *GormFollowRepository) UpdateStatus(followerID, followeeID uint, status model.FollowStatus, acceptedAt *time.Time) error {
	updates := map[string]interface{}{"status": status}
	if acceptedAt != nil {
		updates["accepted_at"] = acceptedAt
	}
	return r.db.Model(&model.Follow{}).
		Where("follower_id = ? AND followee_id = ?", followerID, followeeID).
		Updates(updates).Error
}

func (r *GormFollowRepository) Delete(followerID, followeeID uint) error {
	result := r.db.Where("follower_id = ? AND followee_id = ?", followerID, followeeID).Delete(&model.Follow{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrFollowNotFound
	}
	return nil
}

func (r *GormFollowRepository) FindPendingForFollowee(followeeID uint) ([]model.Follow, error) {
	var follows []model.Follow
	err := r.db.Where("followee_id = ? AND status = ?", followeeID, model.FollowStatusPending).Find(&follows).Error
	return follows, err
}

func (r *GormFollowRepository) FindFollowers(followeeID uint, limit, offset int) ([]model.Follow, int64, error) {
	var follows []model.Follow
	var count int64
	q := r.db.Model(&model.Follow{}).Where("followee_id = ? AND status = ?", followeeID, model.FollowStatusAccepted)
	q.Count(&count)
	err := q.Limit(limit).Offset(offset).Find(&follows).Error
	return follows, count, err
}

func (r *GormFollowRepository) FindFollowing(followerID uint, limit, offset int) ([]model.Follow, int64, error) {
	var follows []model.Follow
	var count int64
	q := r.db.Model(&model.Follow{}).Where("follower_id = ? AND status = ?", followerID, model.FollowStatusAccepted)
	q.Count(&count)
	err := q.Limit(limit).Offset(offset).Find(&follows).Error
	return follows, count, err
}

func (r *GormFollowRepository) CountFollowers(followeeID uint) (int64, error) {
	var count int64
	err := r.db.Model(&model.Follow{}).
		Where("followee_id = ? AND status = ?", followeeID, model.FollowStatusAccepted).
		Count(&count).Error
	return count, err
}

func (r *GormFollowRepository) CountFollowing(followerID uint) (int64, error) {
	var count int64
	err := r.db.Model(&model.Follow{}).
		Where("follower_id = ? AND status = ?", followerID, model.FollowStatusAccepted).
		Count(&count).Error
	return count, err
}

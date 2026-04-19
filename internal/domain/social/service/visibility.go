package service

import (
	"life-tracker-backend/internal/domain/social/model"
	userModel "life-tracker-backend/internal/domain/user/model"

	"gorm.io/gorm"
)

type VisibilityService struct {
	db *gorm.DB
}

func NewVisibilityService(db *gorm.DB) *VisibilityService {
	return &VisibilityService{db: db}
}

// CanViewProfile returns true if viewerID can see ownerID's profile shell and public content.
func (s *VisibilityService) CanViewProfile(viewerID, ownerID uint) (bool, error) {
	if viewerID == ownerID {
		return true, nil
	}

	var owner userModel.User
	if err := s.db.First(&owner, ownerID).Error; err != nil {
		return false, err
	}

	if owner.ProfilePrivacyStatus == "public" {
		return true, nil
	}

	// private profile: viewer must have an accepted follow
	var follow model.Follow
	err := s.db.Where("follower_id = ? AND followee_id = ? AND status = ?",
		viewerID, ownerID, model.FollowStatusAccepted).First(&follow).Error
	if err != nil {
		return false, err
	}
	return true, nil
}

// CanViewContent returns true if viewerID can see a specific piece of content owned by ownerID.
func (s *VisibilityService) CanViewContent(viewerID, ownerID uint, privacy string) (bool, error) {
	if viewerID == ownerID {
		return true, nil
	}
	if privacy == "private" {
		return false, nil
	}
	return s.CanViewProfile(viewerID, ownerID)
}

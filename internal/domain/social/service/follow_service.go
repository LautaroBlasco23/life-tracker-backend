package service

import (
	"errors"
	"time"

	"life-tracker-backend/internal/domain/social/dto"
	"life-tracker-backend/internal/domain/social/model"
	"life-tracker-backend/internal/domain/social/repository"
	userModel "life-tracker-backend/internal/domain/user/model"
	userRepo "life-tracker-backend/internal/domain/user/repository"

	"gorm.io/gorm"
)

type FollowService struct {
	followRepo repository.FollowRepository
	userRepo   userRepo.UserRepository
	visibility *VisibilityService
}

func NewFollowService(db *gorm.DB) *FollowService {
	return &FollowService{
		followRepo: repository.NewFollowRepository(db),
		userRepo:   userRepo.NewUserRepository(db),
		visibility: NewVisibilityService(db),
	}
}

func (s *FollowService) Follow(followerID uint, targetUsername string) (*dto.FollowResponse, error) {
	target, err := s.userRepo.FindByUsername(targetUsername)
	if err != nil {
		if errors.Is(err, userRepo.ErrUserNotFound) {
			return nil, errors.New("user not found")
		}
		return nil, errors.New("failed to find user")
	}

	if followerID == target.ID {
		return nil, errors.New("cannot follow yourself")
	}

	existing, err := s.followRepo.FindFollow(followerID, target.ID)
	if err != nil && !errors.Is(err, repository.ErrFollowNotFound) {
		return nil, err
	}
	if existing != nil {
		return &dto.FollowResponse{Status: string(existing.Status)}, nil
	}

	status := model.FollowStatusAccepted
	if target.ProfilePrivacyStatus == "private" {
		status = model.FollowStatusPending
	}

	now := time.Now()
	follow := &model.Follow{
		FollowerID: followerID,
		FolloweeID: target.ID,
		Status:     status,
		CreatedAt:  now,
	}
	if status == model.FollowStatusAccepted {
		follow.AcceptedAt = &now
	}

	if err := s.followRepo.Create(follow); err != nil {
		return nil, errors.New("failed to create follow")
	}

	return &dto.FollowResponse{Status: string(status)}, nil
}

func (s *FollowService) Unfollow(followerID uint, targetUsername string) error {
	target, err := s.userRepo.FindByUsername(targetUsername)
	if err != nil {
		return errors.New("user not found")
	}
	err = s.followRepo.Delete(followerID, target.ID)
	if errors.Is(err, repository.ErrFollowNotFound) {
		return errors.New("not following this user")
	}
	return err
}

func (s *FollowService) GetPendingRequests(followeeID uint) ([]dto.FollowerItem, error) {
	follows, err := s.followRepo.FindPendingForFollowee(followeeID)
	if err != nil {
		return nil, err
	}
	items := make([]dto.FollowerItem, 0, len(follows))
	for _, f := range follows {
		user, err := s.userRepo.FindByID(f.FollowerID)
		if err != nil {
			continue
		}
		items = append(items, toFollowerItem(user))
	}
	return items, nil
}

func (s *FollowService) AcceptFollow(followeeID, followerID uint) error {
	_, err := s.followRepo.FindFollow(followerID, followeeID)
	if errors.Is(err, repository.ErrFollowNotFound) {
		return errors.New("follow request not found")
	}
	if err != nil {
		return err
	}
	now := time.Now()
	return s.followRepo.UpdateStatus(followerID, followeeID, model.FollowStatusAccepted, &now)
}

func (s *FollowService) RejectFollow(followeeID, followerID uint) error {
	err := s.followRepo.Delete(followerID, followeeID)
	if errors.Is(err, repository.ErrFollowNotFound) {
		return errors.New("follow request not found")
	}
	return err
}

func (s *FollowService) GetFollowers(viewerID uint, targetUsername string, limit, offset int) ([]dto.FollowerItem, int64, error) {
	target, err := s.userRepo.FindByUsername(targetUsername)
	if err != nil {
		return nil, 0, errors.New("user not found")
	}
	canView, err := s.visibility.CanViewProfile(viewerID, target.ID)
	if err != nil || !canView {
		return nil, 0, errors.New("access denied")
	}
	follows, total, err := s.followRepo.FindFollowers(target.ID, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	items := make([]dto.FollowerItem, 0, len(follows))
	for _, f := range follows {
		user, err := s.userRepo.FindByID(f.FollowerID)
		if err != nil {
			continue
		}
		items = append(items, toFollowerItem(user))
	}
	return items, total, nil
}

func (s *FollowService) GetFollowing(viewerID uint, targetUsername string, limit, offset int) ([]dto.FollowerItem, int64, error) {
	target, err := s.userRepo.FindByUsername(targetUsername)
	if err != nil {
		return nil, 0, errors.New("user not found")
	}
	canView, err := s.visibility.CanViewProfile(viewerID, target.ID)
	if err != nil || !canView {
		return nil, 0, errors.New("access denied")
	}
	follows, total, err := s.followRepo.FindFollowing(target.ID, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	items := make([]dto.FollowerItem, 0, len(follows))
	for _, f := range follows {
		user, err := s.userRepo.FindByID(f.FolloweeID)
		if err != nil {
			continue
		}
		items = append(items, toFollowerItem(user))
	}
	return items, total, nil
}

func toFollowerItem(user *userModel.User) dto.FollowerItem {
	return dto.FollowerItem{
		UserID:    user.ID,
		FirstName: user.FirstName,
		LastName:  user.LastName,
		Username:  user.Username,
	}
}

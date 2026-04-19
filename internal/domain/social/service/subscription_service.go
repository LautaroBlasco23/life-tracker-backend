package service

import (
	"errors"

	activityModel "life-tracker-backend/internal/domain/activity/model"
	routineModel "life-tracker-backend/internal/domain/routine/model"
	"life-tracker-backend/internal/domain/social/dto"
	socialModel "life-tracker-backend/internal/domain/social/model"
	"life-tracker-backend/internal/domain/social/repository"
	userRepo "life-tracker-backend/internal/domain/user/repository"

	"gorm.io/gorm"
)

type SubscriptionService struct {
	subRepo    repository.SubscriptionRepository
	userRepo   userRepo.UserRepository
	visibility *VisibilityService
	db         *gorm.DB
}

func NewSubscriptionService(db *gorm.DB) *SubscriptionService {
	return &SubscriptionService{
		subRepo:    repository.NewSubscriptionRepository(db),
		userRepo:   userRepo.NewUserRepository(db),
		visibility: NewVisibilityService(db),
		db:         db,
	}
}

func (s *SubscriptionService) Subscribe(subscriberID uint, req *dto.SubscribeRequest) (*dto.SubscriptionResponse, error) {
	switch req.SourceType {
	case "activity":
		return s.subscribeToActivity(subscriberID, req.SourceID)
	case "routine":
		return s.subscribeToRoutine(subscriberID, req.SourceID)
	default:
		return nil, errors.New("invalid source type")
	}
}

func (s *SubscriptionService) subscribeToActivity(subscriberID, sourceID uint) (*dto.SubscriptionResponse, error) {
	var source activityModel.Activity
	if err := s.db.First(&source, sourceID).Error; err != nil {
		return nil, errors.New("activity not found")
	}

	canView, err := s.visibility.CanViewContent(subscriberID, source.UserID, string(source.PrivacyStatus))
	if err != nil || !canView {
		return nil, errors.New("access denied")
	}

	existing, err := s.subRepo.FindExisting(subscriberID, socialModel.SourceTypeActivity, sourceID)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return toSubscriptionResponse(existing), nil
	}

	var sub *socialModel.Subscription
	txErr := s.db.Transaction(func(tx *gorm.DB) error {
		clone := activityModel.Activity{
			UserID:           subscriberID,
			Title:            source.Title,
			Description:      source.Description,
			Frequency:        source.Frequency,
			DayFrequency:     source.DayFrequency,
			DayTime:          source.DayTime,
			CompletionAmount: source.CompletionAmount,
			IsActive:         false,
			PrivacyStatus:    activityModel.PrivacyPrivate,
		}
		if err := tx.Create(&clone).Error; err != nil {
			return err
		}
		sub = &socialModel.Subscription{
			SubscriberUserID: subscriberID,
			SourceUserID:     source.UserID,
			SourceType:       socialModel.SourceTypeActivity,
			SourceID:         source.ID,
			ClonedID:         clone.ID,
		}
		return tx.Create(sub).Error
	})
	if txErr != nil {
		return nil, errors.New("failed to subscribe")
	}
	return toSubscriptionResponse(sub), nil
}

func (s *SubscriptionService) subscribeToRoutine(subscriberID, sourceID uint) (*dto.SubscriptionResponse, error) {
	var sourceRoutine routineModel.Routine
	if err := s.db.First(&sourceRoutine, sourceID).Error; err != nil {
		return nil, errors.New("routine not found")
	}

	canView, err := s.visibility.CanViewContent(subscriberID, sourceRoutine.UserID, string(sourceRoutine.PrivacyStatus))
	if err != nil || !canView {
		return nil, errors.New("access denied")
	}

	existing, err := s.subRepo.FindExisting(subscriberID, socialModel.SourceTypeRoutine, sourceID)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return toSubscriptionResponse(existing), nil
	}

	var sourceActivities []routineModel.RoutineActivity
	s.db.Where("routine_id = ?", sourceID).Order("position ASC").Find(&sourceActivities)

	activityIDs := make([]uint, len(sourceActivities))
	for i, ra := range sourceActivities {
		activityIDs[i] = ra.ActivityID
	}

	var sourceActs []activityModel.Activity
	if len(activityIDs) > 0 {
		s.db.Where("id IN ?", activityIDs).Find(&sourceActs)
	}
	actMap := make(map[uint]activityModel.Activity, len(sourceActs))
	for _, a := range sourceActs {
		actMap[a.ID] = a
	}

	var sub *socialModel.Subscription
	txErr := s.db.Transaction(func(tx *gorm.DB) error {
		clonedRoutine := routineModel.Routine{
			UserID:        subscriberID,
			Title:         sourceRoutine.Title,
			Description:   sourceRoutine.Description,
			PrivacyStatus: routineModel.PrivacyPrivate,
		}
		if err := tx.Create(&clonedRoutine).Error; err != nil {
			return err
		}

		for i, ra := range sourceActivities {
			srcAct, ok := actMap[ra.ActivityID]
			if !ok {
				continue
			}
			clonedAct := activityModel.Activity{
				UserID:           subscriberID,
				Title:            srcAct.Title,
				Description:      srcAct.Description,
				Frequency:        srcAct.Frequency,
				DayFrequency:     srcAct.DayFrequency,
				DayTime:          srcAct.DayTime,
				CompletionAmount: srcAct.CompletionAmount,
				IsActive:         false,
				PrivacyStatus:    activityModel.PrivacyPrivate,
			}
			if err := tx.Create(&clonedAct).Error; err != nil {
				return err
			}
			newRA := routineModel.RoutineActivity{
				RoutineID:  clonedRoutine.ID,
				ActivityID: clonedAct.ID,
				Position:   i,
			}
			if err := tx.Create(&newRA).Error; err != nil {
				return err
			}
		}

		sub = &socialModel.Subscription{
			SubscriberUserID: subscriberID,
			SourceUserID:     sourceRoutine.UserID,
			SourceType:       socialModel.SourceTypeRoutine,
			SourceID:         sourceRoutine.ID,
			ClonedID:         clonedRoutine.ID,
		}
		return tx.Create(sub).Error
	})
	if txErr != nil {
		return nil, errors.New("failed to subscribe")
	}
	return toSubscriptionResponse(sub), nil
}

func (s *SubscriptionService) GetMySubscriptions(subscriberID uint) ([]dto.SubscriptionResponse, error) {
	subs, err := s.subRepo.FindBySubscriber(subscriberID)
	if err != nil {
		return nil, errors.New("failed to fetch subscriptions")
	}
	responses := make([]dto.SubscriptionResponse, len(subs))
	for i := range subs {
		responses[i] = *toSubscriptionResponse(&subs[i])
	}
	return responses, nil
}

func (s *SubscriptionService) Unsubscribe(subscriberID, subID uint) error {
	if err := s.subRepo.Delete(subID, subscriberID); err != nil {
		return errors.New("failed to delete subscription")
	}
	return nil
}

func toSubscriptionResponse(sub *socialModel.Subscription) *dto.SubscriptionResponse {
	return &dto.SubscriptionResponse{
		ID:         sub.ID,
		SourceType: string(sub.SourceType),
		SourceID:   sub.SourceID,
		ClonedID:   sub.ClonedID,
	}
}

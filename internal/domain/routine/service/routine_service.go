package service

import (
	"errors"

	activityModel "life-tracker-backend/internal/domain/activity/model"
	"life-tracker-backend/internal/domain/routine/dto"
	"life-tracker-backend/internal/domain/routine/model"
	"life-tracker-backend/internal/domain/routine/repository"

	"gorm.io/gorm"
)

type RoutineService struct {
	repo repository.RoutineRepository
	db   *gorm.DB
}

func NewRoutineService(db *gorm.DB) *RoutineService {
	return &RoutineService{
		repo: repository.NewRoutineRepository(db),
		db:   db,
	}
}

func (s *RoutineService) validateActivitiesOwnership(userID uint, activityIDs []dto.ActivityIDPosition) error {
	if len(activityIDs) == 0 {
		return nil
	}
	ids := make([]uint, len(activityIDs))
	for i, a := range activityIDs {
		ids[i] = a.ActivityID
	}
	var count int64
	s.db.Model(&activityModel.Activity{}).
		Where("id IN ? AND user_id = ? AND deleted_at IS NULL", ids, userID).
		Count(&count)
	if int(count) != len(ids) {
		return errors.New("one or more activities not found or not owned by user")
	}
	return nil
}

func (s *RoutineService) fetchActivitiesForResponse(routineID uint) []dto.RoutineActivityItem {
	items, err := s.repo.GetActivities(routineID)
	if err != nil || len(items) == 0 {
		return []dto.RoutineActivityItem{}
	}

	activityIDs := make([]uint, len(items))
	for i, it := range items {
		activityIDs[i] = it.ActivityID
	}

	var activities []activityModel.Activity
	s.db.Where("id IN ?", activityIDs).Find(&activities)

	titleMap := make(map[uint]string, len(activities))
	for _, a := range activities {
		titleMap[a.ID] = a.Title
	}

	result := make([]dto.RoutineActivityItem, len(items))
	for i, it := range items {
		result[i] = dto.RoutineActivityItem{
			ActivityID:    it.ActivityID,
			ActivityTitle: titleMap[it.ActivityID],
			Position:      it.Position,
		}
	}
	return result
}

func (s *RoutineService) CreateRoutine(userID uint, req *dto.CreateRoutineRequest) (*dto.RoutineResponse, error) {
	if err := s.validateActivitiesOwnership(userID, req.ActivityIDs); err != nil {
		return nil, err
	}

	routine := model.Routine{
		UserID:        userID,
		Title:         req.Title,
		Description:   req.Description,
		PrivacyStatus: model.PrivacyStatus(req.PrivacyStatus),
	}
	if err := s.repo.Create(&routine); err != nil {
		return nil, errors.New("failed to create routine")
	}

	items := make([]model.RoutineActivity, len(req.ActivityIDs))
	for i, a := range req.ActivityIDs {
		items[i] = model.RoutineActivity{RoutineID: routine.ID, ActivityID: a.ActivityID, Position: a.Position}
	}
	if err := s.repo.SetActivities(routine.ID, items); err != nil {
		return nil, errors.New("failed to set routine activities")
	}

	return routine.ToResponse(s.fetchActivitiesForResponse(routine.ID)), nil
}

func (s *RoutineService) GetMyRoutines(userID uint) ([]*dto.RoutineResponse, error) {
	routines, err := s.repo.FindByUserID(userID)
	if err != nil {
		return nil, errors.New("failed to fetch routines")
	}
	responses := make([]*dto.RoutineResponse, len(routines))
	for i := range routines {
		responses[i] = routines[i].ToResponse(s.fetchActivitiesForResponse(routines[i].ID))
	}
	return responses, nil
}

func (s *RoutineService) GetRoutine(userID, routineID uint) (*dto.RoutineResponse, error) {
	routine, err := s.repo.FindByIDPublic(routineID)
	if err != nil {
		if errors.Is(err, repository.ErrRoutineNotFound) {
			return nil, errors.New("routine not found")
		}
		return nil, errors.New("failed to fetch routine")
	}
	if routine.UserID != userID && routine.PrivacyStatus != model.PrivacyPublic {
		return nil, errors.New("routine not found")
	}
	return routine.ToResponse(s.fetchActivitiesForResponse(routine.ID)), nil
}

func (s *RoutineService) UpdateRoutine(userID, routineID uint, req *dto.UpdateRoutineRequest) (*dto.RoutineResponse, error) {
	routine, err := s.repo.FindByID(routineID, userID)
	if err != nil {
		if errors.Is(err, repository.ErrRoutineNotFound) {
			return nil, errors.New("routine not found")
		}
		return nil, errors.New("failed to fetch routine")
	}

	updates := map[string]interface{}{}
	if req.Title != nil {
		updates["title"] = *req.Title
	}
	if req.Description != nil {
		updates["description"] = *req.Description
	}
	if req.PrivacyStatus != nil {
		updates["privacy_status"] = *req.PrivacyStatus
	}
	if len(updates) > 0 {
		if err := s.repo.Update(routine, updates); err != nil {
			return nil, errors.New("failed to update routine")
		}
	}

	if req.ActivityIDs != nil {
		if err := s.validateActivitiesOwnership(userID, req.ActivityIDs); err != nil {
			return nil, err
		}
		items := make([]model.RoutineActivity, len(req.ActivityIDs))
		for i, a := range req.ActivityIDs {
			items[i] = model.RoutineActivity{RoutineID: routineID, ActivityID: a.ActivityID, Position: a.Position}
		}
		if err := s.repo.SetActivities(routineID, items); err != nil {
			return nil, errors.New("failed to update activities")
		}
	}

	updated, _ := s.repo.FindByID(routineID, userID)
	return updated.ToResponse(s.fetchActivitiesForResponse(routineID)), nil
}

func (s *RoutineService) DeleteRoutine(userID, routineID uint) error {
	err := s.repo.Delete(routineID, userID)
	if errors.Is(err, repository.ErrRoutineNotFound) {
		return errors.New("routine not found")
	}
	return err
}

func (s *RoutineService) GetPublicByUser(userID uint) ([]*dto.RoutineResponse, error) {
	routines, err := s.repo.FindPublicByUserID(userID)
	if err != nil {
		return nil, errors.New("failed to fetch routines")
	}
	responses := make([]*dto.RoutineResponse, len(routines))
	for i := range routines {
		responses[i] = routines[i].ToResponse(s.fetchActivitiesForResponse(routines[i].ID))
	}
	return responses, nil
}

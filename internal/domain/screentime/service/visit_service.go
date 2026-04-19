package service

import (
	"context"
	"fmt"
	"time"

	"life-tracker-backend/internal/domain/screentime/dto"
	"life-tracker-backend/internal/domain/screentime/model"
	"life-tracker-backend/internal/domain/screentime/repository"

	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type VisitService struct {
	visitRepo repository.VisitRepository
}

func NewVisitService(db *mongo.Database) *VisitService {
	return &VisitService{visitRepo: repository.NewVisitRepository(db)}
}

func (s *VisitService) RecordVisit(userID uint, req *dto.RecordVisitRequest) (*dto.VisitResponse, error) {
	now := time.Now()
	visit := &model.WebVisit{
		ID:        primitive.NewObjectID(),
		UserID:    userID,
		Domain:    req.Domain,
		VisitedAt: req.VisitedAt,
		CreatedAt: now,
	}

	if err := s.visitRepo.Create(context.Background(), visit); err != nil {
		return nil, fmt.Errorf("recording visit: %w", err)
	}

	return visit.ToResponse(), nil
}

func (s *VisitService) GetVisits(userID uint, domain string, from, to *time.Time, limit int) ([]dto.VisitResponse, error) {
	visits, err := s.visitRepo.FindByUser(context.Background(), userID, domain, from, to, limit)
	if err != nil {
		return nil, fmt.Errorf("getting visits: %w", err)
	}

	responses := make([]dto.VisitResponse, 0, len(visits))
	for i := range visits {
		responses = append(responses, *visits[i].ToResponse())
	}
	return responses, nil
}

func (s *VisitService) GetStats(userID uint, from, to *time.Time) ([]dto.DomainStat, error) {
	stats, err := s.visitRepo.StatsByUser(context.Background(), userID, from, to)
	if err != nil {
		return nil, fmt.Errorf("getting stats: %w", err)
	}
	if stats == nil {
		return []dto.DomainStat{}, nil
	}
	return stats, nil
}

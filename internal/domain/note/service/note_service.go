package service

import (
	"errors"

	"life-tracker-backend/internal/domain/note/dto"
	"life-tracker-backend/internal/domain/note/model"
	"life-tracker-backend/internal/domain/note/repository"

	"gorm.io/gorm"
)

var ErrNoteNotFound = errors.New("note not found")

type NoteService struct {
	repo *repository.NoteRepository
}

func NewNoteService(db *gorm.DB) *NoteService {
	return &NoteService{
		repo: repository.NewNoteRepository(db),
	}
}

func (s *NoteService) CreateNote(userID uint, req *dto.CreateNoteRequest) (*dto.NoteResponse, error) {
	note := &model.Note{
		UserID:  userID,
		Title:   req.Title,
		Content: req.Content,
	}

	if err := s.repo.Create(note); err != nil {
		return nil, err
	}

	return note.ToResponse(), nil
}

func (s *NoteService) GetNote(id, userID uint) (*dto.NoteResponse, error) {
	note, err := s.repo.FindByID(id, userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNoteNotFound
		}
		return nil, err
	}

	return note.ToResponse(), nil
}

func (s *NoteService) GetNotes(userID uint, filter *dto.NoteFilter) ([]*dto.NoteResponse, error) {
	notes, err := s.repo.FindByUserID(userID, filter)
	if err != nil {
		return nil, err
	}

	responses := make([]*dto.NoteResponse, len(notes))
	for i, note := range notes {
		responses[i] = note.ToResponse()
	}

	return responses, nil
}

func (s *NoteService) UpdateNote(id, userID uint, req *dto.UpdateNoteRequest) (*dto.NoteResponse, error) {
	note, err := s.repo.FindByID(id, userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNoteNotFound
		}
		return nil, err
	}

	if req.Title != nil {
		note.Title = *req.Title
	}
	if req.Content != nil {
		note.Content = *req.Content
	}

	if err := s.repo.Update(note); err != nil {
		return nil, err
	}

	return note.ToResponse(), nil
}

func (s *NoteService) DeleteNote(id, userID uint) error {
	note, err := s.repo.FindByID(id, userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrNoteNotFound
		}
		return err
	}

	return s.repo.Delete(note.ID, userID)
}

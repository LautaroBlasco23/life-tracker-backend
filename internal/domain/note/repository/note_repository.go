package repository

import (
	"life-tracker-backend/internal/domain/note/dto"
	"life-tracker-backend/internal/domain/note/model"

	"gorm.io/gorm"
)

type NoteRepository struct {
	db *gorm.DB
}

func NewNoteRepository(db *gorm.DB) *NoteRepository {
	return &NoteRepository{db: db}
}

func (r *NoteRepository) Create(note *model.Note) error {
	return r.db.Create(note).Error
}

func (r *NoteRepository) FindByID(id, userID uint) (*model.Note, error) {
	var note model.Note
	err := r.db.Where("id = ? AND user_id = ?", id, userID).First(&note).Error
	if err != nil {
		return nil, err
	}
	return &note, nil
}

func (r *NoteRepository) FindByUserID(userID uint, filter *dto.NoteFilter) ([]*model.Note, error) {
	var notes []*model.Note
	query := r.db.Where("user_id = ?", userID)

	if filter != nil && filter.Query != "" {
		searchPattern := "%" + filter.Query + "%"
		query = query.Where("title ILIKE ? OR content ILIKE ?", searchPattern, searchPattern)
	}

	err := query.Order("updated_at DESC").Find(&notes).Error
	return notes, err
}

func (r *NoteRepository) Update(note *model.Note) error {
	return r.db.Save(note).Error
}

func (r *NoteRepository) Delete(id, userID uint) error {
	return r.db.Where("id = ? AND user_id = ?", id, userID).Delete(&model.Note{}).Error
}

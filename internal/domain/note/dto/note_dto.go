package dto

import "time"

type CreateNoteRequest struct {
	Title   string `json:"title" binding:"required,min=1,max=255"`
	Content string `json:"content" binding:"max=50000"`
}

type UpdateNoteRequest struct {
	Title   *string `json:"title,omitempty" binding:"omitempty,min=1,max=255"`
	Content *string `json:"content,omitempty" binding:"omitempty,max=50000"`
}

type NoteResponse struct {
	CreatedAt time.Time `json:"createdAt"`
	EditedAt  time.Time `json:"editedAt"`
	Title     string    `json:"title"`
	Content   string    `json:"content"`
	ID        uint      `json:"id"`
	UserID    uint      `json:"userId"`
}

type NoteFilter struct {
	Query string `form:"q"`
}

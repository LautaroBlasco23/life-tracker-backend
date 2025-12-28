package controller

import (
	"errors"
	"net/http"
	"strconv"

	"life-tracker-backend/internal/domain/note/dto"
	"life-tracker-backend/internal/domain/note/service"

	"github.com/gin-gonic/gin"
)

type NoteController struct {
	service *service.NoteService
}

func NewNoteController(service *service.NoteService) *NoteController {
	return &NoteController{service: service}
}

func (c *NoteController) CreateNote(ctx *gin.Context) {
	userID := ctx.GetUint("userID")

	var req dto.CreateNoteRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"message": err.Error()})
		return
	}

	note, err := c.service.CreateNote(userID, &req)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to create note"})
		return
	}

	ctx.JSON(http.StatusCreated, gin.H{
		"message": "Note created successfully",
		"data":    note,
	})
}

func (c *NoteController) GetNote(ctx *gin.Context) {
	userID := ctx.GetUint("userID")

	id, err := strconv.ParseUint(ctx.Param("id"), 10, 32)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"message": "Invalid note ID"})
		return
	}

	note, err := c.service.GetNote(uint(id), userID)
	if err != nil {
		if errors.Is(err, service.ErrNoteNotFound) {
			ctx.JSON(http.StatusNotFound, gin.H{"message": "Note not found"})
			return
		}
		ctx.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to fetch note"})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"message": "Note retrieved successfully",
		"data":    note,
	})
}

func (c *NoteController) GetNotes(ctx *gin.Context) {
	userID := ctx.GetUint("userID")

	var filter dto.NoteFilter
	if err := ctx.ShouldBindQuery(&filter); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"message": err.Error()})
		return
	}

	notes, err := c.service.GetNotes(userID, &filter)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to fetch notes"})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"message": "Notes retrieved successfully",
		"data":    notes,
		"count":   len(notes),
	})
}

func (c *NoteController) UpdateNote(ctx *gin.Context) {
	userID := ctx.GetUint("userID")

	id, err := strconv.ParseUint(ctx.Param("id"), 10, 32)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"message": "Invalid note ID"})
		return
	}

	var req dto.UpdateNoteRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"message": err.Error()})
		return
	}

	note, err := c.service.UpdateNote(uint(id), userID, &req)
	if err != nil {
		if errors.Is(err, service.ErrNoteNotFound) {
			ctx.JSON(http.StatusNotFound, gin.H{"message": "Note not found"})
			return
		}
		ctx.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to update note"})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"message": "Note updated successfully",
		"data":    note,
	})
}

func (c *NoteController) DeleteNote(ctx *gin.Context) {
	userID := ctx.GetUint("userID")

	id, err := strconv.ParseUint(ctx.Param("id"), 10, 32)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"message": "Invalid note ID"})
		return
	}

	if err := c.service.DeleteNote(uint(id), userID); err != nil {
		if errors.Is(err, service.ErrNoteNotFound) {
			ctx.JSON(http.StatusNotFound, gin.H{"message": "Note not found"})
			return
		}
		ctx.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to delete note"})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"message": "Note deleted successfully"})
}

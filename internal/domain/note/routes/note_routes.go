package routes

import (
	"life-tracker-backend/internal/domain/note/controller"
	"life-tracker-backend/internal/domain/note/service"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func RegisterNoteRoutes(router *gin.RouterGroup, db *gorm.DB) {
	noteService := service.NewNoteService(db)
	noteController := controller.NewNoteController(noteService)

	noteGroup := router.Group("/notes")
	{
		noteGroup.POST("", noteController.CreateNote)
		noteGroup.GET("", noteController.GetNotes)
		noteGroup.GET("/:id", noteController.GetNote)
		noteGroup.PUT("/:id", noteController.UpdateNote)
		noteGroup.DELETE("/:id", noteController.DeleteNote)
	}
}

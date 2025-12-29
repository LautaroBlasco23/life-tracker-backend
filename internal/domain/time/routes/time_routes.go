package routes

import (
	"life-tracker-backend/internal/domain/time/controller"
	"life-tracker-backend/internal/domain/time/service"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func RegisterTimeRoutes(router *gin.RouterGroup, db *gorm.DB) {
	timeService := service.NewTimeService(db)
	timeController := controller.NewTimeController(timeService)

	timeGroup := router.Group("/time-records")
	{
		timeGroup.POST("", timeController.CreateRecord)
		timeGroup.GET("", timeController.GetRecords)
		timeGroup.GET("/stats", timeController.GetStats)
		timeGroup.GET("/:id", timeController.GetRecord)
		timeGroup.PUT("/:id", timeController.UpdateRecord)
		timeGroup.DELETE("/:id", timeController.DeleteRecord)
	}
}

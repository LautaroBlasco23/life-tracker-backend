package routes

import (
	"life-tracker-backend/internal/domain/activity/controller"
	"life-tracker-backend/internal/domain/activity/service"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/mongo"
	"gorm.io/gorm"
)

func RegisterActivityRoutes(router *gin.RouterGroup, db *gorm.DB, mongoDB *mongo.Database) {
	activityService := service.NewActivityService(db, mongoDB)
	activityController := controller.NewActivityController(activityService)

	activityGroup := router.Group("/activities")
	{
		activityGroup.POST("", activityController.CreateActivity)
		activityGroup.GET("", activityController.GetUserActivities)
		activityGroup.GET("/today", activityController.GetTodayActivities)
		activityGroup.GET("/:id", activityController.GetActivity)
		activityGroup.PUT("/:id", activityController.UpdateActivity)
		activityGroup.DELETE("/:id", activityController.DeleteActivity)

		activityGroup.POST("/:id/record", activityController.RecordActivity)
		activityGroup.GET("/:id/records", activityController.GetActivityRecords)
		activityGroup.GET("/:id/stats", activityController.GetActivityStats)
		activityGroup.DELETE("/:id/revert", activityController.RevertLastCompletion)
	}
}

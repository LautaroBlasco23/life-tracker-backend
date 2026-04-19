package routes

import (
	"life-tracker-backend/internal/domain/screentime/controller"
	"life-tracker-backend/internal/domain/screentime/service"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/mongo"
)

func RegisterScreenTimeRoutes(router *gin.RouterGroup, mongoDB *mongo.Database) {
	visitService := service.NewVisitService(mongoDB)
	visitController := controller.NewVisitController(visitService)

	screenTimeGroup := router.Group("/screen-time")
	{
		screenTimeGroup.POST("/visits", visitController.RecordVisit)
		screenTimeGroup.GET("/visits", visitController.GetVisits)
		screenTimeGroup.GET("/stats", visitController.GetStats)
	}
}

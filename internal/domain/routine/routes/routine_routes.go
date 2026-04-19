package routes

import (
	"life-tracker-backend/internal/domain/routine/controller"
	"life-tracker-backend/internal/domain/routine/service"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func RegisterRoutineRoutes(router *gin.RouterGroup, db *gorm.DB) {
	routineService := service.NewRoutineService(db)
	routineController := controller.NewRoutineController(routineService)

	routineGroup := router.Group("/routines")
	{
		routineGroup.POST("", routineController.CreateRoutine)
		routineGroup.GET("/me", routineController.GetMyRoutines)
		routineGroup.GET("/:id", routineController.GetRoutine)
		routineGroup.PATCH("/:id", routineController.UpdateRoutine)
		routineGroup.DELETE("/:id", routineController.DeleteRoutine)
	}
}

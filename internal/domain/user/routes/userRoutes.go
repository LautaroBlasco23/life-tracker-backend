package routes

import (
	"life-tracker-backend/internal/domain/user/controller"
	"life-tracker-backend/internal/domain/user/service"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func RegisterUserRoutes(router *gin.RouterGroup, db *gorm.DB) {
	userService := service.NewUserService(db)
	userController := controller.NewUserController(userService)

	userGroup := router.Group("/users")
	{
		userGroup.GET("/profile", userController.GetProfile)
		userGroup.PUT("/profile", userController.UpdateProfile)
		userGroup.GET("", userController.GetAllUsers)
		userGroup.GET("/:id", userController.GetUserByID)
		userGroup.DELETE("/:id", userController.DeleteUser)
	}
}

package routes

import (
	"life-tracker-backend/internal/domain/user/controller"
	"life-tracker-backend/internal/domain/user/service"
	"life-tracker-backend/internal/infrastructure/imagestore"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func RegisterUserRoutes(router *gin.RouterGroup, db *gorm.DB, imageClient *imagestore.Client) {
	userService := service.NewUserService(db, imageClient)
	userController := controller.NewUserController(userService)

	userGroup := router.Group("/users")
	{
		userGroup.GET("/profile", userController.GetProfile)
		userGroup.PUT("/profile", userController.UpdateProfile)
		userGroup.GET("", userController.GetAllUsers)
		userGroup.GET("/:id", userController.GetUserByID)
		userGroup.DELETE("/:id", userController.DeleteUser)

		// Image related routes
		userGroup.POST("/profile/image", userController.UploadProfileImage)
		userGroup.DELETE("/profile/image", userController.DeleteProfileImage)
	}
}

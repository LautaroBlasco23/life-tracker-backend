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
		userGroup.GET("/me", userController.GetMyProfile)
		userGroup.PUT("/me", userController.UpdateProfile)
		userGroup.POST("/me/image", userController.UploadProfileImage)
		userGroup.DELETE("/me/image", userController.DeleteProfileImage)

		userGroup.GET("", userController.GetAllUsers)
		userGroup.GET("/search", userController.SearchUsers)
		userGroup.GET("/:username", userController.GetUserOrProfile)
		userGroup.DELETE("/:id", userController.DeleteUser)
	}
}

func RegisterPublicUserRoutes(router *gin.RouterGroup, db *gorm.DB, imageClient *imagestore.Client) {
	userService := service.NewUserService(db, imageClient)
	userController := controller.NewUserController(userService)

	userGroup := router.Group("/users")
	{
		userGroup.GET("/check-username", userController.CheckUsernameAvailability)
	}
}

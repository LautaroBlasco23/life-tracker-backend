package routes

import (
	"life-tracker-backend/internal/config"
	"life-tracker-backend/internal/domain/auth/controller"
	"life-tracker-backend/internal/domain/auth/service"
	userService "life-tracker-backend/internal/domain/user/service"
	"life-tracker-backend/internal/infrastructure/imagestore"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func RegisterAuthRoutes(router *gin.RouterGroup, db *gorm.DB, cfg *config.Config, imageClient *imagestore.Client) {
	authService := service.NewAuthService(db, cfg)
	userService := userService.NewUserService(db, imageClient)
	authController := controller.NewAuthController(authService, userService)

	authGroup := router.Group("/auth")
	{
		authGroup.POST("/register", authController.Register)
		authGroup.POST("/login", authController.Login)
		authGroup.POST("/refresh", authController.RefreshToken)
	}
}

func RegisterProtectedAuthRoutes(router *gin.RouterGroup, db *gorm.DB, cfg *config.Config, imageClient *imagestore.Client) {
	authService := service.NewAuthService(db, cfg)
	userService := userService.NewUserService(db, imageClient)
	authController := controller.NewAuthController(authService, userService)

	authGroup := router.Group("/auth")
	{
		authGroup.PUT("/password", authController.UpdatePassword)
		authGroup.PUT("/email", authController.UpdateEmail)
	}
}

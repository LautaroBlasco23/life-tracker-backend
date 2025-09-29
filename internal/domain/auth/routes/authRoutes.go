package routes

import (
	"life-tracker-backend/internal/config"
	"life-tracker-backend/internal/domain/auth/controller"
	"life-tracker-backend/internal/domain/auth/service"
	userService "life-tracker-backend/internal/domain/user/service"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func RegisterAuthRoutes(router *gin.RouterGroup, db *gorm.DB, cfg *config.Config) {
	// Create services
	authService := service.NewAuthService(db, cfg)
	userService := userService.NewUserService(db)

	// Create controller with both services
	authController := controller.NewAuthController(authService, userService)

	// Register routes
	authGroup := router.Group("/auth")
	{
		authGroup.POST("/register", authController.Register)
		authGroup.POST("/login", authController.Login)
		authGroup.POST("/refresh", authController.RefreshToken)
	}
}

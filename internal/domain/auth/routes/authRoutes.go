package routes

import (
	"life-tracker-backend/internal/config"
	"life-tracker-backend/internal/domain/auth/controller"
	"life-tracker-backend/internal/domain/auth/service"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func RegisterAuthRoutes(router *gin.RouterGroup, db *gorm.DB, cfg *config.Config) {
	authService := service.NewAuthService(db, cfg)
	authController := controller.NewAuthController(authService)

	authGroup := router.Group("/auth")
	{
		authGroup.POST("/register", authController.Register)
		authGroup.POST("/login", authController.Login)
		authGroup.POST("/refresh", authController.RefreshToken)
	}
}

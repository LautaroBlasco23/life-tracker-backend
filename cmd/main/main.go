package main

import (
	"fmt"
	"life-tracker-backend/internal/auth/routes"
	"life-tracker-backend/internal/config"
	"life-tracker-backend/internal/database"
	"life-tracker-backend/internal/middleware"
	"log"
	"net/http"
	"time"

	userRoutes "life-tracker-backend/internal/user/routes"

	"github.com/gin-gonic/gin"
)

func main() {
	// Load configuration
	cfg := config.Load()

	// Initialize database
	db, err := database.Initialize(cfg)
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}

	// Set Gin mode
	gin.SetMode(cfg.GinMode)

	// Initialize router
	r := gin.New()

	// Add middleware
	r.Use(gin.Logger())
	r.Use(gin.Recovery())
	r.Use(middleware.CORSMiddleware())

	// Health check endpoint
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":    "ok",
			"timestamp": time.Now().Unix(),
		})
	})

	// API routes
	api := r.Group("/api/v1")
	{
		// Auth routes (public)
		routes.RegisterAuthRoutes(api, db, cfg)

		// User routes (protected)
		protected := api.Group("")
		protected.Use(middleware.JWTAuthMiddleware(cfg.JWTSecret))
		userRoutes.RegisterUserRoutes(protected, db)
	}

	// Start server
	port := fmt.Sprintf(":%s", cfg.Port)
	log.Printf("Server starting on port %s", cfg.Port)

	if err := r.Run(port); err != nil {
		log.Fatal("Failed to start server:", err)
	}
}

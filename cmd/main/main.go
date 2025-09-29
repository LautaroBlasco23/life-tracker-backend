package main

import (
	"fmt"
	"life-tracker-backend/internal/config"
	"life-tracker-backend/internal/database"
	"life-tracker-backend/internal/domain/auth/routes"
	"life-tracker-backend/internal/middleware"
	"log"
	"net/http"
	"time"

	activityRoutes "life-tracker-backend/internal/domain/activity/routes"

	userRoutes "life-tracker-backend/internal/domain/user/routes"

	"github.com/gin-gonic/gin"
)

func main() {
	// Load configuration
	cfg := config.Load()

	// Initialize databases
	dbs, err := database.Initialize(cfg)
	if err != nil {
		log.Fatal("Failed to connect to databases:", err)
	}

	log.Println("✅ Connected to PostgreSQL successfully")
	log.Println("✅ Connected to MongoDB successfully")

	// Set Gin mode
	gin.SetMode(cfg.GinMode)

	// Initialize router
	r := gin.New()

	// Add middleware
	r.Use(gin.Logger())
	r.Use(gin.Recovery())
	r.Use(middleware.CORSMiddleware())

	// Health check endpoint with database status
	r.GET("/health", func(c *gin.Context) {
		// Test PostgreSQL connection
		sqlDB, err := dbs.PostgreSQL.DB()
		postgresStatus := "ok"
		if err != nil || sqlDB.Ping() != nil {
			postgresStatus = "error"
		}

		// Test MongoDB connection
		mongoStatus := "ok"
		if err := dbs.MongoDB.Client().Ping(c.Request.Context(), nil); err != nil {
			mongoStatus = "error"
		}

		c.JSON(http.StatusOK, gin.H{
			"status":    "ok",
			"timestamp": time.Now().Unix(),
			"databases": gin.H{
				"postgresql": postgresStatus,
				"mongodb":    mongoStatus,
			},
		})
	})

	// API routes
	api := r.Group("/api")
	{
		// Auth routes (public)
		routes.RegisterAuthRoutes(api, dbs.PostgreSQL, cfg)

		// Protected routes
		protected := api.Group("")
		protected.Use(middleware.JWTAuthMiddleware(cfg.JWTSecret))
		{
			// User routes
			userRoutes.RegisterUserRoutes(protected, dbs.PostgreSQL)

			// Activity routes
			activityRoutes.RegisterActivityRoutes(protected, dbs.PostgreSQL, dbs.MongoDB)
		}
	}

	// Start server
	port := fmt.Sprintf(":%s", cfg.Port)
	log.Printf("🚀 Server starting on port %s", cfg.Port)
	log.Printf("📊 Health check available at: http://localhost%s/health", port)
	if err := r.Run(port); err != nil {
		log.Fatal("Failed to start server:", err)
	}
}

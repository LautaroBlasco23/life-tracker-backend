package main

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"life-tracker-backend/internal/config"
	"life-tracker-backend/internal/database"
	authRoutes "life-tracker-backend/internal/domain/auth/routes"
	noteRoutes "life-tracker-backend/internal/domain/note/routes"
	timeRoutes "life-tracker-backend/internal/domain/time/routes"
	"life-tracker-backend/internal/infrastructure/imagestore"
	"life-tracker-backend/internal/middleware"

	activityRoutes "life-tracker-backend/internal/domain/activity/routes"
	financeRoutes "life-tracker-backend/internal/domain/finance/routes"
	screenTimeRoutes "life-tracker-backend/internal/domain/screentime/routes"
	userRoutes "life-tracker-backend/internal/domain/user/routes"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	// Load configuration
	cfg := config.Load()

	// Initialize databases
	dbs, err := database.Initialize(cfg)
	if err != nil {
		return fmt.Errorf("failed to connect to databases: %w", err)
	}

	log.Println("✅ Connected to PostgreSQL successfully")
	log.Println("✅ Connected to MongoDB successfully")

	// Image Service
	imageClient, err := imagestore.NewClient(cfg.ImageStoreAddress)
	if err != nil {
		return fmt.Errorf("failed to connect to imagestore: %w", err)
	}
	defer func() {
		if closeErr := imageClient.Close(); closeErr != nil {
			log.Printf("Error closing image client: %v", closeErr)
		}
	}()

	log.Println("✅ Connected to ImageStore service")

	// Set Gin mode
	gin.SetMode(cfg.GinMode)

	// Initialize router
	r := gin.New()

	err = r.SetTrustedProxies([]string{
		"127.0.0.1",
		"172.16.0.0/12", // docker bridge
	})
	if err != nil {
		return fmt.Errorf("failed to set trusted proxies: %w", err)
	}

	// Add middleware
	r.Use(gin.Logger())
	r.Use(gin.Recovery())
	r.Use(middleware.CORSMiddleware(cfg.CORSAllowedOrigins))
	r.Use(middleware.TimezoneMiddleware())
	r.Use(middleware.PrometheusMiddleware())

	// monitoring metrics endpoint
	r.GET("/metrics", gin.WrapH(promhttp.Handler()))

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
		authRoutes.RegisterAuthRoutes(api, dbs.PostgreSQL, cfg, imageClient)

		// Protected routes
		protected := api.Group("")
		protected.Use(middleware.JWTAuthMiddleware(cfg.JWTSecret))
		{
			authRoutes.RegisterProtectedAuthRoutes(protected, dbs.PostgreSQL, cfg, imageClient)
			userRoutes.RegisterUserRoutes(protected, dbs.PostgreSQL, imageClient)
			activityRoutes.RegisterActivityRoutes(protected, dbs.PostgreSQL, dbs.MongoDB)
			financeRoutes.RegisterFinanceRoutes(protected, dbs.MongoDB)
			screenTimeRoutes.RegisterScreenTimeRoutes(protected, dbs.MongoDB)
			timeRoutes.RegisterTimeRoutes(protected, dbs.PostgreSQL)
			noteRoutes.RegisterNoteRoutes(protected, dbs.PostgreSQL)
		}
	}

	// Start server
	port := fmt.Sprintf(":%s", cfg.Port)
	log.Printf("🚀 Server starting on port %s", cfg.Port)
	log.Print("📊 Metrics available at /metrics")
	log.Print("📊 Health check available at: /health")

	if err := r.Run(port); err != nil {
		return fmt.Errorf("failed to start server: %w", err)
	}

	return nil
}

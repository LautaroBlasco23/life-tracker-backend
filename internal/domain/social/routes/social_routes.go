package routes

import (
	"life-tracker-backend/internal/domain/social/controller"
	"life-tracker-backend/internal/domain/social/service"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func RegisterSocialRoutes(router *gin.RouterGroup, db *gorm.DB) {
	followService := service.NewFollowService(db)
	subscriptionService := service.NewSubscriptionService(db)
	c := controller.NewSocialController(followService, subscriptionService)

	follows := router.Group("/follows")
	{
		follows.POST("/:username", c.Follow)
		follows.DELETE("/:username", c.Unfollow)
		follows.GET("/pending", c.GetPendingRequests)
		follows.POST("/pending/:followerId/accept", c.AcceptFollow)
		follows.POST("/pending/:followerId/reject", c.RejectFollow)
	}

	users := router.Group("/users")
	{
		users.GET("/:username/followers", c.GetFollowers)
		users.GET("/:username/following", c.GetFollowing)
	}

	subs := router.Group("/subscriptions")
	{
		subs.POST("", c.Subscribe)
		subs.GET("", c.GetMySubscriptions)
		subs.DELETE("/:id", c.Unsubscribe)
	}
}

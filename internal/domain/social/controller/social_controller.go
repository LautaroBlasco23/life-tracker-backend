package controller

import (
	"net/http"
	"strconv"

	"life-tracker-backend/internal/domain/social/dto"
	"life-tracker-backend/internal/domain/social/service"
	"life-tracker-backend/internal/infrastructure/pagination"
	"life-tracker-backend/internal/middleware"

	"github.com/gin-gonic/gin"
)

type SocialController struct {
	followService       *service.FollowService
	subscriptionService *service.SubscriptionService
}

func NewSocialController(followService *service.FollowService, subscriptionService *service.SubscriptionService) *SocialController {
	return &SocialController{
		followService:       followService,
		subscriptionService: subscriptionService,
	}
}

func (c *SocialController) Follow(ctx *gin.Context) {
	userID, ok := middleware.GetUserIDFromContext(ctx)
	if !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	resp, err := c.followService.Follow(userID, ctx.Param("username"))
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"data": resp})
}

func (c *SocialController) Unfollow(ctx *gin.Context) {
	userID, ok := middleware.GetUserIDFromContext(ctx)
	if !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	if err := c.followService.Unfollow(userID, ctx.Param("username")); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"message": "Unfollowed successfully"})
}

func (c *SocialController) GetPendingRequests(ctx *gin.Context) {
	userID, ok := middleware.GetUserIDFromContext(ctx)
	if !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	items, err := c.followService.GetPendingRequests(userID)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"data": items, "count": len(items)})
}

func (c *SocialController) AcceptFollow(ctx *gin.Context) {
	userID, ok := middleware.GetUserIDFromContext(ctx)
	if !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	followerID, err := strconv.ParseUint(ctx.Param("followerId"), 10, 32)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid follower ID"})
		return
	}
	if err := c.followService.AcceptFollow(userID, uint(followerID)); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"message": "Follow request accepted"})
}

func (c *SocialController) RejectFollow(ctx *gin.Context) {
	userID, ok := middleware.GetUserIDFromContext(ctx)
	if !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	followerID, err := strconv.ParseUint(ctx.Param("followerId"), 10, 32)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid follower ID"})
		return
	}
	if err := c.followService.RejectFollow(userID, uint(followerID)); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"message": "Follow request rejected"})
}

func (c *SocialController) GetFollowers(ctx *gin.Context) {
	userID, ok := middleware.GetUserIDFromContext(ctx)
	if !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	p := pagination.ParseParams(ctx)
	items, total, err := c.followService.GetFollowers(userID, ctx.Param("username"), p.Limit, p.Offset)
	if err != nil {
		ctx.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"data": pagination.NewPage(items, total, p)})
}

func (c *SocialController) GetFollowing(ctx *gin.Context) {
	userID, ok := middleware.GetUserIDFromContext(ctx)
	if !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	p := pagination.ParseParams(ctx)
	items, total, err := c.followService.GetFollowing(userID, ctx.Param("username"), p.Limit, p.Offset)
	if err != nil {
		ctx.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"data": pagination.NewPage(items, total, p)})
}

func (c *SocialController) Subscribe(ctx *gin.Context) {
	userID, ok := middleware.GetUserIDFromContext(ctx)
	if !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	var req dto.SubscribeRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request", "details": err.Error()})
		return
	}
	resp, err := c.subscriptionService.Subscribe(userID, &req)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	ctx.JSON(http.StatusCreated, gin.H{"data": resp})
}

func (c *SocialController) GetMySubscriptions(ctx *gin.Context) {
	userID, ok := middleware.GetUserIDFromContext(ctx)
	if !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	subs, err := c.subscriptionService.GetMySubscriptions(userID)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"data": subs, "count": len(subs)})
}

func (c *SocialController) Unsubscribe(ctx *gin.Context) {
	userID, ok := middleware.GetUserIDFromContext(ctx)
	if !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	id, err := strconv.ParseUint(ctx.Param("id"), 10, 32)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid subscription ID"})
		return
	}
	if err := c.subscriptionService.Unsubscribe(userID, uint(id)); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"message": "Unsubscribed successfully"})
}

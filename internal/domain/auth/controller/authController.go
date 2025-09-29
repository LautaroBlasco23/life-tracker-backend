package controller

import (
	"life-tracker-backend/internal/domain/auth/dto"
	"life-tracker-backend/internal/domain/auth/service"
	userService "life-tracker-backend/internal/domain/user/service"
	"net/http"

	"github.com/gin-gonic/gin"
)

type AuthController struct {
	authService *service.AuthService
	userService *userService.UserService // Add UserService dependency
}

func NewAuthController(authService *service.AuthService, userService *userService.UserService) *AuthController {
	return &AuthController{
		authService: authService,
		userService: userService,
	}
}

func (c *AuthController) Register(ctx *gin.Context) {
	var req dto.RegisterRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request",
			"details": err.Error(),
		})
		return
	}

	// Register user and get tokens + user ID
	tokens, userID, err := c.authService.Register(&req)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	// Get user profile data using the UserService
	userProfile, err := c.userService.GetProfile(userID)
	if err != nil {
		// If we can't get user profile, still return successful registration with tokens
		ctx.JSON(http.StatusCreated, gin.H{
			"message": "User registered successfully",
			"data":    tokens,
		})
		return
	}

	// Return both tokens and user data
	ctx.JSON(http.StatusCreated, gin.H{
		"message": "User registered successfully",
		"data": gin.H{
			"tokens": tokens,
			"user":   userProfile,
		},
	})
}

func (c *AuthController) Login(ctx *gin.Context) {
	var req dto.LoginRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request",
			"details": err.Error(),
		})
		return
	}

	// Login user and get tokens + user ID
	tokens, userID, err := c.authService.Login(&req)
	if err != nil {
		ctx.JSON(http.StatusUnauthorized, gin.H{
			"error": err.Error(),
		})
		return
	}

	// Get user profile data using the UserService
	userProfile, err := c.userService.GetProfile(userID)
	if err != nil {
		// If we can't get user profile, still return successful login with tokens
		ctx.JSON(http.StatusOK, gin.H{
			"message": "Login successful",
			"data":    tokens,
		})
		return
	}

	// Return both tokens and user data
	ctx.JSON(http.StatusOK, gin.H{
		"message": "Login successful",
		"data": gin.H{
			"tokens": tokens,
			"user":   userProfile,
		},
	})
}

func (c *AuthController) RefreshToken(ctx *gin.Context) {
	var req dto.RefreshRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request",
			"details": err.Error(),
		})
		return
	}

	tokens, err := c.authService.RefreshToken(req.RefreshToken)
	if err != nil {
		ctx.JSON(http.StatusUnauthorized, gin.H{
			"error": err.Error(),
		})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"message": "Token refreshed successfully",
		"data":    tokens,
	})
}

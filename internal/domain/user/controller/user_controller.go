package controller

import (
	"errors"
	"net/http"
	"strconv"

	"life-tracker-backend/internal/domain/user/dto"
	"life-tracker-backend/internal/domain/user/service"

	"github.com/gin-gonic/gin"
)

type UserController struct {
	userService *service.UserService
}

func NewUserController(userService *service.UserService) *UserController {
	return &UserController{
		userService: userService,
	}
}

func getUserID(ctx *gin.Context) (uint, error) {
	value, exists := ctx.Get("userID")
	if !exists {
		return 0, errors.New("user ID not found in context")
	}

	userID, ok := value.(uint)
	if !ok {
		return 0, errors.New("invalid user ID type in context")
	}

	return userID, nil
}

func getUserEmail(ctx *gin.Context) (string, error) {
	value, exists := ctx.Get("email")
	if !exists {
		return "", errors.New("email not found in context")
	}

	email, ok := value.(string)
	if !ok {
		return "", errors.New("invalid email type in context")
	}

	return email, nil
}

func (c *UserController) GetMyProfile(ctx *gin.Context) {
	userID, err := getUserID(ctx)
	if err != nil {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	email, err := getUserEmail(ctx)
	if err != nil {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	user, err := c.userService.GetMyProfile(userID, email)
	if err != nil {
		ctx.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"message": "Profile retrieved successfully",
		"data":    user,
	})
}

func (c *UserController) UpdateProfile(ctx *gin.Context) {
	userID, err := getUserID(ctx)
	if err != nil {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	email, err := getUserEmail(ctx)
	if err != nil {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	// Get username from context if available
	var username string
	if value, exists := ctx.Get("username"); exists {
		if u, ok := value.(string); ok {
			username = u
		}
	}

	var req dto.UpdateUserRequest
	if err = ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request",
			"details": err.Error(),
		})
		return
	}

	user, err := c.userService.UpdateProfile(userID, email, username, &req)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"message": "Profile updated successfully",
		"data":    user,
	})
}

func (c *UserController) GetAllUsers(ctx *gin.Context) {
	users, err := c.userService.GetAllUsers()
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"message": "Users retrieved successfully",
		"data":    users,
		"count":   len(users),
	})
}

func (c *UserController) GetUserByID(ctx *gin.Context) {
	idParam := ctx.Param("id")
	userID, err := strconv.ParseUint(idParam, 10, 32)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	user, err := c.userService.GetUserByID(uint(userID))
	if err != nil {
		ctx.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"message": "User retrieved successfully",
		"data":    user,
	})
}

func (c *UserController) DeleteUser(ctx *gin.Context) {
	idParam := ctx.Param("id")
	userID, err := strconv.ParseUint(idParam, 10, 32)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	if err := c.userService.DeleteUser(uint(userID)); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"message": "User deleted successfully"})
}

func (c *UserController) UploadProfileImage(ctx *gin.Context) {
	userID, err := getUserID(ctx)
	if err != nil {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	email, err := getUserEmail(ctx)
	if err != nil {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	file, err := ctx.FormFile("image")
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "No image file provided"})
		return
	}

	user, err := c.userService.UploadProfileImage(ctx.Request.Context(), userID, email, file)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"message": "Profile image uploaded successfully",
		"data":    user,
	})
}

func (c *UserController) SearchUsers(ctx *gin.Context) {
	var query dto.UserSearchQuery
	if err := ctx.ShouldBindQuery(&query); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid query parameters", "details": err.Error()})
		return
	}

	limit := query.Limit
	if limit == 0 {
		limit = 25
	}

	cards, err := c.userService.SearchUsers(query.Q, limit)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"data": cards, "count": len(cards)})
}

func (c *UserController) GetUserOrProfile(ctx *gin.Context) {
	param := ctx.Param("username")
	if id, err := strconv.ParseUint(param, 10, 32); err == nil {
		user, err := c.userService.GetUserByID(uint(id))
		if err != nil {
			ctx.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		ctx.JSON(http.StatusOK, gin.H{"message": "User retrieved successfully", "data": user})
		return
	}
	user, err := c.userService.GetUserByUsername(param)
	if err != nil {
		ctx.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	card := dto.PublicUserCard{
		ID:                   user.ID,
		FirstName:            user.FirstName,
		LastName:             user.LastName,
		Username:             user.Username,
		ProfilePicURL:        user.ProfilePicURL,
		ProfilePrivacyStatus: user.ProfilePrivacyStatus,
		IsFollowing:          false,
		FollowStatus:         "none",
	}
	ctx.JSON(http.StatusOK, gin.H{"data": dto.PublicProfileResponse{PublicUserCard: card}})
}

func (c *UserController) DeleteProfileImage(ctx *gin.Context) {
	userID, err := getUserID(ctx)
	if err != nil {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	if err := c.userService.DeleteProfileImage(ctx.Request.Context(), userID); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"message": "Profile image deleted successfully"})
}

func (c *UserController) CheckUsernameAvailability(ctx *gin.Context) {
	username := ctx.Query("username")
	if username == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "username query parameter is required"})
		return
	}

	exists, err := c.userService.UsernameExists(username)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "failed to check username availability"})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"available": !exists,
		"username":  username,
	})
}

func (c *UserController) CheckEmailAvailability(ctx *gin.Context) {
	email := ctx.Query("email")
	if email == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "email query parameter is required"})
		return
	}

	exists, err := c.userService.EmailExists(email)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "failed to check email availability"})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"available": !exists,
		"email":     email,
	})
}

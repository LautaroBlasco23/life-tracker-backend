package controller

import (
	"errors"
	"net/http"
	"strconv"

	"life-tracker-backend/internal/domain/activity/dto"
	"life-tracker-backend/internal/domain/activity/service"

	"github.com/gin-gonic/gin"
)

type ActivityController struct {
	activityService *service.ActivityService
}

func NewActivityController(activityService *service.ActivityService) *ActivityController {
	return &ActivityController{
		activityService: activityService,
	}
}

func getActivityUserID(ctx *gin.Context) (uint, error) {
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

func (c *ActivityController) CreateActivity(ctx *gin.Context) {
	userID, err := getActivityUserID(ctx)
	if err != nil {
		ctx.JSON(http.StatusUnauthorized, gin.H{
			"error": err.Error(),
		})
		return
	}

	var req dto.CreateActivityRequest
	if bindErr := ctx.ShouldBindJSON(&req); bindErr != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request",
			"details": bindErr.Error(),
		})
		return
	}

	activity, err := c.activityService.CreateActivity(userID, &req)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	ctx.JSON(http.StatusCreated, gin.H{
		"message": "Activity created successfully",
		"data":    activity,
	})
}

func (c *ActivityController) GetUserActivities(ctx *gin.Context) {
	userID, err := getActivityUserID(ctx)
	if err != nil {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	var filter dto.ActivityFilter
	if err := ctx.ShouldBindQuery(&filter); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid filter parameters"})
		return
	}

	activities, err := c.activityService.GetUserActivitiesFiltered(userID, &filter)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"message": "Activities retrieved successfully",
		"data":    activities,
		"count":   len(activities),
	})
}

func (c *ActivityController) GetTodayActivities(ctx *gin.Context) {
	userID, err := getActivityUserID(ctx)
	if err != nil {
		ctx.JSON(http.StatusUnauthorized, gin.H{
			"error": err.Error(),
		})
		return
	}

	activities, err := c.activityService.GetTodayActivities(userID)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"message": "Today's activities retrieved successfully",
		"data":    activities,
		"count":   len(activities),
	})
}

func (c *ActivityController) GetActivity(ctx *gin.Context) {
	userID, err := getActivityUserID(ctx)
	if err != nil {
		ctx.JSON(http.StatusUnauthorized, gin.H{
			"error": err.Error(),
		})
		return
	}

	idParam := ctx.Param("id")
	activityID, err := strconv.ParseUint(idParam, 10, 32)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid activity ID",
		})
		return
	}

	activity, err := c.activityService.GetActivity(userID, uint(activityID))
	if err != nil {
		ctx.JSON(http.StatusNotFound, gin.H{
			"error": err.Error(),
		})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"message": "Activity retrieved successfully",
		"data":    activity,
	})
}

func (c *ActivityController) UpdateActivity(ctx *gin.Context) {
	userID, err := getActivityUserID(ctx)
	if err != nil {
		ctx.JSON(http.StatusUnauthorized, gin.H{
			"error": err.Error(),
		})
		return
	}

	idParam := ctx.Param("id")
	activityID, err := strconv.ParseUint(idParam, 10, 32)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid activity ID",
		})
		return
	}

	var req dto.UpdateActivityRequest
	if bindErr := ctx.ShouldBindJSON(&req); bindErr != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request",
			"details": bindErr.Error(),
		})
		return
	}

	activity, err := c.activityService.UpdateActivity(userID, uint(activityID), &req)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"message": "Activity updated successfully",
		"data":    activity,
	})
}

func (c *ActivityController) DeleteActivity(ctx *gin.Context) {
	userID, err := getActivityUserID(ctx)
	if err != nil {
		ctx.JSON(http.StatusUnauthorized, gin.H{
			"error": err.Error(),
		})
		return
	}

	idParam := ctx.Param("id")
	activityID, err := strconv.ParseUint(idParam, 10, 32)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid activity ID",
		})
		return
	}

	if deleteErr := c.activityService.DeleteActivity(userID, uint(activityID)); deleteErr != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error": deleteErr.Error(),
		})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"message": "Activity deleted successfully",
	})
}

func (c *ActivityController) RecordActivity(ctx *gin.Context) {
	userID, err := getActivityUserID(ctx)
	if err != nil {
		ctx.JSON(http.StatusUnauthorized, gin.H{
			"error": err.Error(),
		})
		return
	}

	idParam := ctx.Param("id")
	activityID, err := strconv.ParseUint(idParam, 10, 32)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid activity ID",
		})
		return
	}

	var req dto.RecordActivityRequest
	if bindErr := ctx.ShouldBindJSON(&req); bindErr != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request",
			"details": bindErr.Error(),
		})
		return
	}

	record, err := c.activityService.RecordActivity(userID, uint(activityID), &req)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	ctx.JSON(http.StatusCreated, gin.H{
		"message": "Activity completion recorded successfully",
		"data":    record,
	})
}

func (c *ActivityController) GetActivityRecords(ctx *gin.Context) {
	userID, err := getActivityUserID(ctx)
	if err != nil {
		ctx.JSON(http.StatusUnauthorized, gin.H{
			"error": err.Error(),
		})
		return
	}

	idParam := ctx.Param("id")
	activityID, err := strconv.ParseUint(idParam, 10, 32)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid activity ID",
		})
		return
	}

	limit := 50
	if limitParam := ctx.Query("limit"); limitParam != "" {
		if l, parseErr := strconv.Atoi(limitParam); parseErr == nil && l > 0 {
			limit = l
		}
	}

	records, err := c.activityService.GetActivityRecords(userID, uint(activityID), limit)
	if err != nil {
		ctx.JSON(http.StatusNotFound, gin.H{
			"error": err.Error(),
		})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"message": "Activity records retrieved successfully",
		"data":    records,
		"count":   len(records),
	})
}

func (c *ActivityController) GetActivityStats(ctx *gin.Context) {
	userID, err := getActivityUserID(ctx)
	if err != nil {
		ctx.JSON(http.StatusUnauthorized, gin.H{
			"error": err.Error(),
		})
		return
	}

	idParam := ctx.Param("id")
	activityID, err := strconv.ParseUint(idParam, 10, 32)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid activity ID",
		})
		return
	}

	stats, err := c.activityService.GetActivityStats(userID, uint(activityID))
	if err != nil {
		ctx.JSON(http.StatusNotFound, gin.H{
			"error": err.Error(),
		})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"message": "Activity stats retrieved successfully",
		"data":    stats,
	})
}

func (c *ActivityController) RevertLastCompletion(ctx *gin.Context) {
	userID, err := getActivityUserID(ctx)
	if err != nil {
		ctx.JSON(http.StatusUnauthorized, gin.H{
			"error": err.Error(),
		})
		return
	}

	idParam := ctx.Param("id")
	activityID, err := strconv.ParseUint(idParam, 10, 32)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid activity ID",
		})
		return
	}

	if revertErr := c.activityService.RevertLastCompletion(userID, uint(activityID)); revertErr != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error": revertErr.Error(),
		})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"message": "Latest completion reverted successfully",
	})
}

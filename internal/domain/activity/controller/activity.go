package controller

import (
	"life-tracker-backend/internal/domain/activity/dto"
	"life-tracker-backend/internal/domain/activity/service"
	"net/http"
	"strconv"

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

// Activity CRUD endpoints
func (c *ActivityController) CreateActivity(ctx *gin.Context) {
	userID := ctx.MustGet("userID").(uint)

	var req dto.CreateActivityRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request",
			"details": err.Error(),
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
	userID := ctx.MustGet("userID").(uint)

	includeInactive := ctx.Query("include_inactive") == "true"

	activities, err := c.activityService.GetUserActivities(userID, includeInactive)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"message": "Activities retrieved successfully",
		"data":    activities,
		"count":   len(activities),
	})
}

func (c *ActivityController) GetActivity(ctx *gin.Context) {
	userID := ctx.MustGet("userID").(uint)

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
	userID := ctx.MustGet("userID").(uint)

	idParam := ctx.Param("id")
	activityID, err := strconv.ParseUint(idParam, 10, 32)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid activity ID",
		})
		return
	}

	var req dto.UpdateActivityRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request",
			"details": err.Error(),
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
	userID := ctx.MustGet("userID").(uint)

	idParam := ctx.Param("id")
	activityID, err := strconv.ParseUint(idParam, 10, 32)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid activity ID",
		})
		return
	}

	if err := c.activityService.DeleteActivity(userID, uint(activityID)); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"message": "Activity deleted successfully",
	})
}

// Activity Records endpoints
func (c *ActivityController) RecordActivity(ctx *gin.Context) {
	userID := ctx.MustGet("userID").(uint)

	idParam := ctx.Param("id")
	activityID, err := strconv.ParseUint(idParam, 10, 32)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid activity ID",
		})
		return
	}

	var req dto.RecordActivityRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request",
			"details": err.Error(),
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
	userID := ctx.MustGet("userID").(uint)

	idParam := ctx.Param("id")
	activityID, err := strconv.ParseUint(idParam, 10, 32)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid activity ID",
		})
		return
	}

	// Get limit from query params
	limit := 50 // default
	if limitParam := ctx.Query("limit"); limitParam != "" {
		if l, err := strconv.Atoi(limitParam); err == nil && l > 0 {
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
	userID := ctx.MustGet("userID").(uint)

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

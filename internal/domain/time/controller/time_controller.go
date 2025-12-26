package controller

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"life-tracker-backend/internal/domain/time/dto"
	"life-tracker-backend/internal/domain/time/service"
	"life-tracker-backend/internal/middleware"

	"github.com/gin-gonic/gin"
)

type TimeController struct {
	timeService *service.TimeService
}

func NewTimeController(timeService *service.TimeService) *TimeController {
	return &TimeController{timeService: timeService}
}

func getTimeUserID(ctx *gin.Context) (uint, error) {
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

func getTimezone(ctx *gin.Context) *time.Location {
	return middleware.GetTimezoneFromContext(ctx)
}

func (c *TimeController) CreateRecord(ctx *gin.Context) {
	userID, err := getTimeUserID(ctx)
	if err != nil {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	var req dto.CreateTimeRecordRequest
	if err = ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request",
			"details": err.Error(),
		})
		return
	}

	record, err := c.timeService.CreateRecord(userID, &req)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusCreated, gin.H{
		"message": "Time record created successfully",
		"data":    record,
	})
}

func (c *TimeController) GetRecords(ctx *gin.Context) {
	userID, err := getTimeUserID(ctx)
	if err != nil {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	filter := &dto.TimeRecordFilter{
		Category: ctx.Query("category"),
	}

	if monthParam := ctx.Query("month"); monthParam != "" {
		if m, parseErr := strconv.Atoi(monthParam); parseErr == nil && m >= 1 && m <= 12 {
			filter.Month = &m
		}
	}

	if yearParam := ctx.Query("year"); yearParam != "" {
		if y, parseErr := strconv.Atoi(yearParam); parseErr == nil && y >= 2000 && y <= 2100 {
			filter.Year = &y
		}
	}

	if startDate := ctx.Query("start_date"); startDate != "" {
		if parsed, parseErr := time.Parse("2006-01-02", startDate); parseErr == nil {
			filter.StartDate = &parsed
		}
	}

	if endDate := ctx.Query("end_date"); endDate != "" {
		if parsed, parseErr := time.Parse("2006-01-02", endDate); parseErr == nil {
			filter.EndDate = &parsed
		}
	}

	loc := getTimezone(ctx)
	records, err := c.timeService.GetRecords(userID, filter, loc)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"message": "Time records retrieved successfully",
		"data":    records,
		"count":   len(records),
	})
}

func (c *TimeController) GetRecord(ctx *gin.Context) {
	userID, err := getTimeUserID(ctx)
	if err != nil {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	recordID, err := strconv.ParseUint(ctx.Param("id"), 10, 32)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid record ID"})
		return
	}

	record, err := c.timeService.GetRecord(userID, uint(recordID))
	if err != nil {
		ctx.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"message": "Time record retrieved successfully",
		"data":    record,
	})
}

func (c *TimeController) UpdateRecord(ctx *gin.Context) {
	userID, err := getTimeUserID(ctx)
	if err != nil {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	recordID, err := strconv.ParseUint(ctx.Param("id"), 10, 32)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid record ID"})
		return
	}

	var req dto.UpdateTimeRecordRequest
	if err = ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request",
			"details": err.Error(),
		})
		return
	}

	record, err := c.timeService.UpdateRecord(userID, uint(recordID), &req)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"message": "Time record updated successfully",
		"data":    record,
	})
}

func (c *TimeController) DeleteRecord(ctx *gin.Context) {
	userID, err := getTimeUserID(ctx)
	if err != nil {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	recordID, err := strconv.ParseUint(ctx.Param("id"), 10, 32)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid record ID"})
		return
	}

	if err := c.timeService.DeleteRecord(userID, uint(recordID)); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"message": "Time record deleted successfully"})
}

func (c *TimeController) GetStats(ctx *gin.Context) {
	userID, err := getTimeUserID(ctx)
	if err != nil {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	stats, err := c.timeService.GetStats(userID)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"message": "Time stats retrieved successfully",
		"data":    stats,
	})
}

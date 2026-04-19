package controller

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"life-tracker-backend/internal/domain/screentime/dto"
	"life-tracker-backend/internal/domain/screentime/service"

	"github.com/gin-gonic/gin"
)

type VisitController struct {
	visitService *service.VisitService
}

func NewVisitController(s *service.VisitService) *VisitController {
	return &VisitController{visitService: s}
}

func getScreenTimeUserID(ctx *gin.Context) (uint, error) {
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

func parseDate(s string) *time.Time {
	if s == "" {
		return nil
	}
	parsed, err := time.Parse("2006-01-02", s)
	if err != nil {
		return nil
	}
	return &parsed
}

func (c *VisitController) RecordVisit(ctx *gin.Context) {
	userID, err := getScreenTimeUserID(ctx)
	if err != nil {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	var req dto.RecordVisitRequest
	if bindErr := ctx.ShouldBindJSON(&req); bindErr != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request",
			"details": bindErr.Error(),
		})
		return
	}

	result, err := c.visitService.RecordVisit(userID, &req)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusCreated, gin.H{
		"message": "Visit recorded successfully",
		"data":    result,
	})
}

func (c *VisitController) GetVisits(ctx *gin.Context) {
	userID, err := getScreenTimeUserID(ctx)
	if err != nil {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	domain := ctx.Query("domain")
	from := parseDate(ctx.Query("from"))
	to := parseDate(ctx.Query("to"))

	limit := 100
	if l := ctx.Query("limit"); l != "" {
		if v, convErr := strconv.Atoi(l); convErr == nil && v > 0 {
			limit = v
		}
	}

	visits, err := c.visitService.GetVisits(userID, domain, from, to, limit)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"message": "Visits retrieved successfully",
		"data":    visits,
		"count":   len(visits),
	})
}

func (c *VisitController) GetStats(ctx *gin.Context) {
	userID, err := getScreenTimeUserID(ctx)
	if err != nil {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	from := parseDate(ctx.Query("from"))
	to := parseDate(ctx.Query("to"))

	stats, err := c.visitService.GetStats(userID, from, to)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"message": "Stats retrieved successfully",
		"data":    stats,
		"count":   len(stats),
	})
}

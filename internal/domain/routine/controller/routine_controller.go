package controller

import (
	"errors"
	"net/http"
	"strconv"

	"life-tracker-backend/internal/domain/routine/dto"
	"life-tracker-backend/internal/domain/routine/service"

	"github.com/gin-gonic/gin"
)

type RoutineController struct {
	routineService *service.RoutineService
}

func NewRoutineController(routineService *service.RoutineService) *RoutineController {
	return &RoutineController{routineService: routineService}
}

func getRoutineUserID(ctx *gin.Context) (uint, error) {
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

func (c *RoutineController) CreateRoutine(ctx *gin.Context) {
	userID, err := getRoutineUserID(ctx)
	if err != nil {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var req dto.CreateRoutineRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request", "details": err.Error()})
		return
	}

	routine, err := c.routineService.CreateRoutine(userID, &req)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusCreated, gin.H{"message": "Routine created successfully", "data": routine})
}

func (c *RoutineController) GetMyRoutines(ctx *gin.Context) {
	userID, err := getRoutineUserID(ctx)
	if err != nil {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	routines, err := c.routineService.GetMyRoutines(userID)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"data": routines, "count": len(routines)})
}

func (c *RoutineController) GetRoutine(ctx *gin.Context) {
	userID, err := getRoutineUserID(ctx)
	if err != nil {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	id, err := strconv.ParseUint(ctx.Param("id"), 10, 32)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid routine ID"})
		return
	}

	routine, err := c.routineService.GetRoutine(userID, uint(id))
	if err != nil {
		ctx.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"data": routine})
}

func (c *RoutineController) UpdateRoutine(ctx *gin.Context) {
	userID, err := getRoutineUserID(ctx)
	if err != nil {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	id, err := strconv.ParseUint(ctx.Param("id"), 10, 32)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid routine ID"})
		return
	}

	var req dto.UpdateRoutineRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request", "details": err.Error()})
		return
	}

	routine, err := c.routineService.UpdateRoutine(userID, uint(id), &req)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"message": "Routine updated successfully", "data": routine})
}

func (c *RoutineController) DeleteRoutine(ctx *gin.Context) {
	userID, err := getRoutineUserID(ctx)
	if err != nil {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	id, err := strconv.ParseUint(ctx.Param("id"), 10, 32)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid routine ID"})
		return
	}

	if err := c.routineService.DeleteRoutine(userID, uint(id)); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"message": "Routine deleted successfully"})
}

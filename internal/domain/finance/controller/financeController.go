package controller

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"life-tracker-backend/internal/domain/finance/dto"
	"life-tracker-backend/internal/domain/finance/service"

	"github.com/gin-gonic/gin"
)

type FinanceController struct {
	financeService *service.FinanceService
}

func NewFinanceController(financeService *service.FinanceService) *FinanceController {
	return &FinanceController{
		financeService: financeService,
	}
}

func getFinanceUserID(ctx *gin.Context) (uint, error) {
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

func (c *FinanceController) GetCategories(ctx *gin.Context) {
	var transactionType *string
	if t := ctx.Query("type"); t != "" {
		transactionType = &t
	}

	categories, err := c.financeService.GetCategories(transactionType)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"message": "Categories retrieved successfully",
		"data":    categories,
		"count":   len(categories),
	})
}

func (c *FinanceController) CreateTransaction(ctx *gin.Context) {
	userID, err := getFinanceUserID(ctx)
	if err != nil {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	var req dto.CreateTransactionRequest
	if bindErr := ctx.ShouldBindJSON(&req); bindErr != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request",
			"details": bindErr.Error(),
		})
		return
	}

	transaction, err := c.financeService.CreateTransaction(userID, &req)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusCreated, gin.H{
		"message": "Transaction created successfully",
		"data":    transaction,
	})
}

func (c *FinanceController) GetTransactions(ctx *gin.Context) {
	userID, err := getFinanceUserID(ctx)
	if err != nil {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	var transactionType *string
	if t := ctx.Query("type"); t != "" {
		transactionType = &t
	}

	var startDate, endDate *time.Time
	var month, year *int
	var categoryID *uint

	if monthParam := ctx.Query("month"); monthParam != "" {
		if m, parseErr := strconv.Atoi(monthParam); parseErr == nil && m >= 1 && m <= 12 {
			month = &m
		}
	}

	if yearParam := ctx.Query("year"); yearParam != "" {
		if y, parseErr := strconv.Atoi(yearParam); parseErr == nil && y >= 2000 && y <= 2100 {
			year = &y
		}
	}

	if categoryParam := ctx.Query("category_id"); categoryParam != "" {
		if cid, parseErr := strconv.ParseUint(categoryParam, 10, 32); parseErr == nil {
			catID := uint(cid)
			categoryID = &catID
		}
	}

	if start := ctx.Query("start_date"); start != "" {
		if parsed, parseErr := time.Parse("2006-01-02", start); parseErr == nil {
			startDate = &parsed
		}
	}
	if end := ctx.Query("end_date"); end != "" {
		if parsed, parseErr := time.Parse("2006-01-02", end); parseErr == nil {
			endDate = &parsed
		}
	}

	limit := 100
	if limitParam := ctx.Query("limit"); limitParam != "" {
		if l, parseErr := strconv.Atoi(limitParam); parseErr == nil && l > 0 {
			limit = l
		}
	}

	transactions, err := c.financeService.GetTransactions(userID, transactionType, startDate, endDate, month, year, categoryID, limit)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"message": "Transactions retrieved successfully",
		"data":    transactions,
		"count":   len(transactions),
	})
}

func (c *FinanceController) GetTransaction(ctx *gin.Context) {
	userID, err := getFinanceUserID(ctx)
	if err != nil {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	transactionID := ctx.Param("id")
	transaction, err := c.financeService.GetTransaction(userID, transactionID)
	if err != nil {
		ctx.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"message": "Transaction retrieved successfully",
		"data":    transaction,
	})
}

func (c *FinanceController) UpdateTransaction(ctx *gin.Context) {
	userID, err := getFinanceUserID(ctx)
	if err != nil {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	transactionID := ctx.Param("id")

	var req dto.UpdateTransactionRequest
	if bindErr := ctx.ShouldBindJSON(&req); bindErr != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request",
			"details": bindErr.Error(),
		})
		return
	}

	transaction, err := c.financeService.UpdateTransaction(userID, transactionID, &req)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"message": "Transaction updated successfully",
		"data":    transaction,
	})
}

func (c *FinanceController) DeleteTransaction(ctx *gin.Context) {
	userID, err := getFinanceUserID(ctx)
	if err != nil {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	transactionID := ctx.Param("id")
	if deleteErr := c.financeService.DeleteTransaction(userID, transactionID); deleteErr != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": deleteErr.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"message": "Transaction deleted successfully",
	})
}

func (c *FinanceController) GetFinanceSummary(ctx *gin.Context) {
	userID, err := getFinanceUserID(ctx)
	if err != nil {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	startDateStr := ctx.Query("start_date")
	endDateStr := ctx.Query("end_date")

	if startDateStr == "" || endDateStr == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error": "start_date and end_date are required (format: 2006-01-02)",
		})
		return
	}

	startDate, err := time.Parse("2006-01-02", startDateStr)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid start_date format"})
		return
	}

	endDate, err := time.Parse("2006-01-02", endDateStr)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid end_date format"})
		return
	}

	summary, err := c.financeService.GetFinanceSummary(userID, startDate, endDate)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"message": "Finance summary retrieved successfully",
		"data":    summary,
	})
}

func (c *FinanceController) GetMonthlyStats(ctx *gin.Context) {
	userID, err := getFinanceUserID(ctx)
	if err != nil {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	yearParam := ctx.Query("year")
	if yearParam == "" {
		yearParam = strconv.Itoa(time.Now().Year())
	}

	year, err := strconv.Atoi(yearParam)
	if err != nil || year < 2000 || year > 2100 {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid year"})
		return
	}

	stats, err := c.financeService.GetMonthlyStats(userID, year)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"message": "Monthly stats retrieved successfully",
		"data":    stats,
		"year":    year,
	})
}

package controller

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"life-tracker-backend/internal/domain/finance/dto"
	"life-tracker-backend/internal/domain/finance/service"
	"life-tracker-backend/internal/middleware"

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

func getTimezone(ctx *gin.Context) *time.Location {
	return middleware.GetTimezoneFromContext(ctx)
}

type transactionFilters struct {
	transactionType *string
	frequency       *string
	startDate       *time.Time
	endDate         *time.Time
	month           *int
	year            *int
	category        *string
	limit           int
}

func parseTransactionFilters(ctx *gin.Context) transactionFilters {
	f := transactionFilters{limit: 100}

	if t := ctx.Query("type"); t != "" {
		f.transactionType = &t
	}
	if freq := ctx.Query("frequency"); freq != "" {
		f.frequency = &freq
	}

	f.month = parseIntInRange(ctx.Query("month"), 1, 12)
	f.year = parseIntInRange(ctx.Query("year"), 2000, 2100)
	f.category = parseString(ctx.Query("category"))
	f.startDate = parseDate(ctx.Query("start_date"))
	f.endDate = parseDate(ctx.Query("end_date"))

	if l := parseIntPositive(ctx.Query("limit")); l != nil {
		f.limit = *l
	}

	return f
}

func parseIntInRange(s string, min, max int) *int {
	if s == "" {
		return nil
	}
	v, err := strconv.Atoi(s)
	if err != nil || v < min || v > max {
		return nil
	}
	return &v
}

func parseIntPositive(s string) *int {
	if s == "" {
		return nil
	}
	v, err := strconv.Atoi(s)
	if err != nil || v <= 0 {
		return nil
	}
	return &v
}

func parseString(s string) *string {
	if s == "" {
		return nil
	}
	return &s
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

func (c *FinanceController) GetCategories(ctx *gin.Context) {
	var transactionType, frequency *string
	if t := ctx.Query("type"); t != "" {
		transactionType = &t
	}
	if f := ctx.Query("frequency"); f != "" {
		frequency = &f
	}

	categories := c.financeService.GetCategories(transactionType, frequency)

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

	filters := parseTransactionFilters(ctx)
	loc := getTimezone(ctx)

	transactions, err := c.financeService.GetTransactions(
		userID,
		filters.transactionType,
		filters.frequency,
		filters.startDate,
		filters.endDate,
		filters.month,
		filters.year,
		filters.category,
		filters.limit,
		loc,
	)
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

func (c *FinanceController) GetFixedTransactions(ctx *gin.Context) {
	userID, err := getFinanceUserID(ctx)
	if err != nil {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	loc := getTimezone(ctx)
	month := parseIntInRange(ctx.Query("month"), 1, 12)
	year := parseIntInRange(ctx.Query("year"), 2000, 2100)

	transactions, err := c.financeService.GetFixedTransactions(userID, month, year, loc)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"message": "Fixed transactions retrieved successfully",
		"data":    transactions,
		"count":   len(transactions),
	})
}

func (c *FinanceController) GetFixedTransactionWithPayments(ctx *gin.Context) {
	userID, err := getFinanceUserID(ctx)
	if err != nil {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	transactionID := ctx.Param("id")
	transaction, err := c.financeService.GetFixedTransactionWithPayments(userID, transactionID)
	if err != nil {
		ctx.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"message": "Fixed transaction retrieved successfully",
		"data":    transaction,
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

// Payment Endpoints

func (c *FinanceController) CreatePayment(ctx *gin.Context) {
	userID, err := getFinanceUserID(ctx)
	if err != nil {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	var req dto.CreatePaymentRequest
	if bindErr := ctx.ShouldBindJSON(&req); bindErr != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request",
			"details": bindErr.Error(),
		})
		return
	}

	payment, err := c.financeService.CreatePayment(userID, &req)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusCreated, gin.H{
		"message": "Payment created successfully",
		"data":    payment,
	})
}

func (c *FinanceController) GetPayments(ctx *gin.Context) {
	userID, err := getFinanceUserID(ctx)
	if err != nil {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	var transactionID *string
	if tid := ctx.Query("transaction_id"); tid != "" {
		transactionID = &tid
	}

	startDate := parseDate(ctx.Query("start_date"))
	endDate := parseDate(ctx.Query("end_date"))

	limit := 100
	if l := parseIntPositive(ctx.Query("limit")); l != nil {
		limit = *l
	}

	payments, err := c.financeService.GetPayments(userID, transactionID, startDate, endDate, limit)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"message": "Payments retrieved successfully",
		"data":    payments,
		"count":   len(payments),
	})
}

func (c *FinanceController) DeletePayment(ctx *gin.Context) {
	userID, err := getFinanceUserID(ctx)
	if err != nil {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	paymentID := ctx.Param("id")
	if deleteErr := c.financeService.DeletePayment(userID, paymentID); deleteErr != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": deleteErr.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"message": "Payment deleted successfully",
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

	loc := getTimezone(ctx)
	summary, err := c.financeService.GetFinanceSummary(userID, startDate, endDate, loc)
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

	loc := getTimezone(ctx)

	yearParam := ctx.Query("year")
	if yearParam == "" {
		yearParam = strconv.Itoa(time.Now().In(loc).Year())
	}

	year, err := strconv.Atoi(yearParam)
	if err != nil || year < 2000 || year > 2100 {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid year"})
		return
	}

	stats, err := c.financeService.GetMonthlyStats(userID, year, loc)
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

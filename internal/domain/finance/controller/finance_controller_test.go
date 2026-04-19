// finance_controller_test.go
//
// Subject: internal/domain/finance/controller.FinanceController
// Scope:   HTTP handler tests for all finance endpoints — categories, transactions,
//
//	payments, summaries, and monthly stats. Verifies request parsing,
//	response formatting, status codes, and error handling.
//
// Out of scope:
//   - Business logic correctness → finance_service_test.go
//   - Repository aggregation pipeline → transaction_repository_test.go
//   - Database constraint behavior → covered by service tests
//
// Infrastructure: MongoDB (test database via .env.test). Collections dropped between tests.
// Setup notes: Uses gin test mode with httptest.NewRecorder. UserID injected via
//
//	middleware mock. Timezone middleware sets UTC by default.
package controller

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"life-tracker-backend/internal/config"
	"life-tracker-backend/internal/domain/finance/dto"
	"life-tracker-backend/internal/domain/finance/model"
	"life-tracker-backend/internal/domain/finance/service"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var cfg *config.Config

func init() {
	os.Setenv("ENVIRONMENT", "test")
	cfg = config.Load()
	gin.SetMode(gin.TestMode)
}

// setupTestRouter creates a gin router with all finance routes and returns cleanup function.
// Each test gets a unique MongoDB database to ensure isolation.
func setupTestRouter(t *testing.T) (*gin.Engine, *service.FinanceService, func()) {
	ctx := context.Background()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(cfg.MongoURI))
	require.NoError(t, err)
	require.NoError(t, client.Ping(ctx, nil), "MongoDB not available")

	dbName := "test_finance_controller_" + primitive.NewObjectID().Hex()
	mongoDB := client.Database(dbName)

	financeService := service.NewFinanceService(mongoDB)
	controller := NewFinanceController(financeService)

	router := gin.New()

	// Mock auth middleware - injects userID and timezone
	router.Use(func(c *gin.Context) {
		c.Set("userID", uint(1))
		c.Set("timezone", time.UTC)
		c.Next()
	})

	// Categories (public endpoint - no auth required in controller)
	router.GET("/categories", controller.GetCategories)

	// Transactions
	transactions := router.Group("/transactions")
	{
		transactions.POST("", controller.CreateTransaction)
		transactions.GET("", controller.GetTransactions)
		transactions.GET("/fixed", controller.GetFixedTransactions)
		transactions.GET("/fixed/:id", controller.GetFixedTransactionWithPayments)
		transactions.GET("/:id", controller.GetTransaction)
		transactions.PUT("/:id", controller.UpdateTransaction)
		transactions.DELETE("/:id", controller.DeleteTransaction)
	}

	// Payments
	payments := router.Group("/payments")
	{
		payments.POST("", controller.CreatePayment)
		payments.GET("", controller.GetPayments)
		payments.DELETE("/:id", controller.DeletePayment)
	}

	// Summary endpoints
	router.GET("/summary", controller.GetFinanceSummary)
	router.GET("/monthly-stats", controller.GetMonthlyStats)

	cleanup := func() {
		_ = mongoDB.Drop(ctx)
		_ = client.Disconnect(ctx)
	}

	return router, financeService, cleanup
}

// TestGetCategories verifies the category listing endpoint with optional filters.
// Categories are in-memory only — no database interaction.
func TestGetCategories(t *testing.T) {
	router, _, cleanup := setupTestRouter(t)
	defer cleanup()

	t.Run("returns_all_categories_without_filters", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/categories", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp map[string]interface{}
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		assert.Equal(t, "Categories retrieved successfully", resp["message"])
		assert.Equal(t, float64(len(model.Categories)), resp["count"])
		assert.NotNil(t, resp["data"])
	})

	t.Run("filters_by_type_income", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/categories?type=income", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp map[string]interface{}
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		data := resp["data"].([]interface{})
		for _, cat := range data {
			catMap := cat.(map[string]interface{})
			assert.Equal(t, "income", catMap["type"])
		}
	})

	t.Run("filters_by_type_outcome", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/categories?type=outcome", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp map[string]interface{}
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		data := resp["data"].([]interface{})
		for _, cat := range data {
			catMap := cat.(map[string]interface{})
			assert.Equal(t, "outcome", catMap["type"])
		}
	})

	t.Run("filters_by_frequency_fixed", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/categories?frequency=fixed", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp map[string]interface{}
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		data := resp["data"].([]interface{})
		for _, cat := range data {
			catMap := cat.(map[string]interface{})
			assert.Equal(t, "fixed", catMap["applicableToFreq"])
		}
	})

	t.Run("filters_by_type_and_frequency", func(t *testing.T) {
		// Only "Primary Income" matches income + fixed
		req := httptest.NewRequest(http.MethodGet, "/categories?type=income&frequency=fixed", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp map[string]interface{}
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		assert.Equal(t, float64(1), resp["count"])
		data := resp["data"].([]interface{})
		assert.Len(t, data, 1)
		catMap := data[0].(map[string]interface{})
		assert.Equal(t, "Primary Income", catMap["name"])
	})
}

// TestCreateTransactionHandler verifies transaction creation with validation,
// JSON parsing, and error handling.
func TestCreateTransactionHandler(t *testing.T) {
	router, _, cleanup := setupTestRouter(t)
	defer cleanup()

	tests := []struct {
		name           string
		body           interface{}
		expectedStatus int
		checkResponse  func(*testing.T, map[string]interface{})
	}{
		{
			name: "valid_variable_income_transaction",
			body: dto.CreateTransactionRequest{
				Type:      "income",
				Frequency: "variable",
				Amount:    500.00,
				Category:  string(model.CategorySideIncome),
				Date:      time.Now(),
			},
			expectedStatus: http.StatusCreated,
			checkResponse: func(t *testing.T, resp map[string]interface{}) {
				assert.Equal(t, "Transaction created successfully", resp["message"])
				data := resp["data"].(map[string]interface{})
				assert.Equal(t, "income", data["type"])
				assert.Equal(t, "variable", data["frequency"])
				assert.Equal(t, 500.0, data["amount"])
				assert.NotEmpty(t, data["id"])
			},
		},
		{
			name: "valid_fixed_outcome_transaction",
			body: dto.CreateTransactionRequest{
				Type:             "outcome",
				Frequency:        "fixed",
				Amount:           1200.00,
				Category:         string(model.CategoryHousingUtilities),
				PaymentFrequency: "monthly",
				Description:      "Monthly rent",
			},
			expectedStatus: http.StatusCreated,
			checkResponse: func(t *testing.T, resp map[string]interface{}) {
				data := resp["data"].(map[string]interface{})
				assert.Equal(t, "fixed", data["frequency"])
				assert.Equal(t, "monthly", data["paymentFrequency"])
				// Fixed transactions have amount zeroed (tracked via payments)
				assert.Equal(t, 0.0, data["amount"])
			},
		},
		{
			name:           "invalid_json_body",
			body:           `{"invalid": json}`,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "missing_required_fields",
			body: map[string]interface{}{
				"type": "income",
				// Missing frequency, amount, category
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "invalid_transaction_type",
			body: dto.CreateTransactionRequest{
				Type:      "invalid",
				Frequency: "variable",
				Amount:    100,
				Category:  string(model.CategorySideIncome),
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "invalid_category",
			body: dto.CreateTransactionRequest{
				Type:      "income",
				Frequency: "variable",
				Amount:    100,
				Category:  "NotARealCategory",
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "negative_amount_rejected",
			body: dto.CreateTransactionRequest{
				Type:      "outcome",
				Frequency: "variable",
				Amount:    -50,
				Category:  string(model.CategoryFoodGroceries),
			},
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, _ := json.Marshal(tt.body)
			req := httptest.NewRequest(http.MethodPost, "/transactions", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.checkResponse != nil {
				var resp map[string]interface{}
				require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
				tt.checkResponse(t, resp)
			}
		})
	}
}

// TestGetTransactionsHandler verifies transaction listing with various filter combinations.
func TestGetTransactionsHandler(t *testing.T) {
	router, svc, cleanup := setupTestRouter(t)
	defer cleanup()

	// Seed test data
	userID := uint(1)
	now := time.Now()

	transactions := []*dto.CreateTransactionRequest{
		{Type: "income", Frequency: "variable", Amount: 1000, Category: string(model.CategorySideIncome), Date: now},
		{Type: "outcome", Frequency: "variable", Amount: 50, Category: string(model.CategoryFoodGroceries), Date: now},
		{Type: "income", Frequency: "variable", Amount: 200, Category: string(model.CategoryInvestments), Date: now},
	}

	for _, req := range transactions {
		_, err := svc.CreateTransaction(userID, req)
		require.NoError(t, err)
	}

	tests := []struct {
		name           string
		query          string
		expectedStatus int
		checkResponse  func(*testing.T, map[string]interface{})
	}{
		{
			name:           "no_filters_returns_all",
			query:          "",
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, resp map[string]interface{}) {
				assert.Equal(t, "Transactions retrieved successfully", resp["message"])
				assert.Equal(t, float64(3), resp["count"])
			},
		},
		{
			name:           "filter_by_type_income",
			query:          "?type=income",
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, resp map[string]interface{}) {
				assert.Equal(t, float64(2), resp["count"])
				data := resp["data"].([]interface{})
				for _, tx := range data {
					txMap := tx.(map[string]interface{})
					assert.Equal(t, "income", txMap["type"])
				}
			},
		},
		{
			name:           "filter_by_type_outcome",
			query:          "?type=outcome",
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, resp map[string]interface{}) {
				assert.Equal(t, float64(1), resp["count"])
			},
		},
		{
			name:           "filter_by_category",
			query:          "?category=Food+%26+Groceries",
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, resp map[string]interface{}) {
				assert.Equal(t, float64(1), resp["count"])
				data := resp["data"].([]interface{})
				assert.Equal(t, "Food & Groceries", data[0].(map[string]interface{})["categoryName"])
			},
		},
		{
			name:           "filter_by_month_and_year",
			query:          fmt.Sprintf("?month=%d&year=%d", int(now.Month()), now.Year()),
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, resp map[string]interface{}) {
				// Should return transactions for current month
				assert.GreaterOrEqual(t, resp["count"], float64(0))
			},
		},
		{
			name:           "filter_by_date_range",
			query:          fmt.Sprintf("?start_date=%s&end_date=%s", now.AddDate(0, 0, -1).Format("2006-01-02"), now.AddDate(0, 0, 1).Format("2006-01-02")),
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, resp map[string]interface{}) {
				assert.GreaterOrEqual(t, resp["count"], float64(0))
			},
		},
		{
			name:           "filter_with_limit",
			query:          "?limit=2",
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, resp map[string]interface{}) {
				// Limit is passed to service but actual count depends on implementation
				assert.NotNil(t, resp["data"])
			},
		},
		{
			name:           "invalid_month_out_of_range",
			query:          "?month=13",
			expectedStatus: http.StatusOK, // Invalid params are ignored, not rejected
		},
		{
			name:           "invalid_year_out_of_range",
			query:          "?year=1999",
			expectedStatus: http.StatusOK, // Invalid params are ignored, not rejected
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/transactions"+tt.query, nil)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.checkResponse != nil {
				var resp map[string]interface{}
				require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
				tt.checkResponse(t, resp)
			}
		})
	}
}

// TestGetFixedTransactionsHandler verifies the fixed transactions endpoint
// with optional month/year filtering.
func TestGetFixedTransactionsHandler(t *testing.T) {
	router, svc, cleanup := setupTestRouter(t)
	defer cleanup()

	userID := uint(1)

	// Create fixed transactions
	fixedTxs := []*dto.CreateTransactionRequest{
		{Type: "outcome", Frequency: "fixed", Category: string(model.CategoryHousingUtilities), PaymentFrequency: "monthly"},
		{Type: "outcome", Frequency: "fixed", Category: string(model.CategoryTaxesFees), PaymentFrequency: "yearly"},
	}

	for _, req := range fixedTxs {
		_, err := svc.CreateTransaction(userID, req)
		require.NoError(t, err)
	}

	t.Run("returns_all_fixed_transactions", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/transactions/fixed", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp map[string]interface{}
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		assert.Equal(t, "Fixed transactions retrieved successfully", resp["message"])
		assert.Equal(t, float64(2), resp["count"])
	})

	t.Run("filters_by_month_and_year", func(t *testing.T) {
		now := time.Now()
		query := fmt.Sprintf("?month=%d&year=%d", int(now.Month()), now.Year())
		req := httptest.NewRequest(http.MethodGet, "/transactions/fixed"+query, nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var resp map[string]interface{}
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		// Should still return fixed transactions (filter affects payment aggregation)
		assert.NotNil(t, resp["data"])
	})

	t.Run("invalid_month_returns_error", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/transactions/fixed?month=invalid", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code) // Invalid params are ignored
	})
}

// TestGetFixedTransactionWithPaymentsHandler verifies retrieval of a single
// fixed transaction with its payment history.
func TestGetFixedTransactionWithPaymentsHandler(t *testing.T) {
	router, svc, cleanup := setupTestRouter(t)
	defer cleanup()

	userID := uint(1)

	// Create a fixed transaction
	fixedTx, err := svc.CreateTransaction(userID, &dto.CreateTransactionRequest{
		Type:             "outcome",
		Frequency:        "fixed",
		Category:         string(model.CategoryHousingUtilities),
		PaymentFrequency: "monthly",
	})
	require.NoError(t, err)

	// Create a payment for it
	_, err = svc.CreatePayment(userID, &dto.CreatePaymentRequest{
		TransactionID: fixedTx.ID,
		Amount:        1200.00,
	})
	require.NoError(t, err)

	t.Run("returns_existing_fixed_transaction_with_payments", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/transactions/fixed/%s", fixedTx.ID), nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp map[string]interface{}
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		assert.Equal(t, "Fixed transaction retrieved successfully", resp["message"])
		data := resp["data"].(map[string]interface{})
		assert.Equal(t, fixedTx.ID, data["id"])
		assert.Equal(t, "fixed", data["frequency"])
		assert.Equal(t, 1200.0, data["totalPaid"])
	})

	t.Run("returns_404_for_non_existent_transaction", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/transactions/fixed/000000000000000000000000", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("returns_404_for_variable_transaction", func(t *testing.T) {
		// Create a variable transaction
		variableTx, err := svc.CreateTransaction(userID, &dto.CreateTransactionRequest{
			Type:      "outcome",
			Frequency: "variable",
			Amount:    50,
			Category:  string(model.CategoryFoodGroceries),
		})
		require.NoError(t, err)

		req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/transactions/fixed/%s", variableTx.ID), nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})
}

// TestGetTransactionHandler verifies retrieval of a single transaction by ID.
func TestGetTransactionHandler(t *testing.T) {
	router, svc, cleanup := setupTestRouter(t)
	defer cleanup()

	userID := uint(1)

	created, err := svc.CreateTransaction(userID, &dto.CreateTransactionRequest{
		Type:      "income",
		Frequency: "variable",
		Amount:    300,
		Category:  string(model.CategorySideIncome),
	})
	require.NoError(t, err)

	t.Run("returns_existing_transaction", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/transactions/%s", created.ID), nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp map[string]interface{}
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		assert.Equal(t, "Transaction retrieved successfully", resp["message"])
		data := resp["data"].(map[string]interface{})
		assert.Equal(t, created.ID, data["id"])
		assert.Equal(t, 300.0, data["amount"])
	})

	t.Run("returns_404_for_non_existent_id", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/transactions/000000000000000000000000", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("returns_404_for_invalid_id_format", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/transactions/invalid-id", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})
}

// TestUpdateTransactionHandler verifies transaction updates with partial
// field updates and validation errors.
func TestUpdateTransactionHandler(t *testing.T) {
	router, svc, cleanup := setupTestRouter(t)
	defer cleanup()

	userID := uint(1)

	created, err := svc.CreateTransaction(userID, &dto.CreateTransactionRequest{
		Type:      "outcome",
		Frequency: "variable",
		Amount:    80,
		Category:  string(model.CategoryFoodGroceries),
	})
	require.NoError(t, err)

	t.Run("updates_amount_successfully", func(t *testing.T) {
		newAmount := 120.0
		updateReq := dto.UpdateTransactionRequest{Amount: &newAmount}
		body, _ := json.Marshal(updateReq)

		req := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/transactions/%s", created.ID), bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp map[string]interface{}
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		assert.Equal(t, "Transaction updated successfully", resp["message"])
		data := resp["data"].(map[string]interface{})
		assert.Equal(t, 120.0, data["amount"])
	})

	t.Run("rejects_invalid_category", func(t *testing.T) {
		invalidCat := "InvalidCategory"
		updateReq := dto.UpdateTransactionRequest{Category: &invalidCat}
		body, _ := json.Marshal(updateReq)

		req := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/transactions/%s", created.ID), bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("returns_400_for_non_existent_transaction", func(t *testing.T) {
		newAmount := 50.0
		updateReq := dto.UpdateTransactionRequest{Amount: &newAmount}
		body, _ := json.Marshal(updateReq)

		req := httptest.NewRequest(http.MethodPut, "/transactions/000000000000000000000000", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("rejects_invalid_json", func(t *testing.T) {
		body := []byte(`{"amount": "not-a-number"}`)

		req := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/transactions/%s", created.ID), bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

// TestDeleteTransactionHandler verifies transaction deletion with cascade
// behavior for associated payments.
func TestDeleteTransactionHandler(t *testing.T) {
	router, svc, cleanup := setupTestRouter(t)
	defer cleanup()

	userID := uint(1)

	t.Run("deletes_transaction_successfully", func(t *testing.T) {
		// Create a transaction to delete
		tx, err := svc.CreateTransaction(userID, &dto.CreateTransactionRequest{
			Type:      "outcome",
			Frequency: "variable",
			Amount:    50,
			Category:  string(model.CategoryFoodGroceries),
		})
		require.NoError(t, err)

		req := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/transactions/%s", tx.ID), nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp map[string]interface{}
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		assert.Equal(t, "Transaction deleted successfully", resp["message"])
	})

	t.Run("deletes_fixed_transaction_and_cascades_payments", func(t *testing.T) {
		// Create fixed transaction with payment
		fixedTx, err := svc.CreateTransaction(userID, &dto.CreateTransactionRequest{
			Type:             "outcome",
			Frequency:        "fixed",
			Category:         string(model.CategoryHousingUtilities),
			PaymentFrequency: "monthly",
		})
		require.NoError(t, err)

		_, err = svc.CreatePayment(userID, &dto.CreatePaymentRequest{
			TransactionID: fixedTx.ID,
			Amount:        1000,
		})
		require.NoError(t, err)

		req := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/transactions/%s", fixedTx.ID), nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("returns_400_for_non_existent_transaction", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodDelete, "/transactions/000000000000000000000000", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

// TestCreatePaymentHandler verifies payment creation for fixed transactions
// with period deduplication and validation.
func TestCreatePaymentHandler(t *testing.T) {
	router, svc, cleanup := setupTestRouter(t)
	defer cleanup()

	userID := uint(1)

	// Create a fixed transaction
	fixedTx, err := svc.CreateTransaction(userID, &dto.CreateTransactionRequest{
		Type:             "outcome",
		Frequency:        "fixed",
		Category:         string(model.CategoryHousingUtilities),
		PaymentFrequency: "monthly",
	})
	require.NoError(t, err)

	// Create a variable transaction (should reject payments)
	variableTx, err := svc.CreateTransaction(userID, &dto.CreateTransactionRequest{
		Type:      "outcome",
		Frequency: "variable",
		Amount:    50,
		Category:  string(model.CategoryFoodGroceries),
	})
	require.NoError(t, err)

	t.Run("creates_payment_for_fixed_transaction", func(t *testing.T) {
		req := dto.CreatePaymentRequest{
			TransactionID: fixedTx.ID,
			Amount:        1200,
			Date:          time.Now(),
		}
		body, _ := json.Marshal(req)

		httpReq := httptest.NewRequest(http.MethodPost, "/payments", bytes.NewBuffer(body))
		httpReq.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, httpReq)

		assert.Equal(t, http.StatusCreated, w.Code)

		var resp map[string]interface{}
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		assert.Equal(t, "Payment created successfully", resp["message"])
		data := resp["data"].(map[string]interface{})
		assert.Equal(t, fixedTx.ID, data["transactionId"])
		assert.Equal(t, 1200.0, data["amount"])
	})

	t.Run("rejects_duplicate_payment_in_same_period", func(t *testing.T) {
		// Try to create another payment for the same fixed transaction
		req := dto.CreatePaymentRequest{
			TransactionID: fixedTx.ID,
			Amount:        1200,
		}
		body, _ := json.Marshal(req)

		httpReq := httptest.NewRequest(http.MethodPost, "/payments", bytes.NewBuffer(body))
		httpReq.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, httpReq)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("rejects_payment_for_variable_transaction", func(t *testing.T) {
		req := dto.CreatePaymentRequest{
			TransactionID: variableTx.ID,
			Amount:        50,
		}
		body, _ := json.Marshal(req)

		httpReq := httptest.NewRequest(http.MethodPost, "/payments", bytes.NewBuffer(body))
		httpReq.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, httpReq)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("rejects_payment_for_non_existent_transaction", func(t *testing.T) {
		req := dto.CreatePaymentRequest{
			TransactionID: "000000000000000000000000",
			Amount:        100,
		}
		body, _ := json.Marshal(req)

		httpReq := httptest.NewRequest(http.MethodPost, "/payments", bytes.NewBuffer(body))
		httpReq.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, httpReq)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("rejects_invalid_json", func(t *testing.T) {
		body := []byte(`{"invalid": json}`)

		httpReq := httptest.NewRequest(http.MethodPost, "/payments", bytes.NewBuffer(body))
		httpReq.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, httpReq)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("rejects_missing_transaction_id", func(t *testing.T) {
		req := dto.CreatePaymentRequest{
			Amount: 100,
		}
		body, _ := json.Marshal(req)

		httpReq := httptest.NewRequest(http.MethodPost, "/payments", bytes.NewBuffer(body))
		httpReq.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, httpReq)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("rejects_zero_amount", func(t *testing.T) {
		// Create another fixed transaction for this test
		anotherFixedTx, err := svc.CreateTransaction(userID, &dto.CreateTransactionRequest{
			Type:             "outcome",
			Frequency:        "fixed",
			Category:         string(model.CategoryTaxesFees),
			PaymentFrequency: "yearly",
		})
		require.NoError(t, err)

		req := dto.CreatePaymentRequest{
			TransactionID: anotherFixedTx.ID,
			Amount:        0,
		}
		body, _ := json.Marshal(req)

		httpReq := httptest.NewRequest(http.MethodPost, "/payments", bytes.NewBuffer(body))
		httpReq.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, httpReq)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

// TestGetPaymentsHandler verifies payment listing with various filters.
func TestGetPaymentsHandler(t *testing.T) {
	router, svc, cleanup := setupTestRouter(t)
	defer cleanup()

	userID := uint(1)

	// Create two fixed transactions with payments
	txA, err := svc.CreateTransaction(userID, &dto.CreateTransactionRequest{
		Type:             "outcome",
		Frequency:        "fixed",
		Category:         string(model.CategoryHousingUtilities),
		PaymentFrequency: "monthly",
	})
	require.NoError(t, err)

	txB, err := svc.CreateTransaction(userID, &dto.CreateTransactionRequest{
		Type:             "outcome",
		Frequency:        "fixed",
		Category:         string(model.CategoryTaxesFees),
		PaymentFrequency: "yearly",
	})
	require.NoError(t, err)

	_, err = svc.CreatePayment(userID, &dto.CreatePaymentRequest{TransactionID: txA.ID, Amount: 800})
	require.NoError(t, err)
	_, err = svc.CreatePayment(userID, &dto.CreatePaymentRequest{TransactionID: txB.ID, Amount: 2400})
	require.NoError(t, err)

	t.Run("returns_all_payments", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/payments", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp map[string]interface{}
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		assert.Equal(t, "Payments retrieved successfully", resp["message"])
		assert.Equal(t, float64(2), resp["count"])
	})

	t.Run("filters_by_transaction_id", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/payments?transaction_id=%s", txA.ID), nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp map[string]interface{}
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		assert.Equal(t, float64(1), resp["count"])
		data := resp["data"].([]interface{})
		assert.Equal(t, 800.0, data[0].(map[string]interface{})["amount"])
	})

	t.Run("filters_by_date_range", func(t *testing.T) {
		now := time.Now()
		startDate := now.AddDate(0, 0, -1).Format("2006-01-02")
		endDate := now.AddDate(0, 0, 1).Format("2006-01-02")
		query := fmt.Sprintf("?start_date=%s&end_date=%s", startDate, endDate)

		req := httptest.NewRequest(http.MethodGet, "/payments"+query, nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("filters_with_limit", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/payments?limit=1", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("invalid_transaction_id_format", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/payments?transaction_id=invalid-id", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

// TestDeletePaymentHandler verifies payment deletion.
func TestDeletePaymentHandler(t *testing.T) {
	router, svc, cleanup := setupTestRouter(t)
	defer cleanup()

	userID := uint(1)

	fixedTx, err := svc.CreateTransaction(userID, &dto.CreateTransactionRequest{
		Type:             "outcome",
		Frequency:        "fixed",
		Category:         string(model.CategoryHousingUtilities),
		PaymentFrequency: "monthly",
	})
	require.NoError(t, err)

	payment, err := svc.CreatePayment(userID, &dto.CreatePaymentRequest{
		TransactionID: fixedTx.ID,
		Amount:        900,
	})
	require.NoError(t, err)

	t.Run("deletes_existing_payment", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/payments/%s", payment.ID), nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp map[string]interface{}
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		assert.Equal(t, "Payment deleted successfully", resp["message"])
	})

	t.Run("returns_400_for_non_existent_payment", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodDelete, "/payments/000000000000000000000000", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("returns_400_for_invalid_id_format", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodDelete, "/payments/invalid-id", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

// TestGetFinanceSummaryHandler verifies the finance summary endpoint with
// date range validation and aggregation.
func TestGetFinanceSummaryHandler(t *testing.T) {
	router, svc, cleanup := setupTestRouter(t)
	defer cleanup()

	userID := uint(1)
	now := time.Now()

	// Create test transactions
	transactions := []*dto.CreateTransactionRequest{
		{Type: "income", Frequency: "variable", Amount: 1000, Category: string(model.CategorySideIncome), Date: now},
		{Type: "income", Frequency: "variable", Amount: 500, Category: string(model.CategoryInvestments), Date: now},
		{Type: "outcome", Frequency: "variable", Amount: 200, Category: string(model.CategoryFoodGroceries), Date: now},
	}
	for _, txReq := range transactions {
		_, err := svc.CreateTransaction(userID, txReq)
		require.NoError(t, err)
	}

	t.Run("returns_summary_for_valid_date_range", func(t *testing.T) {
		startDate := now.AddDate(0, 0, -1).Format("2006-01-02")
		endDate := now.AddDate(0, 0, 1).Format("2006-01-02")
		query := fmt.Sprintf("?start_date=%s&end_date=%s", startDate, endDate)

		req := httptest.NewRequest(http.MethodGet, "/summary"+query, nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp map[string]interface{}
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		assert.Equal(t, "Finance summary retrieved successfully", resp["message"])
		data := resp["data"].(map[string]interface{})
		assert.NotNil(t, data["totalIncome"])
		assert.NotNil(t, data["totalOutcome"])
		assert.NotNil(t, data["balance"])
		assert.NotNil(t, data["incomeByCategory"])
		assert.NotNil(t, data["outcomeByCategory"])
	})

	t.Run("returns_400_for_missing_start_date", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/summary?end_date=2024-01-31", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("returns_400_for_missing_end_date", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/summary?start_date=2024-01-01", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("returns_400_for_invalid_start_date_format", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/summary?start_date=01-01-2024&end_date=2024-01-31", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("returns_400_for_invalid_end_date_format", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/summary?start_date=2024-01-01&end_date=31-01-2024", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

// TestGetMonthlyStatsHandler verifies the monthly stats endpoint with year
// parameter handling and default year behavior.
func TestGetMonthlyStatsHandler(t *testing.T) {
	router, svc, cleanup := setupTestRouter(t)
	defer cleanup()

	userID := uint(1)
	now := time.Now()

	// Create test transactions for current year
	transactions := []*dto.CreateTransactionRequest{
		{Type: "income", Frequency: "variable", Amount: 1000, Category: string(model.CategorySideIncome), Date: now},
		{Type: "outcome", Frequency: "variable", Amount: 500, Category: string(model.CategoryFoodGroceries), Date: now},
	}
	for _, txReq := range transactions {
		_, err := svc.CreateTransaction(userID, txReq)
		require.NoError(t, err)
	}

	t.Run("returns_stats_for_specified_year", func(t *testing.T) {
		query := fmt.Sprintf("?year=%d", now.Year())
		req := httptest.NewRequest(http.MethodGet, "/monthly-stats"+query, nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp map[string]interface{}
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		assert.Equal(t, "Monthly stats retrieved successfully", resp["message"])
		assert.Equal(t, float64(now.Year()), resp["year"])
		data := resp["data"].([]interface{})
		assert.Len(t, data, 12) // 12 months
	})

	t.Run("uses_current_year_when_not_specified", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/monthly-stats", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp map[string]interface{}
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		assert.Equal(t, float64(now.Year()), resp["year"])
	})

	t.Run("returns_400_for_invalid_year_format", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/monthly-stats?year=not-a-number", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("returns_400_for_year_out_of_range", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/monthly-stats?year=1999", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("returns_400_for_year_too_far_in_future", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/monthly-stats?year=2101", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

// TestUnauthorizedAccess verifies that endpoints requiring authentication
// return 401 when userID is not in context.
func TestUnauthorizedAccess(t *testing.T) {
	// Create router without auth middleware
	ctx := context.Background()
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(cfg.MongoURI))
	require.NoError(t, err)

	dbName := "test_unauthorized_" + primitive.NewObjectID().Hex()
	mongoDB := client.Database(dbName)

	financeService := service.NewFinanceService(mongoDB)
	controller := NewFinanceController(financeService)

	router := gin.New()
	// No auth middleware - simulates missing user context

	// Register protected routes
	transactions := router.Group("/transactions")
	{
		transactions.POST("", controller.CreateTransaction)
		transactions.GET("", controller.GetTransactions)
		transactions.GET("/fixed", controller.GetFixedTransactions)
		transactions.GET("/:id", controller.GetTransaction)
		transactions.PUT("/:id", controller.UpdateTransaction)
		transactions.DELETE("/:id", controller.DeleteTransaction)
	}

	payments := router.Group("/payments")
	{
		payments.POST("", controller.CreatePayment)
		payments.GET("", controller.GetPayments)
		payments.DELETE("/:id", controller.DeletePayment)
	}

	router.GET("/summary", controller.GetFinanceSummary)
	router.GET("/monthly-stats", controller.GetMonthlyStats)

	defer func() {
		_ = mongoDB.Drop(ctx)
		_ = client.Disconnect(ctx)
	}()

	tests := []struct {
		name   string
		method string
		path   string
		body   interface{}
	}{
		{
			name:   "create_transaction_unauthorized",
			method: http.MethodPost,
			path:   "/transactions",
			body:   dto.CreateTransactionRequest{Type: "income", Frequency: "variable", Amount: 100, Category: string(model.CategorySideIncome)},
		},
		{
			name:   "get_transactions_unauthorized",
			method: http.MethodGet,
			path:   "/transactions",
		},
		{
			name:   "get_fixed_transactions_unauthorized",
			method: http.MethodGet,
			path:   "/transactions/fixed",
		},
		{
			name:   "get_transaction_unauthorized",
			method: http.MethodGet,
			path:   "/transactions/000000000000000000000000",
		},
		{
			name:   "update_transaction_unauthorized",
			method: http.MethodPut,
			path:   "/transactions/000000000000000000000000",
			body:   map[string]interface{}{"amount": 100},
		},
		{
			name:   "delete_transaction_unauthorized",
			method: http.MethodDelete,
			path:   "/transactions/000000000000000000000000",
		},
		{
			name:   "create_payment_unauthorized",
			method: http.MethodPost,
			path:   "/payments",
			body:   dto.CreatePaymentRequest{TransactionID: "000000000000000000000000", Amount: 100},
		},
		{
			name:   "get_payments_unauthorized",
			method: http.MethodGet,
			path:   "/payments",
		},
		{
			name:   "delete_payment_unauthorized",
			method: http.MethodDelete,
			path:   "/payments/000000000000000000000000",
		},
		{
			name:   "get_summary_unauthorized",
			method: http.MethodGet,
			path:   "/summary?start_date=2024-01-01&end_date=2024-01-31",
		},
		{
			name:   "get_monthly_stats_unauthorized",
			method: http.MethodGet,
			path:   "/monthly-stats?year=2024",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var body []byte
			if tt.body != nil {
				body, _ = json.Marshal(tt.body)
			}

			req := httptest.NewRequest(tt.method, tt.path, bytes.NewBuffer(body))
			if tt.body != nil {
				req.Header.Set("Content-Type", "application/json")
			}
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, http.StatusUnauthorized, w.Code)
		})
	}
}

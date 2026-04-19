// finance_service_test.go
//
// Subject:      internal/domain/finance/service.FinanceService against real MongoDB.
// Scope:        Transactions and payments — categories, CRUD, fixed-transaction lifecycle,
//               payment creation (period dedup), finance summary aggregation.
// Out of scope: HTTP contract → finance_controller_test.go
//               Repository aggregation pipeline correctness in isolation.
// Infrastructure: MongoDB (test database via .env.test). Collections dropped between tests.
// Data strategy:  Drop all collections before each top-level test function.
// Parallel-safe:  No — shared MongoDB database, global drop between tests.
package service

import (
	"context"
	"os"
	"testing"
	"time"

	"life-tracker-backend/internal/config"
	"life-tracker-backend/internal/domain/finance/dto"
	"life-tracker-backend/internal/domain/finance/model"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var (
	testMongoDB *mongo.Database
	mongoClient *mongo.Client
	cfg         *config.Config
)

func TestMain(m *testing.M) {
	os.Setenv("ENVIRONMENT", "test")
	cfg = config.Load()
	code := m.Run()
	if mongoClient != nil {
		_ = mongoClient.Disconnect(context.Background())
	}
	os.Exit(code)
}

func setupTestDatabase(t *testing.T) {
	t.Helper()
	if testMongoDB != nil {
		return
	}

	ctx := context.Background()
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(cfg.MongoURI))
	require.NoError(t, err)
	require.NoError(t, client.Ping(ctx, nil), "MongoDB not available")

	testMongoDB = client.Database(cfg.MongoDatabase)
	mongoClient = client
}

func cleanDatabase(t *testing.T) {
	t.Helper()
	collections, _ := testMongoDB.ListCollectionNames(context.Background(), bson.M{})
	for _, coll := range collections {
		_ = testMongoDB.Collection(coll).Drop(context.Background())
	}
}

func getTestService(t *testing.T) *FinanceService {
	t.Helper()
	setupTestDatabase(t)
	cleanDatabase(t)
	return NewFinanceService(testMongoDB)
}

// TestGetCategories verifies pure in-memory category filtering by type and frequency.
// No database is touched — this covers the switch logic in GetCategories.
func TestGetCategories(t *testing.T) {
	svc := getTestService(t)

	t.Run("no_filter_returns_all_categories", func(t *testing.T) {
		resp := svc.GetCategories(nil, nil)
		assert.Equal(t, len(model.Categories), len(resp))
	})

	t.Run("filter_by_type_income_returns_only_income", func(t *testing.T) {
		typ := "income"
		resp := svc.GetCategories(&typ, nil)
		for _, cat := range resp {
			assert.Equal(t, "income", cat.Type)
		}
		assert.NotEmpty(t, resp)
	})

	t.Run("filter_by_type_outcome_returns_only_outcome", func(t *testing.T) {
		typ := "outcome"
		resp := svc.GetCategories(&typ, nil)
		for _, cat := range resp {
			assert.Equal(t, "outcome", cat.Type)
		}
		assert.NotEmpty(t, resp)
	})

	t.Run("filter_by_frequency_fixed_returns_only_fixed", func(t *testing.T) {
		freq := "fixed"
		resp := svc.GetCategories(nil, &freq)
		for _, cat := range resp {
			assert.Equal(t, "fixed", cat.ApplicableToFreq)
		}
		assert.NotEmpty(t, resp)
	})

	t.Run("filter_by_type_and_frequency_narrows_result", func(t *testing.T) {
		// Only "Primary Income" is income + fixed.
		typ := "income"
		freq := "fixed"
		resp := svc.GetCategories(&typ, &freq)
		require.Len(t, resp, 1)
		assert.Equal(t, string(model.CategoryPrimaryIncome), resp[0].Name)
	})
}

// TestCreateTransaction verifies that variable and fixed transactions are persisted
// with the correct fields, that invalid categories are rejected, and that fixed
// transactions have their amount zeroed in favour of payment-based tracking.
func TestCreateTransaction(t *testing.T) {
	svc := getTestService(t)

	t.Run("variable_income_transaction_persisted", func(t *testing.T) {
		req := &dto.CreateTransactionRequest{
			Type:      "income",
			Frequency: "variable",
			Amount:    500.00,
			Category:  string(model.CategorySideIncome),
		}
		resp, err := svc.CreateTransaction(10, req)
		assert.NoError(t, err)
		require.NotNil(t, resp)
		assert.Equal(t, "income", resp.Type)
		assert.Equal(t, "variable", resp.Frequency)
		assert.Equal(t, 500.00, resp.Amount)
		assert.NotEmpty(t, resp.ID)
	})

	t.Run("fixed_outcome_transaction_zeroes_amount", func(t *testing.T) {
		// Fixed transactions track amounts via payments, so the base amount is stored as 0.
		req := &dto.CreateTransactionRequest{
			Type:             "outcome",
			Frequency:        "fixed",
			Amount:           1200.00,
			Category:         string(model.CategoryHousingUtilities),
			PaymentFrequency: "monthly",
		}
		resp, err := svc.CreateTransaction(11, req)
		assert.NoError(t, err)
		require.NotNil(t, resp)
		assert.Equal(t, "fixed", resp.Frequency)
		assert.Equal(t, 0.0, resp.Amount)
		assert.Equal(t, "monthly", resp.PaymentFrequency)
	})

	t.Run("invalid_category_returns_error", func(t *testing.T) {
		req := &dto.CreateTransactionRequest{
			Type:      "income",
			Frequency: "variable",
			Amount:    100,
			Category:  "NotACategory",
		}
		resp, err := svc.CreateTransaction(12, req)
		assert.Error(t, err)
		assert.Nil(t, resp)
	})

	t.Run("custom_date_is_used", func(t *testing.T) {
		customDate := time.Date(2025, 3, 15, 0, 0, 0, 0, time.UTC)
		req := &dto.CreateTransactionRequest{
			Type:      "outcome",
			Frequency: "variable",
			Amount:    50.00,
			Category:  string(model.CategoryFoodGroceries),
			Date:      customDate,
		}
		resp, err := svc.CreateTransaction(13, req)
		assert.NoError(t, err)
		assert.WithinDuration(t, customDate, resp.Date, time.Second)
	})
}

// TestGetTransaction verifies retrieval by ID with ownership enforcement.
func TestGetTransaction(t *testing.T) {
	svc := getTestService(t)
	userID := uint(100)

	created, err := svc.CreateTransaction(userID, &dto.CreateTransactionRequest{
		Type:      "income",
		Frequency: "variable",
		Amount:    300.00,
		Category:  string(model.CategorySideIncome),
	})
	require.NoError(t, err)

	t.Run("existing_transaction_returns_response", func(t *testing.T) {
		resp, err := svc.GetTransaction(userID, created.ID)
		assert.NoError(t, err)
		require.NotNil(t, resp)
		assert.Equal(t, created.ID, resp.ID)
		assert.Equal(t, 300.00, resp.Amount)
	})

	t.Run("invalid_object_id_format_returns_error", func(t *testing.T) {
		resp, err := svc.GetTransaction(userID, "not-an-object-id")
		assert.Error(t, err)
		assert.Nil(t, resp)
	})

	t.Run("wrong_user_returns_error", func(t *testing.T) {
		// MongoDB query includes userId; wrong owner → ErrTransactionNotFound.
		resp, err := svc.GetTransaction(999, created.ID)
		assert.Error(t, err)
		assert.Nil(t, resp)
	})
}

// TestGetTransactions verifies list retrieval and optional type filtering.
func TestGetTransactions(t *testing.T) {
	svc := getTestService(t)
	userID := uint(101)

	for _, req := range []*dto.CreateTransactionRequest{
		{Type: "income", Frequency: "variable", Amount: 100, Category: string(model.CategorySideIncome)},
		{Type: "outcome", Frequency: "variable", Amount: 50, Category: string(model.CategoryFoodGroceries)},
		{Type: "income", Frequency: "variable", Amount: 200, Category: string(model.CategoryInvestments)},
	} {
		_, err := svc.CreateTransaction(userID, req)
		require.NoError(t, err)
	}

	t.Run("no_filter_returns_all_user_transactions", func(t *testing.T) {
		resp, err := svc.GetTransactions(userID, nil, nil, nil, nil, nil, nil, nil, 0, time.UTC)
		assert.NoError(t, err)
		assert.Len(t, resp, 3)
	})

	t.Run("filter_by_type_income_narrows_results", func(t *testing.T) {
		typ := "income"
		resp, err := svc.GetTransactions(userID, &typ, nil, nil, nil, nil, nil, nil, 0, time.UTC)
		assert.NoError(t, err)
		assert.Len(t, resp, 2)
		for _, tx := range resp {
			assert.Equal(t, "income", tx.Type)
		}
	})

	t.Run("different_user_sees_no_transactions", func(t *testing.T) {
		resp, err := svc.GetTransactions(888, nil, nil, nil, nil, nil, nil, nil, 0, time.UTC)
		assert.NoError(t, err)
		assert.Empty(t, resp)
	})
}

// TestUpdateTransaction verifies that fields are patched correctly and that
// invalid categories and wrong IDs are rejected.
func TestUpdateTransaction(t *testing.T) {
	svc := getTestService(t)
	userID := uint(102)

	created, err := svc.CreateTransaction(userID, &dto.CreateTransactionRequest{
		Type:      "outcome",
		Frequency: "variable",
		Amount:    80.00,
		Category:  string(model.CategoryFoodGroceries),
	})
	require.NoError(t, err)

	t.Run("update_amount_persisted", func(t *testing.T) {
		newAmount := 120.00
		resp, err := svc.UpdateTransaction(userID, created.ID, &dto.UpdateTransactionRequest{Amount: &newAmount})
		assert.NoError(t, err)
		assert.Equal(t, 120.00, resp.Amount)
	})

	t.Run("update_with_invalid_category_returns_error", func(t *testing.T) {
		invalidCat := "Bogus Category"
		resp, err := svc.UpdateTransaction(userID, created.ID, &dto.UpdateTransactionRequest{Category: &invalidCat})
		assert.Error(t, err)
		assert.Nil(t, resp)
	})

	t.Run("update_non_existent_transaction_returns_error", func(t *testing.T) {
		newAmount := 50.00
		// Use a valid ObjectID format but one that doesn't exist in the DB.
		resp, err := svc.UpdateTransaction(userID, "000000000000000000000001", &dto.UpdateTransactionRequest{Amount: &newAmount})
		assert.Error(t, err)
		assert.Nil(t, resp)
	})
}

// TestDeleteTransaction verifies that deleting a transaction also removes
// associated payments (cascade) and that deleting a non-existent ID fails.
func TestDeleteTransaction(t *testing.T) {
	svc := getTestService(t)
	userID := uint(103)

	t.Run("deletion_succeeds_and_cascades_payments", func(t *testing.T) {
		// Create a fixed transaction so we can attach a payment to it.
		tx, err := svc.CreateTransaction(userID, &dto.CreateTransactionRequest{
			Type:             "outcome",
			Frequency:        "fixed",
			Amount:           0,
			Category:         string(model.CategoryHousingUtilities),
			PaymentFrequency: "monthly",
		})
		require.NoError(t, err)

		_, err = svc.CreatePayment(userID, &dto.CreatePaymentRequest{
			TransactionID: tx.ID,
			Amount:        1200.00,
		})
		require.NoError(t, err)

		err = svc.DeleteTransaction(userID, tx.ID)
		assert.NoError(t, err)

		// Transaction must be gone.
		resp, err := svc.GetTransaction(userID, tx.ID)
		assert.Error(t, err)
		assert.Nil(t, resp)

		// Payments for this transaction must also be removed.
		payments, err := svc.GetPayments(userID, &tx.ID, nil, nil, 0)
		assert.NoError(t, err)
		assert.Empty(t, payments)
	})

	t.Run("delete_non_existent_transaction_returns_error", func(t *testing.T) {
		err := svc.DeleteTransaction(userID, "000000000000000000000002")
		assert.Error(t, err)
	})
}

// TestCreatePayment verifies the full payment lifecycle for fixed transactions:
// valid creation, rejection for variable transactions, and period-dedup enforcement.
func TestCreatePayment(t *testing.T) {
	svc := getTestService(t)
	userID := uint(104)

	fixedTx, err := svc.CreateTransaction(userID, &dto.CreateTransactionRequest{
		Type:             "outcome",
		Frequency:        "fixed",
		Category:         string(model.CategoryHousingUtilities),
		PaymentFrequency: "monthly",
	})
	require.NoError(t, err)

	variableTx, err := svc.CreateTransaction(userID, &dto.CreateTransactionRequest{
		Type:      "outcome",
		Frequency: "variable",
		Amount:    50.00,
		Category:  string(model.CategoryFoodGroceries),
	})
	require.NoError(t, err)

	t.Run("payment_for_fixed_transaction_created", func(t *testing.T) {
		resp, err := svc.CreatePayment(userID, &dto.CreatePaymentRequest{
			TransactionID: fixedTx.ID,
			Amount:        1200.00,
		})
		assert.NoError(t, err)
		require.NotNil(t, resp)
		assert.Equal(t, 1200.00, resp.Amount)
		assert.Equal(t, fixedTx.ID, resp.TransactionID)
	})

	t.Run("duplicate_payment_in_same_period_rejected", func(t *testing.T) {
		// Second payment for the same fixed transaction within the current month is rejected.
		resp, err := svc.CreatePayment(userID, &dto.CreatePaymentRequest{
			TransactionID: fixedTx.ID,
			Amount:        1200.00,
		})
		assert.Error(t, err)
		assert.Nil(t, resp)
	})

	t.Run("payment_for_variable_transaction_rejected", func(t *testing.T) {
		// Payments are only allowed for fixed-frequency transactions.
		resp, err := svc.CreatePayment(userID, &dto.CreatePaymentRequest{
			TransactionID: variableTx.ID,
			Amount:        50.00,
		})
		assert.Error(t, err)
		assert.Nil(t, resp)
	})

	t.Run("payment_for_non_existent_transaction_rejected", func(t *testing.T) {
		resp, err := svc.CreatePayment(userID, &dto.CreatePaymentRequest{
			TransactionID: "000000000000000000000099",
			Amount:        100.00,
		})
		assert.Error(t, err)
		assert.Nil(t, resp)
	})
}

// TestGetPayments verifies that payments are listed correctly and can be
// narrowed by transaction ID.
func TestGetPayments(t *testing.T) {
	svc := getTestService(t)
	userID := uint(105)

	// Two fixed transactions so we can test per-transaction filtering.
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

	t.Run("no_filter_returns_all_user_payments", func(t *testing.T) {
		resp, err := svc.GetPayments(userID, nil, nil, nil, 0)
		assert.NoError(t, err)
		assert.Len(t, resp, 2)
	})

	t.Run("filter_by_transaction_id_narrows_results", func(t *testing.T) {
		resp, err := svc.GetPayments(userID, &txA.ID, nil, nil, 0)
		assert.NoError(t, err)
		require.Len(t, resp, 1)
		assert.Equal(t, 800.0, resp[0].Amount)
	})

	t.Run("different_user_sees_no_payments", func(t *testing.T) {
		resp, err := svc.GetPayments(888, nil, nil, nil, 0)
		assert.NoError(t, err)
		assert.Empty(t, resp)
	})
}

// TestDeletePayment verifies that a payment can be removed and that
// deleting a non-existent payment returns an error.
func TestDeletePayment(t *testing.T) {
	svc := getTestService(t)
	userID := uint(106)

	fixedTx, err := svc.CreateTransaction(userID, &dto.CreateTransactionRequest{
		Type:             "outcome",
		Frequency:        "fixed",
		Category:         string(model.CategoryHousingUtilities),
		PaymentFrequency: "monthly",
	})
	require.NoError(t, err)

	payment, err := svc.CreatePayment(userID, &dto.CreatePaymentRequest{
		TransactionID: fixedTx.ID,
		Amount:        900.00,
	})
	require.NoError(t, err)

	t.Run("existing_payment_deleted_successfully", func(t *testing.T) {
		err := svc.DeletePayment(userID, payment.ID)
		assert.NoError(t, err)

		// After deletion, listing payments for this transaction should be empty.
		payments, err := svc.GetPayments(userID, &fixedTx.ID, nil, nil, 0)
		assert.NoError(t, err)
		assert.Empty(t, payments)
	})

	t.Run("delete_non_existent_payment_returns_error", func(t *testing.T) {
		err := svc.DeletePayment(userID, "000000000000000000000003")
		assert.Error(t, err)
	})
}

// TestGetFinanceSummary verifies that the summary correctly aggregates variable
// transactions into income and outcome totals for a given date range.
func TestGetFinanceSummary(t *testing.T) {
	svc := getTestService(t)
	userID := uint(107)

	now := time.Now()
	startDate := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	endDate := startDate.AddDate(0, 1, 0).Add(-time.Nanosecond)

	transactions := []*dto.CreateTransactionRequest{
		{Type: "income", Frequency: "variable", Amount: 1000, Category: string(model.CategorySideIncome), Date: startDate.Add(time.Hour)},
		{Type: "income", Frequency: "variable", Amount: 500, Category: string(model.CategoryInvestments), Date: startDate.Add(time.Hour)},
		{Type: "outcome", Frequency: "variable", Amount: 200, Category: string(model.CategoryFoodGroceries), Date: startDate.Add(time.Hour)},
	}
	for _, req := range transactions {
		_, err := svc.CreateTransaction(userID, req)
		require.NoError(t, err)
	}

	t.Run("summary_totals_correct_for_current_month", func(t *testing.T) {
		resp, err := svc.GetFinanceSummary(userID, startDate, endDate, time.UTC)
		assert.NoError(t, err)
		require.NotNil(t, resp)

		assert.Equal(t, 1500.0, resp.TotalIncome)
		assert.Equal(t, 200.0, resp.TotalOutcome)
		assert.Equal(t, 1300.0, resp.Balance)
		assert.NotEmpty(t, resp.IncomeByCategory)
		assert.NotEmpty(t, resp.OutcomeByCategory)
	})

	t.Run("different_user_returns_zero_summary", func(t *testing.T) {
		resp, err := svc.GetFinanceSummary(888, startDate, endDate, time.UTC)
		assert.NoError(t, err)
		require.NotNil(t, resp)
		assert.Equal(t, 0.0, resp.TotalIncome)
		assert.Equal(t, 0.0, resp.TotalOutcome)
	})
}

package repository

import (
	"context"
	"os"
	"testing"
	"time"

	"life-tracker-backend/internal/config"
	"life-tracker-backend/internal/domain/finance/model"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// finance_repository_test.go
//
// Subject: TransactionRepository and PaymentRepository (MongoDB)
// Scope:   Repository layer integration tests — CRUD operations, filtering, and aggregations.
// Out of scope:
//   - service-layer business rules   → finance_service_test.go
//   - HTTP contract validation       → finance_controller_test.go
// Infrastructure: MongoDB running locally (via docker-compose or local install)
// Data strategy: DROP MongoDB collections between tests
// Parallel-safe: no — tests share database connections and clean state between runs

var (
	testMongoDB *mongo.Database
	mongoClient *mongo.Client
	testLoc     *time.Location
	cfg         *config.Config
)

func TestMain(m *testing.M) {
	os.Setenv("ENVIRONMENT", "test")
	cfg = config.Load()
	testLoc = time.UTC

	code := m.Run()

	if mongoClient != nil {
		_ = mongoClient.Disconnect(context.Background())
	}

	os.Exit(code)
}

func setupTestDatabase(t *testing.T) {
	if testMongoDB == nil {
		ctx := context.Background()
		client, err := mongo.Connect(ctx, options.Client().ApplyURI(cfg.MongoURI))
		require.NoError(t, err)
		require.NoError(t, client.Ping(ctx, nil), "MongoDB not available")

		testMongoDB = client.Database(cfg.MongoDatabase)
		mongoClient = client
	}
}

func cleanDatabase(t *testing.T) {
	t.Helper()
	ctx := context.Background()
	collections := []string{"transactions", "payments"}
	for _, coll := range collections {
		_ = testMongoDB.Collection(coll).Drop(ctx)
	}
}

type testContext struct {
	transactionRepo TransactionRepository
	paymentRepo     PaymentRepository
	mongoDB         *mongo.Database
}

func getTestContext(t *testing.T) *testContext {
	setupTestDatabase(t)
	cleanDatabase(t)
	return &testContext{
		transactionRepo: NewTransactionRepository(testMongoDB),
		paymentRepo:     NewPaymentRepository(testMongoDB),
		mongoDB:         testMongoDB,
	}
}

// ============================================================================
// TransactionRepository Tests
// ============================================================================

// TransactionRepository_Create verifies that Create persists a new transaction
// with auto-generated ObjectID and timestamps.
func TestTransactionRepository_Create(t *testing.T) {
	tc := getTestContext(t)
	ctx := context.Background()

	transaction := &model.Transaction{
		UserID:    1,
		Type:      model.TransactionTypeIncome,
		Frequency: model.TransactionFrequencyVariable,
		Amount:    100.50,
		Category:  "Primary Income",
		Date:      time.Now().UTC(),
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}

	err := tc.transactionRepo.Create(ctx, transaction)
	require.NoError(t, err)
	assert.NotEqual(t, primitive.NilObjectID, transaction.ID, "ID should be generated")
}

// TransactionRepository_FindByID tests retrieval by ObjectID with user scoping.
// Success case: transaction exists and belongs to user.
// Not found cases: wrong user ID, non-existent ID.
func TestTransactionRepository_FindByID(t *testing.T) {
	tc := getTestContext(t)
	ctx := context.Background()
	userID := uint(10)

	// Create test transaction
	transaction := &model.Transaction{
		UserID:    userID,
		Type:      model.TransactionTypeIncome,
		Frequency: model.TransactionFrequencyVariable,
		Amount:    200.00,
		Category:  "Primary Income",
		Date:      time.Now().UTC(),
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}
	require.NoError(t, tc.transactionRepo.Create(ctx, transaction))

	t.Run("success - existing transaction", func(t *testing.T) {
		found, err := tc.transactionRepo.FindByID(ctx, transaction.ID, userID)
		assert.NoError(t, err)
		assert.NotNil(t, found)
		assert.Equal(t, transaction.Amount, found.Amount)
		assert.Equal(t, userID, found.UserID)
	})

	t.Run("not found - wrong user", func(t *testing.T) {
		found, err := tc.transactionRepo.FindByID(ctx, transaction.ID, 999)
		assert.ErrorIs(t, err, ErrTransactionNotFound)
		assert.Nil(t, found)
	})

	t.Run("not found - non-existent ID", func(t *testing.T) {
		found, err := tc.transactionRepo.FindByID(ctx, primitive.NewObjectID(), userID)
		assert.ErrorIs(t, err, ErrTransactionNotFound)
		assert.Nil(t, found)
	})
}

// TransactionRepository_FindByFilter tests the filter functionality with
// TransactionType, Frequency, Category, StartDate, EndDate, and UserID.
func TestTransactionRepository_FindByFilter(t *testing.T) {
	tc := getTestContext(t)
	ctx := context.Background()
	userID := uint(20)
	now := time.Now().UTC()

	// Create transactions with different properties
	incomeFixed := &model.Transaction{
		UserID:    userID,
		Type:      model.TransactionTypeIncome,
		Frequency: model.TransactionFrequencyFixed,
		Amount:    1000.00,
		Category:  "Primary Income",
		Date:      now.Add(-24 * time.Hour),
		CreatedAt: now,
		UpdatedAt: now,
	}
	require.NoError(t, tc.transactionRepo.Create(ctx, incomeFixed))

	outcomeVariable := &model.Transaction{
		UserID:    userID,
		Type:      model.TransactionTypeOutcome,
		Frequency: model.TransactionFrequencyVariable,
		Amount:    50.00,
		Category:  "Food & Groceries",
		Date:      now,
		CreatedAt: now,
		UpdatedAt: now,
	}
	require.NoError(t, tc.transactionRepo.Create(ctx, outcomeVariable))

	outcomeFixed := &model.Transaction{
		UserID:    userID,
		Type:      model.TransactionTypeOutcome,
		Frequency: model.TransactionFrequencyFixed,
		Amount:    500.00,
		Category:  "Housing & Utilities",
		Date:      now.Add(-48 * time.Hour),
		CreatedAt: now,
		UpdatedAt: now,
	}
	require.NoError(t, tc.transactionRepo.Create(ctx, outcomeFixed))

	// Transaction for different user (should not appear)
	otherUserTransaction := &model.Transaction{
		UserID:    999,
		Type:      model.TransactionTypeIncome,
		Frequency: model.TransactionFrequencyVariable,
		Amount:    9999.00,
		Category:  "Primary Income",
		Date:      now,
		CreatedAt: now,
		UpdatedAt: now,
	}
	require.NoError(t, tc.transactionRepo.Create(ctx, otherUserTransaction))

	t.Run("filter by transaction type only", func(t *testing.T) {
		transactionType := string(model.TransactionTypeIncome)
		filter := TransactionFilter{
			TransactionType: &transactionType,
			UserID:          userID,
		}
		transactions, err := tc.transactionRepo.FindByFilter(ctx, filter, 0)
		assert.NoError(t, err)
		assert.Len(t, transactions, 1)
		assert.Equal(t, model.TransactionTypeIncome, transactions[0].Type)
	})

	t.Run("filter by frequency only", func(t *testing.T) {
		frequency := string(model.TransactionFrequencyFixed)
		filter := TransactionFilter{
			Frequency: &frequency,
			UserID:    userID,
		}
		transactions, err := tc.transactionRepo.FindByFilter(ctx, filter, 0)
		assert.NoError(t, err)
		assert.Len(t, transactions, 2)
		for _, tx := range transactions {
			assert.Equal(t, model.TransactionFrequencyFixed, tx.Frequency)
		}
	})

	t.Run("filter by category only", func(t *testing.T) {
		category := "Food & Groceries"
		filter := TransactionFilter{
			Category: &category,
			UserID:   userID,
		}
		transactions, err := tc.transactionRepo.FindByFilter(ctx, filter, 0)
		assert.NoError(t, err)
		assert.Len(t, transactions, 1)
		assert.Equal(t, "Food & Groceries", transactions[0].Category)
	})

	t.Run("filter by date range", func(t *testing.T) {
		startDate := now.Add(-36 * time.Hour)
		endDate := now.Add(-12 * time.Hour)
		filter := TransactionFilter{
			StartDate: &startDate,
			EndDate:   &endDate,
			UserID:    userID,
		}
		transactions, err := tc.transactionRepo.FindByFilter(ctx, filter, 0)
		assert.NoError(t, err)
		assert.Len(t, transactions, 1)
		assert.Equal(t, incomeFixed.ID, transactions[0].ID)
	})

	t.Run("filter by multiple criteria", func(t *testing.T) {
		transactionType := string(model.TransactionTypeOutcome)
		frequency := string(model.TransactionFrequencyFixed)
		filter := TransactionFilter{
			TransactionType: &transactionType,
			Frequency:       &frequency,
			UserID:          userID,
		}
		transactions, err := tc.transactionRepo.FindByFilter(ctx, filter, 0)
		assert.NoError(t, err)
		assert.Len(t, transactions, 1)
		assert.Equal(t, outcomeFixed.ID, transactions[0].ID)
	})

	t.Run("filter with limit", func(t *testing.T) {
		filter := TransactionFilter{
			UserID: userID,
		}
		transactions, err := tc.transactionRepo.FindByFilter(ctx, filter, 2)
		assert.NoError(t, err)
		assert.Len(t, transactions, 2)
	})

	t.Run("no filter returns all user transactions", func(t *testing.T) {
		filter := TransactionFilter{
			UserID: userID,
		}
		transactions, err := tc.transactionRepo.FindByFilter(ctx, filter, 0)
		assert.NoError(t, err)
		assert.Len(t, transactions, 3)
	})

	t.Run("filter with no matches", func(t *testing.T) {
		category := "NonExistentCategory"
		filter := TransactionFilter{
			Category: &category,
			UserID:   userID,
		}
		transactions, err := tc.transactionRepo.FindByFilter(ctx, filter, 0)
		assert.NoError(t, err)
		assert.Empty(t, transactions)
	})
}

// TransactionRepository_FindFixedTransactions tests retrieval of fixed frequency
// transactions for a specific user.
func TestTransactionRepository_FindFixedTransactions(t *testing.T) {
	tc := getTestContext(t)
	ctx := context.Background()
	userID := uint(30)
	now := time.Now().UTC()

	// Create fixed transaction
	fixedTransaction := &model.Transaction{
		UserID:    userID,
		Type:      model.TransactionTypeOutcome,
		Frequency: model.TransactionFrequencyFixed,
		Amount:    500.00,
		Category:  "Housing & Utilities",
		Date:      now,
		CreatedAt: now,
		UpdatedAt: now,
	}
	require.NoError(t, tc.transactionRepo.Create(ctx, fixedTransaction))

	// Create variable transaction (should be filtered out)
	variableTransaction := &model.Transaction{
		UserID:    userID,
		Type:      model.TransactionTypeIncome,
		Frequency: model.TransactionFrequencyVariable,
		Amount:    100.00,
		Category:  "Primary Income",
		Date:      now,
		CreatedAt: now,
		UpdatedAt: now,
	}
	require.NoError(t, tc.transactionRepo.Create(ctx, variableTransaction))

	// Fixed transaction for different user (should not appear)
	otherUserFixed := &model.Transaction{
		UserID:    999,
		Type:      model.TransactionTypeOutcome,
		Frequency: model.TransactionFrequencyFixed,
		Amount:    9999.00,
		Category:  "Housing & Utilities",
		Date:      now,
		CreatedAt: now,
		UpdatedAt: now,
	}
	require.NoError(t, tc.transactionRepo.Create(ctx, otherUserFixed))

	t.Run("returns only fixed transactions for user", func(t *testing.T) {
		transactions, err := tc.transactionRepo.FindFixedTransactions(ctx, userID)
		assert.NoError(t, err)
		assert.Len(t, transactions, 1)
		assert.Equal(t, fixedTransaction.ID, transactions[0].ID)
		assert.Equal(t, model.TransactionFrequencyFixed, transactions[0].Frequency)
	})

	t.Run("returns empty for user with no fixed transactions", func(t *testing.T) {
		transactions, err := tc.transactionRepo.FindFixedTransactions(ctx, 888)
		assert.NoError(t, err)
		assert.Empty(t, transactions)
	})
}

// TransactionRepository_Update tests partial updates via a bson.M map of fields.
// Verifies that specified fields are updated while others remain unchanged.
func TestTransactionRepository_Update(t *testing.T) {
	tc := getTestContext(t)
	ctx := context.Background()
	userID := uint(40)
	now := time.Now().UTC()

	transaction := &model.Transaction{
		UserID:      userID,
		Type:        model.TransactionTypeIncome,
		Frequency:   model.TransactionFrequencyVariable,
		Amount:      100.00,
		Category:    "Primary Income",
		Description: "Original Description",
		Date:        now,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	require.NoError(t, tc.transactionRepo.Create(ctx, transaction))

	t.Run("update single field", func(t *testing.T) {
		updates := bson.M{
			"amount": 200.00,
		}
		err := tc.transactionRepo.Update(ctx, transaction.ID, updates)
		assert.NoError(t, err)

		// Reload and verify
		updated, err := tc.transactionRepo.FindByID(ctx, transaction.ID, userID)
		require.NoError(t, err)
		assert.Equal(t, 200.00, updated.Amount)
		assert.Equal(t, "Original Description", updated.Description) // unchanged
	})

	t.Run("update multiple fields", func(t *testing.T) {
		updates := bson.M{
			"amount":      300.00,
			"category":    "Side Income",
			"description": "Updated Description",
		}
		err := tc.transactionRepo.Update(ctx, transaction.ID, updates)
		assert.NoError(t, err)

		updated, err := tc.transactionRepo.FindByID(ctx, transaction.ID, userID)
		require.NoError(t, err)
		assert.Equal(t, 300.00, updated.Amount)
		assert.Equal(t, "Side Income", updated.Category)
		assert.Equal(t, "Updated Description", updated.Description)
	})
}

// TransactionRepository_Delete tests deletion with user scoping.
// Success: transaction is deleted.
// Not found: wrong user, non-existent ID.
func TestTransactionRepository_Delete(t *testing.T) {
	tc := getTestContext(t)
	ctx := context.Background()
	userID := uint(50)
	now := time.Now().UTC()

	transaction := &model.Transaction{
		UserID:    userID,
		Type:      model.TransactionTypeIncome,
		Frequency: model.TransactionFrequencyVariable,
		Amount:    100.00,
		Category:  "Primary Income",
		Date:      now,
		CreatedAt: now,
		UpdatedAt: now,
	}
	require.NoError(t, tc.transactionRepo.Create(ctx, transaction))

	t.Run("successful deletion", func(t *testing.T) {
		err := tc.transactionRepo.Delete(ctx, transaction.ID, userID)
		assert.NoError(t, err)

		// Should not be findable
		found, err := tc.transactionRepo.FindByID(ctx, transaction.ID, userID)
		assert.ErrorIs(t, err, ErrTransactionNotFound)
		assert.Nil(t, found)
	})

	t.Run("delete non-existent transaction", func(t *testing.T) {
		err := tc.transactionRepo.Delete(ctx, primitive.NewObjectID(), userID)
		assert.ErrorIs(t, err, ErrTransactionNotFound)
	})

	t.Run("delete with wrong user", func(t *testing.T) {
		// Create new transaction to delete
		transaction2 := &model.Transaction{
			UserID:    userID,
			Type:      model.TransactionTypeIncome,
			Frequency: model.TransactionFrequencyVariable,
			Amount:    200.00,
			Category:  "Primary Income",
			Date:      now,
			CreatedAt: now,
			UpdatedAt: now,
		}
		require.NoError(t, tc.transactionRepo.Create(ctx, transaction2))

		err := tc.transactionRepo.Delete(ctx, transaction2.ID, 999)
		assert.ErrorIs(t, err, ErrTransactionNotFound)

		// Verify transaction still exists
		found, err := tc.transactionRepo.FindByID(ctx, transaction2.ID, userID)
		assert.NoError(t, err)
		assert.NotNil(t, found)
	})
}

// TransactionRepository_Aggregate tests the aggregation pipeline that groups
// transactions by type and category with sum and count.
func TestTransactionRepository_Aggregate(t *testing.T) {
	tc := getTestContext(t)
	ctx := context.Background()
	userID := uint(60)
	now := time.Now().UTC()
	startDate := now.Add(-7 * 24 * time.Hour)
	endDate := now

	// Create income transactions
	for i := 0; i < 3; i++ {
		transaction := &model.Transaction{
			UserID:    userID,
			Type:      model.TransactionTypeIncome,
			Frequency: model.TransactionFrequencyVariable,
			Amount:    100.00,
			Category:  "Primary Income",
			Date:      now.Add(-time.Duration(i) * 24 * time.Hour),
			CreatedAt: now,
			UpdatedAt: now,
		}
		require.NoError(t, tc.transactionRepo.Create(ctx, transaction))
	}

	// Create outcome transactions in different categories
	outcomeCategories := []string{"Food & Groceries", "Food & Groceries", "Housing & Utilities"}
	for i, category := range outcomeCategories {
		transaction := &model.Transaction{
			UserID:    userID,
			Type:      model.TransactionTypeOutcome,
			Frequency: model.TransactionFrequencyVariable,
			Amount:    50.00 * float64(i+1),
			Category:  category,
			Date:      now.Add(-time.Duration(i) * 24 * time.Hour),
			CreatedAt: now,
			UpdatedAt: now,
		}
		require.NoError(t, tc.transactionRepo.Create(ctx, transaction))
	}

	// Transaction outside date range (should not be included)
	oldTransaction := &model.Transaction{
		UserID:    userID,
		Type:      model.TransactionTypeIncome,
		Frequency: model.TransactionFrequencyVariable,
		Amount:    9999.00,
		Category:  "Primary Income",
		Date:      now.Add(-30 * 24 * time.Hour),
		CreatedAt: now,
		UpdatedAt: now,
	}
	require.NoError(t, tc.transactionRepo.Create(ctx, oldTransaction))

	// Transaction for different user (should not be included)
	otherUserTransaction := &model.Transaction{
		UserID:    999,
		Type:      model.TransactionTypeIncome,
		Frequency: model.TransactionFrequencyVariable,
		Amount:    8888.00,
		Category:  "Primary Income",
		Date:      now,
		CreatedAt: now,
		UpdatedAt: now,
	}
	require.NoError(t, tc.transactionRepo.Create(ctx, otherUserTransaction))

	t.Run("aggregate returns grouped results", func(t *testing.T) {
		results, err := tc.transactionRepo.Aggregate(ctx, userID, startDate, endDate)
		assert.NoError(t, err)
		assert.Len(t, results, 3) // Primary Income, Food & Groceries, Housing & Utilities

		// Find Primary Income result
		var incomeResult *AggregationResult
		for _, r := range results {
			if r.Type == string(model.TransactionTypeIncome) && r.Category == "Primary Income" {
				incomeResult = &r
				break
			}
		}
		assert.NotNil(t, incomeResult)
		assert.Equal(t, 300.00, incomeResult.Total)
		assert.Equal(t, int64(3), incomeResult.Count)
	})

	t.Run("aggregate with no matching transactions", func(t *testing.T) {
		futureStart := now.Add(365 * 24 * time.Hour)
		futureEnd := now.Add(366 * 24 * time.Hour)
		results, err := tc.transactionRepo.Aggregate(ctx, userID, futureStart, futureEnd)
		assert.NoError(t, err)
		assert.Empty(t, results)
	})
}

// TransactionRepository_AggregateMonthly tests the monthly aggregation pipeline
// that groups transactions by month and type.
func TestTransactionRepository_AggregateMonthly(t *testing.T) {
	tc := getTestContext(t)
	ctx := context.Background()
	userID := uint(70)
	loc := time.UTC
	now := time.Now().In(loc)

	// Create transactions in January and February
	january := time.Date(now.Year(), 1, 15, 0, 0, 0, 0, loc).UTC()
	february := time.Date(now.Year(), 2, 15, 0, 0, 0, 0, loc).UTC()

	// January income
	for i := 0; i < 2; i++ {
		transaction := &model.Transaction{
			UserID:    userID,
			Type:      model.TransactionTypeIncome,
			Frequency: model.TransactionFrequencyVariable,
			Amount:    100.00,
			Category:  "Primary Income",
			Date:      january,
			CreatedAt: now,
			UpdatedAt: now,
		}
		require.NoError(t, tc.transactionRepo.Create(ctx, transaction))
	}

	// January outcome
	transaction := &model.Transaction{
		UserID:    userID,
		Type:      model.TransactionTypeOutcome,
		Frequency: model.TransactionFrequencyVariable,
		Amount:    50.00,
		Category:  "Food & Groceries",
		Date:      january,
		CreatedAt: now,
		UpdatedAt: now,
	}
	require.NoError(t, tc.transactionRepo.Create(ctx, transaction))

	// February income
	transaction = &model.Transaction{
		UserID:    userID,
		Type:      model.TransactionTypeIncome,
		Frequency: model.TransactionFrequencyVariable,
		Amount:    200.00,
		Category:  "Primary Income",
		Date:      february,
		CreatedAt: now,
		UpdatedAt: now,
	}
	require.NoError(t, tc.transactionRepo.Create(ctx, transaction))

	// Transaction for different user
	otherUserTransaction := &model.Transaction{
		UserID:    999,
		Type:      model.TransactionTypeIncome,
		Frequency: model.TransactionFrequencyVariable,
		Amount:    9999.00,
		Category:  "Primary Income",
		Date:      january,
		CreatedAt: now,
		UpdatedAt: now,
	}
	require.NoError(t, tc.transactionRepo.Create(ctx, otherUserTransaction))

	t.Run("aggregate monthly returns grouped by month", func(t *testing.T) {
		results, err := tc.transactionRepo.AggregateMonthly(ctx, userID, now.Year(), loc)
		assert.NoError(t, err)
		assert.Len(t, results, 3) // January income, January outcome, February income

		// Verify January income
		var janIncome *MonthlyAggregationResult
		for _, r := range results {
			if r.Month == 1 && r.Type == string(model.TransactionTypeIncome) {
				janIncome = &r
				break
			}
		}
		assert.NotNil(t, janIncome)
		assert.Equal(t, 200.00, janIncome.Total)
		assert.Equal(t, int64(2), janIncome.Count)
	})

	t.Run("aggregate monthly with no transactions in year", func(t *testing.T) {
		results, err := tc.transactionRepo.AggregateMonthly(ctx, userID, now.Year()-1, loc)
		assert.NoError(t, err)
		assert.Empty(t, results)
	})
}

// ============================================================================
// PaymentRepository Tests
// ============================================================================

// PaymentRepository_Create verifies that Create persists a new payment
// with auto-generated ObjectID and timestamps.
func TestPaymentRepository_Create(t *testing.T) {
	tc := getTestContext(t)
	ctx := context.Background()
	now := time.Now().UTC()

	payment := &model.Payment{
		TransactionID: primitive.NewObjectID(),
		UserID:        1,
		Amount:        100.50,
		Date:          now,
		CreatedAt:     now,
	}

	err := tc.paymentRepo.Create(ctx, payment)
	require.NoError(t, err)
	assert.NotEqual(t, primitive.NilObjectID, payment.ID, "ID should be generated")
}

// PaymentRepository_FindByID tests retrieval by ObjectID with user scoping.
// Success case: payment exists and belongs to user.
// Not found cases: wrong user ID, non-existent ID.
func TestPaymentRepository_FindByID(t *testing.T) {
	tc := getTestContext(t)
	ctx := context.Background()
	userID := uint(80)
	now := time.Now().UTC()

	payment := &model.Payment{
		TransactionID: primitive.NewObjectID(),
		UserID:        userID,
		Amount:        200.00,
		Date:          now,
		CreatedAt:     now,
	}
	require.NoError(t, tc.paymentRepo.Create(ctx, payment))

	t.Run("success - existing payment", func(t *testing.T) {
		found, err := tc.paymentRepo.FindByID(ctx, payment.ID, userID)
		assert.NoError(t, err)
		assert.NotNil(t, found)
		assert.Equal(t, payment.Amount, found.Amount)
		assert.Equal(t, userID, found.UserID)
	})

	t.Run("not found - wrong user", func(t *testing.T) {
		found, err := tc.paymentRepo.FindByID(ctx, payment.ID, 999)
		assert.ErrorIs(t, err, ErrPaymentNotFound)
		assert.Nil(t, found)
	})

	t.Run("not found - non-existent ID", func(t *testing.T) {
		found, err := tc.paymentRepo.FindByID(ctx, primitive.NewObjectID(), userID)
		assert.ErrorIs(t, err, ErrPaymentNotFound)
		assert.Nil(t, found)
	})
}

// PaymentRepository_FindByTransactionID tests retrieval of all payments
// for a specific transaction ID.
func TestPaymentRepository_FindByTransactionID(t *testing.T) {
	tc := getTestContext(t)
	ctx := context.Background()
	userID := uint(90)
	now := time.Now().UTC()
	transactionID := primitive.NewObjectID()

	// Create payments for the transaction
	for i := 0; i < 3; i++ {
		payment := &model.Payment{
			TransactionID: transactionID,
			UserID:        userID,
			Amount:        100.00 * float64(i+1),
			Date:          now.Add(-time.Duration(i) * 24 * time.Hour),
			CreatedAt:     now,
		}
		require.NoError(t, tc.paymentRepo.Create(ctx, payment))
	}

	// Create payment for different transaction (should not appear)
	otherTransactionPayment := &model.Payment{
		TransactionID: primitive.NewObjectID(),
		UserID:        userID,
		Amount:        999.00,
		Date:          now,
		CreatedAt:     now,
	}
	require.NoError(t, tc.paymentRepo.Create(ctx, otherTransactionPayment))

	// Create payment for different user (should not appear)
	otherUserPayment := &model.Payment{
		TransactionID: transactionID,
		UserID:        999,
		Amount:        888.00,
		Date:          now,
		CreatedAt:     now,
	}
	require.NoError(t, tc.paymentRepo.Create(ctx, otherUserPayment))

	t.Run("returns payments for transaction", func(t *testing.T) {
		payments, err := tc.paymentRepo.FindByTransactionID(ctx, transactionID, userID)
		assert.NoError(t, err)
		assert.Len(t, payments, 3)
		// Verify descending order by date
		assert.True(t, payments[0].Date.After(payments[1].Date))
	})

	t.Run("returns empty for non-existent transaction", func(t *testing.T) {
		payments, err := tc.paymentRepo.FindByTransactionID(ctx, primitive.NewObjectID(), userID)
		assert.NoError(t, err)
		assert.Empty(t, payments)
	})
}

// PaymentRepository_FindByFilter tests the filter functionality with
// TransactionID, StartDate, EndDate, and UserID.
func TestPaymentRepository_FindByFilter(t *testing.T) {
	tc := getTestContext(t)
	ctx := context.Background()
	userID := uint(100)
	now := time.Now().UTC()
	transactionID := primitive.NewObjectID()

	// Create payments at different times
	payment1 := &model.Payment{
		TransactionID: transactionID,
		UserID:        userID,
		Amount:        100.00,
		Date:          now.Add(-48 * time.Hour),
		CreatedAt:     now,
	}
	require.NoError(t, tc.paymentRepo.Create(ctx, payment1))

	payment2 := &model.Payment{
		TransactionID: transactionID,
		UserID:        userID,
		Amount:        200.00,
		Date:          now.Add(-24 * time.Hour),
		CreatedAt:     now,
	}
	require.NoError(t, tc.paymentRepo.Create(ctx, payment2))

	payment3 := &model.Payment{
		TransactionID: primitive.NewObjectID(),
		UserID:        userID,
		Amount:        300.00,
		Date:          now,
		CreatedAt:     now,
	}
	require.NoError(t, tc.paymentRepo.Create(ctx, payment3))

	// Payment for different user
	otherUserPayment := &model.Payment{
		TransactionID: transactionID,
		UserID:        999,
		Amount:        9999.00,
		Date:          now,
		CreatedAt:     now,
	}
	require.NoError(t, tc.paymentRepo.Create(ctx, otherUserPayment))

	t.Run("filter by transaction ID only", func(t *testing.T) {
		filter := PaymentFilter{
			TransactionID: &transactionID,
			UserID:        userID,
		}
		payments, err := tc.paymentRepo.FindByFilter(ctx, filter, 0)
		assert.NoError(t, err)
		assert.Len(t, payments, 2)
	})

	t.Run("filter by date range only", func(t *testing.T) {
		startDate := now.Add(-36 * time.Hour)
		endDate := now.Add(-12 * time.Hour)
		filter := PaymentFilter{
			StartDate: &startDate,
			EndDate:   &endDate,
			UserID:    userID,
		}
		payments, err := tc.paymentRepo.FindByFilter(ctx, filter, 0)
		assert.NoError(t, err)
		assert.Len(t, payments, 1)
		assert.Equal(t, payment2.ID, payments[0].ID)
	})

	t.Run("filter by transaction ID and date range", func(t *testing.T) {
		startDate := now.Add(-72 * time.Hour)
		endDate := now.Add(-12 * time.Hour)
		filter := PaymentFilter{
			TransactionID: &transactionID,
			StartDate:     &startDate,
			EndDate:       &endDate,
			UserID:        userID,
		}
		payments, err := tc.paymentRepo.FindByFilter(ctx, filter, 0)
		assert.NoError(t, err)
		assert.Len(t, payments, 2)
	})

	t.Run("filter with limit", func(t *testing.T) {
		filter := PaymentFilter{
			UserID: userID,
		}
		payments, err := tc.paymentRepo.FindByFilter(ctx, filter, 2)
		assert.NoError(t, err)
		assert.Len(t, payments, 2)
	})

	t.Run("no filter returns all user payments", func(t *testing.T) {
		filter := PaymentFilter{
			UserID: userID,
		}
		payments, err := tc.paymentRepo.FindByFilter(ctx, filter, 0)
		assert.NoError(t, err)
		assert.Len(t, payments, 3)
	})
}

// PaymentRepository_FindCurrentPeriodPayment tests retrieval of the payment
// for the current period based on frequency (monthly, bimonthly, yearly).
func TestPaymentRepository_FindCurrentPeriodPayment(t *testing.T) {
	tc := getTestContext(t)
	ctx := context.Background()
	userID := uint(110)
	now := time.Now().UTC()
	transactionID := primitive.NewObjectID()

	// Create payment for current month
	currentMonthPayment := &model.Payment{
		TransactionID: transactionID,
		UserID:        userID,
		Amount:        100.00,
		Date:          now,
		CreatedAt:     now,
	}
	require.NoError(t, tc.paymentRepo.Create(ctx, currentMonthPayment))

	// Create payment for last month (should not be found)
	lastMonthPayment := &model.Payment{
		TransactionID: transactionID,
		UserID:        userID,
		Amount:        200.00,
		Date:          now.AddDate(0, -1, 0),
		CreatedAt:     now,
	}
	require.NoError(t, tc.paymentRepo.Create(ctx, lastMonthPayment))

	// Create payment for different user (should not be found)
	otherUserPayment := &model.Payment{
		TransactionID: transactionID,
		UserID:        999,
		Amount:        300.00,
		Date:          now,
		CreatedAt:     now,
	}
	require.NoError(t, tc.paymentRepo.Create(ctx, otherUserPayment))

	t.Run("find current period payment - monthly frequency", func(t *testing.T) {
		payment, err := tc.paymentRepo.FindCurrentPeriodPayment(ctx, transactionID, userID, model.PaymentFrequencyMonthly)
		assert.NoError(t, err)
		assert.NotNil(t, payment)
		assert.Equal(t, currentMonthPayment.ID, payment.ID)
	})

	t.Run("find current period payment - bimonthly frequency", func(t *testing.T) {
		payment, err := tc.paymentRepo.FindCurrentPeriodPayment(ctx, transactionID, userID, model.PaymentFrequencyBimonthly)
		assert.NoError(t, err)
		assert.NotNil(t, payment)
	})

	t.Run("find current period payment - yearly frequency", func(t *testing.T) {
		payment, err := tc.paymentRepo.FindCurrentPeriodPayment(ctx, transactionID, userID, model.PaymentFrequencyYearly)
		assert.NoError(t, err)
		assert.NotNil(t, payment)
	})

	t.Run("returns nil if no payment in current period", func(t *testing.T) {
		differentTransactionID := primitive.NewObjectID()
		payment, err := tc.paymentRepo.FindCurrentPeriodPayment(ctx, differentTransactionID, userID, model.PaymentFrequencyMonthly)
		assert.NoError(t, err)
		assert.Nil(t, payment)
	})
}

// PaymentRepository_Delete tests deletion with user scoping.
// Success: payment is deleted.
// Not found: wrong user, non-existent ID.
func TestPaymentRepository_Delete(t *testing.T) {
	tc := getTestContext(t)
	ctx := context.Background()
	userID := uint(120)
	now := time.Now().UTC()

	payment := &model.Payment{
		TransactionID: primitive.NewObjectID(),
		UserID:        userID,
		Amount:        100.00,
		Date:          now,
		CreatedAt:     now,
	}
	require.NoError(t, tc.paymentRepo.Create(ctx, payment))

	t.Run("successful deletion", func(t *testing.T) {
		err := tc.paymentRepo.Delete(ctx, payment.ID, userID)
		assert.NoError(t, err)

		// Should not be findable
		found, err := tc.paymentRepo.FindByID(ctx, payment.ID, userID)
		assert.ErrorIs(t, err, ErrPaymentNotFound)
		assert.Nil(t, found)
	})

	t.Run("delete non-existent payment", func(t *testing.T) {
		err := tc.paymentRepo.Delete(ctx, primitive.NewObjectID(), userID)
		assert.ErrorIs(t, err, ErrPaymentNotFound)
	})

	t.Run("delete with wrong user", func(t *testing.T) {
		// Create new payment to delete
		payment2 := &model.Payment{
			TransactionID: primitive.NewObjectID(),
			UserID:        userID,
			Amount:        200.00,
			Date:          now,
			CreatedAt:     now,
		}
		require.NoError(t, tc.paymentRepo.Create(ctx, payment2))

		err := tc.paymentRepo.Delete(ctx, payment2.ID, 999)
		assert.ErrorIs(t, err, ErrPaymentNotFound)

		// Verify payment still exists
		found, err := tc.paymentRepo.FindByID(ctx, payment2.ID, userID)
		assert.NoError(t, err)
		assert.NotNil(t, found)
	})
}

// PaymentRepository_DeleteByTransactionID tests deletion of all payments
// for a specific transaction ID.
func TestPaymentRepository_DeleteByTransactionID(t *testing.T) {
	tc := getTestContext(t)
	ctx := context.Background()
	userID := uint(130)
	now := time.Now().UTC()
	transactionID := primitive.NewObjectID()

	// Create multiple payments for the transaction
	for i := 0; i < 3; i++ {
		payment := &model.Payment{
			TransactionID: transactionID,
			UserID:        userID,
			Amount:        100.00 * float64(i+1),
			Date:          now.Add(-time.Duration(i) * 24 * time.Hour),
			CreatedAt:     now,
		}
		require.NoError(t, tc.paymentRepo.Create(ctx, payment))
	}

	// Create payment for different transaction (should not be deleted)
	otherTransactionID := primitive.NewObjectID()
	otherPayment := &model.Payment{
		TransactionID: otherTransactionID,
		UserID:        userID,
		Amount:        999.00,
		Date:          now,
		CreatedAt:     now,
	}
	require.NoError(t, tc.paymentRepo.Create(ctx, otherPayment))

	// Create payment for different user (should not be deleted)
	otherUserPayment := &model.Payment{
		TransactionID: transactionID,
		UserID:        999,
		Amount:        888.00,
		Date:          now,
		CreatedAt:     now,
	}
	require.NoError(t, tc.paymentRepo.Create(ctx, otherUserPayment))

	t.Run("deletes all payments for transaction", func(t *testing.T) {
		err := tc.paymentRepo.DeleteByTransactionID(ctx, transactionID, userID)
		assert.NoError(t, err)

		// Verify payments are deleted
		payments, err := tc.paymentRepo.FindByTransactionID(ctx, transactionID, userID)
		assert.NoError(t, err)
		assert.Empty(t, payments)

		// Verify other transaction payment still exists
		found, err := tc.paymentRepo.FindByID(ctx, otherPayment.ID, userID)
		assert.NoError(t, err)
		assert.NotNil(t, found)

		// Verify other user payment still exists
		found, err = tc.paymentRepo.FindByID(ctx, otherUserPayment.ID, 999)
		assert.NoError(t, err)
		assert.NotNil(t, found)
	})
}

// PaymentRepository_AggregateByTransaction tests the aggregation pipeline that
// groups payments by transaction ID with sum and count.
func TestPaymentRepository_AggregateByTransaction(t *testing.T) {
	tc := getTestContext(t)
	ctx := context.Background()
	userID := uint(140)
	now := time.Now().UTC()

	// Create payments for transaction 1
	transactionID1 := primitive.NewObjectID()
	for i := 0; i < 3; i++ {
		payment := &model.Payment{
			TransactionID: transactionID1,
			UserID:        userID,
			Amount:        100.00,
			Date:          now.Add(-time.Duration(i) * 24 * time.Hour),
			CreatedAt:     now,
		}
		require.NoError(t, tc.paymentRepo.Create(ctx, payment))
	}

	// Create payments for transaction 2
	transactionID2 := primitive.NewObjectID()
	for i := 0; i < 2; i++ {
		payment := &model.Payment{
			TransactionID: transactionID2,
			UserID:        userID,
			Amount:        200.00,
			Date:          now.Add(-time.Duration(i) * 24 * time.Hour),
			CreatedAt:     now,
		}
		require.NoError(t, tc.paymentRepo.Create(ctx, payment))
	}

	// Create payment for different user
	otherUserPayment := &model.Payment{
		TransactionID: transactionID1,
		UserID:        999,
		Amount:        9999.00,
		Date:          now,
		CreatedAt:     now,
	}
	require.NoError(t, tc.paymentRepo.Create(ctx, otherUserPayment))

	t.Run("aggregate by transaction returns grouped results", func(t *testing.T) {
		transactionIDs := []primitive.ObjectID{transactionID1, transactionID2}
		results, err := tc.paymentRepo.AggregateByTransaction(ctx, transactionIDs, userID)
		assert.NoError(t, err)
		assert.Len(t, results, 2)

		// Find result for transaction 1
		var result1 *PaymentAggregationResult
		for _, r := range results {
			if r.TransactionID == transactionID1 {
				result1 = &r
				break
			}
		}
		assert.NotNil(t, result1)
		assert.Equal(t, 300.00, result1.Total)
		assert.Equal(t, int64(3), result1.Count)
	})

	t.Run("aggregate with empty transaction IDs returns nil", func(t *testing.T) {
		results, err := tc.paymentRepo.AggregateByTransaction(ctx, []primitive.ObjectID{}, userID)
		assert.NoError(t, err)
		assert.Nil(t, results)
	})

	t.Run("aggregate with non-existent transaction IDs returns empty", func(t *testing.T) {
		results, err := tc.paymentRepo.AggregateByTransaction(ctx, []primitive.ObjectID{primitive.NewObjectID()}, userID)
		assert.NoError(t, err)
		assert.Empty(t, results)
	})
}

// PaymentRepository_AggregateByDateRange tests the aggregation pipeline that
// groups payments by transaction ID within a date range.
func TestPaymentRepository_AggregateByDateRange(t *testing.T) {
	tc := getTestContext(t)
	ctx := context.Background()
	userID := uint(150)
	now := time.Now().UTC()
	transactionID := primitive.NewObjectID()

	// Create payments within date range
	for i := 0; i < 3; i++ {
		payment := &model.Payment{
			TransactionID: transactionID,
			UserID:        userID,
			Amount:        100.00,
			Date:          now.Add(-time.Duration(i) * 24 * time.Hour),
			CreatedAt:     now,
		}
		require.NoError(t, tc.paymentRepo.Create(ctx, payment))
	}

	// Create payment outside date range (should not be included)
	oldPayment := &model.Payment{
		TransactionID: transactionID,
		UserID:        userID,
		Amount:        999.00,
		Date:          now.Add(-30 * 24 * time.Hour),
		CreatedAt:     now,
	}
	require.NoError(t, tc.paymentRepo.Create(ctx, oldPayment))

	// Create payment for different user (should not be included)
	otherUserPayment := &model.Payment{
		TransactionID: transactionID,
		UserID:        999,
		Amount:        888.00,
		Date:          now,
		CreatedAt:     now,
	}
	require.NoError(t, tc.paymentRepo.Create(ctx, otherUserPayment))

	startDate := now.Add(-7 * 24 * time.Hour)
	endDate := now

	t.Run("aggregate by date range returns grouped results", func(t *testing.T) {
		results, err := tc.paymentRepo.AggregateByDateRange(ctx, userID, startDate, endDate)
		assert.NoError(t, err)
		assert.Len(t, results, 1)
		assert.Equal(t, transactionID, results[0].TransactionID)
		assert.Equal(t, 300.00, results[0].Total)
		assert.Equal(t, int64(3), results[0].Count)
	})

	t.Run("aggregate with no payments in range returns empty", func(t *testing.T) {
		futureStart := now.Add(365 * 24 * time.Hour)
		futureEnd := now.Add(366 * 24 * time.Hour)
		results, err := tc.paymentRepo.AggregateByDateRange(ctx, userID, futureStart, futureEnd)
		assert.NoError(t, err)
		assert.Empty(t, results)
	})
}

package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"life-tracker-backend/internal/domain/finance/dto"
	"life-tracker-backend/internal/domain/finance/model"
	"life-tracker-backend/internal/domain/finance/repository"
	"life-tracker-backend/internal/infrastructure/monitoring"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type FinanceService struct {
	transactionRepo repository.TransactionRepository
	paymentRepo     repository.PaymentRepository
}

func NewFinanceService(mongoDB *mongo.Database) *FinanceService {
	return &FinanceService{
		transactionRepo: repository.NewTransactionRepository(mongoDB),
		paymentRepo:     repository.NewPaymentRepository(mongoDB),
	}
}

func (s *FinanceService) GetCategories(transactionType *string, frequency *string) []*dto.CategoryResponse {
	var categories []model.Category

	switch {
	case transactionType != nil && frequency != nil:
		for _, cat := range model.Categories {
			if string(cat.Type) == *transactionType && string(cat.ApplicableToFreq) == *frequency {
				categories = append(categories, cat)
			}
		}
	case transactionType != nil:
		for _, cat := range model.Categories {
			if string(cat.Type) == *transactionType {
				categories = append(categories, cat)
			}
		}
	case frequency != nil:
		for _, cat := range model.Categories {
			if string(cat.ApplicableToFreq) == *frequency {
				categories = append(categories, cat)
			}
		}
	default:
		categories = model.Categories
	}

	responses := make([]*dto.CategoryResponse, len(categories))
	for i := range categories {
		responses[i] = &dto.CategoryResponse{
			Name:             categories[i].Name,
			Type:             string(categories[i].Type),
			Icon:             categories[i].Icon,
			ApplicableToFreq: string(categories[i].ApplicableToFreq),
		}
	}
	return responses
}

func (s *FinanceService) CreateTransaction(userID uint, req *dto.CreateTransactionRequest) (*dto.TransactionResponse, error) {
	if !model.IsValidCategoryName(req.Category) {
		return nil, errors.New("invalid category")
	}

	transactionDate := time.Now()
	if !req.Date.IsZero() {
		transactionDate = req.Date
	}

	transaction := model.Transaction{
		ID:          primitive.NewObjectID(),
		UserID:      userID,
		Type:        model.TransactionType(req.Type),
		Frequency:   model.TransactionFrequency(req.Frequency),
		Amount:      req.Amount,
		Category:    req.Category,
		Description: req.Description,
		Date:        transactionDate,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	if req.Frequency == "fixed" {
		transaction.Amount = 0
		if req.PaymentFrequency != "" {
			transaction.PaymentFrequency = model.PaymentFrequency(req.PaymentFrequency)
		} else {
			transaction.PaymentFrequency = model.PaymentFrequencyMonthly
		}
	}

	if err := s.transactionRepo.Create(context.Background(), &transaction); err != nil {
		return nil, errors.New("failed to create transaction")
	}

	monitoring.TransactionsCreated.WithLabelValues(
		string(transaction.Type),
		transaction.Category,
	).Inc()

	monitoring.TransactionAmount.WithLabelValues(
		string(transaction.Type),
		transaction.Category,
	).Observe(transaction.Amount)

	if transaction.IsFixed() {
		monitoring.FixedTransactionsTotal.WithLabelValues(string(transaction.Type)).Inc()
	}

	return transaction.ToResponse(), nil
}

func (s *FinanceService) GetTransactions(userID uint, transactionType *string, frequency *string, startDate, endDate *time.Time, month, year *int, category *string, limit int, loc *time.Location) ([]*dto.TransactionResponse, error) {
	effectiveStart, effectiveEnd := s.calculateDateRange(startDate, endDate, month, year, loc)

	filter := repository.TransactionFilter{
		UserID:          userID,
		TransactionType: transactionType,
		Frequency:       frequency,
		Category:        category,
		StartDate:       effectiveStart,
		EndDate:         effectiveEnd,
	}

	transactions, err := s.transactionRepo.FindByFilter(context.Background(), filter, limit)
	if err != nil {
		return nil, errors.New("failed to fetch transactions")
	}

	responses := make([]*dto.TransactionResponse, len(transactions))
	for i := range transactions {
		responses[i] = transactions[i].ToResponse()
	}
	return responses, nil
}

func (s *FinanceService) GetFixedTransactions(userID uint, month, year *int, loc *time.Location) ([]*dto.FixedTransactionResponse, error) {
	ctx := context.Background()

	transactions, err := s.transactionRepo.FindFixedTransactions(ctx, userID)
	if err != nil {
		return nil, errors.New("failed to fetch fixed transactions")
	}

	if len(transactions) == 0 {
		return []*dto.FixedTransactionResponse{}, nil
	}

	transactionIDs := make([]primitive.ObjectID, len(transactions))
	for i, t := range transactions {
		transactionIDs[i] = t.ID
	}

	var paymentAggregations []repository.PaymentAggregationResult

	if month != nil && year != nil {
		startDate := time.Date(*year, time.Month(*month), 1, 0, 0, 0, 0, loc)
		endDate := startDate.AddDate(0, 1, 0).Add(-time.Nanosecond)
		paymentAggregations, err = s.paymentRepo.AggregateByDateRange(ctx, userID, startDate, endDate)
	} else {
		paymentAggregations, err = s.paymentRepo.AggregateByTransaction(ctx, transactionIDs, userID)
	}

	if err != nil {
		return nil, errors.New("failed to fetch payment aggregations")
	}

	paymentTotals := make(map[primitive.ObjectID]float64)
	for _, agg := range paymentAggregations {
		paymentTotals[agg.TransactionID] = agg.Total
	}

	responses := make([]*dto.FixedTransactionResponse, len(transactions))
	for i, t := range transactions {
		totalPaid := paymentTotals[t.ID]

		var currentPeriodPayment *dto.PaymentSummary
		payment, err := s.paymentRepo.FindCurrentPeriodPayment(ctx, t.ID, userID, t.PaymentFrequency)
		if err == nil && payment != nil {
			summary := payment.ToSummary()
			currentPeriodPayment = &summary
		}

		responses[i] = t.ToFixedResponse(totalPaid, nil, currentPeriodPayment)
	}

	return responses, nil
}

func (s *FinanceService) GetFixedTransactionWithPayments(userID uint, transactionID string) (*dto.FixedTransactionResponse, error) {
	ctx := context.Background()

	objID, err := primitive.ObjectIDFromHex(transactionID)
	if err != nil {
		return nil, errors.New("invalid transaction ID")
	}

	transaction, err := s.transactionRepo.FindByID(ctx, objID, userID)
	if err != nil {
		if errors.Is(err, repository.ErrTransactionNotFound) {
			return nil, errors.New("transaction not found")
		}
		return nil, errors.New("failed to fetch transaction")
	}

	if !transaction.IsFixed() {
		return nil, errors.New("transaction is not a fixed transaction")
	}

	payments, err := s.paymentRepo.FindByTransactionID(ctx, objID, userID)
	if err != nil {
		return nil, errors.New("failed to fetch payments")
	}

	var totalPaid float64
	paymentSummaries := make([]dto.PaymentSummary, len(payments))
	for i, p := range payments {
		totalPaid += p.Amount
		paymentSummaries[i] = p.ToSummary()
	}

	var currentPeriodPayment *dto.PaymentSummary
	payment, err := s.paymentRepo.FindCurrentPeriodPayment(ctx, objID, userID, transaction.PaymentFrequency)
	if err == nil && payment != nil {
		summary := payment.ToSummary()
		currentPeriodPayment = &summary
	}

	return transaction.ToFixedResponse(totalPaid, paymentSummaries, currentPeriodPayment), nil
}

func (s *FinanceService) GetTransaction(userID uint, transactionID string) (*dto.TransactionResponse, error) {
	objID, err := primitive.ObjectIDFromHex(transactionID)
	if err != nil {
		return nil, errors.New("invalid transaction ID")
	}

	transaction, err := s.transactionRepo.FindByID(context.Background(), objID, userID)
	if err != nil {
		if errors.Is(err, repository.ErrTransactionNotFound) {
			return nil, errors.New("transaction not found")
		}
		return nil, errors.New("failed to fetch transaction")
	}

	return transaction.ToResponse(), nil
}

func (s *FinanceService) UpdateTransaction(userID uint, transactionID string, req *dto.UpdateTransactionRequest) (*dto.TransactionResponse, error) {
	objID, err := primitive.ObjectIDFromHex(transactionID)
	if err != nil {
		return nil, errors.New("invalid transaction ID")
	}

	_, err = s.transactionRepo.FindByID(context.Background(), objID, userID)
	if err != nil {
		return nil, errors.New("transaction not found")
	}

	updates := bson.M{"updatedAt": time.Now()}

	if req.Type != nil {
		updates["type"] = *req.Type
	}
	if req.Frequency != nil {
		updates["frequency"] = *req.Frequency
	}
	if req.PaymentFrequency != nil {
		updates["paymentFrequency"] = *req.PaymentFrequency
	}
	if req.Amount != nil {
		updates["amount"] = *req.Amount
	}
	if req.Category != nil {
		if !model.IsValidCategoryName(*req.Category) {
			return nil, errors.New("invalid category")
		}
		updates["category"] = *req.Category
	}
	if req.Description != nil {
		updates["description"] = *req.Description
	}
	if req.Date != nil {
		updates["date"] = *req.Date
	}

	if err := s.transactionRepo.Update(context.Background(), objID, updates); err != nil {
		return nil, errors.New("failed to update transaction")
	}

	monitoring.TransactionsUpdated.Inc()

	return s.GetTransaction(userID, transactionID)
}

func (s *FinanceService) DeleteTransaction(userID uint, transactionID string) error {
	objID, err := primitive.ObjectIDFromHex(transactionID)
	if err != nil {
		return errors.New("invalid transaction ID")
	}

	ctx := context.Background()

	transaction, err := s.transactionRepo.FindByID(ctx, objID, userID)
	if err != nil {
		if errors.Is(err, repository.ErrTransactionNotFound) {
			return errors.New("transaction not found")
		}
		return errors.New("failed to fetch transaction")
	}

	if err := s.paymentRepo.DeleteByTransactionID(ctx, objID, userID); err != nil {
		return errors.New("failed to delete associated payments")
	}

	if err := s.transactionRepo.Delete(ctx, objID, userID); err != nil {
		if errors.Is(err, repository.ErrTransactionNotFound) {
			return errors.New("transaction not found")
		}
		return errors.New("failed to delete transaction")
	}

	monitoring.TransactionsDeleted.Inc()

	if transaction.IsFixed() {
		monitoring.FixedTransactionsTotal.WithLabelValues(string(transaction.Type)).Dec()
	}

	return nil
}

func (s *FinanceService) CreatePayment(userID uint, req *dto.CreatePaymentRequest) (*dto.PaymentResponse, error) {
	ctx := context.Background()

	transactionID, err := primitive.ObjectIDFromHex(req.TransactionID)
	if err != nil {
		return nil, errors.New("invalid transaction ID")
	}

	transaction, err := s.transactionRepo.FindByID(ctx, transactionID, userID)
	if err != nil {
		if errors.Is(err, repository.ErrTransactionNotFound) {
			return nil, errors.New("transaction not found")
		}
		return nil, errors.New("failed to verify transaction")
	}

	if !transaction.IsFixed() {
		return nil, errors.New("payments can only be created for fixed transactions")
	}

	existingPayment, err := s.paymentRepo.FindCurrentPeriodPayment(ctx, transactionID, userID, transaction.PaymentFrequency)
	if err != nil {
		return nil, errors.New("failed to check existing payment")
	}
	if existingPayment != nil {
		return nil, errors.New("payment already exists for this period")
	}

	paymentDate := time.Now()
	if !req.Date.IsZero() {
		paymentDate = req.Date
	}

	payment := model.Payment{
		ID:            primitive.NewObjectID(),
		TransactionID: transactionID,
		UserID:        userID,
		Amount:        req.Amount,
		Date:          paymentDate,
		CreatedAt:     time.Now(),
	}

	if err := s.paymentRepo.Create(ctx, &payment); err != nil {
		return nil, errors.New("failed to create payment")
	}

	monitoring.PaymentsCreated.WithLabelValues(string(transaction.Type)).Inc()
	monitoring.PaymentAmount.WithLabelValues(string(transaction.Type)).Observe(payment.Amount)

	if transaction.Type == model.TransactionTypeIncome {
		monitoring.TotalIncomeAmount.Add(payment.Amount)
	} else {
		monitoring.TotalOutcomeAmount.Add(payment.Amount)
	}

	return payment.ToResponse(), nil
}

func (s *FinanceService) GetPayments(userID uint, transactionID *string, startDate, endDate *time.Time, limit int) ([]*dto.PaymentResponse, error) {
	ctx := context.Background()

	filter := repository.PaymentFilter{
		UserID:    userID,
		StartDate: startDate,
		EndDate:   endDate,
	}

	if transactionID != nil {
		objID, err := primitive.ObjectIDFromHex(*transactionID)
		if err != nil {
			return nil, errors.New("invalid transaction ID")
		}
		filter.TransactionID = &objID
	}

	payments, err := s.paymentRepo.FindByFilter(ctx, filter, limit)
	if err != nil {
		return nil, errors.New("failed to fetch payments")
	}

	responses := make([]*dto.PaymentResponse, len(payments))
	for i := range payments {
		responses[i] = payments[i].ToResponse()
	}

	return responses, nil
}

func (s *FinanceService) DeletePayment(userID uint, paymentID string) error {
	objID, err := primitive.ObjectIDFromHex(paymentID)
	if err != nil {
		return errors.New("invalid payment ID")
	}

	if err := s.paymentRepo.Delete(context.Background(), objID, userID); err != nil {
		if errors.Is(err, repository.ErrPaymentNotFound) {
			return errors.New("payment not found")
		}
		return errors.New("failed to delete payment")
	}

	monitoring.PaymentsDeleted.Inc()

	return nil
}

func (s *FinanceService) GetFinanceSummary(userID uint, startDate, endDate time.Time, loc *time.Location) (*dto.FinanceSummaryResponse, error) {
	startInLoc := time.Date(startDate.Year(), startDate.Month(), startDate.Day(), 0, 0, 0, 0, loc)
	endInLoc := time.Date(endDate.Year(), endDate.Month(), endDate.Day(), 23, 59, 59, 999999999, loc)
	ctx := context.Background()

	variableResults, err := s.transactionRepo.Aggregate(ctx, userID, startInLoc, endInLoc)
	if err != nil {
		return nil, errors.New("failed to generate summary")
	}

	paymentResults, err := s.paymentRepo.AggregateByDateRange(ctx, userID, startInLoc, endInLoc)
	if err != nil {
		return nil, errors.New("failed to aggregate payments")
	}

	fixedTransactions, err := s.transactionRepo.FindFixedTransactions(ctx, userID)
	if err != nil {
		return nil, errors.New("failed to fetch fixed transactions")
	}

	fixedTxMap := make(map[primitive.ObjectID]*model.Transaction)
	for i := range fixedTransactions {
		fixedTxMap[fixedTransactions[i].ID] = &fixedTransactions[i]
	}

	incomeByCategory, outcomeByCategory, totalIncome, totalOutcome := s.processSummaryData(
		variableResults,
		paymentResults,
		fixedTxMap,
	)

	incomeList := s.buildCategoryList(incomeByCategory, totalIncome)
	outcomeList := s.buildCategoryList(outcomeByCategory, totalOutcome)

	period := fmt.Sprintf("%s to %s", startDate.Format("2006-01-02"), endDate.Format("2006-01-02"))

	return &dto.FinanceSummaryResponse{
		TotalIncome:       totalIncome,
		TotalOutcome:      totalOutcome,
		Balance:           totalIncome - totalOutcome,
		IncomeByCategory:  incomeList,
		OutcomeByCategory: outcomeList,
		Period:            period,
	}, nil
}

func (s *FinanceService) processSummaryData(
	variableResults []repository.AggregationResult,
	paymentResults []repository.PaymentAggregationResult,
	fixedTxMap map[primitive.ObjectID]*model.Transaction,
) (map[string]*dto.CategorySummary, map[string]*dto.CategorySummary, float64, float64) {
	incomeByCategory := make(map[string]*dto.CategorySummary)
	outcomeByCategory := make(map[string]*dto.CategorySummary)
	var totalIncome, totalOutcome float64

	for _, result := range variableResults {
		summary := &dto.CategorySummary{
			CategoryName: result.Category,
			Total:        result.Total,
			Count:        result.Count,
		}
		if result.Type == "income" {
			totalIncome += result.Total
			incomeByCategory[result.Category] = summary
		} else {
			totalOutcome += result.Total
			outcomeByCategory[result.Category] = summary
		}
	}

	for _, paymentAgg := range paymentResults {
		tx, exists := fixedTxMap[paymentAgg.TransactionID]
		if !exists {
			continue
		}

		if tx.Type == model.TransactionTypeIncome {
			totalIncome += paymentAgg.Total
			s.addOrUpdateCategorySummary(incomeByCategory, tx.Category, paymentAgg.Total, paymentAgg.Count)
		} else {
			totalOutcome += paymentAgg.Total
			s.addOrUpdateCategorySummary(outcomeByCategory, tx.Category, paymentAgg.Total, paymentAgg.Count)
		}
	}

	return incomeByCategory, outcomeByCategory, totalIncome, totalOutcome
}

func (s *FinanceService) addOrUpdateCategorySummary(
	summaryMap map[string]*dto.CategorySummary,
	category string,
	total float64,
	count int64,
) {
	if existing, ok := summaryMap[category]; ok {
		existing.Total += total
		existing.Count += count
	} else {
		summaryMap[category] = &dto.CategorySummary{
			CategoryName: category,
			Total:        total,
			Count:        count,
		}
	}
}

func (s *FinanceService) buildCategoryList(categoryMap map[string]*dto.CategorySummary, total float64) []dto.CategorySummary {
	list := make([]dto.CategorySummary, 0, len(categoryMap))
	for _, summary := range categoryMap {
		if total > 0 {
			summary.Percentage = (summary.Total / total) * 100
		}
		list = append(list, *summary)
	}
	return list
}

func (s *FinanceService) GetMonthlyStats(userID uint, year int, loc *time.Location) ([]*dto.MonthlyStatsResponse, error) {
	results, err := s.transactionRepo.AggregateMonthly(context.Background(), userID, year, loc)
	if err != nil {
		return nil, errors.New("failed to generate monthly stats")
	}

	monthlyData := make(map[int]struct {
		income  float64
		outcome float64
		count   int64
	})

	for _, result := range results {
		data := monthlyData[result.Month]
		if result.Type == "income" {
			data.income = result.Total
		} else {
			data.outcome = result.Total
		}
		data.count += result.Count
		monthlyData[result.Month] = data
	}

	stats := make([]*dto.MonthlyStatsResponse, 0, 12)
	for month := 1; month <= 12; month++ {
		data := monthlyData[month]
		stats = append(stats, &dto.MonthlyStatsResponse{
			Month:            time.Month(month).String(),
			Year:             year,
			TotalIncome:      data.income,
			TotalOutcome:     data.outcome,
			Balance:          data.income - data.outcome,
			TransactionCount: data.count,
		})
	}

	return stats, nil
}

func (s *FinanceService) calculateDateRange(startDate, endDate *time.Time, month, year *int, loc *time.Location) (*time.Time, *time.Time) {
	if month != nil && year != nil {
		start := time.Date(*year, time.Month(*month), 1, 0, 0, 0, 0, loc)
		end := start.AddDate(0, 1, 0).Add(-time.Nanosecond)
		return &start, &end
	}

	if month != nil {
		currentYear := time.Now().In(loc).Year()
		start := time.Date(currentYear, time.Month(*month), 1, 0, 0, 0, 0, loc)
		end := start.AddDate(0, 1, 0).Add(-time.Nanosecond)
		return &start, &end
	}

	if year != nil {
		start := time.Date(*year, 1, 1, 0, 0, 0, 0, loc)
		end := time.Date(*year, 12, 31, 23, 59, 59, 999999999, loc)
		return &start, &end
	}

	if startDate != nil || endDate != nil {
		var effectiveStart, effectiveEnd *time.Time
		if startDate != nil {
			start := time.Date(startDate.Year(), startDate.Month(), startDate.Day(), 0, 0, 0, 0, loc)
			effectiveStart = &start
		}
		if endDate != nil {
			end := time.Date(endDate.Year(), endDate.Month(), endDate.Day(), 23, 59, 59, 999999999, loc)
			effectiveEnd = &end
		}
		return effectiveStart, effectiveEnd
	}

	now := time.Now().In(loc)
	start := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, loc)
	end := start.AddDate(0, 1, 0).Add(-time.Nanosecond)
	return &start, &end
}

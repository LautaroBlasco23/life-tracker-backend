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
	"gorm.io/gorm"
)

type FinanceService struct {
	categoryRepo    repository.CategoryRepository
	transactionRepo repository.TransactionRepository
}

func NewFinanceService(db *gorm.DB, mongoDB *mongo.Database) *FinanceService {
	return &FinanceService{
		categoryRepo:    repository.NewCategoryRepository(db),
		transactionRepo: repository.NewTransactionRepository(mongoDB),
	}
}

func (s *FinanceService) InitializeSystemCategories() error {
	for _, cat := range model.SystemCategories {
		_, err := s.categoryRepo.FindByName(cat.Name)
		if err != nil {
			if !errors.Is(err, repository.ErrCategoryNotFound) {
				return fmt.Errorf("failed to check category %s: %w", cat.Name, err)
			}
			if err := s.categoryRepo.Create(&cat); err != nil {
				return fmt.Errorf("failed to create category %s: %w", cat.Name, err)
			}
		}
	}
	return nil
}

func (s *FinanceService) GetCategories(transactionType *string, frequency *string) ([]*dto.CategoryResponse, error) {
	categories, err := s.categoryRepo.FindAll(transactionType, frequency)
	if err != nil {
		return nil, errors.New("failed to fetch categories")
	}

	responses := make([]*dto.CategoryResponse, len(categories))
	for i := range categories {
		responses[i] = s.categoryToResponse(&categories[i])
	}
	return responses, nil
}

func (s *FinanceService) CreateTransaction(userID uint, req *dto.CreateTransactionRequest) (*dto.TransactionResponse, error) {
	category, err := s.categoryRepo.FindByID(req.CategoryID)
	if err != nil {
		return nil, errors.New("category not found")
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
		CategoryID:  req.CategoryID,
		Description: req.Description,
		Date:        transactionDate,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	if err := s.transactionRepo.Create(context.Background(), &transaction); err != nil {
		return nil, errors.New("failed to create transaction")
	}

	monitoring.TransactionsCreated.WithLabelValues(
		string(transaction.Type),
		category.Name,
	).Inc()

	monitoring.TransactionAmount.WithLabelValues(
		string(transaction.Type),
		category.Name,
	).Observe(transaction.Amount)

	return transaction.ToResponse(category.Name), nil
}

func (s *FinanceService) GetTransactions(userID uint, transactionType *string, frequency *string, startDate, endDate *time.Time, month, year *int, categoryID *uint, limit int, loc *time.Location) ([]*dto.TransactionResponse, error) {
	effectiveStart, effectiveEnd := s.calculateDateRange(startDate, endDate, month, year, loc)

	filter := repository.TransactionFilter{
		UserID:          userID,
		TransactionType: transactionType,
		Frequency:       frequency,
		CategoryID:      categoryID,
		StartDate:       effectiveStart,
		EndDate:         effectiveEnd,
	}

	transactions, err := s.transactionRepo.FindByFilter(context.Background(), filter, limit)
	if err != nil {
		return nil, errors.New("failed to fetch transactions")
	}

	categoryMap := s.loadCategoryMap()

	responses := make([]*dto.TransactionResponse, len(transactions))
	for i := range transactions {
		responses[i] = transactions[i].ToResponse(categoryMap[transactions[i].CategoryID])
	}
	return responses, nil
}

func (s *FinanceService) loadCategoryMap() map[uint]string {
	categoryMap := make(map[uint]string)
	categories, err := s.categoryRepo.FindAll(nil, nil)
	if err == nil {
		for _, cat := range categories {
			categoryMap[cat.ID] = cat.Name
		}
	}
	return categoryMap
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

	var categoryName string
	category, err := s.categoryRepo.FindByID(transaction.CategoryID)
	if err == nil {
		categoryName = category.Name
	}

	return transaction.ToResponse(categoryName), nil
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
	if req.Amount != nil {
		updates["amount"] = *req.Amount
	}
	if req.CategoryID != nil {
		if _, err = s.categoryRepo.FindByID(*req.CategoryID); err != nil {
			return nil, errors.New("category not found")
		}
		updates["categoryId"] = *req.CategoryID
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

	return s.GetTransaction(userID, transactionID)
}

func (s *FinanceService) DeleteTransaction(userID uint, transactionID string) error {
	objID, err := primitive.ObjectIDFromHex(transactionID)
	if err != nil {
		return errors.New("invalid transaction ID")
	}

	if err := s.transactionRepo.Delete(context.Background(), objID, userID); err != nil {
		if errors.Is(err, repository.ErrTransactionNotFound) {
			return errors.New("transaction not found")
		}
		return errors.New("failed to delete transaction")
	}

	monitoring.TransactionsDeleted.Inc()
	return nil
}

func (s *FinanceService) GetFinanceSummary(userID uint, startDate, endDate time.Time, loc *time.Location) (*dto.FinanceSummaryResponse, error) {
	startInLoc := time.Date(startDate.Year(), startDate.Month(), startDate.Day(), 0, 0, 0, 0, loc)
	endInLoc := time.Date(endDate.Year(), endDate.Month(), endDate.Day(), 23, 59, 59, 999999999, loc)

	results, err := s.transactionRepo.Aggregate(context.Background(), userID, startInLoc, endInLoc)
	if err != nil {
		return nil, errors.New("failed to generate summary")
	}

	categoryMap := s.loadCategoryMap()

	var totalIncome, totalOutcome float64
	incomeByCategory := make(map[uint]*dto.CategorySummary)
	outcomeByCategory := make(map[uint]*dto.CategorySummary)

	for _, result := range results {
		summary := &dto.CategorySummary{
			CategoryID:   result.CategoryID,
			CategoryName: categoryMap[result.CategoryID],
			Total:        result.Total,
			Count:        result.Count,
		}

		if result.Type == "income" {
			totalIncome += result.Total
			incomeByCategory[result.CategoryID] = summary
		} else {
			totalOutcome += result.Total
			outcomeByCategory[result.CategoryID] = summary
		}
	}

	incomeList := make([]dto.CategorySummary, 0, len(incomeByCategory))
	for _, summary := range incomeByCategory {
		if totalIncome > 0 {
			summary.Percentage = (summary.Total / totalIncome) * 100
		}
		incomeList = append(incomeList, *summary)
	}

	outcomeList := make([]dto.CategorySummary, 0, len(outcomeByCategory))
	for _, summary := range outcomeByCategory {
		if totalOutcome > 0 {
			summary.Percentage = (summary.Total / totalOutcome) * 100
		}
		outcomeList = append(outcomeList, *summary)
	}

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

	return nil, nil
}

func (s *FinanceService) categoryToResponse(category *model.Category) *dto.CategoryResponse {
	return &dto.CategoryResponse{
		ID:               category.ID,
		Name:             category.Name,
		Type:             string(category.Type),
		Icon:             category.Icon,
		ApplicableToFreq: string(category.ApplicableToFreq),
		CreatedAt:        category.CreatedAt,
		UpdatedAt:        category.UpdatedAt,
	}
}

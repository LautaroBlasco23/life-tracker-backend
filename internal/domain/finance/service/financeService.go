package service

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"life-tracker-backend/internal/domain/finance/dto"
	"life-tracker-backend/internal/domain/finance/model"
	"life-tracker-backend/internal/infrastructure/monitoring"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"gorm.io/gorm"
)

type FinanceService struct {
	db      *gorm.DB
	mongoDB *mongo.Database
}

func NewFinanceService(db *gorm.DB, mongoDB *mongo.Database) *FinanceService {
	return &FinanceService{
		db:      db,
		mongoDB: mongoDB,
	}
}

func (s *FinanceService) InitializeSystemCategories() error {
	for _, cat := range model.SystemCategories {
		var existing model.Category
		if err := s.db.Where("name = ?", cat.Name).First(&existing).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				if err := s.db.Create(&cat).Error; err != nil {
					return fmt.Errorf("failed to create category %s: %w", cat.Name, err)
				}

				var createdCat model.Category
				if err := s.db.Where("name = ?", cat.Name).First(&createdCat).Error; err != nil {
					continue
				}

				if subcats, exists := model.SystemSubcategories[cat.Name]; exists {
					for _, subName := range subcats {
						subcat := model.Subcategory{
							CategoryID: createdCat.ID,
							Name:       subName,
						}
						if err := s.db.Create(&subcat).Error; err != nil {
							fmt.Printf("Warning: failed to create subcategory %s: %v\n", subName, err)
						}
					}
				}
			}
		}
	}
	return nil
}

func (s *FinanceService) GetCategories(transactionType *string) ([]*dto.CategoryResponse, error) {
	var categories []model.Category

	query := s.db
	if transactionType != nil {
		query = query.Where("type = ?", *transactionType)
	}

	if err := query.Order("name ASC").Find(&categories).Error; err != nil {
		return nil, errors.New("failed to fetch categories")
	}

	var subcategories []model.Subcategory
	if err := s.db.Find(&subcategories).Error; err != nil {
		return nil, errors.New("failed to fetch subcategories")
	}

	subcatMap := make(map[uint][]model.Subcategory)
	for _, sub := range subcategories {
		subcatMap[sub.CategoryID] = append(subcatMap[sub.CategoryID], sub)
	}

	responses := make([]*dto.CategoryResponse, len(categories))
	for i := range categories {
		responses[i] = s.categoryToResponse(&categories[i])
		if subs, exists := subcatMap[categories[i].ID]; exists {
			for _, sub := range subs {
				responses[i].Subcategories = append(responses[i].Subcategories, dto.SubcategoryResponse{
					ID:         sub.ID,
					CategoryID: sub.CategoryID,
					Name:       sub.Name,
					Icon:       sub.Icon,
					CreatedAt:  sub.CreatedAt,
					UpdatedAt:  sub.UpdatedAt,
				})
			}
		}
	}

	return responses, nil
}

func (s *FinanceService) CreateTransaction(userID uint, req *dto.CreateTransactionRequest) (*dto.TransactionResponse, error) {
	var category model.Category
	if err := s.db.Where("id = ?", req.CategoryID).First(&category).Error; err != nil {
		return nil, errors.New("category not found")
	}

	var subcategory model.Subcategory
	if err := s.db.Where("id = ? AND category_id = ?", req.SubcategoryID, req.CategoryID).First(&subcategory).Error; err != nil {
		return nil, errors.New("subcategory not found or doesn't belong to category")
	}

	transactionDate := time.Now()
	if !req.Date.IsZero() {
		transactionDate = req.Date
	}

	transaction := model.Transaction{
		ID:            primitive.NewObjectID(),
		UserID:        userID,
		Type:          model.TransactionType(req.Type),
		Amount:        req.Amount,
		CategoryID:    req.CategoryID,
		SubcategoryID: req.SubcategoryID,
		Description:   req.Description,
		Date:          transactionDate,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	collection := s.mongoDB.Collection("transactions")
	_, err := collection.InsertOne(context.Background(), transaction)
	if err != nil {
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

	return transaction.ToResponse(category.Name, subcategory.Name), nil
}

func (s *FinanceService) GetTransactions(userID uint, transactionType *string, startDate, endDate *time.Time, month, year *int, categoryID *uint, limit int) ([]*dto.TransactionResponse, error) {
	collection := s.mongoDB.Collection("transactions")
	filter := bson.M{"userId": userID}

	if transactionType != nil {
		filter["type"] = *transactionType
	}

	if categoryID != nil {
		filter["categoryId"] = *categoryID
	}

	effectiveStartDate := startDate
	effectiveEndDate := endDate

	if month != nil && year != nil {
		start := time.Date(*year, time.Month(*month), 1, 0, 0, 0, 0, time.UTC)
		end := start.AddDate(0, 1, 0).Add(-time.Second)
		effectiveStartDate = &start
		effectiveEndDate = &end
	} else if month != nil {
		currentYear := time.Now().Year()
		start := time.Date(currentYear, time.Month(*month), 1, 0, 0, 0, 0, time.UTC)
		end := start.AddDate(0, 1, 0).Add(-time.Second)
		effectiveStartDate = &start
		effectiveEndDate = &end
	} else if year != nil {
		start := time.Date(*year, 1, 1, 0, 0, 0, 0, time.UTC)
		end := time.Date(*year, 12, 31, 23, 59, 59, 999999999, time.UTC)
		effectiveStartDate = &start
		effectiveEndDate = &end
	}

	if effectiveStartDate != nil || effectiveEndDate != nil {
		dateFilter := bson.M{}
		if effectiveStartDate != nil {
			dateFilter["$gte"] = *effectiveStartDate
		}
		if effectiveEndDate != nil {
			dateFilter["$lte"] = *effectiveEndDate
		}
		filter["date"] = dateFilter
	}

	opts := options.Find().SetSort(bson.D{{Key: "date", Value: -1}})
	if limit > 0 {
		opts.SetLimit(int64(limit))
	}

	cursor, err := collection.Find(context.Background(), filter, opts)
	if err != nil {
		return nil, errors.New("failed to fetch transactions")
	}
	defer func() {
		if err := cursor.Close(context.Background()); err != nil {
			log.Printf("failed to close cursor: %v", err)
		}
	}()

	var transactions []model.Transaction
	if err := cursor.All(context.Background(), &transactions); err != nil {
		return nil, errors.New("failed to decode transactions")
	}

	categoryMap := make(map[uint]string)
	subcategoryMap := make(map[uint]string)

	var categories []model.Category
	if err := s.db.Find(&categories).Error; err == nil {
		for _, cat := range categories {
			categoryMap[cat.ID] = cat.Name
		}
	}

	var subcategories []model.Subcategory
	if err := s.db.Find(&subcategories).Error; err == nil {
		for _, sub := range subcategories {
			subcategoryMap[sub.ID] = sub.Name
		}
	}

	responses := make([]*dto.TransactionResponse, len(transactions))
	for i := range transactions {
		catName := categoryMap[transactions[i].CategoryID]
		subName := subcategoryMap[transactions[i].SubcategoryID]
		responses[i] = transactions[i].ToResponse(catName, subName)
	}

	return responses, nil
}

func (s *FinanceService) GetTransaction(userID uint, transactionID string) (*dto.TransactionResponse, error) {
	objID, err := primitive.ObjectIDFromHex(transactionID)
	if err != nil {
		return nil, errors.New("invalid transaction ID")
	}

	collection := s.mongoDB.Collection("transactions")
	var transaction model.Transaction

	err = collection.FindOne(context.Background(), bson.M{"_id": objID, "userId": userID}).Decode(&transaction)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, errors.New("transaction not found")
		}
		return nil, errors.New("failed to fetch transaction")
	}

	var category model.Category
	var subcategory model.Subcategory
	s.db.Where("id = ?", transaction.CategoryID).First(&category)
	s.db.Where("id = ?", transaction.SubcategoryID).First(&subcategory)

	return transaction.ToResponse(category.Name, subcategory.Name), nil
}

func (s *FinanceService) UpdateTransaction(userID uint, transactionID string, req *dto.UpdateTransactionRequest) (*dto.TransactionResponse, error) {
	objID, err := primitive.ObjectIDFromHex(transactionID)
	if err != nil {
		return nil, errors.New("invalid transaction ID")
	}

	collection := s.mongoDB.Collection("transactions")
	var transaction model.Transaction

	err = collection.FindOne(context.Background(), bson.M{"_id": objID, "userId": userID}).Decode(&transaction)
	if err != nil {
		return nil, errors.New("transaction not found")
	}

	updates := bson.M{"updatedAt": time.Now()}

	if req.Type != nil {
		updates["type"] = *req.Type
	}
	if req.Amount != nil {
		updates["amount"] = *req.Amount
	}
	if req.CategoryID != nil {
		var category model.Category
		if err = s.db.Where("id = ?", *req.CategoryID).First(&category).Error; err != nil {
			return nil, errors.New("category not found")
		}
		updates["categoryId"] = *req.CategoryID
	}
	if req.SubcategoryID != nil {
		catID := transaction.CategoryID
		if req.CategoryID != nil {
			catID = *req.CategoryID
		}
		var subcategory model.Subcategory
		if err = s.db.Where("id = ? AND category_id = ?", *req.SubcategoryID, catID).First(&subcategory).Error; err != nil {
			return nil, errors.New("subcategory not found or doesn't belong to category")
		}
		updates["subcategoryId"] = *req.SubcategoryID
	}
	if req.Description != nil {
		updates["description"] = *req.Description
	}
	if req.Date != nil {
		updates["date"] = *req.Date
	}

	_, err = collection.UpdateOne(context.Background(), bson.M{"_id": objID}, bson.M{"$set": updates})
	if err != nil {
		return nil, errors.New("failed to update transaction")
	}

	return s.GetTransaction(userID, transactionID)
}

func (s *FinanceService) DeleteTransaction(userID uint, transactionID string) error {
	objID, err := primitive.ObjectIDFromHex(transactionID)
	if err != nil {
		return errors.New("invalid transaction ID")
	}

	collection := s.mongoDB.Collection("transactions")
	result, err := collection.DeleteOne(context.Background(), bson.M{"_id": objID, "userId": userID})
	if err != nil {
		return errors.New("failed to delete transaction")
	}
	if result.DeletedCount == 0 {
		return errors.New("transaction not found")
	}

	monitoring.TransactionsDeleted.Inc()

	return nil
}

func (s *FinanceService) GetFinanceSummary(userID uint, startDate, endDate time.Time) (*dto.FinanceSummaryResponse, error) {
	collection := s.mongoDB.Collection("transactions")

	pipeline := []bson.M{
		{
			"$match": bson.M{
				"userId": userID,
				"date": bson.M{
					"$gte": startDate,
					"$lte": endDate,
				},
			},
		},
		{
			"$group": bson.M{
				"_id": bson.M{
					"type":       "$type",
					"categoryId": "$categoryId",
				},
				"total": bson.M{"$sum": "$amount"},
				"count": bson.M{"$sum": 1},
			},
		},
	}

	cursor, err := collection.Aggregate(context.Background(), pipeline)
	if err != nil {
		return nil, errors.New("failed to generate summary")
	}
	defer func() {
		if err := cursor.Close(context.Background()); err != nil {
			log.Printf("failed to close cursor: %v", err)
		}
	}()

	var categories []model.Category
	if err := s.db.Find(&categories).Error; err != nil {
		return nil, errors.New("failed to fetch categories")
	}

	categoryMap := make(map[uint]string)
	for _, cat := range categories {
		categoryMap[cat.ID] = cat.Name
	}

	var totalIncome, totalOutcome float64
	incomeByCategory := make(map[uint]*dto.CategorySummary)
	outcomeByCategory := make(map[uint]*dto.CategorySummary)

	for cursor.Next(context.Background()) {
		var result struct {
			ID struct {
				Type       string `bson:"type"`
				CategoryID uint   `bson:"categoryId"`
			} `bson:"_id"`
			Total float64 `bson:"total"`
			Count int64   `bson:"count"`
		}
		if err := cursor.Decode(&result); err != nil {
			continue
		}

		catName := categoryMap[result.ID.CategoryID]
		summary := &dto.CategorySummary{
			CategoryID:   result.ID.CategoryID,
			CategoryName: catName,
			Total:        result.Total,
			Count:        result.Count,
		}

		if result.ID.Type == "income" {
			totalIncome += result.Total
			incomeByCategory[result.ID.CategoryID] = summary
		} else {
			totalOutcome += result.Total
			outcomeByCategory[result.ID.CategoryID] = summary
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

func (s *FinanceService) GetMonthlyStats(userID uint, year int) ([]*dto.MonthlyStatsResponse, error) {
	collection := s.mongoDB.Collection("transactions")

	startDate := time.Date(year, 1, 1, 0, 0, 0, 0, time.UTC)
	endDate := time.Date(year, 12, 31, 23, 59, 59, 999999999, time.UTC)

	pipeline := []bson.M{
		{
			"$match": bson.M{
				"userId": userID,
				"date": bson.M{
					"$gte": startDate,
					"$lte": endDate,
				},
			},
		},
		{
			"$group": bson.M{
				"_id": bson.M{
					"month": bson.M{"$month": "$date"},
					"type":  "$type",
				},
				"total": bson.M{"$sum": "$amount"},
				"count": bson.M{"$sum": 1},
			},
		},
	}

	cursor, err := collection.Aggregate(context.Background(), pipeline)
	if err != nil {
		return nil, errors.New("failed to generate monthly stats")
	}
	defer func() {
		if err := cursor.Close(context.Background()); err != nil {
			log.Printf("failed to close cursor: %v", err)
		}
	}()

	monthlyData := make(map[int]struct {
		income  float64
		outcome float64
		count   int64
	})

	for cursor.Next(context.Background()) {
		var result struct {
			ID struct {
				Type  string `bson:"type"`
				Month int    `bson:"month"`
			} `bson:"_id"`
			Total float64 `bson:"total"`
			Count int64   `bson:"count"`
		}
		if err := cursor.Decode(&result); err != nil {
			continue
		}

		data := monthlyData[result.ID.Month]
		if result.ID.Type == "income" {
			data.income = result.Total
		} else {
			data.outcome = result.Total
		}
		data.count += result.Count
		monthlyData[result.ID.Month] = data
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

func (s *FinanceService) categoryToResponse(category *model.Category) *dto.CategoryResponse {
	return &dto.CategoryResponse{
		ID:            category.ID,
		Name:          category.Name,
		Type:          string(category.Type),
		Icon:          category.Icon,
		Subcategories: []dto.SubcategoryResponse{},
		CreatedAt:     category.CreatedAt,
		UpdatedAt:     category.UpdatedAt,
	}
}

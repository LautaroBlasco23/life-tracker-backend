package database

import (
	"fmt"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"life-tracker-backend/internal/config"
	activityModel "life-tracker-backend/internal/domain/activity/model"
	"life-tracker-backend/internal/domain/auth/model"
	financeModel "life-tracker-backend/internal/domain/finance/model"
	noteModel "life-tracker-backend/internal/domain/note/model"
	timeModel "life-tracker-backend/internal/domain/time/model"
	userModel "life-tracker-backend/internal/domain/user/model"
)

func InitializePostgreSQL(cfg *config.Config) (*gorm.DB, error) {
	dsn := fmt.Sprintf(
		"host=%s user=%s password=%s dbname=%s port=%s sslmode=%s",
		cfg.DBHost, cfg.DBUser, cfg.DBPassword, cfg.DBName, cfg.DBPort, cfg.DBSSLMode,
	)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to PostgreSQL: %w", err)
	}

	if err := db.AutoMigrate(
		&userModel.User{},
		&model.Auth{},
		&activityModel.Activity{},
		&financeModel.Category{},
		&timeModel.TimeRecord{},
		&noteModel.Note{},
	); err != nil {
		return nil, fmt.Errorf("failed to migrate PostgreSQL tables: %w", err)
	}

	if err := seedFinanceData(db); err != nil {
		return nil, fmt.Errorf("failed to seed finance data: %w", err)
	}

	return db, nil
}

func seedFinanceData(db *gorm.DB) error {
	var count int64
	if err := db.Model(&financeModel.Category{}).Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return nil
	}

	if err := db.Create(&financeModel.SystemCategories).Error; err != nil {
		return fmt.Errorf("failed to seed categories: %w", err)
	}

	return nil
}

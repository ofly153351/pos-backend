package database

import (
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"pos-backend/internal/config"
)

func Open(cfg config.Config) (*gorm.DB, error) {
	db, err := gorm.Open(postgres.Open(cfg.DatabaseURL()), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Warn), // log slow queries only
	})
	if err != nil {
		return nil, err
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}

	// Connection pool — tune based on expected concurrent users
	// Rule of thumb: NumCPU * 4 for OLTP workloads
	sqlDB.SetMaxOpenConns(25)            // max concurrent DB connections
	sqlDB.SetMaxIdleConns(10)            // idle connections kept alive
	sqlDB.SetConnMaxLifetime(30 * time.Minute) // recycle connections
	sqlDB.SetConnMaxIdleTime(5 * time.Minute)  // drop idle connections sooner

	if err := sqlDB.Ping(); err != nil {
		_ = sqlDB.Close()
		return nil, err
	}

	return db, nil
}

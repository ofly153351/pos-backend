package database

import (
	"fmt"
	"log"
	"os"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"pos-backend/internal/config"
)

const (
	dbConnectMaxAttempts = 5
	dbConnectRetryDelay  = 2 * time.Second
)

func Open(cfg config.Config) (*gorm.DB, error) {
	var lastErr error

	for attempt := 1; attempt <= dbConnectMaxAttempts; attempt++ {
		gormLogger := logger.New(
			log.New(os.Stdout, "\r\n", log.LstdFlags),
			logger.Config{
				SlowThreshold:             200 * time.Millisecond,
				LogLevel:                  logger.Warn,
				IgnoreRecordNotFoundError: true,
				Colorful:                  false,
			},
		)
		db, err := gorm.Open(postgres.Open(cfg.DatabaseURL()), &gorm.Config{
			Logger: gormLogger,
		})
		if err != nil {
			lastErr = err
		} else {
			sqlDB, err := db.DB()
			if err != nil {
				lastErr = err
			} else {
				// Connection pool tuning for the local OLTP workload.
				sqlDB.SetMaxOpenConns(25)
				sqlDB.SetMaxIdleConns(10)
				sqlDB.SetConnMaxLifetime(30 * time.Minute)
				sqlDB.SetConnMaxIdleTime(5 * time.Minute)

				if err := sqlDB.Ping(); err != nil {
					lastErr = err
					_ = sqlDB.Close()
				} else {
					return db, nil
				}
			}
		}

		if attempt < dbConnectMaxAttempts {
			time.Sleep(dbConnectRetryDelay)
		}
	}

	return nil, fmt.Errorf(
		"failed to connect to PostgreSQL at %s:%s/%s after %d attempts: %w. Start the database service and verify POSTGRES_HOST/POSTGRES_PORT in .env",
		cfg.DBHost,
		cfg.DBPort,
		cfg.DBName,
		dbConnectMaxAttempts,
		lastErr,
	)
}

package vat

import (
	"gorm.io/gorm"
)

type Repository interface {
}

type PostgresRepository struct {
	db *gorm.DB
}

func NewPostgresRepository(db *gorm.DB) PostgresRepository {
	return PostgresRepository{db: db}
}

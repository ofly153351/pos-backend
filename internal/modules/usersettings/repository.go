package usersettings

import (
	"context"
	"errors"

	"gorm.io/gorm"
)

type Repository interface {
	GetCardSettings(ctx context.Context, userID string) (CardSettings, bool, error)
	SaveCardSettings(ctx context.Context, userID string, settings CardSettings) error
}

type PostgresRepository struct {
	db *gorm.DB
}

func NewPostgresRepository(db *gorm.DB) PostgresRepository {
	return PostgresRepository{db: db}
}

// userCardRow maps the JSONB column through the CardSettings Valuer/Scanner.
type userCardRow struct {
	CardSettings CardSettings `gorm:"column:card_settings;type:jsonb"`
}

// GetCardSettings returns the stored settings. The bool is false when the user
// has never saved settings (empty JSON '{}' → zero-value struct) so the caller
// can apply defaults.
func (r PostgresRepository) GetCardSettings(ctx context.Context, userID string) (CardSettings, bool, error) {
	var row userCardRow
	err := r.db.WithContext(ctx).
		Table("users").
		Select("card_settings").
		Where("id = ?", userID).
		Take(&row).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return CardSettings{}, false, nil
		}
		return CardSettings{}, false, err
	}
	// A saved record always has Size set (normalize guarantees it); empty → unset.
	if row.CardSettings.Size == "" {
		return CardSettings{}, false, nil
	}
	return row.CardSettings, true, nil
}

func (r PostgresRepository) SaveCardSettings(ctx context.Context, userID string, settings CardSettings) error {
	// settings.Value() (driver.Valuer) marshals to JSON for the JSONB column.
	return r.db.WithContext(ctx).
		Table("users").
		Where("id = ?", userID).
		Update("card_settings", settings).Error
}

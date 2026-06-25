package receipt_settings

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"
)

type Repository interface {
	GetByStoreID(ctx context.Context, storeID string) (ReceiptSettings, error)
	Insert(ctx context.Context, settings ReceiptSettings) (ReceiptSettings, error)
	Upsert(ctx context.Context, settings ReceiptSettings) (ReceiptSettings, error)
}

type PostgresRepository struct {
	db *gorm.DB
}

func NewPostgresRepository(db *gorm.DB) PostgresRepository {
	return PostgresRepository{db: db}
}

func (r PostgresRepository) GetByStoreID(ctx context.Context, storeID string) (ReceiptSettings, error) {
	var s ReceiptSettings
	err := r.db.WithContext(ctx).
		Where("store_id = ?", storeID).
		Take(&s).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ReceiptSettings{}, ErrNotFound
		}
		return ReceiptSettings{}, err
	}
	return s, nil
}

func (r PostgresRepository) Insert(ctx context.Context, settings ReceiptSettings) (ReceiptSettings, error) {
	if err := r.db.WithContext(ctx).Create(&settings).Error; err != nil {
		return ReceiptSettings{}, err
	}
	return settings, nil
}

func (r PostgresRepository) Upsert(ctx context.Context, settings ReceiptSettings) (ReceiptSettings, error) {
	settings.UpdatedAt = time.Now().UTC()

	result := r.db.WithContext(ctx).
		Table("store_receipt_settings").
		Where("store_id = ?", settings.StoreID).
		Updates(map[string]any{
			"template_key":          settings.TemplateKey,
			"paper_size":            settings.PaperSize,
			"paper_length":          settings.PaperLength,
			"tax_mode":              settings.TaxMode,
			"vat_rate":              settings.VatRate,
			"tax_label":             settings.TaxLabel,
			"show_logo":             settings.ShowLogo,
			"logo_position":         settings.LogoPosition,
			"show_store_name":       settings.ShowStoreName,
			"show_address":          settings.ShowAddress,
			"show_phone":            settings.ShowPhone,
			"show_tax_id":           settings.ShowTaxId,
			"footer_text":           settings.FooterText,
			"payment_channels":      settings.PaymentChannels,
			"printer_type":          settings.PrinterType,
			"printer_name":          settings.PrinterName,
			"auto_print":            settings.AutoPrint,
			"copies":                settings.Copies,
			"show_qr":               settings.ShowQr,
			"qr_size":               settings.QrSize,
			"show_customer_display": settings.ShowCustomerDisplay,
			"show_product_images":   settings.ShowProductImages,
			"date_format":           settings.DateFormat,
			"time_format":           settings.TimeFormat,
			"currency_position":     settings.CurrencyPosition,
			"round_amount":          settings.RoundAmount,
			"updated_at":            settings.UpdatedAt,
		})
	if result.Error != nil {
		return ReceiptSettings{}, result.Error
	}

	if result.RowsAffected == 0 {
		// No existing row — insert defaults
		settings.ID = newID()
		settings.CreatedAt = settings.UpdatedAt
		if err := r.db.WithContext(ctx).Create(&settings).Error; err != nil {
			return ReceiptSettings{}, err
		}
	}

	return r.GetByStoreID(ctx, settings.StoreID)
}

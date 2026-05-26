package sale

import (
	"context"
	"errors"

	"gorm.io/gorm"
)

// dbSettingsRepo adapts the store_receipt_settings table to SettingsRepository
// without importing the receipt_settings module (avoids circular deps).
type dbSettingsRepo struct {
	db *gorm.DB
}

func NewDBSettingsRepo(db *gorm.DB) SettingsRepository {
	return &dbSettingsRepo{db: db}
}

type rawReceiptSettings struct {
	ShowLogo      bool    `gorm:"column:show_logo"`
	LogoPosition  string  `gorm:"column:logo_position"`
	ShowStoreName bool    `gorm:"column:show_store_name"`
	ShowAddress   bool    `gorm:"column:show_address"`
	ShowPhone     bool    `gorm:"column:show_phone"`
	ShowTaxId     bool    `gorm:"column:show_tax_id"`
	TaxMode       string  `gorm:"column:tax_mode"`
	VatRate       float64 `gorm:"column:vat_rate"`
	TaxLabel      string  `gorm:"column:tax_label"`
	FooterText    string  `gorm:"column:footer_text"`
	ShowQr        bool    `gorm:"column:show_qr"`
	QrSize        string  `gorm:"column:qr_size"`
	PaperSize     string  `gorm:"column:paper_size"`
}

func (r *dbSettingsRepo) GetByStoreID(ctx context.Context, storeID string) (ReceiptSettingsView, error) {
	var row rawReceiptSettings
	err := r.db.WithContext(ctx).
		Table("store_receipt_settings").
		Select("show_logo, logo_position, show_store_name, show_address, show_phone, show_tax_id, tax_mode, vat_rate, tax_label, footer_text, show_qr, qr_size, paper_size").
		Where("store_id = ?", storeID).
		Take(&row).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ReceiptSettingsView{}, errors.New("not found")
		}
		return ReceiptSettingsView{}, err
	}
	return ReceiptSettingsView{
		ShowLogo:      row.ShowLogo,
		LogoPosition:  row.LogoPosition,
		ShowStoreName: row.ShowStoreName,
		ShowAddress:   row.ShowAddress,
		ShowPhone:     row.ShowPhone,
		ShowTaxId:     row.ShowTaxId,
		TaxMode:       row.TaxMode,
		VatRate:       row.VatRate,
		TaxLabel:      row.TaxLabel,
		FooterText:    row.FooterText,
		ShowQr:        row.ShowQr,
		QrSize:        row.QrSize,
		PaperSize:     row.PaperSize,
	}, nil
}

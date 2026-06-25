package receipt_settings

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"
)

// PaymentChannelSetting stores which payment channel is enabled.
type PaymentChannelSetting struct {
	Key     string `json:"key"`
	Enabled bool   `json:"enabled"`
}

// PaymentChannels is a JSONB-serialisable slice.
type PaymentChannels []PaymentChannelSetting

func (p PaymentChannels) Value() (driver.Value, error) {
	return json.Marshal(p)
}

func (p *PaymentChannels) Scan(value any) error {
	b, ok := value.([]byte)
	if !ok {
		return fmt.Errorf("receipt_settings: cannot scan %T into PaymentChannels", value)
	}
	return json.Unmarshal(b, p)
}

// ReceiptSettings maps to store_receipt_settings table.
type ReceiptSettings struct {
	ID                  string          `json:"id"                   gorm:"column:id;primaryKey"`
	StoreID             string          `json:"store_id"             gorm:"column:store_id"`
	TemplateKey         string          `json:"template_key"         gorm:"column:template_key"`
	PaperSize           string          `json:"paper_size"           gorm:"column:paper_size"`
	PaperLength         string          `json:"paper_length"         gorm:"column:paper_length"`
	TaxMode             string          `json:"tax_mode"             gorm:"column:tax_mode"`
	VatRate             float64         `json:"vat_rate"             gorm:"column:vat_rate"`
	TaxLabel            string          `json:"tax_label"            gorm:"column:tax_label"`
	ShowLogo            bool            `json:"show_logo"            gorm:"column:show_logo"`
	LogoPosition        string          `json:"logo_position"        gorm:"column:logo_position"`
	ShowStoreName       bool            `json:"show_store_name"      gorm:"column:show_store_name"`
	ShowAddress         bool            `json:"show_address"         gorm:"column:show_address"`
	ShowPhone           bool            `json:"show_phone"           gorm:"column:show_phone"`
	ShowTaxId           bool            `json:"show_tax_id"          gorm:"column:show_tax_id"`
	FooterText          string          `json:"footer_text"          gorm:"column:footer_text"`
	PaymentChannels     PaymentChannels `json:"payment_channels"     gorm:"column:payment_channels;type:jsonb"`
	PrinterType         string          `json:"printer_type"         gorm:"column:printer_type"`
	PrinterName         string          `json:"printer_name"         gorm:"column:printer_name"`
	AutoPrint           bool            `json:"auto_print"           gorm:"column:auto_print"`
	Copies              int             `json:"copies"               gorm:"column:copies"`
	ShowQr              bool            `json:"show_qr"              gorm:"column:show_qr"`
	QrSize              string          `json:"qr_size"              gorm:"column:qr_size"`
	ShowCustomerDisplay bool            `json:"show_customer_display" gorm:"column:show_customer_display"`
	ShowProductImages   bool            `json:"show_product_images"  gorm:"column:show_product_images"`
	DateFormat          string          `json:"date_format"          gorm:"column:date_format"`
	TimeFormat          string          `json:"time_format"          gorm:"column:time_format"`
	CurrencyPosition    string          `json:"currency_position"    gorm:"column:currency_position"`
	RoundAmount         bool            `json:"round_amount"         gorm:"column:round_amount"`
	CreatedAt           time.Time       `json:"created_at"           gorm:"column:created_at"`
	UpdatedAt           time.Time       `json:"updated_at"           gorm:"column:updated_at"`
}

func (ReceiptSettings) TableName() string { return "store_receipt_settings" }

// UpdateReceiptSettingsRequest is the PUT body.
type UpdateReceiptSettingsRequest struct {
	TemplateKey         *string                 `json:"template_key"`
	PaperSize           *string                 `json:"paper_size"`
	PaperLength         *string                 `json:"paper_length"`
	TaxMode             *string                 `json:"tax_mode"`
	VatRate             *float64                `json:"vat_rate"`
	TaxLabel            *string                 `json:"tax_label"`
	ShowLogo            *bool                   `json:"show_logo"`
	LogoPosition        *string                 `json:"logo_position"`
	ShowStoreName       *bool                   `json:"show_store_name"`
	ShowAddress         *bool                   `json:"show_address"`
	ShowPhone           *bool                   `json:"show_phone"`
	ShowTaxId           *bool                   `json:"show_tax_id"`
	FooterText          *string                 `json:"footer_text"`
	PaymentChannels     []PaymentChannelSetting `json:"payment_channels"`
	PrinterType         *string                 `json:"printer_type"`
	PrinterName         *string                 `json:"printer_name"`
	AutoPrint           *bool                   `json:"auto_print"`
	Copies              *int                    `json:"copies"`
	ShowQr              *bool                   `json:"show_qr"`
	QrSize              *string                 `json:"qr_size"`
	ShowCustomerDisplay *bool                   `json:"show_customer_display"`
	ShowProductImages   *bool                   `json:"show_product_images"`
	DateFormat          *string                 `json:"date_format"`
	TimeFormat          *string                 `json:"time_format"`
	CurrencyPosition    *string                 `json:"currency_position"`
	RoundAmount         *bool                   `json:"round_amount"`
}

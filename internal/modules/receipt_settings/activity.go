package receipt_settings

// activitySnapshot captures the editable receipt-settings fields, keyed by the API
// field names the update endpoint accepts, for the Activity Center {before, after}
// diff and restore flow.
func activitySnapshot(s ReceiptSettings) map[string]any {
	return map[string]any{
		"template_key":          s.TemplateKey,
		"paper_size":            s.PaperSize,
		"paper_length":          s.PaperLength,
		"tax_mode":              s.TaxMode,
		"vat_rate":              s.VatRate,
		"tax_label":             s.TaxLabel,
		"show_logo":             s.ShowLogo,
		"logo_position":         s.LogoPosition,
		"show_store_name":       s.ShowStoreName,
		"show_address":          s.ShowAddress,
		"show_phone":            s.ShowPhone,
		"show_tax_id":           s.ShowTaxId,
		"footer_text":           s.FooterText,
		"payment_channels":      s.PaymentChannels,
		"printer_type":          s.PrinterType,
		"printer_name":          s.PrinterName,
		"auto_print":            s.AutoPrint,
		"copies":                s.Copies,
		"show_qr":               s.ShowQr,
		"qr_size":               s.QrSize,
		"show_customer_display": s.ShowCustomerDisplay,
		"show_product_images":   s.ShowProductImages,
		"date_format":           s.DateFormat,
		"time_format":           s.TimeFormat,
		"currency_position":     s.CurrencyPosition,
		"round_amount":          s.RoundAmount,
	}
}

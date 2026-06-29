package store

// activitySnapshot captures the editable store-profile fields, keyed by the API
// field names the update endpoint accepts, for the Activity Center {before, after}
// diff and restore flow. The logo is referenced by URL (binary is not re-uploaded
// on restore).
func activitySnapshot(s Store) map[string]any {
	return map[string]any{
		"name":          s.Name,
		"phone":         s.Phone,
		"fax":           s.Fax,
		"email":         s.Email,
		"website":       s.Website,
		"address":       s.Address,
		"promptpay_id":  s.PromptPayID,
		"tax_id":        s.TaxID,
		"currency_code": s.CurrencyCode,
		"logo_url":      s.LogoURL,
	}
}

package product

// activitySnapshot captures the editable business fields of a product, keyed by the
// API field names the update endpoint accepts. The activity-log middleware diffs the
// before/after snapshots into the audit row's {before, after} change set, which also
// drives the Activity Center "restore previous value" flow.
func activitySnapshot(p Product) map[string]any {
	m := map[string]any{
		"name":             p.Name,
		"brand_id":         p.BrandID,
		"sku":              p.SKU,
		"barcode":          p.Barcode,
		"product_code":     p.ProductCode,
		"description":      p.Description,
		"storage_location": p.StorageLocation,
		"product_type_id":  p.ProductTypeID,
		"product_unit_id":  p.ProductUnitID,
		"base_price":       p.BasePrice,
		"cost_price":       p.CostPrice,
		"min_stock":        p.MinStock,
		"is_active":        p.IsActive,
		"image_url":        p.ImageURL,
	}
	m["default_location_id"] = derefString(p.DefaultLocationID)
	m["max_stock"] = derefInt(p.MaxStock)
	m["special_price"] = derefFloat(p.SpecialPrice)
	return m
}

func derefString(p *string) any {
	if p == nil {
		return nil
	}
	return *p
}

func derefInt(p *int) any {
	if p == nil {
		return nil
	}
	return *p
}

func derefFloat(p *float64) any {
	if p == nil {
		return nil
	}
	return *p
}

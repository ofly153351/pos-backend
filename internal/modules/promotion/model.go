package promotion

import "encoding/json"

// PromotionRow mirrors a few campaign fields into columns for listing/filtering;
// the full client Campaign object is stored verbatim in `data` (TEXT) so the rich,
// type-specific shape round-trips without modelling ~45 fields.
type PromotionRow struct {
	ID        string `gorm:"column:id"`
	StoreID   string `gorm:"column:store_id"`
	Name      string `gorm:"column:name"`
	Type      string `gorm:"column:type"`
	Status    string `gorm:"column:status"`
	Code      string `gorm:"column:code"`
	Data      string `gorm:"column:data"`
	CreatedBy string `gorm:"column:created_by"`
	CreatedAt string `gorm:"column:created_at"`
	UpdatedAt string `gorm:"column:updated_at"`
}

type promotionMeta struct {
	Name   string `json:"name"`
	Type   string `json:"type"`
	Status string `json:"status"`
	Code   string `json:"code"`
}

func extractMeta(data []byte) promotionMeta {
	var m promotionMeta
	_ = json.Unmarshal(data, &m)
	if m.Status == "" {
		m.Status = "draft"
	}
	return m
}

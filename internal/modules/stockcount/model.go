package stockcount

// CountItem mirrors the worksheet row the cashier/counter app edits. JSON keys are
// camelCase to round-trip the frontend CountItem shape verbatim; DB columns are
// snake_case. Client timestamps are stored/returned as ISO strings (TEXT columns)
// so they round-trip without timezone reinterpretation.
type CountItem struct {
	SessionID           string  `json:"-" gorm:"column:session_id"`
	ProductID           string  `json:"productId" gorm:"column:product_id"`
	Name                string  `json:"name" gorm:"column:name"`
	SKU                 string  `json:"sku" gorm:"column:sku"`
	Barcode             string  `json:"barcode" gorm:"column:barcode"`
	SystemQty           int     `json:"systemQty" gorm:"column:system_qty"`
	MinStock            int     `json:"minStock" gorm:"column:min_stock"`
	Location            string  `json:"location" gorm:"column:location"`
	Counted             *int    `json:"counted" gorm:"column:counted"`
	Note                string  `json:"note" gorm:"column:note"`
	Skipped             bool    `json:"skipped" gorm:"column:skipped"`
	VarianceReason      string  `json:"varianceReason" gorm:"column:variance_reason"`
	VarianceReasonOther string  `json:"varianceReasonOther" gorm:"column:variance_reason_other"`
	CountUser           string  `json:"countUser" gorm:"column:count_user"`
	CountedAt           *string `json:"countedAt" gorm:"column:counted_at"`
	CostBasis           float64 `json:"costBasis" gorm:"column:cost_basis"`
	Adjusted            bool    `json:"adjusted" gorm:"column:adjusted"`
	AdjustedAt          *string `json:"adjustedAt" gorm:"column:adjusted_at"`
	AdjustedBy          string  `json:"adjustedBy" gorm:"column:adjusted_by"`
}

// CountSession is one physical-count worksheet (draft → counting → review →
// completed/cancelled). Items are loaded/saved together with the header.
type CountSession struct {
	ID            string      `json:"id" gorm:"column:id"`
	StoreID       string      `json:"-" gorm:"column:store_id"`
	Name          string      `json:"name" gorm:"column:name"`
	WarehouseName *string     `json:"warehouseName" gorm:"column:warehouse_name"`
	Zone          *string     `json:"zone" gorm:"column:zone"`
	CategoryID    *string     `json:"categoryId" gorm:"column:category_id"`
	CategoryName  *string     `json:"categoryName" gorm:"column:category_name"`
	Staff         string      `json:"staff" gorm:"column:staff"`
	Note          string      `json:"note" gorm:"column:note"`
	Status        string      `json:"status" gorm:"column:status"`
	CountType     string      `json:"countType" gorm:"column:count_type"`
	CycleRule     string      `json:"cycleRule" gorm:"column:cycle_rule"`
	BlindCount    bool        `json:"blindCount" gorm:"column:blind_count"`
	CreatedBy     string      `json:"createdBy" gorm:"column:created_by"`
	CompletedBy   string      `json:"completedBy" gorm:"column:completed_by"`
	CompletedAt   *string     `json:"completedAt" gorm:"column:completed_at"`
	CreatedAt     string      `json:"createdAt" gorm:"column:created_at"`
	UpdatedAt     string      `json:"updatedAt" gorm:"column:updated_at"`
	Items         []CountItem `json:"items" gorm:"-"`
}

// ApplyItem is one correction to commit: set the product's on-hand to CountedQty.
type ApplyItem struct {
	ProductID  string `json:"productId"`
	CountedQty int    `json:"countedQty"`
	Note       string `json:"note"`
}

// ApplyRequest is the batch of corrections committed when a count is approved.
// The whole batch is applied in ONE transaction — all-or-nothing (fail-all).
type ApplyRequest struct {
	Items []ApplyItem `json:"items"`
}

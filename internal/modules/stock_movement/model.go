package stock_movement

import (
	"errors"
	"time"
)

// Movement type constants
const (
	MovementTypeIn              = "IN"
	MovementTypeOut             = "OUT"
	MovementTypeTransfer        = "TRANSFER"
	MovementTypeSale            = "SALE"
	MovementTypeAdjust          = "ADJUST"
	MovementTypeReturn          = "RETURN"
	MovementTypeCountCorrection = "COUNT_CORRECTION"
)

type StockMovement struct {
	ID                    string    `json:"id" gorm:"column:id;primaryKey"`
	StoreID               string    `json:"store_id" gorm:"column:store_id"`
	ProductID             string    `json:"product_id" gorm:"column:product_id"`
	LocationID            *string   `json:"location_id,omitempty" gorm:"column:location_id"`
	DestinationLocationID *string   `json:"destination_location_id,omitempty" gorm:"column:destination_location_id"`
	QuantityChange        int       `json:"quantity_change" gorm:"column:quantity_change"`
	Type                  string    `json:"type" gorm:"column:type"`
	ReferenceID           *string   `json:"reference_id,omitempty" gorm:"column:reference_id"`
	Note                  string    `json:"note" gorm:"column:note"`
	CreatedBy             string    `json:"created_by" gorm:"column:created_by"`
	CreatedAt             time.Time `json:"created_at" gorm:"column:created_at"`
	UpdatedAt             time.Time `json:"updated_at" gorm:"column:updated_at"`

	// Relations (read-only)
	ProductName   string `json:"product_name,omitempty"`
	ProductSKU    string `json:"product_sku,omitempty"`
	CreatedByName string `json:"created_by_name,omitempty"`
	LocationName  string `json:"location_name,omitempty"`
}

func (StockMovement) TableName() string {
	return "stock_movements"
}

type AddStockItemRequest struct {
	ProductID  string `json:"product_id"`
	LocationID string `json:"location_id"`
	Quantity   int    `json:"quantity"`
	Note       string `json:"note"`
}

type AddStockRequest struct {
	Items []AddStockItemRequest `json:"items"`
}

type RemoveStockRequest struct {
	ProductID  string `json:"product_id"`
	LocationID string `json:"location_id"`
	Quantity   int    `json:"quantity"`
	Note       string `json:"note"`
}

type TransferStockRequest struct {
	ProductID        string `json:"product_id"`
	SourceLocationID string `json:"source_location_id"`
	DestLocationID   string `json:"dest_location_id"`
	Quantity         int    `json:"quantity"`
	Note             string `json:"note"`
}

type AdjustStockRequest struct {
	ProductID    string `json:"product_id"`
	LocationID   string `json:"location_id"`
	PhysicalQty  int    `json:"physical_quantity"`
	ReferenceID  string `json:"reference_id"`
	MovementType string `json:"movement_type"`
	Note         string `json:"note"`
}

type ListMovementsQuery struct {
	ProductID string `json:"product_id"`
	Page      int    `json:"page"`
	Limit     int    `json:"limit"`
}

type MovementResponse struct {
	Items    []StockMovement `json:"items"`
	Total    int64           `json:"total"`
	Page     int             `json:"page"`
	Limit    int             `json:"limit"`
	LastPage int             `json:"last_page"`
}

type AdditionResult struct {
	Movements []StockMovement `json:"movements"`
}

var (
	ErrStockForbidden    = errors.New("user cannot operate this store")
	ErrStockNoItems      = errors.New("at least one item is required")
	ErrStockBadQty       = errors.New("quantity must be greater than zero")
	ErrProductNotFound   = errors.New("product not found")
	ErrInsufficientStock = errors.New("insufficient stock quantity")
	ErrLocationMismatch  = errors.New("source and destination locations must be different")
)

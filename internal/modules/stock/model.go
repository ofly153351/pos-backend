package stock

import "time"

type Stock struct {
	ID         string    `json:"id" gorm:"column:id;primaryKey"`
	StoreID    string    `json:"store_id" gorm:"column:store_id"`
	ProductID  string    `json:"product_id" gorm:"column:product_id"`
	LocationID string    `json:"location_id" gorm:"column:location_id"`
	Quantity   int       `json:"quantity" gorm:"column:quantity"`
	CreatedAt  time.Time `json:"created_at" gorm:"column:created_at"`
	UpdatedAt  time.Time `json:"updated_at" gorm:"column:updated_at"`
}

func (Stock) TableName() string { return "stocks" }

type StockByLocation struct {
	LocationID    string `json:"location_id"`
	LocationName  string `json:"location_name"`
	WarehouseName string `json:"warehouse_name"`
	Quantity      int    `json:"quantity"`
}

type StockSummary struct {
	ProductID     string           `json:"product_id"`
	TotalQuantity int              `json:"total_quantity"`
	Locations     []StockByLocation `json:"locations"`
}

type LowStockItem struct {
	ProductID   string `json:"product_id"`
	ProductName string `json:"product_name"`
	SKU         string `json:"sku,omitempty"`
	Quantity    int    `json:"quantity"`
	MinStock    int    `json:"min_stock"`
}

type LowStockResult struct {
	Items []LowStockItem `json:"items"`
}

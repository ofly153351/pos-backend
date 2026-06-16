package warehouse

import (
	"errors"
	"sort"
	"strings"
)

// Phase 0 — Warehouse-scoped product inventory (read-only).
//
// The redesigned Warehouse page needs พร้อมขาย / พื้นที่จัดเก็บ / รวมในคลัง scoped to a
// SINGLE selected warehouse. The product_view aggregates (ready_stock / storage_stock)
// are STORE-WIDE (every location in the store) and the legacy total_stock / warehouse_stock
// fields are mislabelled — none of them is safe here. So this feature derives the split
// directly from stocks JOIN locations, scoped by locations.warehouse_id, splitting on
// locations.is_sale_point. The SQL does the warehouse-scoped grouped aggregation
// (ListWarehouseStockRows); this file holds the pure, fully-testable derivation
// (filtering, status, summary, sort, pagination) applied on top of those rows.

const (
	// Stock status (recommended business vocabulary).
	StockStatusAvailable = "available"
	StockStatusLow       = "low_stock"
	StockStatusOut       = "out_of_stock"

	// Location-type filter values.
	LocationTypeAll       = "all"
	LocationTypeSalePoint = "sale_point"
	LocationTypeStorage   = "storage"

	// Sort keys.
	SortName      = "name"
	SortNameDesc  = "name_desc"
	SortTotalAsc  = "total_asc"
	SortTotalDesc = "total_desc"

	defaultInventoryPageSize = 20
	maxInventoryPageSize     = 100
)

// WarehouseStockRow is the raw per-product row returned by the grouped, warehouse-scoped
// SQL query. ready_stock / storage_stock are already split (SUM split on is_sale_point)
// and warehouse-scoped (WHERE locations.warehouse_id = ?); ALL locations (active AND
// inactive) are included in these current-stock totals.
type WarehouseStockRow struct {
	ProductID    string  `gorm:"column:product_id"`
	ProductName  string  `gorm:"column:product_name"`
	SKU          string  `gorm:"column:sku"`
	Barcode      string  `gorm:"column:barcode"`
	CategoryID   string  `gorm:"column:category_id"`
	CategoryName string  `gorm:"column:category_name"`
	Unit         string  `gorm:"column:unit"`
	CostPrice    float64 `gorm:"column:cost_price"`
	SellingPrice float64 `gorm:"column:selling_price"`
	MinStock     int     `gorm:"column:min_stock"`
	ReadyStock   int     `gorm:"column:ready_stock"`
	StorageStock int     `gorm:"column:storage_stock"`
}

// WarehouseInventoryQuery holds the validated, normalized query parameters.
type WarehouseInventoryQuery struct {
	Search       string
	CategoryID   string
	StockStatus  string // "" | available | low_stock | out_of_stock
	LocationType string // "" | all | sale_point | storage
	Sort         string // "" | name | name_desc | total_asc | total_desc
	Page         int
	PageSize     int
}

// WarehouseInventoryProduct is one product's warehouse-scoped inventory line.
type WarehouseInventoryProduct struct {
	ProductID    string  `json:"product_id"`
	ProductName  string  `json:"product_name"`
	SKU          string  `json:"sku"`
	Barcode      string  `json:"barcode"`
	CategoryID   string  `json:"category_id"`
	CategoryName string  `json:"category_name"`
	Unit         string  `json:"unit"`
	CostPrice    float64 `json:"cost_price"`
	SellingPrice float64 `json:"selling_price"`
	MinStock     int     `json:"min_stock"`
	ReadyStock   int     `json:"ready_stock"`   // พร้อมขาย — sale-point locations
	StorageStock int     `json:"storage_stock"` // พื้นที่จัดเก็บ — non-sale-point locations
	TotalStock   int     `json:"total_stock"`   // รวมในคลัง — ready + storage
	Status       string  `json:"status"`
}

// WarehouseInventorySummary aggregates the FILTERED result set (all matching products,
// before pagination).
type WarehouseInventorySummary struct {
	ProductCount    int     `json:"product_count"`
	ReadyStock      int     `json:"ready_stock"`
	StorageStock    int     `json:"storage_stock"`
	TotalStock      int     `json:"total_stock"`
	InventoryValue  float64 `json:"inventory_value"`
	LowStockCount   int     `json:"low_stock_count"`
	OutOfStockCount int     `json:"out_of_stock_count"`
}

// WarehouseInventoryPagination mirrors the request paging plus the filtered total.
type WarehouseInventoryPagination struct {
	Page     int `json:"page"`
	PageSize int `json:"page_size"`
	Total    int `json:"total"`
}

// WarehouseRef is the lightweight warehouse identity echoed in the response.
type WarehouseRef struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Code string `json:"code,omitempty"`
}

// WarehouseInventoryResponse is the endpoint payload.
type WarehouseInventoryResponse struct {
	Warehouse  WarehouseRef                 `json:"warehouse"`
	Summary    WarehouseInventorySummary    `json:"summary"`
	Items      []WarehouseInventoryProduct  `json:"items"`
	Pagination WarehouseInventoryPagination `json:"pagination"`
}

// deriveStockStatus classifies a product using the existing min_stock rule (mirrors the
// warehouse dashboard: low when min_stock > 0 AND total <= min_stock; out when total == 0).
func deriveStockStatus(total, minStock int) string {
	if total <= 0 {
		return StockStatusOut
	}
	if minStock > 0 && total <= minStock {
		return StockStatusLow
	}
	return StockStatusAvailable
}

// validateInventoryQuery enforces the accepted query-parameter values. Returns a non-nil
// error (mapped to HTTP 400) for any invalid value.
func validateInventoryQuery(q WarehouseInventoryQuery) error {
	if q.Page < 1 {
		return errors.New("invalid query: page must be >= 1")
	}
	if q.PageSize < 1 || q.PageSize > maxInventoryPageSize {
		return errors.New("invalid query: page_size must be between 1 and 100")
	}
	switch q.StockStatus {
	case "", StockStatusAvailable, StockStatusLow, StockStatusOut:
	default:
		return errors.New("invalid query: stock_status must be available, low_stock, or out_of_stock")
	}
	switch q.LocationType {
	case "", LocationTypeAll, LocationTypeSalePoint, LocationTypeStorage:
	default:
		return errors.New("invalid query: location_type must be all, sale_point, or storage")
	}
	switch q.Sort {
	case "", SortName, SortNameDesc, SortTotalAsc, SortTotalDesc:
	default:
		return errors.New("invalid query: sort must be name, name_desc, total_asc, or total_desc")
	}
	return nil
}

// buildWarehouseInventory turns warehouse-scoped per-product rows into the response pieces:
// per-product total = ready + storage, status, filtering, summary over the filtered set,
// sort, and pagination. Pure — no I/O — so it is exhaustively unit-tested.
func buildWarehouseInventory(rows []WarehouseStockRow, q WarehouseInventoryQuery) (WarehouseInventorySummary, []WarehouseInventoryProduct, int) {
	search := strings.ToLower(strings.TrimSpace(q.Search))
	filtered := make([]WarehouseInventoryProduct, 0, len(rows))
	var summary WarehouseInventorySummary

	for _, row := range rows {
		total := row.ReadyStock + row.StorageStock
		item := WarehouseInventoryProduct{
			ProductID:    row.ProductID,
			ProductName:  row.ProductName,
			SKU:          row.SKU,
			Barcode:      row.Barcode,
			CategoryID:   row.CategoryID,
			CategoryName: row.CategoryName,
			Unit:         row.Unit,
			CostPrice:    row.CostPrice,
			SellingPrice: row.SellingPrice,
			MinStock:     row.MinStock,
			ReadyStock:   row.ReadyStock,
			StorageStock: row.StorageStock,
			TotalStock:   total,
			Status:       deriveStockStatus(total, row.MinStock),
		}

		if q.CategoryID != "" && item.CategoryID != q.CategoryID {
			continue
		}
		if search != "" {
			hay := strings.ToLower(item.ProductName + " " + item.SKU + " " + item.Barcode)
			if !strings.Contains(hay, search) {
				continue
			}
		}
		switch q.StockStatus {
		case StockStatusAvailable:
			if item.Status != StockStatusAvailable {
				continue
			}
		case StockStatusLow:
			if item.Status != StockStatusLow {
				continue
			}
		case StockStatusOut:
			if item.Status != StockStatusOut {
				continue
			}
		}
		switch q.LocationType {
		case LocationTypeSalePoint:
			if item.ReadyStock <= 0 {
				continue
			}
		case LocationTypeStorage:
			if item.StorageStock <= 0 {
				continue
			}
		}

		summary.ProductCount++
		summary.ReadyStock += item.ReadyStock
		summary.StorageStock += item.StorageStock
		summary.TotalStock += item.TotalStock
		summary.InventoryValue += item.CostPrice * float64(item.TotalStock)
		if item.Status == StockStatusLow {
			summary.LowStockCount++
		}
		if item.Status == StockStatusOut {
			summary.OutOfStockCount++
		}
		filtered = append(filtered, item)
	}

	sortInventory(filtered, q.Sort)

	total := len(filtered)
	start := (q.Page - 1) * q.PageSize
	if start < 0 {
		start = 0
	}
	if start > total {
		start = total
	}
	end := start + q.PageSize
	if end > total {
		end = total
	}
	page := filtered[start:end]
	if page == nil {
		page = []WarehouseInventoryProduct{}
	}
	return summary, page, total
}

func sortInventory(items []WarehouseInventoryProduct, sortKey string) {
	var less func(i, j int) bool
	switch sortKey {
	case SortNameDesc:
		less = func(i, j int) bool {
			return strings.ToLower(items[i].ProductName) > strings.ToLower(items[j].ProductName)
		}
	case SortTotalAsc:
		less = func(i, j int) bool { return items[i].TotalStock < items[j].TotalStock }
	case SortTotalDesc:
		less = func(i, j int) bool { return items[i].TotalStock > items[j].TotalStock }
	default: // SortName / ""
		less = func(i, j int) bool {
			return strings.ToLower(items[i].ProductName) < strings.ToLower(items[j].ProductName)
		}
	}
	sort.SliceStable(items, less)
}

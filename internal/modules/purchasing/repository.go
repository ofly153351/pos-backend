package purchasing

import (
	"context"
	"errors"
	"strings"

	"gorm.io/gorm"
)

type Repository interface {
	// Suppliers
	CreateSupplier(ctx context.Context, supplier Supplier) (Supplier, error)
	ListSuppliers(ctx context.Context, storeID string) ([]Supplier, error)
	GetSupplier(ctx context.Context, storeID, supplierID string) (Supplier, error)
	UpdateSupplier(ctx context.Context, supplier Supplier) (Supplier, error)
	DeleteSupplier(ctx context.Context, storeID, supplierID string) error

	// Supplier Products
	ListSupplierProducts(ctx context.Context, storeID, supplierID string) ([]SupplierProductResponse, error)
	AddSupplierProduct(ctx context.Context, sp SupplierProduct) (SupplierProduct, error)
	GetSupplierProduct(ctx context.Context, supplierID, productID string) (SupplierProduct, error)
	UpdateSupplierProduct(ctx context.Context, sp SupplierProduct) (SupplierProduct, error)
	RemoveSupplierProduct(ctx context.Context, supplierID, productID string) error

	// Purchase Orders
	CreatePO(ctx context.Context, po PurchaseOrder) (PurchaseOrder, error)
	CreatePOItem(ctx context.Context, item PurchaseOrderItem) error
	ListPOs(ctx context.Context, storeID string) ([]PurchaseOrder, error)
	GetPO(ctx context.Context, storeID, poID string) (PurchaseOrder, error)
	UpdatePO(ctx context.Context, po PurchaseOrder) error
	UpdatePOStatus(ctx context.Context, poID string, status PurchaseOrderStatus, receivedAt *timeSetter) error
	GetPOItems(ctx context.Context, poID string) ([]PurchaseOrderItem, error)
	UpdatePOItemReceived(ctx context.Context, itemID string, receivedQty int, lineTotal float64) error
	GetPODailySequence(ctx context.Context, storeID, datePrefix string) (int, error)
	CreatePODailySequence(ctx context.Context, storeID, datePrefix string, seq int) error

	// Products
	GetProduct(ctx context.Context, storeID, productID string) (ProductRef, error)
	UpdateProductStockAndCost(ctx context.Context, storeID, productID string, addQty int, costPrice float64) error
	CreateProductForSupplier(ctx context.Context, storeID string, name, sku, barcode, productTypeID, productUnitID string, basePrice float64) (string, error)

	// Access
	UserCanOperateStore(ctx context.Context, storeID, userID, role string) (bool, error)
}

type timeSetter struct {
	Time  interface{}
	Valid bool
}

type ProductRef struct {
	ID        string
	StoreID   string
	Name      string
	Quantity  int
	CostPrice float64
}

type PostgresRepository struct {
	db *gorm.DB
}

func NewPostgresRepository(db *gorm.DB) PostgresRepository {
	return PostgresRepository{db: db}
}

// ---- Suppliers ----

func (r PostgresRepository) CreateSupplier(ctx context.Context, supplier Supplier) (Supplier, error) {
	payload := map[string]any{
		"id":             supplier.ID,
		"store_id":       supplier.StoreID,
		"name":           supplier.Name,
		"is_active":      supplier.IsActive,
		"created_at":     supplier.CreatedAt,
		"updated_at":     supplier.CreatedAt,
	}
	if strings.TrimSpace(supplier.Phone) == "" {
		payload["phone"] = nil
	} else {
		payload["phone"] = strings.TrimSpace(supplier.Phone)
	}
	if strings.TrimSpace(supplier.Address) == "" {
		payload["address"] = nil
	} else {
		payload["address"] = strings.TrimSpace(supplier.Address)
	}
	if strings.TrimSpace(supplier.TaxID) == "" {
		payload["tax_id"] = nil
	} else {
		payload["tax_id"] = strings.TrimSpace(supplier.TaxID)
	}
	if strings.TrimSpace(supplier.ContactPerson) == "" {
		payload["contact_person"] = nil
	} else {
		payload["contact_person"] = strings.TrimSpace(supplier.ContactPerson)
	}
	if strings.TrimSpace(supplier.Note) == "" {
		payload["note"] = nil
	} else {
		payload["note"] = strings.TrimSpace(supplier.Note)
	}

	if err := r.db.WithContext(ctx).Table("suppliers").Create(payload).Error; err != nil {
		return Supplier{}, err
	}
	supplier.UpdatedAt = supplier.CreatedAt
	return supplier, nil
}

func (r PostgresRepository) ListSuppliers(ctx context.Context, storeID string) ([]Supplier, error) {
	var items []Supplier
	err := r.db.WithContext(ctx).
		Model(&Supplier{}).
		Where("store_id = ?", storeID).
		Order("created_at DESC").
		Find(&items).Error
	return items, err
}

func (r PostgresRepository) GetSupplier(ctx context.Context, storeID, supplierID string) (Supplier, error) {
	var item Supplier
	err := r.db.WithContext(ctx).
		Model(&Supplier{}).
		Where("store_id = ? AND id = ?", storeID, supplierID).
		Take(&item).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return Supplier{}, ErrSupplierNotFound
		}
		return Supplier{}, err
	}
	return item, nil
}

func (r PostgresRepository) UpdateSupplier(ctx context.Context, supplier Supplier) (Supplier, error) {
	updates := map[string]any{
		"name":       supplier.Name,
		"is_active":  supplier.IsActive,
		"updated_at": supplier.UpdatedAt,
	}
	if strings.TrimSpace(supplier.Phone) == "" {
		updates["phone"] = nil
	} else {
		updates["phone"] = strings.TrimSpace(supplier.Phone)
	}
	if strings.TrimSpace(supplier.Address) == "" {
		updates["address"] = nil
	} else {
		updates["address"] = strings.TrimSpace(supplier.Address)
	}
	if strings.TrimSpace(supplier.TaxID) == "" {
		updates["tax_id"] = nil
	} else {
		updates["tax_id"] = strings.TrimSpace(supplier.TaxID)
	}
	if strings.TrimSpace(supplier.ContactPerson) == "" {
		updates["contact_person"] = nil
	} else {
		updates["contact_person"] = strings.TrimSpace(supplier.ContactPerson)
	}
	if strings.TrimSpace(supplier.Note) == "" {
		updates["note"] = nil
	} else {
		updates["note"] = strings.TrimSpace(supplier.Note)
	}

	result := r.db.WithContext(ctx).
		Model(&Supplier{}).
		Where("store_id = ? AND id = ?", supplier.StoreID, supplier.ID).
		Updates(updates)
	if result.Error != nil {
		return Supplier{}, result.Error
	}
	if result.RowsAffected == 0 {
		return Supplier{}, ErrSupplierNotFound
	}
	return supplier, nil
}

func (r PostgresRepository) DeleteSupplier(ctx context.Context, storeID, supplierID string) error {
	result := r.db.WithContext(ctx).
		Where("store_id = ? AND id = ?", storeID, supplierID).
		Delete(&Supplier{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrSupplierNotFound
	}
	return nil
}

// ---- Purchase Orders ----

func (r PostgresRepository) CreatePO(ctx context.Context, po PurchaseOrder) (PurchaseOrder, error) {
	payload := map[string]any{
		"id":           po.ID,
		"store_id":     po.StoreID,
		"order_number": po.OrderNumber,
		"status":       string(po.Status),
		"total_cost":   po.TotalCost,
		"created_at":   po.CreatedAt,
		"updated_at":   po.CreatedAt,
	}
	if strings.TrimSpace(po.SupplierID) == "" {
		payload["supplier_id"] = nil
	} else {
		payload["supplier_id"] = po.SupplierID
	}
	if strings.TrimSpace(po.Notes) == "" {
		payload["notes"] = nil
	} else {
		payload["notes"] = strings.TrimSpace(po.Notes)
	}

	if err := r.db.WithContext(ctx).Table("purchase_orders").Create(payload).Error; err != nil {
		return PurchaseOrder{}, err
	}
	po.UpdatedAt = po.CreatedAt
	return po, nil
}

func (r PostgresRepository) CreatePOItem(ctx context.Context, item PurchaseOrderItem) error {
	payload := map[string]any{
		"id":                item.ID,
		"purchase_order_id": item.PurchaseOrderID,
		"product_id":        item.ProductID,
		"quantity":          item.Quantity,
		"received_quantity": 0,
		"unit_cost":         item.UnitCost,
		"line_total":        item.LineTotal,
		"created_at":        item.CreatedAt,
	}
	return r.db.WithContext(ctx).Table("purchase_order_items").Create(payload).Error
}

func (r PostgresRepository) ListPOs(ctx context.Context, storeID string) ([]PurchaseOrder, error) {
	var items []PurchaseOrder
	err := r.db.WithContext(ctx).
		Model(&PurchaseOrder{}).
		Preload("Supplier").
		Where("store_id = ?", storeID).
		Order("created_at DESC").
		Find(&items).Error
	return items, err
}

func (r PostgresRepository) GetPO(ctx context.Context, storeID, poID string) (PurchaseOrder, error) {
	var item PurchaseOrder
	err := r.db.WithContext(ctx).
		Model(&PurchaseOrder{}).
		Preload("Supplier").
		Preload("Items").
		Where("store_id = ? AND id = ?", storeID, poID).
		Take(&item).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return PurchaseOrder{}, ErrPONotFound
		}
		return PurchaseOrder{}, err
	}
	return item, nil
}

func (r PostgresRepository) UpdatePO(ctx context.Context, po PurchaseOrder) error {
	updates := map[string]any{
		"updated_at": po.UpdatedAt,
	}
	if po.SupplierID != "" {
		updates["supplier_id"] = po.SupplierID
	}
	if po.Notes != "" {
		updates["notes"] = po.Notes
	}

	result := r.db.WithContext(ctx).
		Model(&PurchaseOrder{}).
		Where("store_id = ? AND id = ?", po.StoreID, po.ID).
		Updates(updates)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrPONotFound
	}
	return nil
}

func (r PostgresRepository) UpdatePOStatus(ctx context.Context, poID string, status PurchaseOrderStatus, receivedAt *timeSetter) error {
	updates := map[string]any{
		"status":     string(status),
		"updated_at": gorm.Expr("NOW()"),
	}
	if receivedAt != nil && receivedAt.Valid {
		updates["received_at"] = receivedAt.Time
	}
	return r.db.WithContext(ctx).
		Model(&PurchaseOrder{}).
		Where("id = ?", poID).
		Updates(updates).Error
}

func (r PostgresRepository) GetPOItems(ctx context.Context, poID string) ([]PurchaseOrderItem, error) {
	var items []PurchaseOrderItem
	err := r.db.WithContext(ctx).
		Model(&PurchaseOrderItem{}).
		Where("purchase_order_id = ?", poID).
		Find(&items).Error
	return items, err
}

func (r PostgresRepository) UpdatePOItemReceived(ctx context.Context, itemID string, receivedQty int, lineTotal float64) error {
	return r.db.WithContext(ctx).
		Model(&PurchaseOrderItem{}).
		Where("id = ?", itemID).
		Updates(map[string]any{
			"received_quantity": receivedQty,
			"line_total":        lineTotal,
		}).Error
}

func (r PostgresRepository) GetPODailySequence(ctx context.Context, storeID, datePrefix string) (int, error) {
	var seq struct {
		Seq int
	}
	err := r.db.WithContext(ctx).
		Table("purchase_orders").
		Select("COUNT(*) as seq").
		Where("store_id = ? AND order_number LIKE ?", storeID, "PO-"+datePrefix+"-%").
		Take(&seq).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return 0, nil
		}
		return 0, err
	}
	return seq.Seq, nil
}

func (r PostgresRepository) CreatePODailySequence(ctx context.Context, storeID, datePrefix string, seq int) error {
	// We use order_number pattern to derive seq, no separate table needed
	return nil
}

// ---- Products ----

func (r PostgresRepository) GetProduct(ctx context.Context, storeID, productID string) (ProductRef, error) {
	var prod struct {
		ID        string
		StoreID   string
		Name      string
		Quantity  int
		CostPrice float64
	}
	err := r.db.WithContext(ctx).
		Table("product_view").
		Select("id, store_id, name, COALESCE(total_stock, 0) AS quantity, cost_price").
		Where("id = ? AND store_id = ?", productID, storeID).
		Take(&prod).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ProductRef{}, ErrPOProductNotFound
		}
		return ProductRef{}, err
	}
	return ProductRef{
		ID:        prod.ID,
		StoreID:   prod.StoreID,
		Name:      prod.Name,
		Quantity:  prod.Quantity,
		CostPrice: prod.CostPrice,
	}, nil
}

func (r PostgresRepository) UpdateProductStockAndCost(ctx context.Context, storeID, productID string, addQty int, costPrice float64) error {
	// Find or create a default receiving location in this store
	var locID string
	err := r.db.WithContext(ctx).
		Table("locations").
		Where("store_id = ? AND is_sale_point = ?", storeID, false).
		Order("created_at ASC").
		Select("id").
		Take(&locID).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// Try any location (for stores with only sale_point locations)
			if err2 := r.db.WithContext(ctx).
				Table("locations").
				Where("store_id = ?", storeID).
				Order("created_at ASC").
				Select("id").
				Take(&locID).Error; err2 == nil {
				goto found
			}
			// Auto-create a default receiving location
			locID = newID()
			var whID string
			if err := r.db.WithContext(ctx).
				Table("warehouses").
				Where("store_id = ?", storeID).
				Order("created_at ASC").
				Select("id").
				Take(&whID).Error; err != nil {
				return err
			}
			if err := r.db.WithContext(ctx).
				Table("locations").
				Create(map[string]any{
					"id":           locID,
					"store_id":     storeID,
					"warehouse_id": whID,
					"name":         "รับสินค้าเข้า",
					"is_sale_point": false,
					"is_active":    true,
					"created_at":   gorm.Expr("NOW()"),
					"updated_at":   gorm.Expr("NOW()"),
				}).Error; err != nil {
				return err
			}
		} else {
			return err
		}
	}
found:

	// Upsert stock at location, update cost_price on product
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec(
			`INSERT INTO stocks (id, store_id, product_id, location_id, quantity, created_at, updated_at)
			 VALUES (?, ?, ?, ?, GREATEST(0, ?), NOW(), NOW())
			 ON CONFLICT (product_id, location_id)
			 DO UPDATE SET quantity = stocks.quantity + ?, updated_at = NOW()
			 WHERE (stocks.quantity + ?) >= 0`,
			newID(), storeID, productID, locID, addQty, addQty, addQty,
		).Error; err != nil {
			return err
		}
		return tx.Table("products").
			Where("id = ?", productID).
			Update("cost_price", costPrice).Error
	})
}

func (r PostgresRepository) CreateProductForSupplier(ctx context.Context, storeID string, name, sku, barcode, productTypeID, productUnitID string, basePrice float64) (string, error) {
	id := newID()
	now := gorm.Expr("NOW()")
	payload := map[string]any{
		"id":          id,
		"store_id":    storeID,
		"name":        name,
		"sku":         sku,
		"barcode":     barcode,
		"base_price":  basePrice,
		"is_active":   true,
		"created_at":  now,
		"updated_at":  now,
	}
	if strings.TrimSpace(productTypeID) != "" {
		payload["product_type_id"] = productTypeID
	}
	if strings.TrimSpace(productUnitID) != "" {
		payload["product_unit_id"] = productUnitID
	} else {
		// product_unit_id is NOT NULL — find or create a default unit for this store
		var defaultUnitID string
		unitID := newID()
		err := r.db.WithContext(ctx).
			Table("product_units").
			Where("store_id = ?", storeID).
			Order("created_at ASC").
			Select("id").
			Take(&defaultUnitID).Error
		if err != nil {
			// Create a default 'unit' for this store
			defaultUnitID = unitID
			if err := r.db.WithContext(ctx).Table("product_units").Create(map[string]any{
				"id":         defaultUnitID,
				"store_id":   storeID,
				"name":       "unit",
				"is_active":  true,
				"created_at": now,
				"updated_at": now,
			}).Error; err != nil {
				return "", err
			}
		}
		payload["product_unit_id"] = defaultUnitID
	}
	if err := r.db.WithContext(ctx).Table("products").Create(payload).Error; err != nil {
		return "", err
	}
	return id, nil
}

// ---- Access ----

func (r PostgresRepository) UserCanOperateStore(ctx context.Context, storeID, userID, role string) (bool, error) {
	if role == "platform_admin" {
		return true, nil
	}
	var count int64
	err := r.db.WithContext(ctx).
		Table("store_members").
		Where("store_id = ? AND user_id = ? AND role IN ?", storeID, userID, []string{"owner", "manager", "cashier"}).
		Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// ---- Supplier Products ----

func (r PostgresRepository) ListSupplierProducts(ctx context.Context, storeID, supplierID string) ([]SupplierProductResponse, error) {
	var items []SupplierProductResponse
	err := r.db.WithContext(ctx).
		Table("supplier_products").
		Select(`
			supplier_products.id,
			supplier_products.supplier_id,
			supplier_products.product_id,
			COALESCE(products.name, '') AS product_name,
			COALESCE(products.sku, '') AS product_sku,
			supplier_products.supplier_sku,
			supplier_products.supplier_price,
			supplier_products.created_at
		`).
		Joins("LEFT JOIN products ON products.id = supplier_products.product_id AND products.store_id = ?", storeID).
		Where("supplier_products.supplier_id = ?", supplierID).
		Order("supplier_products.created_at DESC").
		Scan(&items).Error
	if err != nil {
		return nil, err
	}
	return items, nil
}

func (r PostgresRepository) AddSupplierProduct(ctx context.Context, sp SupplierProduct) (SupplierProduct, error) {
	payload := map[string]any{
		"id":             sp.ID,
		"supplier_id":    sp.SupplierID,
		"product_id":     sp.ProductID,
		"supplier_sku":   sp.SupplierSKU,
		"supplier_price": sp.SupplierPrice,
		"created_at":     sp.CreatedAt,
		"updated_at":     sp.CreatedAt,
	}
	if err := r.db.WithContext(ctx).Table("supplier_products").Create(payload).Error; err != nil {
		return SupplierProduct{}, err
	}
	sp.UpdatedAt = sp.CreatedAt
	return sp, nil
}

func (r PostgresRepository) GetSupplierProduct(ctx context.Context, supplierID, productID string) (SupplierProduct, error) {
	var item SupplierProduct
	err := r.db.WithContext(ctx).
		Model(&SupplierProduct{}).
		Where("supplier_id = ? AND product_id = ?", supplierID, productID).
		Take(&item).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return SupplierProduct{}, ErrSupplierProductNotFound
		}
		return SupplierProduct{}, err
	}
	return item, nil
}

func (r PostgresRepository) UpdateSupplierProduct(ctx context.Context, sp SupplierProduct) (SupplierProduct, error) {
	updates := map[string]any{
		"supplier_sku":   sp.SupplierSKU,
		"supplier_price": sp.SupplierPrice,
		"updated_at":     sp.UpdatedAt,
	}
	result := r.db.WithContext(ctx).
		Model(&SupplierProduct{}).
		Where("supplier_id = ? AND product_id = ?", sp.SupplierID, sp.ProductID).
		Updates(updates)
	if result.Error != nil {
		return SupplierProduct{}, result.Error
	}
	if result.RowsAffected == 0 {
		return SupplierProduct{}, ErrSupplierProductNotFound
	}
	return sp, nil
}

func (r PostgresRepository) RemoveSupplierProduct(ctx context.Context, supplierID, productID string) error {
	result := r.db.WithContext(ctx).
		Where("supplier_id = ? AND product_id = ?", supplierID, productID).
		Delete(&SupplierProduct{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrSupplierProductNotFound
	}
	return nil
}

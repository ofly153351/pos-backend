package warehouse_receipt

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"
)

type Repository interface {
	WithTx(tx *gorm.DB) Repository
	FindPrimaryStoreIDByUserID(ctx context.Context, userID string) (string, error)
	UserCanOperateStore(ctx context.Context, storeID, userID, role string) (bool, error)
	UserCanManageStore(ctx context.Context, storeID, userID, role string) (bool, error)
	GenerateDocumentNo(ctx context.Context, now time.Time) (string, error)
	ListByStore(ctx context.Context, storeID string, status *ReceiptStatus, page, limit int) ([]WarehouseReceipt, int64, error)
	Create(ctx context.Context, receipt WarehouseReceipt) (WarehouseReceipt, error)
	GetByID(ctx context.Context, receiptID string) (WarehouseReceipt, error)
	Update(ctx context.Context, receipt WarehouseReceipt) (WarehouseReceipt, error)
	GetItemByID(ctx context.Context, receiptID, itemID string) (WarehouseReceiptItem, error)
	DeleteItemsByReceiptID(ctx context.Context, receiptID string) error
	CreateItems(ctx context.Context, items []WarehouseReceiptItem) error
	UpdateItem(ctx context.Context, item WarehouseReceiptItem) error
	DeleteItem(ctx context.Context, receiptID, itemID string) error
	RecalculateTotals(ctx context.Context, receiptID string) error
	GetProductSnapshot(ctx context.Context, storeID, productID string) (receiptProductSnapshot, error)
	GetLocationSnapshot(ctx context.Context, storeID, locationID string) (locationSnapshot, error)
	WarehouseExists(ctx context.Context, storeID, warehouseID string) (bool, error)
	SupplierExists(ctx context.Context, storeID, supplierID string) (bool, error)
	GetPurchaseOrder(ctx context.Context, storeID, purchaseOrderID string) (purchaseOrderSnapshot, error)
	GetPurchaseOrderItems(ctx context.Context, purchaseOrderID string) ([]purchaseOrderItemSnapshot, error)
	UpdateAttachment(ctx context.Context, receiptID, url, mimeType, name string, size int64, updatedBy string) error
	AppendAudit(ctx context.Context, audit WarehouseReceiptAudit) error
	ComputePreview(ctx context.Context, receiptID string) ([]StockImpactPreview, error)
	ConfirmDraft(ctx context.Context, receiptID, actorID string) error
	CancelDraft(ctx context.Context, receiptID, actorID string) error
}

type PostgresRepository struct{ db *gorm.DB }

func NewPostgresRepository(db *gorm.DB) PostgresRepository { return PostgresRepository{db: db} }
func (r PostgresRepository) WithTx(tx *gorm.DB) Repository { return PostgresRepository{db: tx} }

func (r PostgresRepository) FindPrimaryStoreIDByUserID(ctx context.Context, userID string) (string, error) {
	type storeMember struct {
		StoreID string `gorm:"column:store_id"`
	}

	var member storeMember
	err := r.db.WithContext(ctx).
		Table("store_members").
		Select("store_id").
		Where("user_id = ?", userID).
		Order("created_at ASC").
		Limit(1).
		Take(&member).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "", nil
		}
		return "", err
	}

	return member.StoreID, nil
}

func (r PostgresRepository) UserCanOperateStore(ctx context.Context, storeID, userID, role string) (bool, error) {
	if role == "platform_admin" {
		return true, nil
	}
	var count int64
	err := r.db.WithContext(ctx).Table("store_members").Where("store_id = ? AND user_id = ? AND role IN ?", storeID, userID, []string{"owner", "manager", "cashier"}).Count(&count).Error
	return count > 0, err
}

func (r PostgresRepository) UserCanManageStore(ctx context.Context, storeID, userID, role string) (bool, error) {
	if role == "platform_admin" {
		return true, nil
	}
	var count int64
	err := r.db.WithContext(ctx).Table("store_members").Where("store_id = ? AND user_id = ? AND role IN ?", storeID, userID, []string{"owner", "manager"}).Count(&count).Error
	return count > 0, err
}

func (r PostgresRepository) GenerateDocumentNo(ctx context.Context, now time.Time) (string, error) {
	dateKey := documentDateKey(now)
	var seq int
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec("SELECT pg_advisory_xact_lock(hashtext(?))", "warehouse-receipt-doc-no-"+dateKey).Error; err != nil {
			return err
		}
		row := tx.Raw(`
			INSERT INTO warehouse_receipt_document_sequences (date_key, last_sequence, created_at, updated_at)
			VALUES (?, 1, NOW(), NOW())
			ON CONFLICT (date_key)
			DO UPDATE SET last_sequence = warehouse_receipt_document_sequences.last_sequence + 1, updated_at = NOW()
			RETURNING last_sequence
		`, dateKey).Row()
		return row.Scan(&seq)
	})
	if err != nil {
		return "", err
	}
	return formatReceiptDocumentNo(now, seq), nil
}

func (r PostgresRepository) ListByStore(ctx context.Context, storeID string, status *ReceiptStatus, page, limit int) ([]WarehouseReceipt, int64, error) {
	query := r.db.WithContext(ctx).Table("warehouse_receipts wr").Where("wr.store_id = ?", storeID)
	if status != nil {
		query = query.Where("wr.status = ?", string(*status))
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var items []WarehouseReceipt
	err := query.Select(`
		wr.id, wr.store_id, wr.warehouse_id, COALESCE(wr.supplier_id, '') AS supplier_id,
		COALESCE(wr.purchase_order_id, '') AS purchase_order_id, wr.document_no, wr.status,
		wr.received_at, COALESCE(wr.reference_no, '') AS reference_no, COALESCE(wr.note, '') AS note,
		wr.vat_included, wr.vat_percent, wr.total_items, wr.subtotal_amount, wr.discount_amount,
		wr.net_amount, wr.vat_amount, wr.total_amount, COALESCE(wr.attachment_url, '') AS attachment_url,
		COALESCE(wr.attachment_mime_type, '') AS attachment_mime_type, COALESCE(wr.attachment_name, '') AS attachment_name,
		COALESCE(wr.attachment_size, 0) AS attachment_size, wr.created_by, COALESCE(wr.confirmed_by, '') AS confirmed_by,
		COALESCE(wr.cancelled_by, '') AS cancelled_by, wr.confirmed_at, wr.cancelled_at, wr.created_at, wr.updated_at,
		COALESCE(wh.name, '') AS warehouse_name, COALESCE(sp.name, '') AS supplier_name,
		COALESCE(cu.full_name, '') AS created_by_name, COALESCE(cf.full_name, '') AS confirmed_by_name,
		COALESCE(cc.full_name, '') AS cancelled_by_name, COALESCE(po.order_number, '') AS purchase_order_no
	`).Joins("LEFT JOIN warehouses wh ON wh.id = wr.warehouse_id").Joins("LEFT JOIN suppliers sp ON sp.id = wr.supplier_id").Joins("LEFT JOIN users cu ON cu.id = wr.created_by").Joins("LEFT JOIN users cf ON cf.id = wr.confirmed_by").Joins("LEFT JOIN users cc ON cc.id = wr.cancelled_by").Joins("LEFT JOIN purchase_orders po ON po.id = wr.purchase_order_id").Order("wr.received_at DESC").Order("wr.created_at DESC").Limit(limit).Offset((page - 1) * limit).Find(&items).Error
	if err != nil {
		return nil, 0, err
	}
	if items == nil {
		items = []WarehouseReceipt{}
	}
	for i := range items {
		items[i].Items = []WarehouseReceiptItem{}
		items[i].Preview = []StockImpactPreview{}
		items[i].Audits = []WarehouseReceiptAudit{}
	}
	return items, total, nil
}

func (r PostgresRepository) Create(ctx context.Context, receipt WarehouseReceipt) (WarehouseReceipt, error) {
	payload := map[string]any{
		"id":              receipt.ID,
		"store_id":        receipt.StoreID,
		"warehouse_id":    receipt.WarehouseID,
		"document_no":     receipt.DocumentNo,
		"status":          string(receipt.Status),
		"received_at":     receipt.ReceivedAt,
		"reference_no":    nilIfEmpty(receipt.ReferenceNo),
		"note":            nilIfEmpty(receipt.Note),
		"vat_included":    receipt.VATIncluded,
		"vat_percent":     receipt.VATPercent,
		"total_items":     receipt.TotalItems,
		"subtotal_amount": receipt.SubtotalAmount,
		"discount_amount": receipt.DiscountAmount,
		"net_amount":      receipt.NetAmount,
		"vat_amount":      receipt.VATAmount,
		"total_amount":    receipt.TotalAmount,
		"created_by":      receipt.CreatedBy,
		"created_at":      receipt.CreatedAt,
		"updated_at":      receipt.CreatedAt,
	}
	if strings.TrimSpace(receipt.SupplierID) != "" {
		payload["supplier_id"] = receipt.SupplierID
	}
	if strings.TrimSpace(receipt.PurchaseOrderID) != "" {
		payload["purchase_order_id"] = receipt.PurchaseOrderID
	}
	if err := r.db.WithContext(ctx).Table("warehouse_receipts").Create(payload).Error; err != nil {
		return WarehouseReceipt{}, mapWriteError(err)
	}
	return r.GetByID(ctx, receipt.ID)
}

func (r PostgresRepository) GetByID(ctx context.Context, receiptID string) (WarehouseReceipt, error) {
	var receipt WarehouseReceipt
	err := r.db.WithContext(ctx).Table("warehouse_receipts wr").Select(`
		wr.id, wr.store_id, wr.warehouse_id, COALESCE(wr.supplier_id, '') AS supplier_id,
		COALESCE(wr.purchase_order_id, '') AS purchase_order_id, wr.document_no, wr.status,
		wr.received_at, COALESCE(wr.reference_no, '') AS reference_no, COALESCE(wr.note, '') AS note,
		wr.vat_included, wr.vat_percent, wr.total_items, wr.subtotal_amount, wr.discount_amount,
		wr.net_amount, wr.vat_amount, wr.total_amount, COALESCE(wr.attachment_url, '') AS attachment_url,
		COALESCE(wr.attachment_mime_type, '') AS attachment_mime_type, COALESCE(wr.attachment_name, '') AS attachment_name,
		COALESCE(wr.attachment_size, 0) AS attachment_size, wr.created_by, COALESCE(wr.confirmed_by, '') AS confirmed_by,
		COALESCE(wr.cancelled_by, '') AS cancelled_by, wr.confirmed_at, wr.cancelled_at, wr.created_at, wr.updated_at,
		COALESCE(wh.name, '') AS warehouse_name, COALESCE(sp.name, '') AS supplier_name,
		COALESCE(cu.full_name, '') AS created_by_name, COALESCE(cf.full_name, '') AS confirmed_by_name,
		COALESCE(cc.full_name, '') AS cancelled_by_name, COALESCE(po.order_number, '') AS purchase_order_no
	`).Joins("LEFT JOIN warehouses wh ON wh.id = wr.warehouse_id").Joins("LEFT JOIN suppliers sp ON sp.id = wr.supplier_id").Joins("LEFT JOIN users cu ON cu.id = wr.created_by").Joins("LEFT JOIN users cf ON cf.id = wr.confirmed_by").Joins("LEFT JOIN users cc ON cc.id = wr.cancelled_by").Joins("LEFT JOIN purchase_orders po ON po.id = wr.purchase_order_id").Where("wr.id = ?", receiptID).Take(&receipt).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return WarehouseReceipt{}, ErrReceiptNotFound
		}
		return WarehouseReceipt{}, err
	}
	items, err := r.listItems(ctx, receiptID)
	if err != nil {
		return WarehouseReceipt{}, err
	}
	receipt.Items = items
	audits, err := r.listAudits(ctx, receiptID)
	if err != nil {
		return WarehouseReceipt{}, err
	}
	receipt.Audits = audits
	preview, err := r.ComputePreview(ctx, receiptID)
	if err != nil {
		return WarehouseReceipt{}, err
	}
	receipt.Preview = preview
	if receipt.Items == nil {
		receipt.Items = []WarehouseReceiptItem{}
	}
	if receipt.Preview == nil {
		receipt.Preview = []StockImpactPreview{}
	}
	if receipt.Audits == nil {
		receipt.Audits = []WarehouseReceiptAudit{}
	}
	return receipt, nil
}

func (r PostgresRepository) Update(ctx context.Context, receipt WarehouseReceipt) (WarehouseReceipt, error) {
	updates := map[string]any{
		"warehouse_id":      receipt.WarehouseID,
		"supplier_id":       nilIfEmpty(receipt.SupplierID),
		"purchase_order_id": nilIfEmpty(receipt.PurchaseOrderID),
		"document_no":       receipt.DocumentNo,
		"received_at":       receipt.ReceivedAt,
		"reference_no":      nilIfEmpty(receipt.ReferenceNo),
		"note":              nilIfEmpty(receipt.Note),
		"vat_included":      receipt.VATIncluded,
		"vat_percent":       receipt.VATPercent,
		"updated_at":        receipt.UpdatedAt,
	}
	result := r.db.WithContext(ctx).Table("warehouse_receipts").Where("id = ?", receipt.ID).Updates(updates)
	if result.Error != nil {
		return WarehouseReceipt{}, mapWriteError(result.Error)
	}
	if result.RowsAffected == 0 {
		return WarehouseReceipt{}, ErrReceiptNotFound
	}
	return r.GetByID(ctx, receipt.ID)
}

func (r PostgresRepository) GetItemByID(ctx context.Context, receiptID, itemID string) (WarehouseReceiptItem, error) {
	var item WarehouseReceiptItem
	err := r.db.WithContext(ctx).Table("warehouse_receipt_items").Where("receipt_id = ? AND id = ?", receiptID, itemID).Take(&item).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return WarehouseReceiptItem{}, ErrReceiptItemNotFound
		}
		return WarehouseReceiptItem{}, err
	}
	return item, nil
}

func (r PostgresRepository) DeleteItemsByReceiptID(ctx context.Context, receiptID string) error {
	return r.db.WithContext(ctx).Table("warehouse_receipt_items").Where("receipt_id = ?", receiptID).Delete(nil).Error
}

func (r PostgresRepository) CreateItems(ctx context.Context, items []WarehouseReceiptItem) error {
	if len(items) == 0 {
		return nil
	}
	payload := make([]map[string]any, 0, len(items))
	for _, item := range items {
		row := map[string]any{
			"id":              item.ID,
			"receipt_id":      item.ReceiptID,
			"product_id":      item.ProductID,
			"location_id":     item.LocationID,
			"warehouse_id":    item.WarehouseID,
			"zone_name":       nilIfEmpty(item.ZoneName),
			"floor_name":      nilIfEmpty(item.FloorName),
			"location_name":   item.LocationName,
			"product_name":    item.ProductName,
			"sku":             nilIfEmpty(item.SKU),
			"barcode":         nilIfEmpty(item.Barcode),
			"unit_name":       nilIfEmpty(item.UnitName),
			"quantity":        item.Quantity,
			"unit_price":      item.UnitPrice,
			"discount_type":   nilIfEmpty(item.DiscountType),
			"discount_value":  nilFloat(item.DiscountValue),
			"discount_amount": item.DiscountAmount,
			"line_subtotal":   item.LineSubtotal,
			"line_net":        item.LineNet,
			"created_at":      item.CreatedAt,
			"updated_at":      item.UpdatedAt,
		}
		payload = append(payload, row)
	}
	if err := r.db.WithContext(ctx).Table("warehouse_receipt_items").Create(&payload).Error; err != nil {
		return mapWriteError(err)
	}
	return nil
}

func (r PostgresRepository) UpdateItem(ctx context.Context, item WarehouseReceiptItem) error {
	result := r.db.WithContext(ctx).Table("warehouse_receipt_items").Where("receipt_id = ? AND id = ?", item.ReceiptID, item.ID).Updates(map[string]any{
		"product_id":      item.ProductID,
		"location_id":     item.LocationID,
		"warehouse_id":    item.WarehouseID,
		"zone_name":       nilIfEmpty(item.ZoneName),
		"floor_name":      nilIfEmpty(item.FloorName),
		"location_name":   item.LocationName,
		"product_name":    item.ProductName,
		"sku":             nilIfEmpty(item.SKU),
		"barcode":         nilIfEmpty(item.Barcode),
		"unit_name":       nilIfEmpty(item.UnitName),
		"quantity":        item.Quantity,
		"unit_price":      item.UnitPrice,
		"discount_type":   nilIfEmpty(item.DiscountType),
		"discount_value":  nilFloat(item.DiscountValue),
		"discount_amount": item.DiscountAmount,
		"line_subtotal":   item.LineSubtotal,
		"line_net":        item.LineNet,
		"updated_at":      item.UpdatedAt,
	})
	if result.Error != nil {
		return mapWriteError(result.Error)
	}
	if result.RowsAffected == 0 {
		return ErrReceiptItemNotFound
	}
	return nil
}

func (r PostgresRepository) DeleteItem(ctx context.Context, receiptID, itemID string) error {
	result := r.db.WithContext(ctx).Table("warehouse_receipt_items").Where("receipt_id = ? AND id = ?", receiptID, itemID).Delete(nil)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrReceiptItemNotFound
	}
	return nil
}

func (r PostgresRepository) RecalculateTotals(ctx context.Context, receiptID string) error {
	var totals struct {
		TotalItems     int     `gorm:"column:total_items"`
		SubtotalAmount float64 `gorm:"column:subtotal_amount"`
		DiscountAmount float64 `gorm:"column:discount_amount"`
	}
	if err := r.db.WithContext(ctx).Table("warehouse_receipt_items").Select(`
		COALESCE(COUNT(*), 0) AS total_items,
		COALESCE(SUM(line_subtotal), 0) AS subtotal_amount,
		COALESCE(SUM(discount_amount * quantity), 0) AS discount_amount
	`).Where("receipt_id = ?", receiptID).Take(&totals).Error; err != nil {
		return err
	}
	var header struct {
		VATIncluded bool    `gorm:"column:vat_included"`
		VATPercent  float64 `gorm:"column:vat_percent"`
	}
	if err := r.db.WithContext(ctx).Table("warehouse_receipts").Select("vat_included, vat_percent").Where("id = ?", receiptID).Take(&header).Error; err != nil {
		return err
	}
	netAmount := totals.SubtotalAmount - totals.DiscountAmount
	if netAmount < 0 {
		netAmount = 0
	}
	vatAmount := 0.0
	totalAmount := netAmount
	if header.VATPercent > 0 {
		if header.VATIncluded {
			vatAmount = netAmount * header.VATPercent / (100 + header.VATPercent)
		} else {
			vatAmount = netAmount * header.VATPercent / 100
			totalAmount = netAmount + vatAmount
		}
	}
	return r.db.WithContext(ctx).Table("warehouse_receipts").Where("id = ?", receiptID).Updates(map[string]any{
		"total_items":     totals.TotalItems,
		"subtotal_amount": roundMoney(totals.SubtotalAmount),
		"discount_amount": roundMoney(totals.DiscountAmount),
		"net_amount":      roundMoney(netAmount),
		"vat_amount":      roundMoney(vatAmount),
		"total_amount":    roundMoney(totalAmount),
		"updated_at":      gorm.Expr("NOW()"),
	}).Error
}

func (r PostgresRepository) GetProductSnapshot(ctx context.Context, storeID, productID string) (receiptProductSnapshot, error) {
	var item receiptProductSnapshot
	err := r.db.WithContext(ctx).Table("product_view").Select("id, store_id, name, COALESCE(sku, '') AS sku, COALESCE(barcode, '') AS barcode, COALESCE(product_unit_name, '') AS unit_name, is_active, COALESCE(cost_price, 0) AS cost_price").Where("id = ? AND store_id = ?", productID, storeID).Take(&item).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return receiptProductSnapshot{}, ErrReceiptProductNotFound
		}
		return receiptProductSnapshot{}, err
	}
	return item, nil
}

func (r PostgresRepository) GetLocationSnapshot(ctx context.Context, storeID, locationID string) (locationSnapshot, error) {
	var item locationSnapshot
	err := r.db.WithContext(ctx).Table("locations").Select("id, store_id, warehouse_id, name, COALESCE(zone_name, '') AS zone_name, COALESCE(floor_name, '') AS floor_name, is_active, is_sale_point").Where("id = ? AND store_id = ?", locationID, storeID).Take(&item).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return locationSnapshot{}, ErrReceiptLocationNotFound
		}
		return locationSnapshot{}, err
	}
	return item, nil
}

func (r PostgresRepository) WarehouseExists(ctx context.Context, storeID, warehouseID string) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Table("warehouses").Where("id = ? AND store_id = ?", warehouseID, storeID).Count(&count).Error
	return count > 0, err
}

func (r PostgresRepository) SupplierExists(ctx context.Context, storeID, supplierID string) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Table("suppliers").Where("id = ? AND store_id = ?", supplierID, storeID).Count(&count).Error
	return count > 0, err
}

func (r PostgresRepository) GetPurchaseOrder(ctx context.Context, storeID, purchaseOrderID string) (purchaseOrderSnapshot, error) {
	var item purchaseOrderSnapshot
	err := r.db.WithContext(ctx).Table("purchase_orders").Select("id, store_id, COALESCE(supplier_id, '') AS supplier_id, order_number, status").Where("id = ? AND store_id = ?", purchaseOrderID, storeID).Take(&item).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return purchaseOrderSnapshot{}, ErrReceiptPurchaseOrderNotFound
		}
		return purchaseOrderSnapshot{}, err
	}
	return item, nil
}

func (r PostgresRepository) GetPurchaseOrderItems(ctx context.Context, purchaseOrderID string) ([]purchaseOrderItemSnapshot, error) {
	var items []purchaseOrderItemSnapshot
	err := r.db.WithContext(ctx).Table("purchase_order_items").Select("id, purchase_order_id, product_id, quantity, received_quantity, unit_cost").Where("purchase_order_id = ?", purchaseOrderID).Find(&items).Error
	return items, err
}

func (r PostgresRepository) UpdateAttachment(ctx context.Context, receiptID, url, mimeType, name string, size int64, updatedBy string) error {
	return r.db.WithContext(ctx).Table("warehouse_receipts").Where("id = ?", receiptID).Updates(map[string]any{
		"attachment_url":       nilIfEmpty(url),
		"attachment_mime_type": nilIfEmpty(mimeType),
		"attachment_name":      nilIfEmpty(name),
		"attachment_size":      size,
		"updated_at":           gorm.Expr("NOW()"),
	}).Error
}

func (r PostgresRepository) AppendAudit(ctx context.Context, audit WarehouseReceiptAudit) error {
	return r.db.WithContext(ctx).Table("warehouse_receipt_audits").Create(map[string]any{
		"id":          audit.ID,
		"receipt_id":  audit.ReceiptID,
		"action":      audit.Action,
		"description": nilIfEmpty(audit.Description),
		"actor_id":    audit.ActorID,
		"created_at":  audit.CreatedAt,
	}).Error
}

func (r PostgresRepository) ComputePreview(ctx context.Context, receiptID string) ([]StockImpactPreview, error) {
	items, err := r.listItems(ctx, receiptID)
	if err != nil {
		return nil, err
	}
	preview := make([]StockImpactPreview, 0, len(items))
	for _, item := range items {
		var qty int
		if err := r.db.WithContext(ctx).Table("stocks").Select("COALESCE(quantity, 0)").Where("product_id = ? AND location_id = ?", item.ProductID, item.LocationID).Take(&qty).Error; err != nil {
			if !errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, err
			}
			qty = 0
		}
		beforeValue := roundMoney(float64(qty) * item.UnitPrice)
		afterQty := qty
		if item.Quantity > 0 {
			afterQty = qty + item.Quantity
		}
		afterValue := roundMoney(float64(afterQty) * item.UnitPrice)
		preview = append(preview, StockImpactPreview{
			ItemID:         item.ID,
			ProductID:      item.ProductID,
			ProductName:    item.ProductName,
			LocationID:     item.LocationID,
			LocationName:   item.LocationName,
			Quantity:       item.Quantity,
			BeforeQuantity: qty,
			AfterQuantity:  afterQty,
			BeforeValue:    beforeValue,
			AfterValue:     afterValue,
			ValueChange:    roundMoney(afterValue - beforeValue),
		})
	}
	return preview, nil
}

func (r PostgresRepository) ConfirmDraft(ctx context.Context, receiptID, actorID string) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		repo := PostgresRepository{db: tx}
		var receipt WarehouseReceipt
		if err := tx.Table("warehouse_receipts").Where("id = ?", receiptID).Take(&receipt).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrReceiptNotFound
			}
			return err
		}
		if receipt.Status != ReceiptStatusDraft {
			return ErrReceiptConfirmOnlyDraft
		}
		items, err := repo.listItems(ctx, receiptID)
		if err != nil {
			return err
		}
		if len(items) == 0 {
			return ErrReceiptItemsRequired
		}
		if strings.TrimSpace(receipt.PurchaseOrderID) != "" {
			poItems, err := repo.GetPurchaseOrderItems(ctx, receipt.PurchaseOrderID)
			if err != nil {
				return err
			}
			outstanding := map[string]int{}
			for _, poItem := range poItems {
				outstanding[poItem.ProductID] = poItem.Quantity - poItem.ReceivedQuantity
			}
			receivedByProduct := map[string]int{}
			for _, item := range items {
				receivedByProduct[item.ProductID] += item.Quantity
			}
			for productID, qty := range receivedByProduct {
				if qty > outstanding[productID] {
					return ErrReceiptPOQuantityExceeded
				}
			}
			for _, item := range items {
				if err := tx.Table("purchase_order_items").Where("purchase_order_id = ? AND product_id = ?", receipt.PurchaseOrderID, item.ProductID).Update("received_quantity", gorm.Expr("received_quantity + ?", item.Quantity)).Error; err != nil {
					return err
				}
			}
			var pendingCount int64
			if err := tx.Table("purchase_order_items").Where("purchase_order_id = ? AND received_quantity < quantity", receipt.PurchaseOrderID).Count(&pendingCount).Error; err != nil {
				return err
			}
			newStatus := "partial"
			if pendingCount == 0 {
				newStatus = "completed"
			}
			if err := tx.Table("purchase_orders").Where("id = ?", receipt.PurchaseOrderID).Updates(map[string]any{"status": newStatus, "received_at": time.Now().UTC(), "updated_at": gorm.Expr("NOW()")}).Error; err != nil {
				return err
			}
		}
		now := time.Now().UTC()
		for _, item := range items {
			if err := tx.Exec(`
				INSERT INTO warehouse_inventory (id, store_id, warehouse_id, product_id, quantity, transferred_at, created_at, updated_at)
				VALUES (?, ?, ?, ?, ?, NOW(), NOW(), NOW())
				ON CONFLICT (warehouse_id, product_id)
				DO UPDATE SET quantity = warehouse_inventory.quantity + EXCLUDED.quantity, updated_at = NOW()
			`, newID(), receipt.StoreID, receipt.WarehouseID, item.ProductID, item.Quantity).Error; err != nil {
				return err
			}
			if err := tx.Exec(`
				INSERT INTO stocks (id, store_id, product_id, location_id, quantity, created_at, updated_at)
				VALUES (?, ?, ?, ?, ?, NOW(), NOW())
				ON CONFLICT (product_id, location_id)
				DO UPDATE SET quantity = stocks.quantity + EXCLUDED.quantity, updated_at = NOW()
			`, newID(), receipt.StoreID, item.ProductID, item.LocationID, item.Quantity).Error; err != nil {
				return err
			}
			locationID := item.LocationID
			referenceID := receipt.ID
			if err := tx.Table("stock_movements").Create(map[string]any{
				"id":              newStockMovementID(),
				"store_id":        receipt.StoreID,
				"product_id":      item.ProductID,
				"location_id":     locationID,
				"quantity_change": item.Quantity,
				"type":            "IN",
				"reference_id":    referenceID,
				"note":            fmt.Sprintf("warehouse receipt %s", receipt.DocumentNo),
				"created_by":      actorID,
				"created_at":      now,
				"updated_at":      now,
			}).Error; err != nil {
				return err
			}
		}
		if err := tx.Table("warehouse_receipts").Where("id = ?", receiptID).Updates(map[string]any{
			"status":       string(ReceiptStatusConfirmed),
			"confirmed_by": actorID,
			"confirmed_at": now,
			"updated_at":   now,
		}).Error; err != nil {
			return err
		}
		return repo.AppendAudit(ctx, WarehouseReceiptAudit{ID: newAuditID(), ReceiptID: receiptID, Action: "confirm", Description: "receipt confirmed", ActorID: actorID, CreatedAt: now})
	})
}

func (r PostgresRepository) CancelDraft(ctx context.Context, receiptID, actorID string) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var status string
		if err := tx.Table("warehouse_receipts").Select("status").Where("id = ?", receiptID).Take(&status).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrReceiptNotFound
			}
			return err
		}
		if status != string(ReceiptStatusDraft) {
			return ErrReceiptCancelOnlyDraft
		}
		now := time.Now().UTC()
		if err := tx.Table("warehouse_receipts").Where("id = ?", receiptID).Updates(map[string]any{
			"status":       string(ReceiptStatusCancelled),
			"cancelled_by": actorID,
			"cancelled_at": now,
			"updated_at":   now,
		}).Error; err != nil {
			return err
		}
		return PostgresRepository{db: tx}.AppendAudit(ctx, WarehouseReceiptAudit{ID: newAuditID(), ReceiptID: receiptID, Action: "cancel", Description: "receipt cancelled", ActorID: actorID, CreatedAt: now})
	})
}

func (r PostgresRepository) listItems(ctx context.Context, receiptID string) ([]WarehouseReceiptItem, error) {
	var items []WarehouseReceiptItem
	err := r.db.WithContext(ctx).Table("warehouse_receipt_items").Where("receipt_id = ?", receiptID).Order("created_at ASC, id ASC").Find(&items).Error
	if err != nil {
		return nil, err
	}
	if items == nil {
		items = []WarehouseReceiptItem{}
	}
	return items, nil
}

func (r PostgresRepository) listAudits(ctx context.Context, receiptID string) ([]WarehouseReceiptAudit, error) {
	var audits []WarehouseReceiptAudit
	err := r.db.WithContext(ctx).Table("warehouse_receipt_audits a").Select("a.id, a.receipt_id, a.action, COALESCE(a.description, '') AS description, a.actor_id, a.created_at, COALESCE(u.full_name, '') AS actor_name").Joins("LEFT JOIN users u ON u.id = a.actor_id").Where("a.receipt_id = ?", receiptID).Order("a.created_at ASC").Scan(&audits).Error
	if err != nil {
		return nil, err
	}
	if audits == nil {
		audits = []WarehouseReceiptAudit{}
	}
	return audits, nil
}

func nilIfEmpty(value string) any {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return strings.TrimSpace(value)
}

func nilFloat(value *float64) any {
	if value == nil {
		return nil
	}
	return *value
}

func mapWriteError(err error) error {
	if err == nil {
		return nil
	}
	message := strings.ToLower(err.Error())
	if strings.Contains(message, "document_no") || strings.Contains(message, "warehouse_receipts_document_no_key") {
		return ErrReceiptDuplicateDocumentNo
	}
	if strings.Contains(message, "warehouse_receipt_items_receipt_id_product_id_location_id_key") {
		return ErrReceiptDuplicateItem
	}
	return err
}

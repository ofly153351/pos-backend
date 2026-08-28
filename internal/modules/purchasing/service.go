package purchasing

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"

	"pos-backend/internal/modules/auth"
	"pos-backend/internal/modules/warehouse_receipt"
)

type Service struct {
	repo    Repository
	db      *gorm.DB
	storage SupplierLogoStorage
}

func NewService(repo Repository, db *gorm.DB, storage SupplierLogoStorage) Service {
	return Service{repo: repo, db: db, storage: storage}
}

// ---- Suppliers ----

func (s Service) CreateSupplier(ctx context.Context, actor auth.Claims, storeID string, input CreateSupplierRequest) (Supplier, error) {
	if strings.TrimSpace(storeID) == "" {
		return Supplier{}, ErrSupplierStoreIDReq
	}
	if strings.TrimSpace(input.Name) == "" {
		return Supplier{}, ErrSupplierNameRequired
	}

	isActive := true
	if input.IsActive != nil {
		isActive = *input.IsActive
	}

	logoURL, err := s.storage.SaveSupplierLogo(input.LogoFile)
	if err != nil {
		return Supplier{}, fmt.Errorf("save supplier logo: %w", err)
	}

	item := Supplier{
		ID:                newSupplierID(),
		StoreID:           storeID,
		Name:              strings.TrimSpace(input.Name),
		Phone:             strings.TrimSpace(input.Phone),
		Address:           strings.TrimSpace(input.Address),
		TaxID:             strings.TrimSpace(input.TaxID),
		ContactPerson:     strings.TrimSpace(input.ContactPerson),
		Note:              strings.TrimSpace(input.Note),
		IsActive:          isActive,
		Email:             strings.TrimSpace(input.Email),
		LineID:            strings.TrimSpace(input.LineID),
		PaymentMethod:     strings.TrimSpace(input.PaymentMethod),
		PromptpayNumber:   strings.TrimSpace(input.PromptpayNumber),
		BankName:          strings.TrimSpace(input.BankName),
		BankAccountNumber: strings.TrimSpace(input.BankAccountNumber),
		BankAccountName:   strings.TrimSpace(input.BankAccountName),
		CreditDays:        input.CreditDays,
		LogoURL:           logoURL,
		CreatedAt:         time.Now().UTC(),
	}
	result, err := s.repo.CreateSupplier(ctx, item)
	if err != nil && logoURL != "" {
		_ = s.storage.DeleteSupplierLogo(logoURL)
	}
	return result, err
}

func (s Service) ListSuppliers(ctx context.Context, actor auth.Claims, storeID string) ([]Supplier, error) {
	return s.repo.ListSuppliers(ctx, storeID)
}

func (s Service) GetSupplier(ctx context.Context, actor auth.Claims, storeID, supplierID string) (Supplier, error) {
	return s.repo.GetSupplier(ctx, storeID, supplierID)
}

func (s Service) UpdateSupplier(ctx context.Context, actor auth.Claims, storeID, supplierID string, input UpdateSupplierRequest) (Supplier, error) {

	existing, err := s.repo.GetSupplier(ctx, storeID, supplierID)
	if err != nil {
		return Supplier{}, err
	}

	if input.Name != nil {
		existing.Name = strings.TrimSpace(*input.Name)
	}
	if strings.TrimSpace(existing.Name) == "" {
		return Supplier{}, ErrSupplierNameRequired
	}
	if input.Phone != nil {
		existing.Phone = strings.TrimSpace(*input.Phone)
	}
	if input.Address != nil {
		existing.Address = strings.TrimSpace(*input.Address)
	}
	if input.TaxID != nil {
		existing.TaxID = strings.TrimSpace(*input.TaxID)
	}
	if input.ContactPerson != nil {
		existing.ContactPerson = strings.TrimSpace(*input.ContactPerson)
	}
	if input.Note != nil {
		existing.Note = strings.TrimSpace(*input.Note)
	}
	if input.IsActive != nil {
		existing.IsActive = *input.IsActive
	}
	if input.Email != nil {
		existing.Email = strings.TrimSpace(*input.Email)
	}
	if input.LineID != nil {
		existing.LineID = strings.TrimSpace(*input.LineID)
	}
	if input.PaymentMethod != nil {
		existing.PaymentMethod = strings.TrimSpace(*input.PaymentMethod)
	}
	if input.PromptpayNumber != nil {
		existing.PromptpayNumber = strings.TrimSpace(*input.PromptpayNumber)
	}
	if input.BankName != nil {
		existing.BankName = strings.TrimSpace(*input.BankName)
	}
	if input.BankAccountNumber != nil {
		existing.BankAccountNumber = strings.TrimSpace(*input.BankAccountNumber)
	}
	if input.BankAccountName != nil {
		existing.BankAccountName = strings.TrimSpace(*input.BankAccountName)
	}
	if input.CreditDays != nil {
		existing.CreditDays = *input.CreditDays
	}

	oldLogoURL := existing.LogoURL
	if input.RemoveLogo {
		existing.LogoURL = ""
	} else if input.LogoFile != nil {
		newLogoURL, err := s.storage.SaveSupplierLogo(input.LogoFile)
		if err != nil {
			return Supplier{}, fmt.Errorf("save supplier logo: %w", err)
		}
		existing.LogoURL = newLogoURL
	}

	existing.UpdatedAt = time.Now().UTC()
	result, err := s.repo.UpdateSupplier(ctx, existing)
	if err != nil {
		if input.LogoFile != nil && existing.LogoURL != oldLogoURL {
			_ = s.storage.DeleteSupplierLogo(existing.LogoURL)
		}
		return Supplier{}, err
	}
	if oldLogoURL != "" && existing.LogoURL != oldLogoURL {
		_ = s.storage.DeleteSupplierLogo(oldLogoURL)
	}
	return result, nil
}

func (s Service) DeleteSupplier(ctx context.Context, actor auth.Claims, storeID, supplierID string) error {
	return s.repo.DeleteSupplier(ctx, storeID, supplierID)
}

// ---- Purchase Orders ----

func (s Service) CreatePO(ctx context.Context, actor auth.Claims, storeID string, input CreatePORequest) (PurchaseOrder, error) {
	if strings.TrimSpace(storeID) == "" {
		return PurchaseOrder{}, ErrPOStoreIDRequired
	}

	if len(input.Items) == 0 {
		return PurchaseOrder{}, ErrPOItemsRequired
	}

	// Validate items once, before the retry loop (idempotent).
	for _, item := range input.Items {
		if item.Quantity <= 0 {
			return PurchaseOrder{}, ErrPOInvalidQuantity
		}
		if item.UnitCost < 0 {
			return PurchaseOrder{}, ErrPOInvalidUnitCost
		}
		// Verify product exists in store
		if _, err := s.repo.GetProduct(ctx, storeID, item.ProductID); err != nil {
			return PurchaseOrder{}, err
		}
	}

	now := time.Now().UTC()
	datePrefix := now.Format("20060102")

	// Total is independent of the order number; compute it once.
	var totalCost float64
	for _, item := range input.Items {
		totalCost += float64(item.Quantity) * item.UnitCost
	}

	// The order number is derived from a COUNT (GetPODailySequence) that is not
	// atomic with the insert, so two concurrent creates for the same store/day can
	// race to the same PO-<date>-NNNNN. Generating the sequence INSIDE the insert
	// tx narrows the window; the UNIQUE (store_id, order_number) index (migration
	// 030) is the correctness backstop — it rejects the loser with SQLSTATE 23505,
	// and we recompute the sequence and retry so the race stays invisible to the
	// caller. Only after exhausting the attempts do we surface a conflict.
	const maxOrderNumberAttempts = 5
	var createdID string
	for attempt := 0; attempt < maxOrderNumberAttempts; attempt++ {
		txErr := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
			txRepo := NewPostgresRepository(tx)

			seq, gerr := txRepo.GetPODailySequence(ctx, storeID, datePrefix)
			if gerr != nil {
				return ErrPOOrderNumberGenerate
			}

			po := PurchaseOrder{
				ID:          newID(),
				StoreID:     storeID,
				SupplierID:  strings.TrimSpace(input.SupplierID),
				OrderNumber: fmt.Sprintf("PO-%s-%05d", datePrefix, seq+1),
				Status:      POStatusPending,
				Notes:       strings.TrimSpace(input.Notes),
				TotalCost:   totalCost,
				CreatedBy:   actor.UserID,
				CreatedAt:   now,
			}

			created, cerr := txRepo.CreatePO(ctx, po)
			if cerr != nil {
				return cerr // unique-violation bubbles up here on a collision
			}
			createdID = created.ID

			for _, item := range input.Items {
				itemLineTotal := float64(item.Quantity) * item.UnitCost
				poItem := PurchaseOrderItem{
					ID:              newPOItemID(),
					PurchaseOrderID: created.ID,
					ProductID:       item.ProductID,
					Quantity:        item.Quantity,
					UnitCost:        item.UnitCost,
					LineTotal:       itemLineTotal,
					CreatedAt:       now,
				}
				if ierr := txRepo.CreatePOItem(ctx, poItem); ierr != nil {
					return ierr
				}
			}
			return nil
		})

		if txErr == nil {
			break
		}
		if isUniqueViolation(txErr) {
			if attempt < maxOrderNumberAttempts-1 {
				continue // collision — recompute the sequence and retry
			}
			return PurchaseOrder{}, ErrPOOrderNumberConflict
		}
		return PurchaseOrder{}, txErr
	}

	// Reload with relations
	return s.repo.GetPO(ctx, storeID, createdID)
}

func (s Service) ListPOs(ctx context.Context, actor auth.Claims, storeID string) ([]PurchaseOrder, error) {
	return s.repo.ListPOs(ctx, storeID)
}

func (s Service) GetPO(ctx context.Context, actor auth.Claims, storeID, poID string) (PurchaseOrder, error) {
	return s.repo.GetPO(ctx, storeID, poID)
}

func (s Service) UpdatePO(ctx context.Context, actor auth.Claims, storeID, poID string, input UpdatePORequest) (PurchaseOrder, error) {

	existing, err := s.repo.GetPO(ctx, storeID, poID)
	if err != nil {
		return PurchaseOrder{}, err
	}

	if existing.Status == POStatusCompleted {
		return PurchaseOrder{}, ErrPOAlreadyCompleted
	}
	if existing.Status == POStatusCancelled {
		return PurchaseOrder{}, ErrPOAlreadyCancelled
	}

	if input.SupplierID != nil {
		existing.SupplierID = *input.SupplierID
	}
	if input.Notes != nil {
		existing.Notes = *input.Notes
	}
	if input.Status != nil {
		status := PurchaseOrderStatus(*input.Status)
		switch status {
		case POStatusPending, POStatusPartial, POStatusCompleted, POStatusCancelled:
			existing.Status = status
		default:
			return PurchaseOrder{}, ErrPOInvalidStatus
		}
	}

	existing.UpdatedAt = time.Now().UTC()
	if err := s.repo.UpdatePO(ctx, existing); err != nil {
		return PurchaseOrder{}, err
	}
	return s.repo.GetPO(ctx, storeID, poID)
}

func (s Service) ReceiveStock(ctx context.Context, actor auth.Claims, storeID, poID string, input ReceivePORequest) (PurchaseOrder, error) {

	existing, err := s.repo.GetPO(ctx, storeID, poID)
	if err != nil {
		return PurchaseOrder{}, err
	}

	if existing.Status == POStatusCompleted {
		return PurchaseOrder{}, ErrPOAlreadyCompleted
	}
	if existing.Status == POStatusCancelled {
		return PurchaseOrder{}, ErrPOAlreadyCancelled
	}

	// Build map of received quantities
	receiveMap := make(map[string]int)
	for _, item := range input.Items {
		receiveMap[item.ProductID] = item.Quantity
	}

	now := time.Now().UTC()

	// All received lines, their stock/cost updates, the IN movements, and the PO
	// status update run in ONE transaction: a failure on any line rolls back the
	// entire receive, so stock, the PO, and the movement ledger never diverge.
	txErr := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		txRepo := NewPostgresRepository(tx)
		allCompleted := true
		anyReceived := false

		for i := range existing.Items {
			item := &existing.Items[i]
			reqQty, ok := receiveMap[item.ProductID]
			if !ok {
				// Not in receive request, skip
				if item.ReceivedQuantity < item.Quantity {
					allCompleted = false
				}
				continue
			}

			if reqQty < 0 {
				return ErrPOReceiveInvalidQty
			}

			newReceived := item.ReceivedQuantity + reqQty
			if newReceived > item.Quantity {
				return ErrPOReceiveInvalidQty
			}

			newLineTotal := float64(newReceived) * item.UnitCost
			if err := txRepo.UpdatePOItemReceived(ctx, item.ID, newReceived, newLineTotal); err != nil {
				return err
			}
			// Resolve the destination from the product's AUTHORITATIVE default location
			// (products.default_location_id) via the shared warehouse-receipt resolver —
			// the same source of truth and validation as canonical Goods Receiving. This
			// runs inside the transaction immediately before the stock mutation; any
			// invalid default rolls back the entire receive.
			loc, resolveErr := warehouse_receipt.ResolveValidReceivingLocation(ctx, tx, storeID, input.WarehouseID, item.ProductID)
			if resolveErr != nil {
				return translateReceivingLocationError(resolveErr)
			}
			if err := txRepo.UpdateProductStockAndCost(ctx, storeID, item.ProductID, loc.ID, loc.WarehouseID, reqQty, item.UnitCost, actor.UserID, poID); err != nil {
				return err
			}

			item.ReceivedQuantity = newReceived
			anyReceived = true

			if newReceived < item.Quantity {
				allCompleted = false
			}
		}

		if !anyReceived {
			return ErrPOReceiveInvalidQty
		}

		newStatus := POStatusPartial
		if allCompleted {
			newStatus = POStatusCompleted
		}
		ts := &timeSetter{Time: now, Valid: true}
		return txRepo.UpdatePOStatus(ctx, poID, newStatus, ts, actor.UserID)
	})
	if txErr != nil {
		return PurchaseOrder{}, txErr
	}

	return s.repo.GetPO(ctx, storeID, poID)
}

// translateReceivingLocationError maps the shared warehouse-receipt resolver errors
// onto purchasing-domain errors so Quick Receive returns clear HTTP 400 responses
// without leaking the warehouse-receipt error vocabulary.
func translateReceivingLocationError(err error) error {
	switch {
	case errors.Is(err, warehouse_receipt.ErrReceiptItemLocationMissing):
		return ErrPOReceiveNoDefaultLocation
	case errors.Is(err, warehouse_receipt.ErrReceiptLocationNotFound):
		return ErrPOReceiveLocationNotFound
	case errors.Is(err, warehouse_receipt.ErrReceiptLocationInactive):
		return ErrPOReceiveLocationInactive
	case errors.Is(err, warehouse_receipt.ErrReceiptLocationSalePoint):
		return ErrPOReceiveLocationSalePoint
	case errors.Is(err, warehouse_receipt.ErrReceiptLocationWrongWarehouse):
		return ErrPOReceiveLocationWrongWarehouse
	case errors.Is(err, warehouse_receipt.ErrReceiptProductNotFound):
		return ErrPOProductNotFound
	default:
		return err
	}
}

func (s Service) CancelPO(ctx context.Context, actor auth.Claims, storeID, poID string) (PurchaseOrder, error) {

	existing, err := s.repo.GetPO(ctx, storeID, poID)
	if err != nil {
		return PurchaseOrder{}, err
	}

	if existing.Status == POStatusCompleted {
		return PurchaseOrder{}, ErrPOAlreadyCompleted
	}
	if existing.Status == POStatusCancelled {
		return PurchaseOrder{}, ErrPOAlreadyCancelled
	}

	if err := s.repo.UpdatePOStatus(ctx, poID, POStatusCancelled, nil, ""); err != nil {
		return PurchaseOrder{}, err
	}

	return s.repo.GetPO(ctx, storeID, poID)
}

// ---- Supplier Products ----

func (s Service) ListSupplierProducts(ctx context.Context, actor auth.Claims, storeID, supplierID string) ([]SupplierProductResponse, error) {
	// Verify supplier belongs to store
	if _, err := s.repo.GetSupplier(ctx, storeID, supplierID); err != nil {
		return nil, err
	}
	return s.repo.ListSupplierProducts(ctx, storeID, supplierID)
}

func (s Service) AddSupplierProduct(ctx context.Context, actor auth.Claims, storeID, supplierID string, input AddSupplierProductRequest) (SupplierProductResponse, error) {

	if strings.TrimSpace(input.ProductID) == "" {
		return SupplierProductResponse{}, ErrSupplierProductRequired
	}

	// Verify supplier belongs to store
	if _, err := s.repo.GetSupplier(ctx, storeID, supplierID); err != nil {
		return SupplierProductResponse{}, err
	}

	// Verify product exists in store
	if _, err := s.repo.GetProduct(ctx, storeID, input.ProductID); err != nil {
		return SupplierProductResponse{}, err
	}

	// Check duplicate
	if existing, err := s.repo.GetSupplierProduct(ctx, supplierID, input.ProductID); err == nil && existing.ID != "" {
		return SupplierProductResponse{}, ErrSupplierProductExists
	}

	now := time.Now().UTC()
	sp := SupplierProduct{
		ID:            newSupplierProductID(),
		SupplierID:    supplierID,
		ProductID:     input.ProductID,
		SupplierSKU:   strings.TrimSpace(input.SupplierSKU),
		SupplierPrice: input.SupplierPrice,
		CreatedAt:     now,
	}

	created, err := s.repo.AddSupplierProduct(ctx, sp)
	if err != nil {
		return SupplierProductResponse{}, err
	}

	// Fetch the joined response
	items, err := s.repo.ListSupplierProducts(ctx, storeID, supplierID)
	if err != nil {
		return SupplierProductResponse{}, err
	}
	for _, item := range items {
		if item.ID == created.ID {
			return item, nil
		}
	}
	// Fallback: build minimal response
	return SupplierProductResponse{
		ID:            created.ID,
		SupplierID:    created.SupplierID,
		ProductID:     created.ProductID,
		SupplierSKU:   created.SupplierSKU,
		SupplierPrice: created.SupplierPrice,
		CreatedAt:     created.CreatedAt.Format(time.RFC3339),
	}, nil
}

func (s Service) UpdateSupplierProduct(ctx context.Context, actor auth.Claims, storeID, supplierID, productID string, input UpdateSupplierProductRequest) (SupplierProductResponse, error) {

	// Verify supplier belongs to store
	if _, err := s.repo.GetSupplier(ctx, storeID, supplierID); err != nil {
		return SupplierProductResponse{}, err
	}

	// Verify the link exists
	if _, err := s.repo.GetSupplierProduct(ctx, supplierID, productID); err != nil {
		return SupplierProductResponse{}, err
	}

	sp := SupplierProduct{
		SupplierID:    supplierID,
		ProductID:     productID,
		SupplierSKU:   strings.TrimSpace(input.SupplierSKU),
		SupplierPrice: input.SupplierPrice,
		UpdatedAt:     time.Now().UTC(),
	}

	if _, err := s.repo.UpdateSupplierProduct(ctx, sp); err != nil {
		return SupplierProductResponse{}, err
	}

	// Fetch the updated response
	items, err := s.repo.ListSupplierProducts(ctx, storeID, supplierID)
	if err != nil {
		return SupplierProductResponse{}, err
	}
	for _, item := range items {
		if item.ProductID == productID {
			return item, nil
		}
	}

	return SupplierProductResponse{}, ErrSupplierProductNotFound
}

func (s Service) RemoveSupplierProduct(ctx context.Context, actor auth.Claims, storeID, supplierID, productID string) error {
	// Verify supplier belongs to store
	if _, err := s.repo.GetSupplier(ctx, storeID, supplierID); err != nil {
		return err
	}
	return s.repo.RemoveSupplierProduct(ctx, supplierID, productID)
}

func (s Service) CreateSupplierProductAndLink(ctx context.Context, actor auth.Claims, storeID, supplierID string, input CreateSupplierProductAndLinkRequest) (SupplierProductResponse, error) {

	if strings.TrimSpace(input.Name) == "" {
		return SupplierProductResponse{}, ErrSupplierProductNameReq
	}

	// Verify supplier belongs to store
	if _, err := s.repo.GetSupplier(ctx, storeID, supplierID); err != nil {
		return SupplierProductResponse{}, err
	}

	// Create product
	productID, err := s.repo.CreateProductForSupplier(ctx, storeID,
		strings.TrimSpace(input.Name),
		strings.TrimSpace(input.SKU),
		strings.TrimSpace(input.Barcode),
		strings.TrimSpace(input.ProductTypeID),
		strings.TrimSpace(input.ProductUnitID),
		input.BasePrice,
	)
	if err != nil {
		return SupplierProductResponse{}, err
	}

	// Create supplier product link
	now := time.Now().UTC()
	sp := SupplierProduct{
		ID:            newSupplierProductID(),
		SupplierID:    supplierID,
		ProductID:     productID,
		SupplierSKU:   strings.TrimSpace(input.SupplierSKU),
		SupplierPrice: input.SupplierPrice,
		CreatedAt:     now,
	}

	if _, err := s.repo.AddSupplierProduct(ctx, sp); err != nil {
		return SupplierProductResponse{}, err
	}

	// Fetch the joined response
	items, err := s.repo.ListSupplierProducts(ctx, storeID, supplierID)
	if err != nil {
		return SupplierProductResponse{}, err
	}
	for _, item := range items {
		if item.ProductID == productID {
			return item, nil
		}
	}

	return SupplierProductResponse{
		ID:            sp.ID,
		SupplierID:    sp.SupplierID,
		ProductID:     sp.ProductID,
		ProductName:   input.Name,
		SupplierSKU:   sp.SupplierSKU,
		SupplierPrice: sp.SupplierPrice,
		CreatedAt:     sp.CreatedAt.Format(time.RFC3339),
	}, nil
}

// ensureAccess gates operate-level actions (list/get + receive stock): owner,
// manager, cashier, and warehouse members all pass.

// ensureManageAccess gates management actions (create/edit/cancel/delete of
// suppliers, supplier-product links, and purchase orders): only owner/manager
// (or platform_admin) pass. `forbidden` is the domain-specific 403 error to
// return so supplier vs PO call sites surface the right message.

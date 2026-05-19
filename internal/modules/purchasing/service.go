package purchasing

import (
	"context"
	"fmt"
	"strings"
	"time"

	"pos-backend/internal/modules/auth"
)

type Service struct {
	repo Repository
}

func NewService(repo Repository) Service {
	return Service{repo: repo}
}

// ---- Suppliers ----

func (s Service) CreateSupplier(ctx context.Context, actor auth.Claims, storeID string, input CreateSupplierRequest) (Supplier, error) {
	if strings.TrimSpace(storeID) == "" {
		return Supplier{}, ErrSupplierStoreIDReq
	}
	if err := s.ensureAccess(ctx, actor, storeID); err != nil {
		return Supplier{}, err
	}
	if strings.TrimSpace(input.Name) == "" {
		return Supplier{}, ErrSupplierNameRequired
	}

	isActive := true
	if input.IsActive != nil {
		isActive = *input.IsActive
	}

	item := Supplier{
		ID:            newID(),
		StoreID:       storeID,
		Name:          strings.TrimSpace(input.Name),
		Phone:         strings.TrimSpace(input.Phone),
		Address:       strings.TrimSpace(input.Address),
		TaxID:         strings.TrimSpace(input.TaxID),
		ContactPerson: strings.TrimSpace(input.ContactPerson),
		Note:          strings.TrimSpace(input.Note),
		IsActive:      isActive,
		CreatedAt:     time.Now().UTC(),
	}
	return s.repo.CreateSupplier(ctx, item)
}

func (s Service) ListSuppliers(ctx context.Context, actor auth.Claims, storeID string) ([]Supplier, error) {
	if err := s.ensureAccess(ctx, actor, storeID); err != nil {
		return nil, err
	}
	return s.repo.ListSuppliers(ctx, storeID)
}

func (s Service) GetSupplier(ctx context.Context, actor auth.Claims, storeID, supplierID string) (Supplier, error) {
	if err := s.ensureAccess(ctx, actor, storeID); err != nil {
		return Supplier{}, err
	}
	return s.repo.GetSupplier(ctx, storeID, supplierID)
}

func (s Service) UpdateSupplier(ctx context.Context, actor auth.Claims, storeID, supplierID string, input UpdateSupplierRequest) (Supplier, error) {
	if err := s.ensureAccess(ctx, actor, storeID); err != nil {
		return Supplier{}, err
	}

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

	existing.UpdatedAt = time.Now().UTC()
	return s.repo.UpdateSupplier(ctx, existing)
}

func (s Service) DeleteSupplier(ctx context.Context, actor auth.Claims, storeID, supplierID string) error {
	if err := s.ensureAccess(ctx, actor, storeID); err != nil {
		return err
	}
	return s.repo.DeleteSupplier(ctx, storeID, supplierID)
}

// ---- Purchase Orders ----

func (s Service) CreatePO(ctx context.Context, actor auth.Claims, storeID string, input CreatePORequest) (PurchaseOrder, error) {
	if strings.TrimSpace(storeID) == "" {
		return PurchaseOrder{}, ErrPOStoreIDRequired
	}
	if err := s.ensureAccess(ctx, actor, storeID); err != nil {
		return PurchaseOrder{}, err
	}

	if len(input.Items) == 0 {
		return PurchaseOrder{}, ErrPOItemsRequired
	}

	// Validate items
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

	// Generate order number
	now := time.Now().UTC()
	datePrefix := now.Format("20060102")
	seq, err := s.repo.GetPODailySequence(ctx, storeID, datePrefix)
	if err != nil {
		return PurchaseOrder{}, ErrPOOrderNumberGenerate
	}
	orderNumber := fmt.Sprintf("PO-%s-%05d", datePrefix, seq+1)

	// Calculate total
	var totalCost float64
	for _, item := range input.Items {
		totalCost += float64(item.Quantity) * item.UnitCost
	}

	po := PurchaseOrder{
		ID:          newID(),
		StoreID:     storeID,
		SupplierID:  strings.TrimSpace(input.SupplierID),
		OrderNumber: orderNumber,
		Status:      POStatusPending,
		Notes:       strings.TrimSpace(input.Notes),
		TotalCost:   totalCost,
		CreatedAt:   now,
	}

	created, err := s.repo.CreatePO(ctx, po)
	if err != nil {
		return PurchaseOrder{}, err
	}

	// Create items
	for _, item := range input.Items {
		itemLineTotal := float64(item.Quantity) * item.UnitCost
		poItem := PurchaseOrderItem{
			ID:              newID(),
			PurchaseOrderID: created.ID,
			ProductID:       item.ProductID,
			Quantity:        item.Quantity,
			UnitCost:        item.UnitCost,
			LineTotal:       itemLineTotal,
			CreatedAt:       now,
		}
		if err := s.repo.CreatePOItem(ctx, poItem); err != nil {
			return PurchaseOrder{}, err
		}
	}

	// Reload with relations
	return s.repo.GetPO(ctx, storeID, created.ID)
}

func (s Service) ListPOs(ctx context.Context, actor auth.Claims, storeID string) ([]PurchaseOrder, error) {
	if err := s.ensureAccess(ctx, actor, storeID); err != nil {
		return nil, err
	}
	return s.repo.ListPOs(ctx, storeID)
}

func (s Service) GetPO(ctx context.Context, actor auth.Claims, storeID, poID string) (PurchaseOrder, error) {
	if err := s.ensureAccess(ctx, actor, storeID); err != nil {
		return PurchaseOrder{}, err
	}
	return s.repo.GetPO(ctx, storeID, poID)
}

func (s Service) UpdatePO(ctx context.Context, actor auth.Claims, storeID, poID string, input UpdatePORequest) (PurchaseOrder, error) {
	if err := s.ensureAccess(ctx, actor, storeID); err != nil {
		return PurchaseOrder{}, err
	}

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
	if err := s.ensureAccess(ctx, actor, storeID); err != nil {
		return PurchaseOrder{}, err
	}

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

	allCompleted := true
	anyReceived := false
	now := time.Now().UTC()

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
			return PurchaseOrder{}, ErrPOReceiveInvalidQty
		}

		newReceived := item.ReceivedQuantity + reqQty
		if newReceived > item.Quantity {
			return PurchaseOrder{}, ErrPOReceiveInvalidQty
		}

		// Update PO item received quantity
		newLineTotal := float64(newReceived) * item.UnitCost
		if err := s.repo.UpdatePOItemReceived(ctx, item.ID, newReceived, newLineTotal); err != nil {
			return PurchaseOrder{}, err
		}

		// Update product stock and cost price
		if err := s.repo.UpdateProductStockAndCost(ctx, item.ProductID, reqQty, item.UnitCost); err != nil {
			return PurchaseOrder{}, err
		}

		item.ReceivedQuantity = newReceived
		anyReceived = true

		if newReceived < item.Quantity {
			allCompleted = false
		}
	}

	if !anyReceived {
		return PurchaseOrder{}, ErrPOReceiveInvalidQty
	}

	// Update PO status
	newStatus := POStatusPartial
	if allCompleted {
		newStatus = POStatusCompleted
	}

	ts := &timeSetter{Time: now, Valid: true}
	if err := s.repo.UpdatePOStatus(ctx, poID, newStatus, ts); err != nil {
		return PurchaseOrder{}, err
	}

	return s.repo.GetPO(ctx, storeID, poID)
}

func (s Service) CancelPO(ctx context.Context, actor auth.Claims, storeID, poID string) (PurchaseOrder, error) {
	if err := s.ensureAccess(ctx, actor, storeID); err != nil {
		return PurchaseOrder{}, err
	}

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

	if err := s.repo.UpdatePOStatus(ctx, poID, POStatusCancelled, nil); err != nil {
		return PurchaseOrder{}, err
	}

	return s.repo.GetPO(ctx, storeID, poID)
}

// ---- Supplier Products ----

func (s Service) ListSupplierProducts(ctx context.Context, actor auth.Claims, storeID, supplierID string) ([]SupplierProductResponse, error) {
	if err := s.ensureAccess(ctx, actor, storeID); err != nil {
		return nil, err
	}
	// Verify supplier belongs to store
	if _, err := s.repo.GetSupplier(ctx, storeID, supplierID); err != nil {
		return nil, err
	}
	return s.repo.ListSupplierProducts(ctx, storeID, supplierID)
}

func (s Service) AddSupplierProduct(ctx context.Context, actor auth.Claims, storeID, supplierID string, input AddSupplierProductRequest) (SupplierProductResponse, error) {
	if err := s.ensureAccess(ctx, actor, storeID); err != nil {
		return SupplierProductResponse{}, err
	}

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
		ID:            newID(),
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
	if err := s.ensureAccess(ctx, actor, storeID); err != nil {
		return SupplierProductResponse{}, err
	}

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
	if err := s.ensureAccess(ctx, actor, storeID); err != nil {
		return err
	}
	// Verify supplier belongs to store
	if _, err := s.repo.GetSupplier(ctx, storeID, supplierID); err != nil {
		return err
	}
	return s.repo.RemoveSupplierProduct(ctx, supplierID, productID)
}

func (s Service) CreateSupplierProductAndLink(ctx context.Context, actor auth.Claims, storeID, supplierID string, input CreateSupplierProductAndLinkRequest) (SupplierProductResponse, error) {
	if err := s.ensureAccess(ctx, actor, storeID); err != nil {
		return SupplierProductResponse{}, err
	}

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
		ID:            newID(),
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

func (s Service) ensureAccess(ctx context.Context, actor auth.Claims, storeID string) error {
	ok, err := s.repo.UserCanOperateStore(ctx, storeID, actor.UserID, actor.Role)
	if err != nil {
		return err
	}
	if !ok {
		return ErrSupplierForbidden
	}
	return nil
}

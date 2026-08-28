package warehouse_receipt

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"math"
	"net"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"time"

	"gorm.io/gorm"

	"pos-backend/internal/modules/auth"
)

type Service struct {
	repo    Repository
	db      *gorm.DB
	storage AttachmentStorage
}

func NewService(repo Repository, db *gorm.DB, storage AttachmentStorage) Service {
	return Service{repo: repo, db: db, storage: storage}
}

func (s Service) List(ctx context.Context, actor auth.Claims, query ListReceiptsQuery) (ReceiptListResult, error) {
	storeID := strings.TrimSpace(query.StoreID)
	if storeID == "" {
		resolvedStoreID, err := s.repo.FindPrimaryStoreIDByUserID(ctx, actor.UserID)
		if err != nil {
			return ReceiptListResult{}, err
		}
		storeID = strings.TrimSpace(resolvedStoreID)
	}
	if storeID == "" {
		return ReceiptListResult{}, ErrReceiptStoreContextRequired
	}
	if err := s.ensureOperateAccess(ctx, actor, storeID); err != nil {
		return ReceiptListResult{}, err
	}
	if query.Page < 1 || query.Limit < 1 {
		return ReceiptListResult{}, ErrInvalidPagination
	}

	items, total, err := s.repo.ListByStore(ctx, storeID, query.Status, query.Page, query.Limit)
	if err != nil {
		return ReceiptListResult{}, err
	}
	totalPages := int(math.Ceil(float64(total) / float64(query.Limit)))
	if totalPages == 0 {
		totalPages = 1
	}
	return ReceiptListResult{
		Items:      items,
		Page:       query.Page,
		Limit:      query.Limit,
		Total:      total,
		TotalPages: totalPages,
		HasNext:    query.Page < totalPages,
		HasPrev:    query.Page > 1,
	}, nil
}

func (s Service) Create(ctx context.Context, actor auth.Claims, input CreateReceiptRequest) (WarehouseReceipt, error) {
	storeID := strings.TrimSpace(input.StoreID)
	if storeID == "" {
		return WarehouseReceipt{}, ErrReceiptStoreIDRequired
	}
	if err := s.ensureOperateAccess(ctx, actor, storeID); err != nil {
		return WarehouseReceipt{}, err
	}
	receipt, err := s.buildHeader(ctx, s.repo, WarehouseReceipt{}, actor.UserID, storeID, input.WarehouseID, input.SupplierID, input.PurchaseOrderID, input.DocumentNo, input.ReceivedAt, input.ReferenceNo, input.Note, input.VATIncluded, input.VATPercent, true)
	if err != nil {
		return WarehouseReceipt{}, err
	}
	receipt.ID = newID()
	receipt.Status = ReceiptStatusDraft
	receipt.CreatedBy = actor.UserID
	receipt.CreatedAt = time.Now().UTC()
	receipt.UpdatedAt = receipt.CreatedAt
	created, err := s.repo.Create(ctx, receipt)
	if err != nil {
		return WarehouseReceipt{}, err
	}
	if err := s.repo.AppendAudit(ctx, WarehouseReceiptAudit{ID: newAuditID(), ReceiptID: created.ID, Action: "create", Description: "receipt created", ActorID: actor.UserID, CreatedAt: time.Now().UTC()}); err != nil {
		return WarehouseReceipt{}, err
	}
	return s.repo.GetByID(ctx, created.ID)
}

func (s Service) GetByID(ctx context.Context, actor auth.Claims, receiptID string) (WarehouseReceipt, error) {
	receipt, err := s.repo.GetByID(ctx, receiptID)
	if err != nil {
		return WarehouseReceipt{}, err
	}
	if err := s.ensureOperateAccess(ctx, actor, receipt.StoreID); err != nil {
		return WarehouseReceipt{}, err
	}
	return receipt, nil
}

func (s Service) Update(ctx context.Context, actor auth.Claims, receiptID string, input UpdateReceiptRequest) (WarehouseReceipt, error) {
	current, err := s.repo.GetByID(ctx, receiptID)
	if err != nil {
		return WarehouseReceipt{}, err
	}
	if err := s.ensureOperateAccess(ctx, actor, current.StoreID); err != nil {
		return WarehouseReceipt{}, err
	}
	if current.Status != ReceiptStatusDraft {
		return WarehouseReceipt{}, ErrReceiptImmutable
	}

	warehouseID := current.WarehouseID
	if input.WarehouseID != nil {
		warehouseID = strings.TrimSpace(*input.WarehouseID)
	}
	supplierID := current.SupplierID
	if input.SupplierID != nil {
		supplierID = strings.TrimSpace(*input.SupplierID)
	}
	purchaseOrderID := current.PurchaseOrderID
	if input.PurchaseOrderID != nil {
		purchaseOrderID = strings.TrimSpace(*input.PurchaseOrderID)
	}
	documentNo := current.DocumentNo
	if input.DocumentNo != nil {
		documentNo = strings.TrimSpace(*input.DocumentNo)
	}
	receivedAt := &current.ReceivedAt
	if input.ReceivedAt != nil {
		receivedAt = input.ReceivedAt
	}
	referenceNo := current.ReferenceNo
	if input.ReferenceNo != nil {
		referenceNo = *input.ReferenceNo
	}
	note := current.Note
	if input.Note != nil {
		note = *input.Note
	}
	vatIncluded := &current.VATIncluded
	if input.VATIncluded != nil {
		vatIncluded = input.VATIncluded
	}
	vatPercent := &current.VATPercent
	if input.VATPercent != nil {
		vatPercent = input.VATPercent
	}

	updated, err := s.buildHeader(ctx, s.repo, current, actor.UserID, current.StoreID, warehouseID, supplierID, purchaseOrderID, documentNo, receivedAt, referenceNo, note, vatIncluded, vatPercent, false)
	if err != nil {
		return WarehouseReceipt{}, err
	}
	updated.UpdatedAt = time.Now().UTC()
	result, err := s.repo.Update(ctx, updated)
	if err != nil {
		return WarehouseReceipt{}, err
	}
	if err := s.repo.AppendAudit(ctx, WarehouseReceiptAudit{ID: newAuditID(), ReceiptID: result.ID, Action: "update", Description: "receipt updated", ActorID: actor.UserID, CreatedAt: time.Now().UTC()}); err != nil {
		return WarehouseReceipt{}, err
	}
	return s.repo.GetByID(ctx, result.ID)
}

func (s Service) AddItems(ctx context.Context, actor auth.Claims, receiptID string, input AddReceiptItemsRequest) (WarehouseReceipt, error) {
	current, err := s.repo.GetByID(ctx, receiptID)
	if err != nil {
		return WarehouseReceipt{}, err
	}
	if err := s.ensureOperateAccess(ctx, actor, current.StoreID); err != nil {
		return WarehouseReceipt{}, err
	}
	if current.Status != ReceiptStatusDraft {
		return WarehouseReceipt{}, ErrReceiptImmutable
	}
	if len(input.Items) == 0 {
		return WarehouseReceipt{}, ErrReceiptItemsRequired
	}

	returnResult := WarehouseReceipt{}
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		txRepo := s.repo.WithTx(tx)
		items, err := s.materializeItems(ctx, txRepo, current, input.Items)
		if err != nil {
			return err
		}
		if input.ReplaceExisting {
			if err := txRepo.DeleteItemsByReceiptID(ctx, current.ID); err != nil {
				return err
			}
		} else {
			items = append(current.Items, items...)
			if err := ensureUniqueItems(items); err != nil {
				return err
			}
			if err := txRepo.DeleteItemsByReceiptID(ctx, current.ID); err != nil {
				return err
			}
			for i := range items {
				if strings.TrimSpace(items[i].ID) == "" {
					items[i].ID = newItemID()
				}
				items[i].UpdatedAt = time.Now().UTC()
				if items[i].CreatedAt.IsZero() {
					items[i].CreatedAt = items[i].UpdatedAt
				}
			}
		}
		if err := txRepo.CreateItems(ctx, items); err != nil {
			return err
		}
		if err := txRepo.RecalculateTotals(ctx, current.ID); err != nil {
			return err
		}
		if err := txRepo.AppendAudit(ctx, WarehouseReceiptAudit{ID: newAuditID(), ReceiptID: current.ID, Action: "add_item", Description: fmt.Sprintf("%d receipt items saved", len(input.Items)), ActorID: actor.UserID, CreatedAt: time.Now().UTC()}); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return WarehouseReceipt{}, err
	}
	returnResult, err = s.repo.GetByID(ctx, current.ID)
	return returnResult, err
}

func (s Service) UpdateItem(ctx context.Context, actor auth.Claims, receiptID, itemID string, input UpdateReceiptItemRequest) (WarehouseReceipt, error) {
	current, err := s.repo.GetByID(ctx, receiptID)
	if err != nil {
		return WarehouseReceipt{}, err
	}
	if err := s.ensureOperateAccess(ctx, actor, current.StoreID); err != nil {
		return WarehouseReceipt{}, err
	}
	if current.Status != ReceiptStatusDraft {
		return WarehouseReceipt{}, ErrReceiptImmutable
	}
	item, err := s.repo.GetItemByID(ctx, receiptID, itemID)
	if err != nil {
		return WarehouseReceipt{}, err
	}
	lineInput := ReceiptItemInput{
		ProductID:     item.ProductID,
		LocationID:    item.LocationID,
		Quantity:      item.Quantity,
		UnitPrice:     &item.UnitPrice,
		DiscountType:  item.DiscountType,
		DiscountValue: item.DiscountValue,
	}
	if input.ProductID != nil {
		lineInput.ProductID = strings.TrimSpace(*input.ProductID)
	}
	if input.LocationID != nil {
		lineInput.LocationID = strings.TrimSpace(*input.LocationID)
	}
	if input.Quantity != nil {
		lineInput.Quantity = *input.Quantity
	}
	if input.UnitPrice != nil {
		lineInput.UnitPrice = input.UnitPrice
	}
	if input.DiscountType != nil {
		lineInput.DiscountType = *input.DiscountType
	}
	if input.DiscountValue != nil || input.DiscountType != nil {
		lineInput.DiscountValue = input.DiscountValue
	}

	returnResult := WarehouseReceipt{}
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		txRepo := s.repo.WithTx(tx)
		updatedItem, err := s.materializeItem(ctx, txRepo, current, lineInput)
		if err != nil {
			return err
		}
		updatedItem.ID = item.ID
		updatedItem.ReceiptID = receiptID
		updatedItem.CreatedAt = item.CreatedAt
		updatedItem.UpdatedAt = time.Now().UTC()
		for _, existing := range current.Items {
			if existing.ID == item.ID {
				continue
			}
			if existing.ProductID == updatedItem.ProductID && existing.LocationID == updatedItem.LocationID {
				return ErrReceiptDuplicateItem
			}
		}
		if err := txRepo.UpdateItem(ctx, updatedItem); err != nil {
			return err
		}
		if err := txRepo.RecalculateTotals(ctx, receiptID); err != nil {
			return err
		}
		if err := txRepo.AppendAudit(ctx, WarehouseReceiptAudit{ID: newAuditID(), ReceiptID: receiptID, Action: "edit_item", Description: "receipt item updated", ActorID: actor.UserID, CreatedAt: time.Now().UTC()}); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return WarehouseReceipt{}, err
	}
	returnResult, err = s.repo.GetByID(ctx, receiptID)
	return returnResult, err
}

func (s Service) DeleteItem(ctx context.Context, actor auth.Claims, receiptID, itemID string) (WarehouseReceipt, error) {
	current, err := s.repo.GetByID(ctx, receiptID)
	if err != nil {
		return WarehouseReceipt{}, err
	}
	if err := s.ensureOperateAccess(ctx, actor, current.StoreID); err != nil {
		return WarehouseReceipt{}, err
	}
	if current.Status != ReceiptStatusDraft {
		return WarehouseReceipt{}, ErrReceiptImmutable
	}
	if err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		txRepo := s.repo.WithTx(tx)
		if err := txRepo.DeleteItem(ctx, receiptID, itemID); err != nil {
			return err
		}
		if err := txRepo.RecalculateTotals(ctx, receiptID); err != nil {
			return err
		}
		return txRepo.AppendAudit(ctx, WarehouseReceiptAudit{ID: newAuditID(), ReceiptID: receiptID, Action: "remove_item", Description: "receipt item deleted", ActorID: actor.UserID, CreatedAt: time.Now().UTC()})
	}); err != nil {
		return WarehouseReceipt{}, err
	}
	return s.repo.GetByID(ctx, receiptID)
}

func (s Service) Confirm(ctx context.Context, actor auth.Claims, receiptID, idempotencyKey string) (WarehouseReceipt, error) {
	current, err := s.repo.GetByID(ctx, receiptID)
	if err != nil {
		return WarehouseReceipt{}, err
	}
	if err := s.ensureManageAccess(ctx, actor, current.StoreID, true); err != nil {
		return WarehouseReceipt{}, err
	}
	key := strings.TrimSpace(idempotencyKey)
	// Resolve each positive line's EFFECTIVE receiving location once; it feeds the
	// fingerprint AND (via ConfirmDraft) the persisted item location + stock/movement writes.
	effective, err := s.resolveEffectiveLocations(ctx, current)
	if err != nil {
		return WarehouseReceipt{}, err
	}
	fingerprint := confirmFingerprint(current, actor.UserID, effective)
	// Idempotency pre-check: an already-confirmed receipt returns its original result for
	// an identical retry (same key + fingerprint); otherwise it is "already confirmed".
	if current.Status == ReceiptStatusConfirmed {
		if key != "" && current.ConfirmIdempotencyKey == key && current.ConfirmRequestFingerprint == fingerprint {
			return current, nil
		}
		return WarehouseReceipt{}, ErrReceiptAlreadyConfirmed
	}
	pendingAttachments, err := s.repo.ListPendingAttachments(ctx, receiptID)
	if err != nil {
		return WarehouseReceipt{}, err
	}
	if err := s.flushPendingAttachments(ctx, receiptID, actor.UserID, pendingAttachments); err != nil {
		return WarehouseReceipt{}, err
	}
	if err := s.repo.ConfirmDraft(ctx, receiptID, actor.UserID, key, fingerprint, effective); err != nil {
		// Concurrent loser: another confirm committed between our read and the lock. If it
		// was the same logical request (same key + fingerprint) return the original result;
		// otherwise surface "already confirmed".
		if errors.Is(err, ErrReceiptConfirmInvalidStatus) {
			if refreshed, ferr := s.repo.GetByID(ctx, receiptID); ferr == nil && refreshed.Status == ReceiptStatusConfirmed {
				if key != "" && refreshed.ConfirmIdempotencyKey == key && refreshed.ConfirmRequestFingerprint == fingerprint {
					return refreshed, nil
				}
				return WarehouseReceipt{}, ErrReceiptAlreadyConfirmed
			}
		}
		return WarehouseReceipt{}, err
	}
	return s.repo.GetByID(ctx, receiptID)
}

// confirmFingerprint is a deterministic hash of a confirm request's business-significant
// fields: store, receipt, confirming user, and — per positive line, sorted by line id —
// the receipt-item id, product id, EFFECTIVE receiving location, received quantity, and net
// received unit cost (unit_price − per-unit discount). The effective location (eff[itemID])
// is the location the confirm actually receives into: an explicit valid location as-is, or
// the product default resolved by the canonical policy. Using the EFFECTIVE (not the raw
// draft) location keeps the destination business-significant — a blank-draft line that
// resolves to a concrete location yields the same hash on a post-confirm retry (its
// location is persisted), while the SAME key against a DIFFERENT effective destination is a
// genuine conflict. The live status is deliberately omitted so a post-confirm retry still
// matches. Joined with control separators and SHA-256 hashed (stable, locale-independent).
func confirmFingerprint(r WarehouseReceipt, userID string, eff map[string]string) string {
	type line struct {
		id, product, loc string
		qty              int
		cost             float64
	}
	lines := make([]line, 0, len(r.Items))
	for _, it := range r.Items {
		if it.Quantity <= 0 {
			continue
		}
		lines = append(lines, line{id: it.ID, product: it.ProductID, loc: eff[it.ID], qty: it.Quantity, cost: it.UnitPrice - it.DiscountAmount})
	}
	sort.Slice(lines, func(i, j int) bool { return lines[i].id < lines[j].id })
	var b strings.Builder
	b.WriteString(r.StoreID)
	b.WriteByte(0x1f)
	b.WriteString(r.ID)
	b.WriteByte(0x1f)
	b.WriteString(userID)
	for _, l := range lines {
		b.WriteByte(0x1f)
		b.WriteString(l.id)
		b.WriteByte(0x1e)
		b.WriteString(l.product)
		b.WriteByte(0x1e)
		b.WriteString(l.loc)
		b.WriteByte(0x1e)
		b.WriteString(strconv.Itoa(l.qty))
		b.WriteByte(0x1e)
		b.WriteString(strconv.FormatFloat(l.cost, 'f', 2, 64))
	}
	sum := sha256.Sum256([]byte(b.String()))
	return hex.EncodeToString(sum[:])
}

// resolveEffectiveLocations computes each positive line's EFFECTIVE receiving location —
// an explicit valid location as-is, otherwise the product default resolved by the canonical
// policy (store, warehouse, active location, active warehouse; sale point allowed). The same
// effective location is used for the confirm fingerprint, the persisted item location, and
// the stock/movement writes (resolved once, used everywhere). For an already-confirmed
// receipt the persisted location IS the effective location (frozen history), so it is used
// directly without re-resolution.
func (s Service) resolveEffectiveLocations(ctx context.Context, r WarehouseReceipt) (map[string]string, error) {
	eff := make(map[string]string, len(r.Items))
	confirmed := r.Status == ReceiptStatusConfirmed
	for _, it := range r.Items {
		if it.Quantity <= 0 {
			continue
		}
		if confirmed {
			eff[it.ID] = it.LocationID
			continue
		}
		loc, err := s.repo.ResolveLineReceivingLocation(ctx, r.StoreID, r.WarehouseID, it.ProductID, it.LocationID)
		if err != nil {
			return nil, err
		}
		eff[it.ID] = loc.ID
	}
	return eff, nil
}

func (s Service) Submit(ctx context.Context, actor auth.Claims, receiptID string) (WarehouseReceipt, error) {
	current, err := s.repo.GetByID(ctx, receiptID)
	if err != nil {
		return WarehouseReceipt{}, err
	}
	if err := s.ensureOperateAccess(ctx, actor, current.StoreID); err != nil {
		return WarehouseReceipt{}, err
	}
	if current.Status != ReceiptStatusDraft {
		return WarehouseReceipt{}, ErrReceiptSubmitOnlyDraft
	}
	if len(current.Items) == 0 {
		return WarehouseReceipt{}, ErrReceiptItemsRequired
	}
	// A receipt cannot move to pending_review while any positive-quantity item lacks a
	// valid receiving location. Honour the per-line location: validate an explicit one,
	// or fall back to the product default (sale points allowed; §3C/§3D errors surfaced).
	for _, item := range current.Items {
		if item.Quantity <= 0 {
			continue
		}
		if _, err := s.repo.ResolveLineReceivingLocation(ctx, current.StoreID, current.WarehouseID, item.ProductID, item.LocationID); err != nil {
			return WarehouseReceipt{}, err
		}
	}
	if err := s.repo.SubmitForReview(ctx, receiptID, actor.UserID); err != nil {
		return WarehouseReceipt{}, err
	}
	return s.repo.GetByID(ctx, receiptID)
}

func (s Service) Reopen(ctx context.Context, actor auth.Claims, receiptID string) (WarehouseReceipt, error) {
	current, err := s.repo.GetByID(ctx, receiptID)
	if err != nil {
		return WarehouseReceipt{}, err
	}
	role, err := s.resolveStoreRole(ctx, current.StoreID, actor)
	if err != nil {
		return WarehouseReceipt{}, err
	}
	if !auth.RoleSatisfies(role, auth.StoreLevelManage) {
		return WarehouseReceipt{}, ErrReceiptReopenForbidden
	}
	if err := s.repo.ReopenToDraft(ctx, receiptID, actor.UserID); err != nil {
		return WarehouseReceipt{}, err
	}
	return s.repo.GetByID(ctx, receiptID)
}

func (s Service) Cancel(ctx context.Context, actor auth.Claims, receiptID string) (WarehouseReceipt, error) {
	current, err := s.repo.GetByID(ctx, receiptID)
	if err != nil {
		return WarehouseReceipt{}, err
	}
	if err := s.ensureManageAccess(ctx, actor, current.StoreID, false); err != nil {
		return WarehouseReceipt{}, err
	}
	if err := s.repo.CancelDraft(ctx, receiptID, actor.UserID); err != nil {
		return WarehouseReceipt{}, err
	}
	return s.repo.GetByID(ctx, receiptID)
}

func (s Service) UploadAttachment(ctx context.Context, actor auth.Claims, receiptID string, req UploadAttachmentRequest) (WarehouseReceipt, error) {
	current, err := s.repo.GetByID(ctx, receiptID)
	if err != nil {
		return WarehouseReceipt{}, err
	}
	if err := s.ensureOperateAccess(ctx, actor, current.StoreID); err != nil {
		return WarehouseReceipt{}, err
	}
	if current.Status != ReceiptStatusDraft {
		return WarehouseReceipt{}, ErrReceiptImmutable
	}
	meta, err := validateAttachmentFile(req.File)
	if err != nil {
		return WarehouseReceipt{}, err
	}
	src, err := req.File.Open()
	if err != nil {
		return WarehouseReceipt{}, err
	}
	defer src.Close()
	data, err := io.ReadAll(src)
	if err != nil {
		return WarehouseReceipt{}, err
	}
	if err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		txRepo := s.repo.WithTx(tx)
		if err := txRepo.CreatePendingAttachment(ctx, WarehouseReceiptPendingAttachment{
			ID:        newAttachmentToken(),
			ReceiptID: receiptID,
			MimeType:  meta.MimeType,
			Name:      meta.Name,
			Size:      meta.Size,
			Data:      data,
			CreatedBy: actor.UserID,
			CreatedAt: time.Now().UTC(),
		}); err != nil {
			return err
		}
		return txRepo.AppendAudit(ctx, WarehouseReceiptAudit{ID: newAuditID(), ReceiptID: receiptID, Action: "add_pending_attachment", Description: meta.Name, ActorID: actor.UserID, CreatedAt: time.Now().UTC()})
	}); err != nil {
		return WarehouseReceipt{}, err
	}
	return s.repo.GetByID(ctx, receiptID)
}

func (s Service) flushPendingAttachments(ctx context.Context, receiptID, actorID string, pending []WarehouseReceiptPendingAttachment) error {
	if len(pending) == 0 {
		return nil
	}
	uploadedURLs := make([]string, 0, len(pending))
	toPersist := make([]WarehouseReceiptAttachment, 0, len(pending))
	pendingIDs := make([]string, 0, len(pending))
	now := time.Now().UTC()
	for _, item := range pending {
		url, err := s.storage.SaveAttachmentBytes(item.Name, item.MimeType, item.Data)
		if err != nil {
			s.cleanupUploaded(uploadedURLs)
			if isStorageUnavailableError(err) {
				return fmt.Errorf("%w: %v", ErrReceiptAttachmentStorageUnavailable, err)
			}
			return fmt.Errorf("%w: %v", ErrReceiptAttachmentUploadFailed, err)
		}
		uploadedURLs = append(uploadedURLs, url)
		toPersist = append(toPersist, WarehouseReceiptAttachment{
			ID:         newAttachmentToken(),
			ReceiptID:  receiptID,
			URL:        url,
			MimeType:   item.MimeType,
			Name:       item.Name,
			Size:       item.Size,
			UploadedBy: actorID,
			UploadedAt: now,
		})
		pendingIDs = append(pendingIDs, item.ID)
	}
	if err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		txRepo := s.repo.WithTx(tx)
		if err := txRepo.CreateAttachments(ctx, toPersist); err != nil {
			return err
		}
		last := toPersist[len(toPersist)-1]
		if err := txRepo.UpdateAttachment(ctx, receiptID, last.URL, last.MimeType, last.Name, last.Size, actorID); err != nil {
			return err
		}
		if err := txRepo.DeletePendingAttachments(ctx, receiptID, pendingIDs); err != nil {
			return err
		}
		return txRepo.AppendAudit(ctx, WarehouseReceiptAudit{ID: newAuditID(), ReceiptID: receiptID, Action: "upload_attachment", Description: fmt.Sprintf("%d attachment(s) uploaded", len(toPersist)), ActorID: actorID, CreatedAt: now})
	}); err != nil {
		s.cleanupUploaded(uploadedURLs)
		return err
	}
	return nil
}

func (s Service) cleanupUploaded(urls []string) {
	for _, url := range urls {
		_ = s.storage.DeleteByURL(url)
	}
}

func isStorageUnavailableError(err error) bool {
	var netErr net.Error
	if errors.As(err, &netErr) {
		return true
	}
	if errors.Is(err, syscall.ECONNREFUSED) || errors.Is(err, syscall.ECONNRESET) {
		return true
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "connection refused") ||
		strings.Contains(msg, "no such host") ||
		strings.Contains(msg, "timeout") ||
		strings.Contains(msg, "connection reset")
}

func (s Service) GenerateDocumentNo(ctx context.Context, actor auth.Claims, input GenerateDocumentNoRequest) (GenerateDocumentNoResponse, error) {
	storeID := strings.TrimSpace(input.StoreID)
	if storeID == "" {
		return GenerateDocumentNoResponse{}, ErrReceiptStoreIDRequired
	}
	if err := s.ensureOperateAccess(ctx, actor, storeID); err != nil {
		return GenerateDocumentNoResponse{}, err
	}
	documentNo, err := s.repo.GenerateDocumentNo(ctx, time.Now())
	if err != nil {
		return GenerateDocumentNoResponse{}, err
	}
	return GenerateDocumentNoResponse{DocumentNo: documentNo}, nil
}

func (s Service) StockImpact(ctx context.Context, actor auth.Claims, receiptID string) ([]StockImpactPreview, error) {
	receipt, err := s.GetByID(ctx, actor, receiptID)
	if err != nil {
		return nil, err
	}
	preview, err := s.repo.ComputePreview(ctx, receipt.ID)
	if err != nil {
		return nil, err
	}
	return preview, nil
}

func (s Service) Print(ctx context.Context, actor auth.Claims, receiptID string) (PrintReceiptResponse, error) {
	receipt, err := s.GetByID(ctx, actor, receiptID)
	if err != nil {
		return PrintReceiptResponse{}, err
	}
	if err := s.repo.AppendAudit(ctx, WarehouseReceiptAudit{ID: newAuditID(), ReceiptID: receiptID, Action: "print", Description: "receipt printed", ActorID: actor.UserID, CreatedAt: time.Now().UTC()}); err != nil {
		return PrintReceiptResponse{}, err
	}
	html := renderReceiptHTML(receipt)
	return PrintReceiptResponse{
		ReceiptID:   receipt.ID,
		DocumentNo:  receipt.DocumentNo,
		FileName:    fmt.Sprintf("warehouse-receipt-%s.html", receipt.DocumentNo),
		ContentType: "text/html; charset=utf-8",
		HTML:        html,
	}, nil
}

// resolveStoreRole returns the caller's active store role for storeID ("" if no
// active membership). warehouse_receipt's storeID lives on the receipt resource
// rather than in the URL path, so membership is resolved here instead of via the
// store-authorization middleware; the role→level decision still uses
// auth.RoleSatisfies (single source of truth).
func (s Service) resolveStoreRole(ctx context.Context, storeID string, actor auth.Claims) (string, error) {
	if actor.Role == auth.RolePlatformAdmin {
		return auth.RoleOwner, nil
	}
	var member struct {
		Role   string `gorm:"column:role"`
		Status string `gorm:"column:status"`
	}
	err := s.db.WithContext(ctx).
		Table("store_members").
		Select("role, status").
		Where("store_id = ? AND user_id = ?", storeID, actor.UserID).
		Take(&member).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "", nil
		}
		return "", err
	}
	if member.Status != "active" {
		return "", nil
	}
	return member.Role, nil
}

func (s Service) ensureOperateAccess(ctx context.Context, actor auth.Claims, storeID string) error {
	role, err := s.resolveStoreRole(ctx, storeID, actor)
	if err != nil {
		return err
	}
	if !auth.RoleSatisfies(role, auth.StoreLevelAccess) {
		return ErrReceiptForbidden
	}
	return nil
}

func (s Service) ensureManageAccess(ctx context.Context, actor auth.Claims, storeID string, confirm bool) error {
	role, err := s.resolveStoreRole(ctx, storeID, actor)
	if err != nil {
		return err
	}
	if !auth.RoleSatisfies(role, auth.StoreLevelManage) {
		if confirm {
			return ErrReceiptConfirmForbidden
		}
		return ErrReceiptCancelForbidden
	}
	return nil
}

func (s Service) buildHeader(ctx context.Context, repo Repository, current WarehouseReceipt, actorID, storeID, warehouseID, supplierID, purchaseOrderID, documentNo string, receivedAt *time.Time, referenceNo, note string, vatIncluded *bool, vatPercent *float64, allowGenerate bool) (WarehouseReceipt, error) {
	warehouseID = strings.TrimSpace(warehouseID)
	if warehouseID == "" {
		return WarehouseReceipt{}, ErrReceiptWarehouseRequired
	}
	exists, err := repo.WarehouseExists(ctx, storeID, warehouseID)
	if err != nil {
		return WarehouseReceipt{}, err
	}
	if !exists {
		return WarehouseReceipt{}, ErrReceiptWarehouseNotFound
	}
	supplierID = strings.TrimSpace(supplierID)
	if supplierID != "" {
		exists, err := repo.SupplierExists(ctx, storeID, supplierID)
		if err != nil {
			return WarehouseReceipt{}, err
		}
		if !exists {
			return WarehouseReceipt{}, ErrReceiptSupplierNotFound
		}
	}
	purchaseOrderID = strings.TrimSpace(purchaseOrderID)
	if purchaseOrderID != "" {
		po, err := repo.GetPurchaseOrder(ctx, storeID, purchaseOrderID)
		if err != nil {
			return WarehouseReceipt{}, err
		}
		if supplierID != "" && po.SupplierID != "" && supplierID != po.SupplierID {
			return WarehouseReceipt{}, ErrReceiptPurchaseOrderNotFound
		}
	}
	documentNo = strings.TrimSpace(documentNo)
	if documentNo == "" {
		if !allowGenerate {
			return WarehouseReceipt{}, ErrReceiptDocumentNoRequired
		}
		documentNo, err = repo.GenerateDocumentNo(ctx, time.Now())
		if err != nil {
			return WarehouseReceipt{}, err
		}
	}
	vatIncludedValue := true
	if vatIncluded != nil {
		vatIncludedValue = *vatIncluded
	} else if current.ID != "" {
		vatIncludedValue = current.VATIncluded
	}
	vatPercentValue := 7.0
	if vatPercent != nil {
		vatPercentValue = *vatPercent
	} else if current.ID != "" && current.VATPercent > 0 {
		vatPercentValue = current.VATPercent
	}
	if vatPercentValue < 0 || vatPercentValue > 100 {
		return WarehouseReceipt{}, ErrReceiptInvalidVATPercent
	}
	receivedAtValue := time.Now().UTC()
	if receivedAt != nil && !receivedAt.IsZero() {
		receivedAtValue = receivedAt.UTC()
	} else if !current.ReceivedAt.IsZero() {
		receivedAtValue = current.ReceivedAt
	}
	result := current
	result.StoreID = storeID
	result.WarehouseID = warehouseID
	result.SupplierID = supplierID
	result.PurchaseOrderID = purchaseOrderID
	result.DocumentNo = documentNo
	result.ReceivedAt = receivedAtValue
	result.ReferenceNo = strings.TrimSpace(referenceNo)
	result.Note = strings.TrimSpace(note)
	result.VATIncluded = vatIncludedValue
	result.VATPercent = vatPercentValue
	if strings.TrimSpace(result.CreatedBy) == "" {
		result.CreatedBy = actorID
	}
	return result, nil
}

func (s Service) materializeItems(ctx context.Context, repo Repository, receipt WarehouseReceipt, inputs []ReceiptItemInput) ([]WarehouseReceiptItem, error) {
	items := make([]WarehouseReceiptItem, 0, len(inputs))
	for _, input := range inputs {
		item, err := s.materializeItem(ctx, repo, receipt, input)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	if err := ensureUniqueItems(items); err != nil {
		return nil, err
	}
	return items, nil
}

func (s Service) materializeItem(ctx context.Context, repo Repository, receipt WarehouseReceipt, input ReceiptItemInput) (WarehouseReceiptItem, error) {
	productID := strings.TrimSpace(input.ProductID)
	if productID == "" {
		return WarehouseReceiptItem{}, ErrReceiptItemProductRequired
	}
	if input.Quantity < 1 {
		return WarehouseReceiptItem{}, ErrReceiptItemQuantityRequired
	}
	product, err := repo.GetProductSnapshot(ctx, receipt.StoreID, productID)
	if err != nil {
		return WarehouseReceiptItem{}, err
	}
	if !product.IsActive {
		return WarehouseReceiptItem{}, ErrReceiptProductInactive
	}
	// Phase W3 §3: each line gets one authoritative receiving location.
	//   A) an explicit per-line location → validated against the receipt warehouse
	//      (active, same store, same warehouse; a SALE POINT is allowed). Invalid → error.
	//   B) otherwise the product's default location, when valid for the receipt warehouse
	//      (sale point allowed). A missing/dangling/other-warehouse default leaves the
	//      draft line unresolved (NULL) for correction; submit/confirm enforce a location.
	var locationID, locationName, zoneName, floorName string
	if explicit := strings.TrimSpace(input.LocationID); explicit != "" {
		loc, err := repo.ValidateExplicitReceivingLocation(ctx, receipt.StoreID, receipt.WarehouseID, explicit)
		if err != nil {
			return WarehouseReceiptItem{}, err
		}
		locationID, locationName, zoneName, floorName = loc.ID, loc.Name, loc.ZoneName, loc.FloorName
	} else if strings.TrimSpace(product.DefaultLocationID) != "" {
		loc, err := repo.ResolveValidReceivingLocation(ctx, receipt.StoreID, receipt.WarehouseID, product.ID)
		switch {
		case err == nil:
			locationID, locationName, zoneName, floorName = loc.ID, loc.Name, loc.ZoneName, loc.FloorName
		case errors.Is(err, ErrReceiptLocationWrongWarehouse),
			errors.Is(err, ErrReceiptLocationNotFound),
			errors.Is(err, ErrReceiptItemLocationMissing),
			errors.Is(err, ErrReceiptLocationInactive):
			// Unresolved default → leave the draft line without a location.
		default:
			return WarehouseReceiptItem{}, err
		}
	}
	unitPrice := product.CostPrice
	if input.UnitPrice != nil {
		unitPrice = *input.UnitPrice
	}
	if unitPrice < 0 {
		return WarehouseReceiptItem{}, ErrReceiptItemUnitPriceInvalid
	}
	discountAmount, err := calculateDiscount(unitPrice, input.DiscountType, input.DiscountValue)
	if err != nil {
		return WarehouseReceiptItem{}, err
	}
	lineSubtotal := roundMoney(float64(input.Quantity) * unitPrice)
	lineNet := roundMoney(float64(input.Quantity) * (unitPrice - discountAmount))
	now := time.Now().UTC()
	item := WarehouseReceiptItem{
		ID:             newItemID(),
		ReceiptID:      receipt.ID,
		ProductID:      product.ID,
		LocationID:     locationID,
		WarehouseID:    receipt.WarehouseID,
		ZoneName:       zoneName,
		FloorName:      floorName,
		LocationName:   locationName,
		ProductName:    product.Name,
		SKU:            product.SKU,
		Barcode:        product.Barcode,
		UnitName:       product.UnitName,
		Quantity:       input.Quantity,
		UnitPrice:      roundMoney(unitPrice),
		DiscountType:   normalizeDiscountType(input.DiscountType),
		DiscountValue:  input.DiscountValue,
		DiscountAmount: roundMoney(discountAmount),
		LineSubtotal:   lineSubtotal,
		LineNet:        lineNet,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	if err := s.ensurePOOutstanding(ctx, repo, receipt, item); err != nil {
		return WarehouseReceiptItem{}, err
	}
	return item, nil
}

func (s Service) ensurePOOutstanding(ctx context.Context, repo Repository, receipt WarehouseReceipt, item WarehouseReceiptItem) error {
	if strings.TrimSpace(receipt.PurchaseOrderID) == "" {
		return nil
	}
	poItems, err := repo.GetPurchaseOrderItems(ctx, receipt.PurchaseOrderID)
	if err != nil {
		return err
	}
	for _, poItem := range poItems {
		if poItem.ProductID != item.ProductID {
			continue
		}
		if item.Quantity > (poItem.Quantity - poItem.ReceivedQuantity) {
			return ErrReceiptPOQuantityExceeded
		}
		return nil
	}
	return ErrReceiptPOQuantityExceeded
}

func ensureUniqueItems(items []WarehouseReceiptItem) error {
	seen := map[string]struct{}{}
	for _, item := range items {
		key := item.ProductID + "::" + item.LocationID
		if _, ok := seen[key]; ok {
			return ErrReceiptDuplicateItem
		}
		seen[key] = struct{}{}
	}
	return nil
}

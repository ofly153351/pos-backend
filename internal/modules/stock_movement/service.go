package stock_movement

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"gorm.io/gorm"

	"pos-backend/internal/modules/auth"
)

type Service struct {
	repo Repository
	db   *gorm.DB
}

func NewService(repo Repository, db *gorm.DB) Service {
	return Service{repo: repo, db: db}
}

func (s Service) canManage(ctx context.Context, storeID, userID, role string) (bool, error) {
	if role == "platform_admin" {
		return true, nil
	}
	var count int64
	err := s.db.WithContext(ctx).
		Table("store_members").
		Where("store_id = ? AND user_id = ? AND role IN ? AND status <> 'suspended'", storeID, userID, []string{"owner", "manager"}).
		Count(&count).Error
	return count > 0, err
}

func (s Service) productExistsInStore(ctx context.Context, storeID, productID string) (bool, error) {
	var count int64
	err := s.db.WithContext(ctx).
		Table("products").
		Where("id = ? AND store_id = ?", productID, storeID).
		Count(&count).Error
	return count > 0, err
}

// resolveStockLocation picks the location a stock operation targets, WITHOUT ever
// guessing by row order (Phase W2 §15/§19). Precedence: an explicit location wins;
// else the product's own default_location_id (only if it is still an active location);
// else the store's authoritative default sale location (is_default_sale + active +
// sale point). An empty result means the caller must reject with ErrStockLocationRequired
// rather than silently pick an arbitrary first row.
func (s Service) resolveStockLocation(ctx context.Context, storeID, productID, explicit string) (string, error) {
	if e := strings.TrimSpace(explicit); e != "" {
		return e, nil
	}
	// Product default, only if it points at an active location in this store.
	var locID string
	err := s.db.WithContext(ctx).
		Table("products p").
		Select("p.default_location_id").
		Joins("JOIN locations l ON l.id = p.default_location_id AND l.is_active = TRUE").
		Where("p.id = ? AND p.store_id = ?", productID, storeID).
		Take(&locID).Error
	if err == nil && strings.TrimSpace(locID) != "" {
		return locID, nil
	}
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return "", err
	}
	// Store authoritative default sale location (W1 flag).
	err = s.db.WithContext(ctx).
		Table("locations").
		Select("id").
		Where("store_id = ? AND is_default_sale = TRUE AND is_active = TRUE AND is_sale_point = TRUE", storeID).
		Take(&locID).Error
	if err == nil {
		return locID, nil
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return "", nil
	}
	return "", err
}

// findByIdempotencyKey returns a prior movement for this (store, key), or nil if none.
// Used to make a retried adjustment submit a no-op that returns the original result.
func (s Service) findByIdempotencyKey(ctx context.Context, storeID, key string) (*StockMovement, error) {
	if strings.TrimSpace(key) == "" {
		return nil, nil
	}
	var m StockMovement
	err := s.db.WithContext(ctx).Table("stock_movements").
		Where("store_id = ? AND idempotency_key = ?", storeID, key).
		Take(&m).Error
	if err == nil {
		return &m, nil
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return nil, err
}

// isUniqueViolation reports whether err is a Postgres unique_violation (SQLSTATE 23505) —
// the idempotency-key index firing when two identical submits race past the pre-check.
func isUniqueViolation(err error) bool {
	return err != nil && strings.Contains(err.Error(), "SQLSTATE 23505")
}

// normalizeNote is the documented note-normalization rule for the idempotency
// fingerprint: surrounding whitespace is trimmed and meaningful internal text is
// preserved; an empty note and an absent/null note are EQUIVALENT (both → "").
func normalizeNote(note string) string {
	return strings.TrimSpace(note)
}

// requestFingerprint is a deterministic hash of a stock-adjustment request's
// business-significant fields, representing the ORIGINAL user intent. A reused
// idempotency key whose fingerprint matches is a safe retry; a different fingerprint is
// a conflict. For SET_ACTUAL, requestedQty is the REQUESTED physical quantity — never the
// live-stock-dependent computed delta — so a SET 10 vs SET 20 (same key) is a mismatch.
//
// Fields (in fixed order): store_id, product_id, location_id, mode (ADD/SUBTRACT/
// SET_ACTUAL), requested_quantity, reason code, normalized note, user_id. They are joined
// with the ASCII Unit Separator (0x1f, which cannot appear in normal input) and SHA-256
// hashed — a stable, order-independent, locale-independent serialization (no JSON map
// ordering).
func requestFingerprint(storeID, productID, locationID, mode string, requestedQty int, reason, note, userID string) string {
	parts := []string{
		storeID,
		productID,
		locationID,
		mode,
		strconv.Itoa(requestedQty),
		strings.TrimSpace(reason),
		normalizeNote(note),
		userID,
	}
	sum := sha256.Sum256([]byte(strings.Join(parts, "\x1f")))
	return hex.EncodeToString(sum[:])
}

// AddStock creates IN movements and adds stock to locations
func (s Service) AddStock(ctx context.Context, actor auth.Claims, storeID string, input AddStockRequest) (AdditionResult, error) {
	if strings.TrimSpace(storeID) == "" {
		return AdditionResult{}, fmt.Errorf("storeID is required")
	}

	allowed, err := s.canManage(ctx, storeID, actor.UserID, actor.Role)
	if err != nil {
		return AdditionResult{}, err
	}
	if !allowed {
		return AdditionResult{}, ErrStockForbidden
	}

	if len(input.Items) == 0 {
		return AdditionResult{}, ErrStockNoItems
	}

	for _, item := range input.Items {
		if item.Quantity <= 0 {
			return AdditionResult{}, ErrStockBadQty
		}
		// A reason is mandatory for an ADD (§13); OTHER requires a free-text note.
		if err := validateReason(opAdd, item.Reason, item.Note); err != nil {
			return AdditionResult{}, err
		}
	}

	now := time.Now().UTC()
	var movements []StockMovement

	// Wrap every movement + stock write in one transaction so a mid-loop failure
	// rolls back all of them (no movement-without-stock drift, no partial add).
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		txRepo := NewPostgresRepository(tx)
		for _, item := range input.Items {
			exists, err := s.productExistsInStore(ctx, storeID, item.ProductID)
			if err != nil {
				return err
			}
			if !exists {
				return ErrProductNotFound
			}

			// Deterministic location (explicit → product default → store default sale).
			resolvedLocID, err := s.resolveStockLocation(ctx, storeID, item.ProductID, item.LocationID)
			if err != nil {
				return err
			}
			if resolvedLocID == "" {
				return ErrStockLocationRequired
			}
			locPtr := &resolvedLocID

			// Idempotency: a retried submit with the same key + SAME fingerprint returns the
			// original movement (no second stock change); the same key with a DIFFERENT
			// fingerprint (intent) is rejected as a conflict.
			key := strings.TrimSpace(item.IdempotencyKey)
			fp := requestFingerprint(storeID, item.ProductID, resolvedLocID, opAdd, item.Quantity, item.Reason, item.Note, actor.UserID)
			if key != "" {
				var existing StockMovement
				e := tx.WithContext(ctx).Table("stock_movements").
					Where("store_id = ? AND idempotency_key = ?", storeID, key).Take(&existing).Error
				if e == nil {
					if existing.RequestFingerprint == fp {
						movements = append(movements, existing)
						continue
					}
					return ErrStockIdempotencyConflict
				}
				if !errors.Is(e, gorm.ErrRecordNotFound) {
					return e
				}
			}

			mg := StockMovement{
				ID:             newID(),
				StoreID:        storeID,
				ProductID:      item.ProductID,
				LocationID:     locPtr,
				QuantityChange: item.Quantity,
				Type:           MovementTypeIn,
				Reason:         strings.TrimSpace(item.Reason),
				Note:           strings.TrimSpace(item.Note),
				CreatedBy:      actor.UserID,
				CreatedAt:      now,
			}
			if key != "" {
				mg.IdempotencyKey = &key
				mg.RequestFingerprint = fp
			}

			created, err := txRepo.Create(ctx, mg)
			if err != nil {
				return err
			}
			if err := txRepo.UpsertStock(ctx, storeID, item.ProductID, resolvedLocID, item.Quantity); err != nil {
				return err
			}
			movements = append(movements, created)
		}
		return nil
	})
	if err != nil {
		// A true simultaneous-race duplicate (both submits passed the pre-check) is
		// caught by the idempotency-key index; resolve a single-item add to the winner
		// only when the payload matches, else surface the conflict.
		if isUniqueViolation(err) && len(input.Items) == 1 {
			it := input.Items[0]
			if existing, e := s.findByIdempotencyKey(ctx, storeID, strings.TrimSpace(it.IdempotencyKey)); e == nil && existing != nil {
				loc, _ := s.resolveStockLocation(ctx, storeID, it.ProductID, it.LocationID)
				fp := requestFingerprint(storeID, it.ProductID, loc, opAdd, it.Quantity, it.Reason, it.Note, actor.UserID)
				if existing.RequestFingerprint == fp {
					return AdditionResult{Movements: []StockMovement{*existing}}, nil
				}
				return AdditionResult{}, ErrStockIdempotencyConflict
			}
		}
		return AdditionResult{}, err
	}

	return AdditionResult{Movements: movements}, nil
}

// RemoveStock creates OUT movements and deducts stock from locations
func (s Service) RemoveStock(ctx context.Context, actor auth.Claims, storeID string, req RemoveStockRequest) (StockMovement, error) {
	if strings.TrimSpace(storeID) == "" {
		return StockMovement{}, fmt.Errorf("storeID is required")
	}
	allowed, err := s.canManage(ctx, storeID, actor.UserID, actor.Role)
	if err != nil {
		return StockMovement{}, err
	}
	if !allowed {
		return StockMovement{}, ErrStockForbidden
	}
	if req.Quantity <= 0 {
		return StockMovement{}, ErrStockBadQty
	}
	// A reason is mandatory for a SUBTRACT (§13); OTHER requires a free-text note.
	if err := validateReason(opSubtract, req.Reason, req.Note); err != nil {
		return StockMovement{}, err
	}

	exists, err := s.productExistsInStore(ctx, storeID, req.ProductID)
	if err != nil {
		return StockMovement{}, err
	}
	if !exists {
		return StockMovement{}, ErrProductNotFound
	}
	// Removing stock requires knowing the source location — no guessing.
	if strings.TrimSpace(req.LocationID) == "" {
		return StockMovement{}, ErrStockLocationRequired
	}

	// Idempotency: same key + same fingerprint returns the original (no double
	// deduction); same key + different fingerprint is a conflict.
	key := strings.TrimSpace(req.IdempotencyKey)
	fp := requestFingerprint(storeID, req.ProductID, req.LocationID, opSubtract, req.Quantity, req.Reason, req.Note, actor.UserID)
	if key != "" {
		existing, err := s.findByIdempotencyKey(ctx, storeID, key)
		if err != nil {
			return StockMovement{}, err
		}
		if existing != nil {
			if existing.RequestFingerprint == fp {
				return *existing, nil
			}
			return StockMovement{}, ErrStockIdempotencyConflict
		}
	}

	now := time.Now().UTC()
	locID := &req.LocationID

	mg := StockMovement{
		ID:             newID(),
		StoreID:        storeID,
		ProductID:      req.ProductID,
		LocationID:     locID,
		QuantityChange: -req.Quantity,
		Type:           MovementTypeOut,
		Reason:         strings.TrimSpace(req.Reason),
		Note:           strings.TrimSpace(req.Note),
		CreatedBy:      actor.UserID,
		CreatedAt:      now,
	}
	if key != "" {
		mg.IdempotencyKey = &key
		mg.RequestFingerprint = fp
	}

	// Movement + stock deduction in one transaction: if the guarded UpsertStock
	// rejects an over-removal, the OUT movement is rolled back too (no drift).
	var created StockMovement
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		txRepo := NewPostgresRepository(tx)
		var err error
		created, err = txRepo.Create(ctx, mg)
		if err != nil {
			return err
		}
		return txRepo.UpsertStock(ctx, storeID, req.ProductID, req.LocationID, -req.Quantity)
	})
	if err != nil {
		if isUniqueViolation(err) && key != "" {
			if existing, e := s.findByIdempotencyKey(ctx, storeID, key); e == nil && existing != nil {
				if existing.RequestFingerprint == fp {
					return *existing, nil
				}
				return StockMovement{}, ErrStockIdempotencyConflict
			}
		}
		// Surface a clear "you asked to remove more than is on hand here" message.
		if errors.Is(err, ErrInsufficientStock) {
			return StockMovement{}, ErrStockExceedsAvailable
		}
		return StockMovement{}, err
	}

	return created, nil
}

// validateTransferLocation enforces the canonical receiving/transfer location rules for one
// endpoint of a transfer (Phase W4A §4): the location must exist, belong to THIS store, be
// active, and sit in an active warehouse. A location in another store yields the explicit
// cross-store error (not a generic "not found"). Sale points are allowed on both ends.
func (s Service) validateTransferLocation(ctx context.Context, storeID, locationID string, missingErr error) error {
	var row struct {
		StoreID         string `gorm:"column:store_id"`
		IsActive        bool   `gorm:"column:is_active"`
		WarehouseActive bool   `gorm:"column:warehouse_active"`
	}
	err := s.db.WithContext(ctx).Table("locations l").
		Select("l.store_id, l.is_active, w.is_active AS warehouse_active").
		Joins("JOIN warehouses w ON w.id = l.warehouse_id").
		Where("l.id = ?", locationID).Take(&row).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return missingErr
		}
		return err
	}
	if row.StoreID != storeID {
		return ErrTransferCrossStore
	}
	if !row.IsActive || !row.WarehouseActive {
		return ErrTransferLocationInactive
	}
	return nil
}

// transferFingerprint is the deterministic hash of a transfer's business-significant intent
// (Phase W4A §9): store, product, source, destination, requested quantity, reason, normalized
// note, user. A reused idempotency key whose fingerprint matches is a safe retry; a different
// fingerprint is a conflict.
func transferFingerprint(storeID, productID, srcLoc, destLoc string, qty int, reason, note, userID string) string {
	parts := []string{storeID, productID, srcLoc, destLoc, opTransfer, strconv.Itoa(qty), strings.TrimSpace(reason), normalizeNote(note), userID}
	sum := sha256.Sum256([]byte(strings.Join(parts, "\x1f")))
	return hex.EncodeToString(sum[:])
}

func (s Service) findTransferByKey(ctx context.Context, storeID, key string) (*StockTransfer, error) {
	if strings.TrimSpace(key) == "" {
		return nil, nil
	}
	var t StockTransfer
	err := s.db.WithContext(ctx).Table("stock_transfers").
		Where("store_id = ? AND idempotency_key = ?", storeID, key).Take(&t).Error
	if err == nil {
		return &t, nil
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return nil, err
}

// loadTransferResult rebuilds a transfer's canonical result (header + paired movements) from
// a stored header — used to return the ORIGINAL result on an idempotent retry.
func (s Service) loadTransferResult(ctx context.Context, t StockTransfer) (TransferResult, error) {
	var movements []StockMovement
	if err := s.db.WithContext(ctx).Table("stock_movements").
		Where("store_id = ? AND reference_id = ?", t.StoreID, t.ID).
		Order("quantity_change ASC").Find(&movements).Error; err != nil {
		return TransferResult{}, err
	}
	return TransferResult{Transfer: t, Movements: movements}, nil
}

// TransferStock is the canonical location→location stock transfer (Phase W4A): it moves a
// positive quantity of one Product from one explicit source Location to one explicit
// destination Location in the SAME store, atomically, preserving total Product stock. It
// validates both Locations, locks both stock rows in deterministic (sorted) order, rejects
// insufficient source quantity, writes paired TRANSFER_OUT/TRANSFER_IN movements that share
// the transfer reference, records a stock_transfers header, and is idempotent + concurrency-
// safe. It never auto-selects a source/destination, auto-creates a warehouse/location, clones
// a product, transfers across stores, or changes products.default_location_id.
func (s Service) TransferStock(ctx context.Context, actor auth.Claims, storeID string, req TransferStockRequest) (TransferResult, error) {
	if strings.TrimSpace(storeID) == "" {
		return TransferResult{}, fmt.Errorf("storeID is required")
	}
	allowed, err := s.canManage(ctx, storeID, actor.UserID, actor.Role)
	if err != nil {
		return TransferResult{}, err
	}
	if !allowed {
		return TransferResult{}, ErrTransferForbidden
	}
	if req.Quantity <= 0 {
		return TransferResult{}, ErrStockBadQty
	}
	src := strings.TrimSpace(req.SourceLocationID)
	dst := strings.TrimSpace(req.DestLocationID)
	if src == "" {
		return TransferResult{}, ErrTransferSourceRequired
	}
	if dst == "" {
		return TransferResult{}, ErrTransferDestRequired
	}
	if src == dst {
		return TransferResult{}, ErrTransferSameLocation
	}
	exists, err := s.productExistsInStore(ctx, storeID, req.ProductID)
	if err != nil {
		return TransferResult{}, err
	}
	if !exists {
		return TransferResult{}, ErrProductNotFound
	}
	if err := validateReason(opTransfer, req.Reason, req.Note); err != nil {
		return TransferResult{}, err
	}
	if err := s.validateTransferLocation(ctx, storeID, src, ErrTransferLocationNotInStore); err != nil {
		return TransferResult{}, err
	}
	if err := s.validateTransferLocation(ctx, storeID, dst, ErrTransferLocationNotInStore); err != nil {
		return TransferResult{}, err
	}

	key := strings.TrimSpace(req.IdempotencyKey)
	fp := transferFingerprint(storeID, req.ProductID, src, dst, req.Quantity, req.Reason, req.Note, actor.UserID)
	// Idempotency pre-check: same key + same fingerprint returns the original transfer; same
	// key + different fingerprint is a conflict (Phase W4A §9).
	if existing, ferr := s.findTransferByKey(ctx, storeID, key); ferr != nil {
		return TransferResult{}, ferr
	} else if existing != nil {
		if existing.RequestFingerprint == fp {
			return s.loadTransferResult(ctx, *existing)
		}
		return TransferResult{}, ErrTransferIdempotencyConflict
	}

	transferID := newID()
	now := time.Now().UTC()
	reason := strings.TrimSpace(req.Reason)
	note := strings.TrimSpace(req.Note)
	header := StockTransfer{
		ID: transferID, StoreID: storeID, ProductID: req.ProductID,
		SourceLocationID: src, DestLocationID: dst, Quantity: req.Quantity,
		Reason: reason, Note: note, CreatedBy: actor.UserID, CreatedAt: now,
	}
	if key != "" {
		header.IdempotencyKey = &key
		header.RequestFingerprint = fp
	}
	var result TransferResult

	// One transaction: lock → read → deduct → add → paired movements → header. A failure at
	// any step rolls everything back (no split stock, no orphan movement, no header).
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		txRepo := NewPostgresRepository(tx)
		// ── Authoritative in-transaction validation + locking (Phase W4A Blocker 1) ──
		// Re-validate AND lock both locations + their parent warehouses INSIDE the tx, reading
		// active-state under the lock. This closes the TOCTOU window: a location/warehouse
		// deactivated after the advisory pre-check is caught here, because FOR UPDATE serializes
		// against the concurrent deactivation (we either see it committed → reject, or hold the
		// lock so it cannot commit until we do). The pre-tx checks above are advisory only.
		{
			var pCount int64
			if err := tx.Table("products").Where("id = ? AND store_id = ?", req.ProductID, storeID).Count(&pCount).Error; err != nil {
				return err
			}
			if pCount == 0 {
				return ErrProductNotFound
			}
			loLoc, hiLoc := src, dst
			if loLoc > hiLoc {
				loLoc, hiLoc = hiLoc, loLoc
			}
			type lockedLoc struct {
				ID          string `gorm:"column:id"`
				StoreID     string `gorm:"column:store_id"`
				IsActive    bool   `gorm:"column:is_active"`
				WarehouseID string `gorm:"column:warehouse_id"`
			}
			var locs []lockedLoc
			if err := tx.Raw(`SELECT id, store_id, is_active, warehouse_id FROM locations WHERE id IN (?, ?) ORDER BY id FOR UPDATE`, loLoc, hiLoc).Scan(&locs).Error; err != nil {
				return err
			}
			byID := map[string]lockedLoc{}
			for _, l := range locs {
				byID[l.ID] = l
			}
			sLoc, sOK := byID[src]
			dLoc, dOK := byID[dst]
			if !sOK || !dOK {
				return ErrTransferLocationNotInStore
			}
			if sLoc.StoreID != storeID || dLoc.StoreID != storeID {
				return ErrTransferCrossStore
			}
			whA, whB := sLoc.WarehouseID, dLoc.WarehouseID
			if whA > whB {
				whA, whB = whB, whA
			}
			type lockedWh struct {
				ID       string `gorm:"column:id"`
				IsActive bool   `gorm:"column:is_active"`
			}
			var whs []lockedWh
			if err := tx.Raw(`SELECT id, is_active FROM warehouses WHERE id IN (?, ?) ORDER BY id FOR UPDATE`, whA, whB).Scan(&whs).Error; err != nil {
				return err
			}
			whActive := map[string]bool{}
			for _, w := range whs {
				whActive[w.ID] = w.IsActive
			}
			if !sLoc.IsActive || !dLoc.IsActive || !whActive[sLoc.WarehouseID] || !whActive[dLoc.WarehouseID] {
				return ErrTransferLocationInactive
			}
		}
		// Lock the EXISTING source/destination stock rows FOR UPDATE in deterministic
		// location-id order so two concurrent transfers touching the same rows can never
		// deadlock or lose an update. Note: FOR UPDATE can only lock rows that already exist —
		// a brand-new destination location has no row yet, so only the source is locked here;
		// that case is still safe because (a) an absent source short-circuits below as
		// insufficient, and (b) the destination INSERT ... ON CONFLICT in UpsertStock is
		// serialized by the UNIQUE (product_id, location_id) constraint (block-then-DO-UPDATE,
		// no lost update). ORDER BY governs lock-acquisition order for the rows that do exist.
		if err := tx.Exec(`SELECT 1 FROM stocks WHERE store_id = ? AND product_id = ? AND location_id IN (?, ?) ORDER BY location_id FOR UPDATE`,
			storeID, req.ProductID, src, dst).Error; err != nil {
			return err
		}
		// Read source quantity under lock; reject insufficient (zero if no source row exists).
		var srcQty int
		if err := tx.Table("stocks").Select("COALESCE(quantity, 0)").
			Where("store_id = ? AND product_id = ? AND location_id = ?", storeID, req.ProductID, src).
			Scan(&srcQty).Error; err != nil {
			return err
		}
		if srcQty < req.Quantity {
			return ErrTransferInsufficient
		}
		// Deduct source (guarded: returns ErrInsufficientStock if a race would overdraw).
		if err := txRepo.UpsertStock(ctx, storeID, req.ProductID, src, -req.Quantity); err != nil {
			if errors.Is(err, ErrInsufficientStock) {
				return ErrTransferInsufficient
			}
			return err
		}
		// Add destination (creates the row if absent).
		if err := txRepo.UpsertStock(ctx, storeID, req.ProductID, dst, req.Quantity); err != nil {
			return err
		}
		ref := transferID
		outMg := StockMovement{ID: newID(), StoreID: storeID, ProductID: req.ProductID, LocationID: &src, DestinationLocationID: &dst, QuantityChange: -req.Quantity, Type: MovementTypeTransferOut, Reason: reason, ReferenceID: &ref, Note: note, CreatedBy: actor.UserID, CreatedAt: now}
		inMg := StockMovement{ID: newID(), StoreID: storeID, ProductID: req.ProductID, LocationID: &dst, QuantityChange: req.Quantity, Type: MovementTypeTransferIn, Reason: reason, ReferenceID: &ref, Note: note, CreatedBy: actor.UserID, CreatedAt: now}
		outCreated, err := txRepo.Create(ctx, outMg)
		if err != nil {
			return err
		}
		inCreated, err := txRepo.Create(ctx, inMg)
		if err != nil {
			return err
		}
		headerPayload := map[string]any{
			"id": header.ID, "store_id": header.StoreID, "product_id": header.ProductID,
			"source_location_id": header.SourceLocationID, "dest_location_id": header.DestLocationID,
			"quantity": header.Quantity, "reason": header.Reason, "note": header.Note,
			"created_by": header.CreatedBy, "created_at": header.CreatedAt,
		}
		if header.IdempotencyKey != nil {
			headerPayload["idempotency_key"] = *header.IdempotencyKey
			headerPayload["request_fingerprint"] = header.RequestFingerprint
		}
		if err := tx.Table("stock_transfers").Create(headerPayload).Error; err != nil {
			return err
		}
		result = TransferResult{Transfer: header, Movements: []StockMovement{outCreated, inCreated}}
		return nil
	})
	if err != nil {
		// Idempotency race: two identical transfers passed the pre-check; the loser's header
		// insert hit the unique index → its whole tx rolled back. Return the winner's result.
		if isUniqueViolation(err) {
			if existing, ferr := s.findTransferByKey(ctx, storeID, key); ferr == nil && existing != nil {
				if existing.RequestFingerprint == fp {
					return s.loadTransferResult(ctx, *existing)
				}
				return TransferResult{}, ErrTransferIdempotencyConflict
			}
		}
		return TransferResult{}, err
	}
	return result, nil
}

// AdjustStock sets physical stock count at a location, creates ADJUST movement
func (s Service) AdjustStock(ctx context.Context, actor auth.Claims, storeID string, req AdjustStockRequest) (StockMovement, error) {
	if strings.TrimSpace(storeID) == "" {
		return StockMovement{}, fmt.Errorf("storeID is required")
	}
	allowed, err := s.canManage(ctx, storeID, actor.UserID, actor.Role)
	if err != nil {
		return StockMovement{}, err
	}
	if !allowed {
		return StockMovement{}, ErrStockForbidden
	}

	if req.PhysicalQty < 0 {
		return StockMovement{}, ErrStockBadQty
	}
	// A reason is mandatory for SET_ACTUAL (§13); OTHER requires a free-text note.
	if err := validateReason(opSetActual, req.Reason, req.Note); err != nil {
		return StockMovement{}, err
	}

	exists, err := s.productExistsInStore(ctx, storeID, req.ProductID)
	if err != nil {
		return StockMovement{}, err
	}
	if !exists {
		return StockMovement{}, ErrProductNotFound
	}

	// Deterministic location (explicit → product default → store default sale).
	resolvedLocID, err := s.resolveStockLocation(ctx, storeID, req.ProductID, req.LocationID)
	if err != nil {
		return StockMovement{}, err
	}
	if resolvedLocID == "" {
		return StockMovement{}, ErrStockLocationRequired
	}
	locPtr := &resolvedLocID

	now := time.Now().UTC()
	movementType := strings.TrimSpace(req.MovementType)
	if movementType == "" {
		movementType = MovementTypeCountCorrection
	}
	var refPtr *string

	// Idempotency: same key + same fingerprint returns the original (never re-sets
	// stock); same key + different fingerprint is a conflict. The fingerprint hashes the
	// REQUESTED physical quantity (req.PhysicalQty) — not the live-stock-dependent delta —
	// so SET 10 vs SET 20 under one key is a mismatch.
	key := strings.TrimSpace(req.IdempotencyKey)
	fp := requestFingerprint(storeID, req.ProductID, resolvedLocID, opSetActual, req.PhysicalQty, req.Reason, req.Note, actor.UserID)
	if key != "" {
		existing, ferr := s.findByIdempotencyKey(ctx, storeID, key)
		if ferr != nil {
			return StockMovement{}, ferr
		}
		if existing != nil {
			if existing.RequestFingerprint == fp {
				return *existing, nil
			}
			return StockMovement{}, ErrStockIdempotencyConflict
		}
	}
	if ref := strings.TrimSpace(req.ReferenceID); ref != "" {
		refPtr = &ref
	}

	// Lock the stock row, read current qty, write the movement, and set the new
	// absolute quantity — all in one transaction so a concurrent sale/adjust cannot
	// interleave (closes the read-modify-write race) and the movement is never
	// recorded without the matching stock change.
	var created StockMovement
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Row lock (no-op if the row doesn't exist yet; SetStockQuantity creates it).
		if err := tx.Exec(
			"SELECT 1 FROM stocks WHERE store_id = ? AND product_id = ? AND location_id = ? FOR UPDATE",
			storeID, req.ProductID, resolvedLocID,
		).Error; err != nil {
			return err
		}
		txRepo := NewPostgresRepository(tx)
		currentQty, err := txRepo.GetCurrentStockQty(ctx, storeID, req.ProductID, resolvedLocID)
		if err != nil {
			return err
		}
		// SET_ACTUAL equal to current → no DB mutation, no zero-delta movement (§8).
		if req.PhysicalQty == currentQty {
			return ErrStockNoChange
		}
		mg := StockMovement{
			ID:             newID(),
			StoreID:        storeID,
			ProductID:      req.ProductID,
			LocationID:     locPtr,
			QuantityChange: req.PhysicalQty - currentQty,
			Type:           movementType,
			ReferenceID:    refPtr,
			Reason:         strings.TrimSpace(req.Reason),
			Note:           fmt.Sprintf("adjusted from %d to %d. %s", currentQty, req.PhysicalQty, strings.TrimSpace(req.Note)),
			CreatedBy:      actor.UserID,
			CreatedAt:      now,
		}
		if key != "" {
			mg.IdempotencyKey = &key
			mg.RequestFingerprint = fp
		}
		c, err := txRepo.Create(ctx, mg)
		if err != nil {
			return err
		}
		created = c
		return txRepo.SetStockQuantity(ctx, storeID, req.ProductID, resolvedLocID, req.PhysicalQty)
	})
	if err != nil {
		if isUniqueViolation(err) && key != "" {
			if existing, e := s.findByIdempotencyKey(ctx, storeID, key); e == nil && existing != nil {
				if existing.RequestFingerprint == fp {
					return *existing, nil
				}
				return StockMovement{}, ErrStockIdempotencyConflict
			}
		}
		return StockMovement{}, err
	}

	return created, nil
}

// RecordSaleMovement creates a SALE movement and deducts stock
func (s Service) RecordSaleMovement(ctx context.Context, actor auth.Claims, storeID, productID, locationID string, qty int, referenceID string) error {
	if qty <= 0 {
		return ErrStockBadQty
	}

	now := time.Now().UTC()
	mg := StockMovement{
		ID:             newID(),
		StoreID:        storeID,
		ProductID:      productID,
		LocationID:     &locationID,
		QuantityChange: -qty,
		Type:           MovementTypeSale,
		ReferenceID:    &referenceID,
		Note:           "sale deduction",
		CreatedBy:      actor.UserID,
		CreatedAt:      now,
	}

	if _, err := s.repo.Create(ctx, mg); err != nil {
		return err
	}

	if err := s.repo.UpsertStock(ctx, storeID, productID, locationID, -qty); err != nil {
		return err
	}

	return nil
}

// ListMovements lists stock movements for a store
func (s Service) ListMovements(ctx context.Context, actor auth.Claims, storeID string, q ListMovementsQuery) (MovementResponse, error) {
	if strings.TrimSpace(storeID) == "" {
		return MovementResponse{}, fmt.Errorf("storeID is required")
	}
	allowed, err := s.canManage(ctx, storeID, actor.UserID, actor.Role)
	if err != nil {
		return MovementResponse{}, err
	}
	if !allowed {
		return MovementResponse{}, ErrStockForbidden
	}
	return s.repo.ListByStore(ctx, storeID, q)
}

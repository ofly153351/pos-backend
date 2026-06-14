// Package provisioning is the single source of truth for a store's default
// warehouse ("คลังหลัก") and default sale-point location ("หน้าร้าน") — Phase W1.
//
// Every active store-creation path and the startup backfill route through
// EnsureDefaults / EnsureDefaultsTx, so the default-provisioning logic lives in
// exactly one place rather than being duplicated across handlers.
//
// Identity model (see migration 037): defaults are identified by the boolean flags
// warehouses.is_default and locations.is_default_sale, each enforced "one per store"
// by a partial UNIQUE index. The flag — never the Thai display name — is the
// authoritative identifier, so a store may rename คลังหลัก/หน้าร้าน freely. A reserved
// internal code (ReservedDefault*Code) gives a second, deterministic, store-unique
// identifier for records THIS package creates, used only for self-healing.
package provisioning

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"gorm.io/gorm"

	"pos-backend/internal/idgen"
)

// isUniqueViolation reports whether err is a Postgres unique_violation (SQLSTATE
// 23505). Used as a defensive backstop: the per-store FOR UPDATE lock already
// serializes provisioning, but if a default were ever created outside this package
// (bypassing the lock), the partial UNIQUE index would still reject a duplicate — and
// we re-resolve to the existing default instead of surfacing a raw error.
func isUniqueViolation(err error) bool {
	return err != nil && strings.Contains(err.Error(), "SQLSTATE 23505")
}

const (
	// ReservedDefaultWarehouseCode / ReservedDefaultSaleLocationCode are the stable,
	// language-independent internal codes assigned ONLY to records this package
	// creates. They are deterministic and unique per store (warehouses UNIQUE(store_id,
	// code); locations UNIQUE(store_id, warehouse_id, code)) and are never derived from
	// the Thai display names below — renaming the display name keeps these codes (and
	// the is_default/is_default_sale flags) intact.
	ReservedDefaultWarehouseCode    = "__default__"
	ReservedDefaultSaleLocationCode = "__pos_default__"

	// Default user-facing names. Users may rename these afterwards without breaking
	// the default relationship (identity is the flag, not the name).
	DefaultWarehouseName    = "คลังหลัก"
	DefaultSaleLocationName = "หน้าร้าน"
)

var (
	// ErrStoreNotFound is returned when provisioning is asked to operate on a store id
	// that does not exist.
	ErrStoreNotFound = errors.New("provisioning: store not found")

	// ErrInvalidDefaultWarehouse / ErrInvalidDefaultSaleLocation are returned when a row
	// is FLAGGED as the store's default but violates the W1 invariant (a default
	// warehouse must be is_active; a default sale location must be is_active AND
	// is_sale_point). The app guards prevent reaching these states normally, so this only
	// happens via direct SQL / external corruption. Provisioning refuses to silently use
	// the invalid row, and does NOT create a duplicate (the partial unique index still
	// marks the invalid row as default). RECOVERY: reactivate / restore the row, or clear
	// its is_default(/_sale) flag, then re-provision.
	ErrInvalidDefaultWarehouse    = errors.New("provisioning: store's flagged default warehouse is inactive; reactivate it or clear is_default, then re-provision")
	ErrInvalidDefaultSaleLocation = errors.New("provisioning: store's flagged default sale location is inactive or not a sale point; fix it or clear is_default_sale, then re-provision")
)

// Defaults is the resolved default structure for a store.
type Defaults struct {
	WarehouseID      string
	WarehouseName    string
	SaleLocationID   string
	SaleLocationName string
}

// EnsureDefaults idempotently provisions a store's default warehouse and default sale
// location in its OWN transaction. Safe to call repeatedly and concurrently — calling
// it again returns the same records and never creates duplicates.
func EnsureDefaults(ctx context.Context, db *gorm.DB, storeID string) (Defaults, error) {
	var out Defaults
	err := db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		d, err := EnsureDefaultsTx(ctx, tx, storeID)
		if err != nil {
			return err
		}
		out = d
		return nil
	})
	return out, err
}

// EnsureDefaultsTx is EnsureDefaults but runs inside a transaction the CALLER owns, so
// new-store creation can provision defaults atomically alongside the store row (if this
// fails, the caller's rollback unwinds the whole store creation — no half-configured
// store, and the failure is surfaced, not hidden).
func EnsureDefaultsTx(ctx context.Context, tx *gorm.DB, storeID string) (Defaults, error) {
	// Serialize provisioning per store with a row lock: two concurrent callers
	// (parallel onboarding requests, or several app instances running the startup
	// backfill) take turns. The second blocks until the first commits, then observes
	// the now-existing defaults and returns them — so no duplicate default is created
	// and the partial UNIQUE indexes are never violated at the user's expense.
	var lockedID string
	if err := tx.WithContext(ctx).
		Raw("SELECT id FROM stores WHERE id = ? FOR UPDATE", storeID).
		Scan(&lockedID).Error; err != nil {
		return Defaults{}, err
	}
	if lockedID == "" {
		return Defaults{}, ErrStoreNotFound
	}

	whID, whName, err := resolveDefaultWarehouse(ctx, tx, storeID)
	if err != nil {
		return Defaults{}, err
	}
	locID, locName, err := resolveDefaultSaleLocation(ctx, tx, storeID, whID)
	if err != nil {
		return Defaults{}, err
	}
	return Defaults{
		WarehouseID:      whID,
		WarehouseName:    whName,
		SaleLocationID:   locID,
		SaleLocationName: locName,
	}, nil
}

type namedRow struct {
	ID          string `gorm:"column:id"`
	Name        string `gorm:"column:name"`
	IsActive    bool   `gorm:"column:is_active"`
	IsSalePoint bool   `gorm:"column:is_sale_point"`
}

// resolveDefaultWarehouse returns the store's default warehouse, creating or adopting
// one deterministically. Resolution order (never an arbitrary first-row pick):
//
//  1. an already-flagged default warehouse (authoritative)
//  2. exactly one active warehouse  → adopt it (set the flag only; data untouched)
//  3. a prior provisioned warehouse found by its reserved code (self-heal the flag)
//  4. otherwise create a dedicated "คลังหลัก"
func resolveDefaultWarehouse(ctx context.Context, tx *gorm.DB, storeID string) (string, string, error) {
	// 1. Authoritative flag. A flagged default warehouse is valid only while active —
	//    refuse an inactive flagged row (only reachable via direct SQL) instead of
	//    silently using it or creating a duplicate the partial unique index would block.
	var w namedRow
	err := tx.WithContext(ctx).Table("warehouses").
		Select("id, name, is_active").
		Where("store_id = ? AND is_default = TRUE", storeID).
		Take(&w).Error
	if err == nil {
		if !w.IsActive {
			return "", "", fmt.Errorf("%w: warehouse %s", ErrInvalidDefaultWarehouse, w.ID)
		}
		return w.ID, w.Name, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return "", "", err
	}

	// 2. Exactly one active warehouse → adopt it.
	var active []namedRow
	if err := tx.WithContext(ctx).Table("warehouses").
		Select("id, name").
		Where("store_id = ? AND is_active = TRUE", storeID).
		Order("created_at ASC").
		Find(&active).Error; err != nil {
		return "", "", err
	}
	if len(active) == 1 {
		if err := tx.WithContext(ctx).Table("warehouses").
			Where("id = ?", active[0].ID).
			Update("is_default", true).Error; err != nil {
			return "", "", err
		}
		return active[0].ID, active[0].Name, nil
	}

	// 3. Self-heal: a record we previously created, by its stable reserved code. Only an
	//    active one is adopted, so self-healing never produces an invalid default.
	err = tx.WithContext(ctx).Table("warehouses").
		Select("id, name").
		Where("store_id = ? AND code = ? AND is_active = TRUE", storeID, ReservedDefaultWarehouseCode).
		Take(&w).Error
	if err == nil {
		if uerr := tx.WithContext(ctx).Table("warehouses").
			Where("id = ?", w.ID).
			Update("is_default", true).Error; uerr != nil {
			return "", "", uerr
		}
		return w.ID, w.Name, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return "", "", err
	}

	// 4. Create a dedicated default warehouse. Zero or several-ambiguous warehouses all
	//    land here — we never silently pick one of several existing warehouses.
	id := idgen.Generate(idgen.PrefixWarehouse)
	now := time.Now().UTC()
	if err := tx.WithContext(ctx).Table("warehouses").Create(map[string]any{
		"id":         id,
		"store_id":   storeID,
		"name":       DefaultWarehouseName,
		"code":       ReservedDefaultWarehouseCode,
		"is_active":  true,
		"is_default": true,
		"created_at": now,
		"updated_at": now,
	}).Error; err != nil {
		// Defensive: a default created outside the store lock would trip the partial
		// UNIQUE index — re-resolve to it rather than leak a raw 23505.
		if isUniqueViolation(err) {
			var existing namedRow
			if e := tx.WithContext(ctx).Table("warehouses").
				Select("id, name").
				Where("store_id = ? AND is_default = TRUE", storeID).
				Take(&existing).Error; e == nil {
				return existing.ID, existing.Name, nil
			}
		}
		return "", "", err
	}
	return id, DefaultWarehouseName, nil
}

// resolveDefaultSaleLocation returns the store's default sale-point location, creating
// or adopting one deterministically. Resolution order:
//
//  1. an already-flagged default sale location (authoritative)
//  2. a prior provisioned location found by its reserved code (self-heal the flag)
//  3. exactly one active sale-point location → adopt it (store-scoped, so valid)
//  4. otherwise (zero, or several ambiguous sale points) create a dedicated "หน้าร้าน"
//     under the resolved default warehouse; existing sale points are left untouched.
func resolveDefaultSaleLocation(ctx context.Context, tx *gorm.DB, storeID, defaultWarehouseID string) (string, string, error) {
	// 1. Authoritative flag. A flagged default sale location is valid only while it is
	//    active AND still a sale point — refuse an invalid flagged row (only reachable via
	//    direct SQL) instead of silently using it or creating a duplicate the partial
	//    unique index would block.
	var l namedRow
	err := tx.WithContext(ctx).Table("locations").
		Select("id, name, is_active, is_sale_point").
		Where("store_id = ? AND is_default_sale = TRUE", storeID).
		Take(&l).Error
	if err == nil {
		if !l.IsActive || !l.IsSalePoint {
			return "", "", fmt.Errorf("%w: location %s", ErrInvalidDefaultSaleLocation, l.ID)
		}
		return l.ID, l.Name, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return "", "", err
	}

	// 2. Self-heal by reserved code. Only an active sale-point row is adopted, so
	//    self-healing never produces an invalid default.
	err = tx.WithContext(ctx).Table("locations").
		Select("id, name").
		Where("store_id = ? AND code = ? AND is_active = TRUE AND is_sale_point = TRUE", storeID, ReservedDefaultSaleLocationCode).
		Take(&l).Error
	if err == nil {
		if uerr := tx.WithContext(ctx).Table("locations").
			Where("id = ?", l.ID).
			Update("is_default_sale", true).Error; uerr != nil {
			return "", "", uerr
		}
		return l.ID, l.Name, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return "", "", err
	}

	// 3. Exactly one active sale-point location → adopt it. Two or more: do NOT guess.
	var salePoints []namedRow
	if err := tx.WithContext(ctx).Table("locations").
		Select("id, name").
		Where("store_id = ? AND is_sale_point = TRUE AND is_active = TRUE", storeID).
		Order("created_at ASC").
		Find(&salePoints).Error; err != nil {
		return "", "", err
	}
	if len(salePoints) == 1 {
		if err := tx.WithContext(ctx).Table("locations").
			Where("id = ?", salePoints[0].ID).
			Update("is_default_sale", true).Error; err != nil {
			return "", "", err
		}
		return salePoints[0].ID, salePoints[0].Name, nil
	}

	// 4. Create a dedicated default sale location under the default warehouse.
	id := idgen.Generate(idgen.PrefixLocation)
	now := time.Now().UTC()
	if err := tx.WithContext(ctx).Table("locations").Create(map[string]any{
		"id":              id,
		"store_id":        storeID,
		"warehouse_id":    defaultWarehouseID,
		"name":            DefaultSaleLocationName,
		"code":            ReservedDefaultSaleLocationCode,
		"is_sale_point":   true,
		"is_active":       true,
		"is_default_sale": true,
		"created_at":      now,
		"updated_at":      now,
	}).Error; err != nil {
		// Defensive: a default created outside the store lock would trip the partial
		// UNIQUE index — re-resolve to it rather than leak a raw 23505.
		if isUniqueViolation(err) {
			var existing namedRow
			if e := tx.WithContext(ctx).Table("locations").
				Select("id, name").
				Where("store_id = ? AND is_default_sale = TRUE", storeID).
				Take(&existing).Error; e == nil {
				return existing.ID, existing.Name, nil
			}
		}
		return "", "", err
	}
	return id, DefaultSaleLocationName, nil
}

// BackfillAllStores ensures every existing store has its default warehouse and default
// sale location. Best-effort and idempotent: a per-store failure is logged and the loop
// continues so one bad store never blocks server startup. Run once at boot AFTER
// migrations, by which point most stores already have defaults and the work is a no-op.
func BackfillAllStores(ctx context.Context, db *gorm.DB) {
	var storeIDs []string
	if err := db.WithContext(ctx).Raw(`
		SELECT s.id FROM stores s
		WHERE NOT EXISTS (SELECT 1 FROM warehouses w WHERE w.store_id = s.id AND w.is_default = TRUE)
		   OR NOT EXISTS (SELECT 1 FROM locations  l WHERE l.store_id = s.id AND l.is_default_sale = TRUE)
		ORDER BY s.id
	`).Scan(&storeIDs).Error; err != nil {
		log.Printf("provisioning backfill: failed to list stores needing defaults: %v", err)
		return
	}
	if len(storeIDs) == 0 {
		return
	}
	log.Printf("provisioning backfill: %d store(s) need a default warehouse/sale location", len(storeIDs))
	provisioned := 0
	for _, id := range storeIDs {
		if _, err := EnsureDefaults(ctx, db, id); err != nil {
			log.Printf("provisioning backfill: store %s failed (skipped): %v", id, err)
			continue
		}
		provisioned++
	}
	log.Printf("provisioning backfill: provisioned %d/%d store(s)", provisioned, len(storeIDs))
}

// BackfillProductDefaultLocations assigns the store's default sale location to products
// that still have a NULL default_location_id (Phase W2 §5). Metadata-only: it never
// moves stock, creates stock rows or movements, and never touches a product that already
// has a default_location_id — including an invalid/cross-store one (those are reported,
// not silently overwritten). Best-effort, idempotent, store-isolated; safe on every boot.
// Runs after BackfillAllStores so every store already has a valid default sale location.
func BackfillProductDefaultLocations(ctx context.Context, db *gorm.DB) {
	var storeIDs []string
	if err := db.WithContext(ctx).Raw(
		`SELECT DISTINCT store_id FROM products WHERE default_location_id IS NULL ORDER BY store_id`,
	).Scan(&storeIDs).Error; err != nil {
		log.Printf("product-default backfill: failed to list stores: %v", err)
		return
	}
	if len(storeIDs) == 0 {
		return
	}
	assigned := 0
	for _, storeID := range storeIDs {
		var locID string
		err := db.WithContext(ctx).Table("locations").Select("id").
			Where("store_id = ? AND is_default_sale = TRUE AND is_active = TRUE AND is_sale_point = TRUE", storeID).
			Take(&locID).Error
		if err != nil {
			// No valid default sale location → skip (do not guess); the store's W1
			// provisioning should have created one. Logged for follow-up.
			log.Printf("product-default backfill: store %s has no valid default sale location, skipped: %v", storeID, err)
			continue
		}
		res := db.WithContext(ctx).Exec(
			`UPDATE products SET default_location_id = ?, updated_at = NOW() WHERE store_id = ? AND default_location_id IS NULL`,
			locID, storeID,
		)
		if res.Error != nil {
			log.Printf("product-default backfill: store %s update failed: %v", storeID, res.Error)
			continue
		}
		assigned += int(res.RowsAffected)
	}
	if assigned > 0 {
		log.Printf("product-default backfill: assigned default location to %d product(s)", assigned)
	}
}

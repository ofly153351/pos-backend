package stock_movement

// Phase W5 — authoritative stock-movement taxonomy.
//
// This is the single source of truth for every value written to stock_movements.type.
// It consolidates the codes that were previously scattered across modules (sale,
// creditsale, warehouse_receipt, stock_movement, warehouse, provisioning) and records,
// for each, its signed direction, which stock aggregate it affects, the operational
// source, the reference_id meaning, whether it is system- or user-created, and the
// canonical Thai/English labels. Presentation layers (movement history, activity feed,
// filters, dashboards) should derive labels and signs from here rather than hard-coding.

// MovementDirection is the sign a movement applies to on-hand stock.
type MovementDirection string

const (
	DirectionIn      MovementDirection = "in"      // increases stock (quantity_change > 0)
	DirectionOut     MovementDirection = "out"     // decreases stock (quantity_change < 0)
	DirectionNeutral MovementDirection = "neutral" // sign depends on the operation (±)
)

// StockEffect names which product aggregate a movement shifts. A move between a sale
// point and storage changes ready/storage composition but not the grand total.
type StockEffect string

const (
	EffectReady       StockEffect = "ready"       // affects sale-point (ready_stock)
	EffectStorage     StockEffect = "storage"     // affects non-sale-point (storage_stock)
	EffectEither      StockEffect = "either"      // depends on the location's is_sale_point
	EffectTotalOnly   StockEffect = "total"       // changes the grand total (in/out of the store)
	EffectComposition StockEffect = "composition" // moves between ready/storage, total unchanged
)

// MovementTypeMeta is the descriptor for one stock_movements.type code.
type MovementTypeMeta struct {
	Code          string            `json:"code"`
	LabelTH       string            `json:"label_th"`
	LabelEN       string            `json:"label_en"`
	Direction     MovementDirection `json:"direction"`
	Effect        StockEffect       `json:"effect"`
	Source        string            `json:"source"`         // operational origin module
	ReferenceType string            `json:"reference_type"` // what reference_id points at ("" = none)
	SystemCreated bool              `json:"system_created"` // true = written by the system, not a user action
	ShowInFeed    bool              `json:"show_in_feed"`   // surfaced in dashboards / activity feed
}

// MovementTaxonomy is the ordered authoritative list. Keep in sync with the writers.
var MovementTaxonomy = []MovementTypeMeta{
	{MovementTypeSale, "ขาย", "Sale", DirectionOut, EffectReady, "sale (POS checkout)", "sale.id", false, true},
	{MovementTypeReturn, "คืนสินค้า", "Return", DirectionIn, EffectReady, "credit-sale cancellation", "credit_sale.id", false, true},
	{MovementTypeIn, "รับเข้า", "Stock In", DirectionIn, EffectEither, "purchase / goods receipt", "warehouse_receipt.id", false, true},
	{MovementTypeOut, "นำออก", "Stock Out", DirectionOut, EffectEither, "manual stock issue", "", true, true},
	{MovementTypeAdjust, "ปรับปรุง", "Adjustment", DirectionNeutral, EffectEither, "manual physical adjustment", "", false, true},
	{MovementTypeCountCorrection, "ปรับจากการนับสต็อก", "Count Correction", DirectionNeutral, EffectEither, "stock count apply", "stock_count_session.id", false, true},
	{MovementTypeTransferOut, "โอนออก", "Transfer Out", DirectionOut, EffectComposition, "location transfer (source leg)", "stock_transfer.id", false, true},
	{MovementTypeTransferIn, "โอนเข้า", "Transfer In", DirectionIn, EffectComposition, "location transfer (destination leg)", "stock_transfer.id", false, true},
	{MovementTypeTransfer, "โอนย้าย (เดิม)", "Transfer (legacy)", DirectionNeutral, EffectComposition, "legacy warehouse transfer (disabled)", "", false, false},
	{MovementTypeAllocate, "จัดสรรจากคลัง", "Allocate", DirectionIn, EffectEither, "warehouse_inventory → stocks", "warehouse_inventory.id", true, false},
	// Initial/seed stock is NOT a distinct type: it is written as type=IN with
	// reason="OPENING_BALANCE" (see reasons.go), so it is covered by the IN entry above.
}

// movementMetaByCode indexes the taxonomy for O(1) lookup.
var movementMetaByCode = func() map[string]MovementTypeMeta {
	m := make(map[string]MovementTypeMeta, len(MovementTaxonomy))
	for _, t := range MovementTaxonomy {
		m[t.Code] = t
	}
	return m
}()

// MovementMeta returns the descriptor for a movement type code and whether it is known.
func MovementMeta(code string) (MovementTypeMeta, bool) {
	meta, ok := movementMetaByCode[code]
	return meta, ok
}

// A logical Transfer writes TWO rows (TRANSFER_OUT + TRANSFER_IN) that share one
// reference_id (stock_transfer.id). To count transfers as one business event, dashboards
// must count the shared reference once — e.g. count only the TRANSFER_IN (positive) leg,
// or COUNT(DISTINCT reference_id). IsTransferInLeg marks the canonical countable leg.
func IsTransferInLeg(code string) bool { return code == MovementTypeTransferIn }

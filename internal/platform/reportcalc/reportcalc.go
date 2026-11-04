// Package reportcalc is the single definition of report figures (B14), shared
// by finance, dashboard and warehouse reports so equal scopes give equal totals.
//
// Owner-approved basis (2026-11-04):
//   - Revenue is recognised from the sales table only, at sold_at, pre-VAT,
//     excluding voided sales and loans (credit_sales.type='loan'). Documents
//     (invoices, tax invoices, receipts, credit notes) are paperwork, not revenue.
//   - Refunds (sale_returns) reduce revenue by their pre-VAT share.
//   - COGS uses the cost snapshotted at sale time (sale_items.unit_cost). A line
//     without it is reported as missing — never silently valued at today's cost
//     or at a selling price. Returned quantities reverse their COGS.
//   - Inventory value = positive on-hand quantity (stocks) × current cost_price
//     of non-deleted products; products without cost are counted separately.
//
// All SQL here aliases sales as s, sale_items as si.
package reportcalc

// NotLoanSale excludes stock-borrow sales (they are not revenue).
const NotLoanSale = `NOT EXISTS (SELECT 1 FROM credit_sales cs WHERE cs.sale_id = s.id AND cs.type = 'loan')`

// NetSales is the pre-VAT revenue of one sale row.
const NetSales = `(s.total_amount - COALESCE(s.vat_amount, 0))`

// RefundPreVAT is the pre-VAT part of a refund row sr joined to its sale s.
const RefundPreVAT = `(sr.refund_amount * CASE WHEN s.total_amount > 0 THEN (s.total_amount - COALESCE(s.vat_amount, 0)) / s.total_amount ELSE 1 END)`

// LineCOGS values a sold line at its sale-time cost only.
const LineCOGS = `(si.quantity * COALESCE(si.unit_cost, 0))`

// LineMissingCost flags a sold line with no sale-time cost.
const LineMissingCost = `(CASE WHEN COALESCE(si.unit_cost, 0) <= 0 THEN 1 ELSE 0 END)`

// ReturnedCOGS reverses cost for returned quantities (sri joined to si).
const ReturnedCOGS = `(sri.quantity * COALESCE(si.unit_cost, 0))`

// InventoryValueSQL returns inventory_value, priced_units, missing_cost_products
// for one store (positional arg: store id).
const InventoryValueSQL = `
SELECT
	COALESCE(SUM(st.quantity * COALESCE(p.cost_price, 0)) FILTER (WHERE st.quantity > 0), 0) AS inventory_value,
	COALESCE(SUM(st.quantity) FILTER (WHERE st.quantity > 0), 0)                             AS on_hand_units,
	COUNT(DISTINCT p.id) FILTER (WHERE st.quantity > 0 AND COALESCE(p.cost_price, 0) <= 0)  AS missing_cost_products
FROM stocks st
JOIN products p ON p.id = st.product_id
WHERE st.store_id = ? AND p.deleted_at IS NULL`

-- 046_products_deleted_at.sql
-- Soft-delete marker for products.
--
-- This is DISTINCT from is_active: is_active is the deactivate/reactivate status that
-- intentionally keeps a product visible (and reactivatable) on the product-management
-- page. deleted_at marks a product as DELETED — hidden from every active product view
-- and selector — while its row and ALL historical references (sales, sale_items,
-- receiving, stock_movements, transfers, documents) are preserved for reporting and
-- foreign-key integrity. Deletion never destroys operational data.
--
-- Additive + idempotent (nullable column, IF NOT EXISTS); no view is recreated — active
-- product queries join products and filter `deleted_at IS NULL`.
ALTER TABLE products ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMPTZ;

-- Partial index keeps the common "active (non-deleted) products in a store" lookups fast;
-- deleted rows are excluded from the index entirely.
CREATE INDEX IF NOT EXISTS idx_products_store_not_deleted
    ON products (store_id)
    WHERE deleted_at IS NULL;

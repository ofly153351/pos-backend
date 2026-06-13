-- 030_purchase_order_number_unique.sql
-- P2.6: enforce per-store uniqueness of purchase-order numbers.
--
-- order_number was NOT NULL but had no UNIQUE constraint (001_schema.sql), so two
-- concurrent CreatePO calls for the same store/day could both derive the same
-- sequence (GetPODailySequence COUNTs outside the insert tx) and insert duplicate
-- numbers silently. The number is generated per store, so the key is per store:
-- two different stores legitimately produce PO-20260613-00001 on the same day.
--
-- Idempotent: CREATE UNIQUE INDEX IF NOT EXISTS. Runs inside the migrator's tx.
-- The de-dup below renumbers any pre-existing collisions so the unique index can
-- build even on a database that already accumulated duplicates; on a clean DB it
-- is a no-op. Suffix is "-D<rn>" on the later rows (kept, never deleted).

WITH ranked AS (
    SELECT id,
           order_number,
           ROW_NUMBER() OVER (
               PARTITION BY store_id, order_number ORDER BY created_at, id
           ) AS rn
    FROM purchase_orders
)
UPDATE purchase_orders po
SET order_number = ranked.order_number || '-D' || ranked.rn
FROM ranked
WHERE po.id = ranked.id
  AND ranked.rn > 1;

CREATE UNIQUE INDEX IF NOT EXISTS uq_purchase_orders_store_order_number
    ON purchase_orders (store_id, order_number);

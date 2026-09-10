-- 056: immutable promotion attribution snapshot
--
-- Promotion usage history must remain explainable after a campaign is edited or
-- soft-deleted. These nullable columns preserve the campaign identity at the
-- moment the sale used it; old rows remain NULL and are intentionally historical
-- records from before this migration.
ALTER TABLE promotion_usages
    ADD COLUMN IF NOT EXISTS promotion_name TEXT,
    ADD COLUMN IF NOT EXISTS promotion_type TEXT,
    ADD COLUMN IF NOT EXISTS revenue_amount NUMERIC(14,2);

-- Best-effort migration for legacy usage rows. Exact historical values cannot
-- be reconstructed because the old schema stored no campaign snapshot; new rows
-- are immutable snapshots written by the sale transaction.
UPDATE promotion_usages pu
SET promotion_name = p.name,
    promotion_type = p.type
FROM promotions p
WHERE pu.promotion_name IS NULL
  AND pu.promotion_type IS NULL
  AND p.id = pu.promotion_id
  AND p.store_id = pu.store_id;

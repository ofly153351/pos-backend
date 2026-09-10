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

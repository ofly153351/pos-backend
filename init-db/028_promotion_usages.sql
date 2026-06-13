-- 028: Per-application promotion usage ledger — the reliable analytics source.
--
-- One row is written for each promotion applied on a sale, with the discount
-- attributed to it. This is the audit trail behind promotions.usage_count /
-- discount_given_total and lets reporting reconstruct which sales used a promo.
--
-- promotion_id is intentionally NOT a foreign key: promotions can be soft-deleted
-- (027) while their historical usage rows must survive. store_id cascades.
--
-- Idempotent (CREATE TABLE/INDEX IF NOT EXISTS).
CREATE TABLE IF NOT EXISTS promotion_usages (
    id              TEXT PRIMARY KEY,
    store_id        TEXT NOT NULL REFERENCES stores(id) ON DELETE CASCADE,
    promotion_id    TEXT NOT NULL,
    sale_id         TEXT NOT NULL,
    discount_amount NUMERIC(14,2) NOT NULL DEFAULT 0,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_promotion_usages_promo ON promotion_usages(promotion_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_promotion_usages_store ON promotion_usages(store_id, created_at DESC);

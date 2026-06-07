-- Per-user POS product-card display preferences (JSON blob).
ALTER TABLE users
    ADD COLUMN IF NOT EXISTS card_settings JSONB NOT NULL DEFAULT '{}'::jsonb;

CREATE TABLE IF NOT EXISTS store_bank_accounts (
    id          VARCHAR(30) PRIMARY KEY,
    store_id    VARCHAR(30) NOT NULL REFERENCES stores(id) ON DELETE CASCADE,
    bank_code   VARCHAR(20) NOT NULL DEFAULT '',
    bank_name   VARCHAR(100) NOT NULL DEFAULT '',
    account_no  VARCHAR(50) NOT NULL DEFAULT '',
    account_name VARCHAR(100) NOT NULL DEFAULT '',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_store_bank_accounts_store ON store_bank_accounts(store_id);

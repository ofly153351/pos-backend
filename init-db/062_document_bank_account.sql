ALTER TABLE documents
    ADD COLUMN IF NOT EXISTS bank_account_id VARCHAR(30)
    REFERENCES store_bank_accounts(id) ON DELETE SET NULL;

CREATE INDEX IF NOT EXISTS idx_documents_bank_account_id
    ON documents(bank_account_id);

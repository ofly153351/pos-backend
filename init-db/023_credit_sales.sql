-- 023_credit_sales.sql — Credit / loan receivables (Accounts Receivable ledger).
-- A credit sale ALSO creates a real `sales` row (payment_method='credit') so it
-- deducts stock and is recognized as accrual revenue by the finance module with
-- no finance changes. These tables track ONLY the receivable + collections; they
-- never duplicate revenue. Mirrors the proven invoices/invoice_payments shape.

CREATE TABLE IF NOT EXISTS credit_sales (
    id               TEXT          PRIMARY KEY,
    store_id         TEXT          NOT NULL REFERENCES stores(id) ON DELETE CASCADE,
    sale_id          TEXT          NOT NULL REFERENCES sales(id),
    customer_id      TEXT          NOT NULL REFERENCES customers(id),
    document_number  TEXT          NOT NULL DEFAULT '',
    type             TEXT          NOT NULL DEFAULT 'credit',
    total_amount     NUMERIC(14,2) NOT NULL DEFAULT 0,
    paid_amount      NUMERIC(14,2) NOT NULL DEFAULT 0,
    remaining_amount NUMERIC(14,2) NOT NULL DEFAULT 0,
    status           TEXT          NOT NULL DEFAULT 'pending',
    due_date         TEXT          NOT NULL DEFAULT '',
    note             TEXT          NOT NULL DEFAULT '',
    created_by       TEXT          NOT NULL DEFAULT '',
    cancelled_at     TEXT,
    created_at       TEXT          NOT NULL DEFAULT '',
    updated_at       TEXT          NOT NULL DEFAULT ''
);

CREATE INDEX IF NOT EXISTS idx_credit_sales_store    ON credit_sales(store_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_credit_sales_customer ON credit_sales(store_id, customer_id);

CREATE TABLE IF NOT EXISTS credit_payments (
    id             TEXT          PRIMARY KEY,
    credit_sale_id TEXT          NOT NULL REFERENCES credit_sales(id) ON DELETE CASCADE,
    store_id       TEXT          NOT NULL,
    amount         NUMERIC(14,2) NOT NULL CHECK (amount > 0),
    method         TEXT          NOT NULL DEFAULT 'cash',
    note           TEXT          NOT NULL DEFAULT '',
    paid_at        TEXT          NOT NULL DEFAULT '',
    created_by     TEXT          NOT NULL DEFAULT ''
);

CREATE INDEX IF NOT EXISTS idx_credit_payments_sale ON credit_payments(credit_sale_id);

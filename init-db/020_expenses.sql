-- 020_expenses.sql — Expense records (Finance Phase 2a)
-- expense_categories: per-store configurable categories (soft-disable via is_active)
-- expenses: store-scoped expense records (soft-delete via voided_at)

CREATE TABLE IF NOT EXISTS expense_categories (
    id          TEXT        PRIMARY KEY,
    store_id    TEXT        NOT NULL REFERENCES stores(id) ON DELETE CASCADE,
    name        TEXT        NOT NULL,
    is_active   BOOLEAN     NOT NULL DEFAULT TRUE,
    sort_order  INTEGER     NOT NULL DEFAULT 0,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (store_id, name)
);

CREATE INDEX IF NOT EXISTS idx_expense_categories_store ON expense_categories(store_id);

CREATE TABLE IF NOT EXISTS expenses (
    id             TEXT          PRIMARY KEY,
    store_id       TEXT          NOT NULL REFERENCES stores(id) ON DELETE CASCADE,
    expense_date   DATE          NOT NULL,
    category_id    TEXT          NOT NULL REFERENCES expense_categories(id),
    description    TEXT          NOT NULL,
    amount         NUMERIC(14,2) NOT NULL CHECK (amount >= 0),
    payment_method TEXT          NOT NULL DEFAULT 'cash',
    note           TEXT,
    status         TEXT          NOT NULL DEFAULT 'approved',
    created_by     TEXT          NOT NULL REFERENCES users(id),
    approved_by    TEXT          REFERENCES users(id),
    approved_at    TIMESTAMPTZ,
    voided_at      TIMESTAMPTZ,
    voided_by      TEXT          REFERENCES users(id),
    created_at     TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    updated_at     TIMESTAMPTZ   NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_expenses_store_date   ON expenses(store_id, expense_date DESC);
CREATE INDEX IF NOT EXISTS idx_expenses_store_cat    ON expenses(store_id, category_id);
CREATE INDEX IF NOT EXISTS idx_expenses_store_status ON expenses(store_id, status) WHERE voided_at IS NULL;

-- Seed the 9 default categories for every EXISTING store (idempotent).
-- New stores are lazy-seeded by the expense service on first category list.
INSERT INTO expense_categories (id, store_id, name, sort_order)
SELECT 'exc-' || lpad((90000000 + row_number() OVER ())::text, 8, '0'), s.id, d.name, d.ord
FROM stores s
CROSS JOIN (VALUES
    ('ค่าเช่า', 1),
    ('ค่าน้ำค่าไฟ', 2),
    ('ค่าน้ำมัน', 3),
    ('ค่าขนส่ง', 4),
    ('เงินเดือน', 5),
    ('ค่าซ่อมบำรุง', 6),
    ('อุปกรณ์สำนักงาน', 7),
    ('การตลาด', 8),
    ('อื่นๆ', 9)
) AS d(name, ord)
ON CONFLICT (store_id, name) DO NOTHING;

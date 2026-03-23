CREATE TABLE IF NOT EXISTS invoices (
    id TEXT PRIMARY KEY,
    store_id TEXT NOT NULL REFERENCES stores(id) ON DELETE CASCADE,
    invoice_number TEXT NOT NULL UNIQUE,
    customer_id TEXT NOT NULL REFERENCES customers(id) ON DELETE RESTRICT,
    cashier_user_id TEXT NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    status TEXT NOT NULL CHECK (status IN ('unpaid', 'partially_paid', 'paid', 'cancelled')),
    payment_method TEXT,
    note TEXT,
    due_at TIMESTAMPTZ,
    customer_level INTEGER,
    network_discount_percent NUMERIC(5,2) NOT NULL DEFAULT 0 CHECK (network_discount_percent >= 0 AND network_discount_percent <= 100),
    total_items INTEGER NOT NULL CHECK (total_items > 0),
    subtotal_amount NUMERIC(12,2) NOT NULL CHECK (subtotal_amount >= 0),
    discount_amount NUMERIC(12,2) NOT NULL CHECK (discount_amount >= 0),
    total_amount NUMERIC(12,2) NOT NULL CHECK (total_amount >= 0),
    paid_amount NUMERIC(12,2) NOT NULL DEFAULT 0 CHECK (paid_amount >= 0),
    remaining_amount NUMERIC(12,2) NOT NULL CHECK (remaining_amount >= 0),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS invoice_items (
    id TEXT PRIMARY KEY,
    invoice_id TEXT NOT NULL REFERENCES invoices(id) ON DELETE CASCADE,
    product_id TEXT NOT NULL REFERENCES products(id) ON DELETE RESTRICT,
    product_name TEXT NOT NULL,
    sku TEXT,
    unit_type TEXT,
    quantity INTEGER NOT NULL CHECK (quantity > 0),
    unit_price NUMERIC(12,2) NOT NULL CHECK (unit_price >= 0),
    discount_type TEXT,
    discount_value NUMERIC(12,4),
    discount_amount_per_unit NUMERIC(12,2) NOT NULL DEFAULT 0,
    line_subtotal NUMERIC(12,2) NOT NULL CHECK (line_subtotal >= 0),
    line_discount_total NUMERIC(12,2) NOT NULL CHECK (line_discount_total >= 0),
    line_total NUMERIC(12,2) NOT NULL CHECK (line_total >= 0),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS invoice_payments (
    id TEXT PRIMARY KEY,
    invoice_id TEXT NOT NULL REFERENCES invoices(id) ON DELETE CASCADE,
    paid_amount NUMERIC(12,2) NOT NULL CHECK (paid_amount > 0),
    payment_method TEXT NOT NULL,
    note TEXT,
    paid_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_invoices_store_id ON invoices (store_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_invoices_customer_id ON invoices (customer_id);
CREATE INDEX IF NOT EXISTS idx_invoice_items_invoice_id ON invoice_items (invoice_id);
CREATE INDEX IF NOT EXISTS idx_invoice_payments_invoice_id ON invoice_payments (invoice_id);

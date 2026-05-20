-- Parked bills (parked/held POS transactions)
DROP TABLE IF EXISTS parked_bill_items;
DROP TABLE IF EXISTS parked_bills;
CREATE TABLE IF NOT EXISTS parked_bills (
    id TEXT PRIMARY KEY,
    store_id TEXT NOT NULL REFERENCES stores(id) ON DELETE CASCADE,
    cashier_user_id TEXT NOT NULL,
    label TEXT DEFAULT '',
    note TEXT DEFAULT '',
    bill_discount_amount DECIMAL(12,2) DEFAULT 0,
    bill_discount_type TEXT DEFAULT 'amount',
    bill_discount_percent DECIMAL(5,2) DEFAULT 0,
    customer_id TEXT DEFAULT '',
    customer_settlement_mode TEXT DEFAULT 'cash_now',
    payment_method TEXT DEFAULT 'cash',
    vat_included BOOLEAN DEFAULT true,
    vat_percent DECIMAL(5,2) DEFAULT 7,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS parked_bill_items (
    id TEXT PRIMARY KEY,
    parked_bill_id TEXT NOT NULL REFERENCES parked_bills(id) ON DELETE CASCADE,
    product_id TEXT NOT NULL,
    product_name TEXT NOT NULL DEFAULT '',
    product_sku TEXT DEFAULT '',
    price DECIMAL(12,2) NOT NULL DEFAULT 0,
    quantity INT NOT NULL,
    discount_type TEXT,
    discount_value DECIMAL(12,2)
);

CREATE INDEX IF NOT EXISTS idx_parked_bills_store_id ON parked_bills(store_id);
CREATE INDEX IF NOT EXISTS idx_parked_bill_items_parked_bill_id ON parked_bill_items(parked_bill_id);

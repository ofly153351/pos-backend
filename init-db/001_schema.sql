-- ============================================================
-- POS System — Clean Schema (single file)
-- ============================================================

-- ── 1. USERS ─────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS users (
    id            TEXT        PRIMARY KEY,
    full_name     TEXT        NOT NULL,
    email         TEXT        NOT NULL UNIQUE,
    password_hash TEXT        NOT NULL,
    token_version BIGINT      NOT NULL DEFAULT 0,
    role          TEXT        NOT NULL CHECK (role IN ('platform_admin','owner','manager','cashier')),
    status        TEXT        NOT NULL DEFAULT 'active' CHECK (status IN ('active','disabled')),
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- ── 2. STORES ────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS stores (
    id            TEXT        PRIMARY KEY,
    owner_user_id TEXT        NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    name          TEXT        NOT NULL,
    logo_url      TEXT,
    phone         TEXT,
    address       TEXT,
    promptpay_id  TEXT,
    currency_code TEXT        NOT NULL DEFAULT 'THB',
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS store_members (
    id         TEXT        PRIMARY KEY,
    store_id   TEXT        NOT NULL REFERENCES stores(id) ON DELETE CASCADE,
    user_id    TEXT        NOT NULL REFERENCES users(id)  ON DELETE CASCADE,
    role       TEXT        NOT NULL CHECK (role IN ('owner','manager','cashier')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (store_id, user_id)
);

-- ── 3. SUBSCRIPTIONS ─────────────────────────────────────────
CREATE TABLE IF NOT EXISTS subscription_plans (
    id            TEXT           PRIMARY KEY,
    code          TEXT           NOT NULL UNIQUE,
    name          TEXT           NOT NULL,
    description   TEXT,
    duration_days INTEGER        NOT NULL CHECK (duration_days > 0),
    price_amount  NUMERIC(12,2)  NOT NULL CHECK (price_amount >= 0),
    currency_code TEXT           NOT NULL DEFAULT 'THB',
    is_active     BOOLEAN        NOT NULL DEFAULT TRUE,
    created_at    TIMESTAMPTZ    NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS store_subscriptions (
    id                   TEXT        PRIMARY KEY,
    store_id             TEXT        NOT NULL REFERENCES stores(id) ON DELETE CASCADE,
    plan_id              TEXT        NOT NULL REFERENCES subscription_plans(id) ON DELETE RESTRICT,
    status               TEXT        NOT NULL CHECK (status IN ('trialing','active','past_due','cancelled','expired')),
    current_period_start TIMESTAMPTZ NOT NULL,
    current_period_end   TIMESTAMPTZ NOT NULL,
    created_at           TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- ── 4. PRODUCT CATALOG ───────────────────────────────────────
CREATE TABLE IF NOT EXISTS product_types (
    id          TEXT        PRIMARY KEY,
    store_id    TEXT        NOT NULL REFERENCES stores(id) ON DELETE CASCADE,
    name        TEXT        NOT NULL,
    description TEXT,
    is_active   BOOLEAN     NOT NULL DEFAULT TRUE,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (store_id, name)
);

CREATE TABLE IF NOT EXISTS product_units (
    id          TEXT        PRIMARY KEY,
    store_id    TEXT        NOT NULL REFERENCES stores(id) ON DELETE CASCADE,
    name        TEXT        NOT NULL,
    description TEXT,
    is_active   BOOLEAN     NOT NULL DEFAULT TRUE,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (store_id, name)
);

CREATE TABLE IF NOT EXISTS product_brands (
    id         TEXT        PRIMARY KEY,
    store_id   TEXT        NOT NULL REFERENCES stores(id) ON DELETE CASCADE,
    name       TEXT        NOT NULL,
    is_active  BOOLEAN     NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (store_id, name)
);

CREATE TABLE IF NOT EXISTS products (
    id                      TEXT           PRIMARY KEY,
    store_id                TEXT           NOT NULL REFERENCES stores(id)        ON DELETE CASCADE,
    product_type_id         TEXT           REFERENCES product_types(id)          ON DELETE SET NULL,
    product_unit_id         TEXT           NOT NULL REFERENCES product_units(id) ON DELETE RESTRICT,
    brand_id                TEXT           REFERENCES product_brands(id)         ON DELETE SET NULL,
    name                    TEXT           NOT NULL,
    sku                     TEXT,
    barcode                 TEXT,
    product_code            TEXT,
    description             TEXT,
    storage_location        TEXT,
    base_price              NUMERIC(12,2)  NOT NULL CHECK (base_price >= 0),
    cost_price              NUMERIC(12,2)  NOT NULL DEFAULT 0,
    special_price           NUMERIC(12,2),
    special_price_start_at  TIMESTAMPTZ,
    special_price_end_at    TIMESTAMPTZ,
    image_url               TEXT,
    min_stock               INTEGER        NOT NULL DEFAULT 0 CHECK (min_stock >= 0),
    max_stock               INTEGER        CHECK (max_stock IS NULL OR max_stock >= 0),
    is_active               BOOLEAN        NOT NULL DEFAULT TRUE,
    created_at              TIMESTAMPTZ    NOT NULL DEFAULT NOW(),
    updated_at              TIMESTAMPTZ    NOT NULL DEFAULT NOW(),
    UNIQUE (store_id, sku),
    CONSTRAINT products_special_price_check CHECK (
        special_price IS NULL OR (special_price >= 0 AND special_price <= base_price)
    ),
    CONSTRAINT products_special_price_window_check CHECK (
        special_price_start_at IS NULL OR special_price_end_at IS NULL
        OR special_price_end_at >= special_price_start_at
    )
);

-- ── 5. CUSTOMERS ─────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS customers (
    id             TEXT        PRIMARY KEY,
    store_id       TEXT        NOT NULL REFERENCES stores(id) ON DELETE CASCADE,
    customer_level INTEGER     NOT NULL DEFAULT 1 CHECK (customer_level > 0),
    full_name      TEXT        NOT NULL,
    phone          TEXT,
    email          TEXT,
    address        TEXT,
    note           TEXT,
    is_active      BOOLEAN     NOT NULL DEFAULT TRUE,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS customer_level_discounts (
    store_id         TEXT           NOT NULL REFERENCES stores(id) ON DELETE CASCADE,
    level            INTEGER        NOT NULL CHECK (level > 0),
    discount_percent NUMERIC(5,2)   NOT NULL CHECK (discount_percent >= 0 AND discount_percent <= 100),
    created_at       TIMESTAMPTZ    NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMPTZ    NOT NULL DEFAULT NOW(),
    PRIMARY KEY (store_id, level)
);

-- ── 6. PURCHASING ────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS suppliers (
    id             TEXT        PRIMARY KEY,
    store_id       TEXT        NOT NULL REFERENCES stores(id) ON DELETE CASCADE,
    name           TEXT        NOT NULL,
    phone          TEXT,
    address        TEXT,
    tax_id         TEXT,
    contact_person TEXT,
    note           TEXT,
    is_active      BOOLEAN     NOT NULL DEFAULT TRUE,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS purchase_orders (
    id           TEXT          PRIMARY KEY,
    store_id     TEXT          NOT NULL REFERENCES stores(id)    ON DELETE CASCADE,
    supplier_id  TEXT          REFERENCES suppliers(id)          ON DELETE SET NULL,
    order_number TEXT          NOT NULL,
    status       TEXT          NOT NULL DEFAULT 'pending' CHECK (status IN ('pending','partial','completed','cancelled')),
    notes        TEXT,
    total_cost   NUMERIC(12,2) NOT NULL DEFAULT 0,
    received_at  TIMESTAMPTZ,
    created_at   TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ   NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS purchase_order_items (
    id                TEXT          PRIMARY KEY,
    purchase_order_id TEXT          NOT NULL REFERENCES purchase_orders(id) ON DELETE CASCADE,
    product_id        TEXT          NOT NULL REFERENCES products(id)        ON DELETE RESTRICT,
    quantity          INTEGER       NOT NULL DEFAULT 1 CHECK (quantity > 0),
    received_quantity INTEGER       NOT NULL DEFAULT 0 CHECK (received_quantity >= 0),
    unit_cost         NUMERIC(12,2) NOT NULL DEFAULT 0,
    line_total        NUMERIC(12,2) NOT NULL DEFAULT 0,
    created_at        TIMESTAMPTZ   NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS supplier_products (
    id             TEXT          PRIMARY KEY,
    supplier_id    TEXT          NOT NULL REFERENCES suppliers(id) ON DELETE CASCADE,
    product_id     TEXT          NOT NULL REFERENCES products(id)  ON DELETE CASCADE,
    supplier_sku   TEXT          DEFAULT '',
    supplier_price NUMERIC(12,2) DEFAULT 0,
    created_at     TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    updated_at     TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    UNIQUE (supplier_id, product_id)
);

-- ── 7. WAREHOUSES & LOCATIONS & STOCK ────────────────────────
CREATE TABLE IF NOT EXISTS warehouses (
    id                   TEXT        PRIMARY KEY,
    store_id             TEXT        NOT NULL REFERENCES stores(id) ON DELETE CASCADE,
    name                 TEXT        NOT NULL,
    code                 TEXT,
    address              TEXT,
    phone                TEXT,
    contact_name         TEXT,
    is_active            BOOLEAN     NOT NULL DEFAULT TRUE,
    source_store_id      TEXT        REFERENCES stores(id) ON DELETE SET NULL,
    source_warehouse_id  TEXT,
    created_at           TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at           TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (store_id, code)
);

CREATE TABLE IF NOT EXISTS locations (
    id           TEXT        PRIMARY KEY,
    store_id     TEXT        NOT NULL REFERENCES stores(id)     ON DELETE CASCADE,
    warehouse_id TEXT        NOT NULL REFERENCES warehouses(id) ON DELETE CASCADE,
    name         TEXT        NOT NULL,
    code         TEXT,
    is_sale_point BOOLEAN    NOT NULL DEFAULT FALSE,
    is_active    BOOLEAN     NOT NULL DEFAULT TRUE,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (store_id, warehouse_id, code)
);

CREATE TABLE IF NOT EXISTS stocks (
    id          TEXT        PRIMARY KEY,
    store_id    TEXT        NOT NULL REFERENCES stores(id)    ON DELETE CASCADE,
    product_id  TEXT        NOT NULL REFERENCES products(id)  ON DELETE CASCADE,
    location_id TEXT        NOT NULL REFERENCES locations(id) ON DELETE CASCADE,
    quantity    INTEGER     NOT NULL DEFAULT 0 CHECK (quantity >= 0),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (product_id, location_id)
);

CREATE TABLE IF NOT EXISTS warehouse_inventory (
    id                   TEXT        PRIMARY KEY,
    store_id             TEXT        NOT NULL REFERENCES stores(id)     ON DELETE CASCADE,
    warehouse_id         TEXT        NOT NULL REFERENCES warehouses(id) ON DELETE CASCADE,
    product_id           TEXT        NOT NULL REFERENCES products(id)   ON DELETE CASCADE,
    quantity             INTEGER     NOT NULL DEFAULT 0 CHECK (quantity >= 0),
    source_store_id      TEXT        REFERENCES stores(id) ON DELETE SET NULL,
    source_warehouse_id  TEXT,
    transferred_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at           TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at           TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (warehouse_id, product_id)
);

-- ── 8. PRODUCT VIEW (after stocks + locations exist) ─────────
CREATE OR REPLACE VIEW product_view AS
SELECT
    p.id,
    p.store_id,
    p.product_type_id,
    COALESCE(pt.name, '')  AS product_type_name,
    p.product_unit_id,
    COALESCE(pu.name, '')  AS product_unit_name,
    p.brand_id,
    COALESCE(pb.name, '')  AS brand_name,
    p.name,
    p.sku,
    p.barcode,
    p.product_code,
    p.description,
    p.storage_location,
    p.image_url,
    p.min_stock,
    p.max_stock,
    COALESCE((
        SELECT SUM(s.quantity)
        FROM stocks s
        JOIN locations l ON l.id = s.location_id AND l.is_sale_point = TRUE
        WHERE s.product_id = p.id
    ), 0)::integer AS total_stock,
    COALESCE((
        SELECT SUM(s.quantity)
        FROM stocks s
        WHERE s.product_id = p.id
    ), 0)::integer AS warehouse_stock,
    p.base_price,
    p.cost_price,
    p.special_price,
    p.special_price_start_at,
    p.special_price_end_at,
    p.is_active,
    p.created_at,
    p.updated_at
FROM products p
LEFT JOIN product_types  pt ON pt.id = p.product_type_id
LEFT JOIN product_units  pu ON pu.id = p.product_unit_id
LEFT JOIN product_brands pb ON pb.id = p.brand_id;

-- ── 10. SALES ────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS sales (
    id                      TEXT          PRIMARY KEY,
    store_id                TEXT          NOT NULL REFERENCES stores(id) ON DELETE CASCADE,
    sale_number             TEXT          NOT NULL UNIQUE,
    cashier_user_id         TEXT          NOT NULL REFERENCES users(id)  ON DELETE RESTRICT,
    customer_id             TEXT          REFERENCES customers(id)        ON DELETE SET NULL,
    customer_level          INTEGER,
    network_discount_percent NUMERIC(5,2) NOT NULL DEFAULT 0,
    status                  TEXT          NOT NULL CHECK (status IN ('completed')),
    payment_method          TEXT,
    note                    TEXT,
    total_items             INTEGER       NOT NULL CHECK (total_items > 0),
    subtotal_amount         NUMERIC(12,2) NOT NULL DEFAULT 0 CHECK (subtotal_amount >= 0),
    discount_amount         NUMERIC(12,2) NOT NULL DEFAULT 0 CHECK (discount_amount >= 0),
    bill_discount_amount    NUMERIC(12,2) NOT NULL DEFAULT 0,
    vat_included            BOOLEAN       NOT NULL DEFAULT TRUE,
    vat_percent             NUMERIC(5,2)  NOT NULL DEFAULT 7 CHECK (vat_percent >= 0 AND vat_percent <= 100),
    vat_amount              NUMERIC(12,2) NOT NULL DEFAULT 0 CHECK (vat_amount >= 0),
    total_amount            NUMERIC(12,2) NOT NULL CHECK (total_amount >= 0),
    paid_amount             NUMERIC(12,2) NOT NULL CHECK (paid_amount >= 0),
    change_amount           NUMERIC(12,2) NOT NULL,
    sold_at                 TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    created_at              TIMESTAMPTZ   NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS sale_items (
    id                     TEXT          PRIMARY KEY,
    sale_id                TEXT          NOT NULL REFERENCES sales(id)   ON DELETE CASCADE,
    product_id             TEXT          REFERENCES products(id)         ON DELETE SET NULL,
    product_name           TEXT          NOT NULL,
    sku                    TEXT,
    unit_type              TEXT,
    quantity               INTEGER       NOT NULL CHECK (quantity > 0),
    unit_price             NUMERIC(12,2) NOT NULL CHECK (unit_price >= 0),
    discount_type          TEXT,
    discount_value         NUMERIC(12,4),
    discount_amount_per_unit NUMERIC(12,2) NOT NULL DEFAULT 0,
    line_subtotal          NUMERIC(12,2) NOT NULL DEFAULT 0,
    line_discount_total    NUMERIC(12,2) NOT NULL DEFAULT 0,
    line_total             NUMERIC(12,2) NOT NULL CHECK (line_total >= 0),
    created_at             TIMESTAMPTZ   NOT NULL DEFAULT NOW()
);

-- ── 11. INVOICES ─────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS invoices (
    id                       TEXT          PRIMARY KEY,
    store_id                 TEXT          NOT NULL REFERENCES stores(id)    ON DELETE CASCADE,
    invoice_number           TEXT          NOT NULL UNIQUE,
    customer_id              TEXT          NOT NULL REFERENCES customers(id) ON DELETE RESTRICT,
    cashier_user_id          TEXT          NOT NULL REFERENCES users(id)     ON DELETE RESTRICT,
    status                   TEXT          NOT NULL CHECK (status IN ('unpaid','partially_paid','paid','cancelled')),
    payment_method           TEXT,
    note                     TEXT,
    due_at                   TIMESTAMPTZ,
    customer_level           INTEGER,
    network_discount_percent NUMERIC(5,2)  NOT NULL DEFAULT 0 CHECK (network_discount_percent >= 0 AND network_discount_percent <= 100),
    total_items              INTEGER       NOT NULL CHECK (total_items > 0),
    subtotal_amount          NUMERIC(12,2) NOT NULL CHECK (subtotal_amount >= 0),
    discount_amount          NUMERIC(12,2) NOT NULL CHECK (discount_amount >= 0),
    vat_included             BOOLEAN       NOT NULL DEFAULT TRUE,
    vat_percent              NUMERIC(5,2)  NOT NULL DEFAULT 7 CHECK (vat_percent >= 0 AND vat_percent <= 100),
    vat_amount               NUMERIC(12,2) NOT NULL DEFAULT 0 CHECK (vat_amount >= 0),
    total_amount             NUMERIC(12,2) NOT NULL CHECK (total_amount >= 0),
    paid_amount              NUMERIC(12,2) NOT NULL DEFAULT 0 CHECK (paid_amount >= 0),
    remaining_amount         NUMERIC(12,2) NOT NULL CHECK (remaining_amount >= 0),
    created_at               TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    updated_at               TIMESTAMPTZ   NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS invoice_items (
    id                     TEXT          PRIMARY KEY,
    invoice_id             TEXT          NOT NULL REFERENCES invoices(id) ON DELETE CASCADE,
    product_id             TEXT          REFERENCES products(id)          ON DELETE SET NULL,
    product_name           TEXT          NOT NULL,
    sku                    TEXT,
    unit_type              TEXT,
    quantity               INTEGER       NOT NULL CHECK (quantity > 0),
    unit_price             NUMERIC(12,2) NOT NULL CHECK (unit_price >= 0),
    discount_type          TEXT,
    discount_value         NUMERIC(12,4),
    discount_amount_per_unit NUMERIC(12,2) NOT NULL DEFAULT 0,
    line_subtotal          NUMERIC(12,2) NOT NULL CHECK (line_subtotal >= 0),
    line_discount_total    NUMERIC(12,2) NOT NULL CHECK (line_discount_total >= 0),
    line_total             NUMERIC(12,2) NOT NULL CHECK (line_total >= 0),
    created_at             TIMESTAMPTZ   NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS invoice_payments (
    id                 TEXT          PRIMARY KEY,
    invoice_id         TEXT          NOT NULL REFERENCES invoices(id) ON DELETE CASCADE,
    paid_amount        NUMERIC(12,2) NOT NULL CHECK (paid_amount > 0),
    payment_method     TEXT          NOT NULL,
    note               TEXT,
    proof_url          TEXT,
    proof_mime_type    TEXT,
    proof_file_name    TEXT,
    is_voided          BOOLEAN       NOT NULL DEFAULT FALSE,
    voided_at          TIMESTAMPTZ,
    voided_by_user_id  TEXT          REFERENCES users(id) ON DELETE SET NULL,
    void_reason        TEXT,
    paid_at            TIMESTAMPTZ   NOT NULL,
    created_at         TIMESTAMPTZ   NOT NULL DEFAULT NOW()
);

-- ── 12. PARKED BILLS ─────────────────────────────────────────
CREATE TABLE IF NOT EXISTS parked_bills (
    id                      TEXT          PRIMARY KEY,
    store_id                TEXT          NOT NULL REFERENCES stores(id) ON DELETE CASCADE,
    cashier_user_id         TEXT          NOT NULL,
    label                   TEXT          NOT NULL DEFAULT '',
    note                    TEXT          NOT NULL DEFAULT '',
    bill_discount_amount    NUMERIC(12,2) NOT NULL DEFAULT 0,
    bill_discount_type      TEXT          NOT NULL DEFAULT 'amount',
    bill_discount_percent   NUMERIC(5,2)  NOT NULL DEFAULT 0,
    customer_id             TEXT          NOT NULL DEFAULT '',
    customer_settlement_mode TEXT         NOT NULL DEFAULT 'cash_now',
    payment_method          TEXT          NOT NULL DEFAULT 'cash',
    vat_included            BOOLEAN       NOT NULL DEFAULT TRUE,
    vat_percent             NUMERIC(5,2)  NOT NULL DEFAULT 7,
    created_at              TIMESTAMPTZ   NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS parked_bill_items (
    id             TEXT          PRIMARY KEY,
    parked_bill_id TEXT          NOT NULL REFERENCES parked_bills(id) ON DELETE CASCADE,
    product_id     TEXT          NOT NULL,
    product_name   TEXT          NOT NULL DEFAULT '',
    product_sku    TEXT          NOT NULL DEFAULT '',
    price          NUMERIC(12,2) NOT NULL DEFAULT 0,
    quantity       INTEGER       NOT NULL,
    discount_type  TEXT,
    discount_value NUMERIC(12,2)
);

-- ── 13. STOCK MOVEMENTS ──────────────────────────────────────
CREATE TABLE IF NOT EXISTS stock_movements (
    id                      TEXT        PRIMARY KEY,
    store_id                TEXT        NOT NULL REFERENCES stores(id)    ON DELETE CASCADE,
    product_id              TEXT        NOT NULL REFERENCES products(id)  ON DELETE CASCADE,
    location_id             TEXT        REFERENCES locations(id)          ON DELETE SET NULL,
    destination_location_id TEXT        REFERENCES locations(id)          ON DELETE SET NULL,
    inventory_id            TEXT        REFERENCES warehouse_inventory(id) ON DELETE SET NULL,
    quantity_change         INTEGER     NOT NULL,
    type                    TEXT        NOT NULL DEFAULT 'adjustment',
    reference_id            TEXT,
    note                    TEXT        NOT NULL DEFAULT '',
    created_by              TEXT        NOT NULL REFERENCES users(id),
    created_at              TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at              TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- ── 14. INDEXES ──────────────────────────────────────────────
CREATE INDEX IF NOT EXISTS idx_store_members_user_id            ON store_members (user_id);
CREATE INDEX IF NOT EXISTS idx_store_subscriptions_store_id     ON store_subscriptions (store_id);
CREATE INDEX IF NOT EXISTS idx_product_types_store_id           ON product_types (store_id);
CREATE INDEX IF NOT EXISTS idx_product_units_store_id           ON product_units (store_id);
CREATE INDEX IF NOT EXISTS idx_product_brands_store_id          ON product_brands (store_id);
CREATE INDEX IF NOT EXISTS idx_products_store_id                ON products (store_id);
CREATE INDEX IF NOT EXISTS idx_products_product_unit_id         ON products (product_unit_id);
CREATE INDEX IF NOT EXISTS idx_customers_store_id               ON customers (store_id);
CREATE INDEX IF NOT EXISTS idx_suppliers_store_id               ON suppliers (store_id);
CREATE INDEX IF NOT EXISTS idx_purchase_orders_store_id         ON purchase_orders (store_id);
CREATE INDEX IF NOT EXISTS idx_purchase_orders_supplier_id      ON purchase_orders (supplier_id);
CREATE INDEX IF NOT EXISTS idx_purchase_order_items_order_id    ON purchase_order_items (purchase_order_id);
CREATE INDEX IF NOT EXISTS idx_purchase_order_items_product_id  ON purchase_order_items (product_id);
CREATE INDEX IF NOT EXISTS idx_supplier_products_supplier       ON supplier_products (supplier_id);
CREATE INDEX IF NOT EXISTS idx_warehouses_store_id              ON warehouses (store_id);
CREATE INDEX IF NOT EXISTS idx_warehouses_source_store          ON warehouses (source_store_id);
CREATE INDEX IF NOT EXISTS idx_locations_store_id               ON locations (store_id);
CREATE INDEX IF NOT EXISTS idx_locations_warehouse_id           ON locations (warehouse_id);
CREATE INDEX IF NOT EXISTS idx_stocks_store_id                  ON stocks (store_id);
CREATE INDEX IF NOT EXISTS idx_stocks_product_id                ON stocks (product_id);
CREATE INDEX IF NOT EXISTS idx_stocks_location_id               ON stocks (location_id);
CREATE INDEX IF NOT EXISTS idx_warehouse_inventory_store        ON warehouse_inventory (store_id);
CREATE INDEX IF NOT EXISTS idx_warehouse_inventory_warehouse    ON warehouse_inventory (warehouse_id);
CREATE INDEX IF NOT EXISTS idx_warehouse_inventory_product      ON warehouse_inventory (product_id);
CREATE INDEX IF NOT EXISTS idx_warehouse_inventory_source_store ON warehouse_inventory (source_store_id);
CREATE INDEX IF NOT EXISTS idx_sales_store_id                   ON sales (store_id, sold_at DESC);
CREATE INDEX IF NOT EXISTS idx_sale_items_sale_id               ON sale_items (sale_id);
CREATE INDEX IF NOT EXISTS idx_invoices_store_id                ON invoices (store_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_invoices_customer_id             ON invoices (customer_id);
CREATE INDEX IF NOT EXISTS idx_invoice_items_invoice_id         ON invoice_items (invoice_id);
CREATE INDEX IF NOT EXISTS idx_invoice_payments_invoice_id      ON invoice_payments (invoice_id);
CREATE INDEX IF NOT EXISTS idx_parked_bills_store_id            ON parked_bills (store_id);
CREATE INDEX IF NOT EXISTS idx_parked_bill_items_bill_id        ON parked_bill_items (parked_bill_id);
CREATE INDEX IF NOT EXISTS idx_stock_movements_store            ON stock_movements (store_id);
CREATE INDEX IF NOT EXISTS idx_stock_movements_product          ON stock_movements (product_id);
CREATE INDEX IF NOT EXISTS idx_stock_movements_location         ON stock_movements (location_id);

-- ── 15. SEED DATA ────────────────────────────────────────────
INSERT INTO subscription_plans (id, code, name, description, duration_days, price_amount, currency_code)
VALUES
    ('plan_starter_monthly', 'starter', 'Starter', 'Single-store starter plan', 30,  299.00, 'THB'),
    ('plan_growth_monthly',  'growth',  'Growth',  'Growing store plan',        30,  799.00, 'THB'),
    ('plan_pro_yearly',      'pro',     'Pro',     'Annual professional plan',  365, 7990.00,'THB')
ON CONFLICT (code) DO NOTHING;

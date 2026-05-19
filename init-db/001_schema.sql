CREATE TABLE IF NOT EXISTS users (
    id TEXT PRIMARY KEY,
    full_name TEXT NOT NULL,
    email TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    token_version BIGINT NOT NULL DEFAULT 0,
    role TEXT NOT NULL CHECK (role IN ('platform_admin', 'owner', 'manager', 'cashier')),
    status TEXT NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'disabled')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS stores (
    id TEXT PRIMARY KEY,
    owner_user_id TEXT NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    name TEXT NOT NULL,
    logo_url TEXT,
    phone TEXT,
    address TEXT,
    promptpay_id TEXT,
    currency_code TEXT NOT NULL DEFAULT 'THB',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS store_members (
    id TEXT PRIMARY KEY,
    store_id TEXT NOT NULL REFERENCES stores(id) ON DELETE CASCADE,
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role TEXT NOT NULL CHECK (role IN ('owner', 'manager', 'cashier')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (store_id, user_id)
);

CREATE TABLE IF NOT EXISTS subscription_plans (
    id TEXT PRIMARY KEY,
    code TEXT NOT NULL UNIQUE,
    name TEXT NOT NULL,
    description TEXT,
    duration_days INTEGER NOT NULL CHECK (duration_days > 0),
    price_amount NUMERIC(12,2) NOT NULL CHECK (price_amount >= 0),
    currency_code TEXT NOT NULL DEFAULT 'THB',
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS store_subscriptions (
    id TEXT PRIMARY KEY,
    store_id TEXT NOT NULL REFERENCES stores(id) ON DELETE CASCADE,
    plan_id TEXT NOT NULL REFERENCES subscription_plans(id) ON DELETE RESTRICT,
    status TEXT NOT NULL CHECK (status IN ('trialing', 'active', 'past_due', 'cancelled', 'expired')),
    current_period_start TIMESTAMPTZ NOT NULL,
    current_period_end TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS product_types (
    id TEXT PRIMARY KEY,
    store_id TEXT NOT NULL REFERENCES stores(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    slug TEXT NOT NULL,
    description TEXT,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT product_types_store_slug_unique UNIQUE (store_id, slug)
);

CREATE TABLE IF NOT EXISTS products (
    id TEXT PRIMARY KEY,
    store_id TEXT NOT NULL REFERENCES stores(id) ON DELETE CASCADE,
    product_type_id TEXT REFERENCES product_types(id) ON DELETE SET NULL,
    name TEXT NOT NULL,
    sku TEXT,
    barcode TEXT,
    unit_type TEXT NOT NULL DEFAULT 'piece',
    base_price NUMERIC(12,2) NOT NULL CHECK (base_price >= 0),
    special_price NUMERIC(12,2),
    special_price_start_at TIMESTAMPTZ,
    special_price_end_at TIMESTAMPTZ,
    image_url TEXT,
    min_stock INTEGER NOT NULL DEFAULT 0 CHECK (min_stock >= 0),
    max_stock INTEGER CHECK (max_stock IS NULL OR max_stock >= 0),
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT products_store_sku_unique UNIQUE (store_id, sku),
    CONSTRAINT products_special_price_check CHECK (
        special_price IS NULL OR (special_price >= 0 AND special_price <= base_price)
    ),
    CONSTRAINT products_special_price_window_check CHECK (
        special_price_start_at IS NULL OR special_price_end_at IS NULL OR special_price_end_at >= special_price_start_at
    )
);

CREATE TABLE IF NOT EXISTS customers (
    id TEXT PRIMARY KEY,
    store_id TEXT NOT NULL REFERENCES stores(id) ON DELETE CASCADE,
    customer_level INTEGER NOT NULL DEFAULT 1 CHECK (customer_level > 0),
    full_name TEXT NOT NULL,
    phone TEXT,
    email TEXT,
    address TEXT,
    note TEXT,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_store_members_user_id ON store_members (user_id);
CREATE INDEX IF NOT EXISTS idx_store_subscriptions_store_id ON store_subscriptions (store_id);
CREATE INDEX IF NOT EXISTS idx_product_types_store_id ON product_types (store_id);
CREATE INDEX IF NOT EXISTS idx_products_store_id ON products (store_id);
CREATE INDEX IF NOT EXISTS idx_customers_store_id ON customers (store_id);

-- Purchasing module: suppliers and purchase orders
CREATE TABLE IF NOT EXISTS suppliers (
    id TEXT PRIMARY KEY,
    store_id TEXT NOT NULL REFERENCES stores(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    phone TEXT,
    address TEXT,
    tax_id TEXT,
    contact_person TEXT,
    note TEXT,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS purchase_orders (
    id TEXT PRIMARY KEY,
    store_id TEXT NOT NULL REFERENCES stores(id) ON DELETE CASCADE,
    supplier_id TEXT REFERENCES suppliers(id) ON DELETE SET NULL,
    order_number TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'partial', 'completed', 'cancelled')),
    notes TEXT,
    total_cost NUMERIC(12,2) NOT NULL DEFAULT 0,
    received_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS purchase_order_items (
    id TEXT PRIMARY KEY,
    purchase_order_id TEXT NOT NULL REFERENCES purchase_orders(id) ON DELETE CASCADE,
    product_id TEXT NOT NULL REFERENCES products(id) ON DELETE RESTRICT,
    quantity INTEGER NOT NULL DEFAULT 1 CHECK (quantity > 0),
    received_quantity INTEGER NOT NULL DEFAULT 0 CHECK (received_quantity >= 0),
    unit_cost NUMERIC(12,2) NOT NULL DEFAULT 0,
    line_total NUMERIC(12,2) NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Add cost_price to products if not exists
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'products' AND column_name = 'cost_price'
    ) THEN
        ALTER TABLE products ADD COLUMN cost_price NUMERIC(12,2) NOT NULL DEFAULT 0;
    END IF;
END $$;

CREATE INDEX IF NOT EXISTS idx_suppliers_store_id ON suppliers (store_id);
CREATE INDEX IF NOT EXISTS idx_purchase_orders_store_id ON purchase_orders (store_id);
CREATE INDEX IF NOT EXISTS idx_purchase_orders_supplier_id ON purchase_orders (supplier_id);
CREATE INDEX IF NOT EXISTS idx_purchase_order_items_order_id ON purchase_order_items (purchase_order_id);
CREATE INDEX IF NOT EXISTS idx_purchase_order_items_product_id ON purchase_order_items (product_id);

INSERT INTO subscription_plans (id, code, name, description, duration_days, price_amount, currency_code)
VALUES
    ('plan_starter_monthly', 'starter', 'Starter', 'Single-store starter plan', 30, 299.00, 'THB'),
    ('plan_growth_monthly', 'growth', 'Growth', 'Growing store plan', 30, 799.00, 'THB'),
    ('plan_pro_yearly', 'pro', 'Pro', 'Annual professional plan', 365, 7990.00, 'THB')
ON CONFLICT (code) DO NOTHING;

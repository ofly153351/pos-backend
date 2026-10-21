-- Seed a complete quotation -> invoice -> delivery order -> bill -> receipt flow.
-- Uses the first existing store/customer/staff and 40 existing active products.
-- This is a deliberate SQL seed for local/dev data; it does not create products.

BEGIN;

-- Keep this seed self-contained when local migrations have not yet applied the
-- document relation migrations.
ALTER TABLE documents
    ADD COLUMN IF NOT EXISTS source_document_id VARCHAR(30)
    REFERENCES documents(id) ON DELETE SET NULL;

CREATE TABLE IF NOT EXISTS bill_delivery_order_links (
    bill_document_id  VARCHAR(30) NOT NULL REFERENCES documents(id) ON DELETE CASCADE,
    delivery_order_id VARCHAR(30) NOT NULL REFERENCES documents(id) ON DELETE RESTRICT,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (bill_document_id, delivery_order_id)
);

CREATE TABLE IF NOT EXISTS receipt_settlements (
    id                   VARCHAR(30) PRIMARY KEY,
    receipt_document_id  VARCHAR(30) NOT NULL REFERENCES documents(id) ON DELETE CASCADE,
    billing_document_id  VARCHAR(30) REFERENCES documents(id) ON DELETE RESTRICT,
    delivery_order_id    VARCHAR(30) REFERENCES documents(id) ON DELETE RESTRICT,
    applied_amount       NUMERIC(14,2) NOT NULL,
    sort_order           INTEGER NOT NULL DEFAULT 1,
    created_at           TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT seed_receipt_settlements_source_check CHECK (
        billing_document_id IS NOT NULL OR delivery_order_id IS NOT NULL
    ),
    CONSTRAINT seed_receipt_settlements_amount_check CHECK (applied_amount > 0)
);

DO $seed$
DECLARE
    v_store_id       TEXT;
    v_staff_id       TEXT;
    v_staff_name     TEXT;
    v_customer_id    TEXT;
    v_customer_name  TEXT;
    v_product_count  INTEGER;
BEGIN
    SELECT s.id, s.owner_user_id, u.full_name
      INTO STRICT v_store_id, v_staff_id, v_staff_name
      FROM stores s
      JOIN users u ON u.id = s.owner_user_id
     ORDER BY s.created_at, s.id
     LIMIT 1;

    SELECT c.id, c.full_name
      INTO STRICT v_customer_id, v_customer_name
      FROM customers c
     WHERE c.store_id = v_store_id
       AND c.is_active = TRUE
     ORDER BY c.created_at, c.id
     LIMIT 1;

    CREATE TEMP TABLE flow_products ON COMMIT DROP AS
    SELECT
        row_number() OVER (ORDER BY p.created_at, p.id)::INTEGER AS row_no,
        p.id AS product_id,
        p.name AS description,
        COALESCE(p.special_price, p.base_price)::NUMERIC(14,2) AS unit_price
      FROM products p
     WHERE p.store_id = v_store_id
       AND p.is_active = TRUE
     ORDER BY p.created_at, p.id
     LIMIT 40;

    SELECT COUNT(*) INTO v_product_count FROM flow_products;
    IF v_product_count < 40 THEN
        RAISE EXCEPTION
            'Need at least 40 active products in store %, found %',
            v_store_id, v_product_count;
    END IF;

    CREATE TEMP TABLE flow_totals ON COMMIT DROP AS
    SELECT
        ROUND(SUM(unit_price), 2)::NUMERIC(14,2) AS subtotal,
        ROUND(SUM(unit_price) * 0.07, 2)::NUMERIC(14,2) AS vat_amount,
        ROUND(SUM(unit_price) * 1.07, 2)::NUMERIC(14,2) AS total_amount
      FROM flow_products;

    -- Keep the selected runtime values available to the INSERT statements.
    CREATE TEMP TABLE flow_context ON COMMIT DROP AS
    SELECT
        v_store_id AS store_id,
        v_staff_id AS staff_id,
        v_staff_name AS staff_name,
        v_customer_id AS customer_id,
        v_customer_name AS customer_name;

    IF EXISTS (
        SELECT 1
          FROM documents
         WHERE document_no IN (
            'QT256910-9001',
            'INV256910-9001',
            'DO256910-9001',
            'BILL256910-9001',
            'RCT256910-9001'
         )
    ) THEN
        RAISE EXCEPTION 'Seed document numbers already exist; choose a new -9001 suffix';
    END IF;
END
$seed$;

-- 1) Quotation
INSERT INTO documents (
    id, store_id, document_no, document_no_full, type, status, payment_status,
    customer_id, customer_name, customer_tax_id, staff_id, staff_name,
    document_date, due_date, subtotal, vat_rate, vat_amount, total_amount,
    notes, created_by
)
SELECT
    'seed-flow-qt-9001', c.store_id, 'QT256910-9001', 'QT256910-9001',
    'QUOTATION', 'COMPLETED', 'UNPAID', c.customer_id, c.customer_name, NULL,
    c.staff_id, c.staff_name, '2026-10-21 09:00:00+07', '2026-11-20 23:59:59+07',
    t.subtotal, 7, t.vat_amount, t.total_amount,
    'SQL seed: quotation with 40 products', c.staff_id
FROM flow_context c CROSS JOIN flow_totals t;

-- 2) Invoice, sourced from the quotation
INSERT INTO documents (
    id, store_id, document_no, document_no_full, type, status, payment_status,
    customer_id, customer_name, customer_tax_id, staff_id, staff_name,
    document_date, due_date, subtotal, vat_rate, vat_amount, total_amount,
    source_document_id, notes, created_by
)
SELECT
    'seed-flow-inv-9001', c.store_id, 'INV256910-9001', 'INV256910-9001',
    'INVOICE', 'COMPLETED', 'UNPAID', c.customer_id, c.customer_name, NULL,
    c.staff_id, c.staff_name, '2026-10-21 09:05:00+07', '2026-11-20 23:59:59+07',
    t.subtotal, 7, t.vat_amount, t.total_amount,
    'seed-flow-qt-9001', 'SQL seed: invoice from quotation', c.staff_id
FROM flow_context c CROSS JOIN flow_totals t;

-- 3) Delivery order, sourced from the invoice
INSERT INTO documents (
    id, store_id, document_no, document_no_full, type, status, payment_status,
    customer_id, customer_name, customer_tax_id, staff_id, staff_name,
    document_date, due_date, subtotal, vat_rate, vat_amount, total_amount,
    source_document_id, notes, created_by
)
SELECT
    'seed-flow-do-9001', c.store_id, 'DO256910-9001', 'DO256910-9001',
    'DELIVERY_ORDER', 'COMPLETED', 'UNPAID', c.customer_id, c.customer_name, NULL,
    c.staff_id, c.staff_name, '2026-10-21 09:10:00+07', '2026-11-20 23:59:59+07',
    t.subtotal, 7, t.vat_amount, t.total_amount,
    'seed-flow-inv-9001', 'SQL seed: delivery order from invoice', c.staff_id
FROM flow_context c CROSS JOIN flow_totals t;

-- 4) Bill, directly related to the delivery order through the join table
INSERT INTO documents (
    id, store_id, document_no, document_no_full, type, status, payment_status,
    customer_id, customer_name, customer_tax_id, staff_id, staff_name,
    document_date, due_date, subtotal, vat_rate, vat_amount, total_amount,
    notes, created_by
)
SELECT
    'seed-flow-bill-9001', c.store_id, 'BILL256910-9001', 'BILL256910-9001',
    'BILL', 'COMPLETED', 'UNPAID', c.customer_id, c.customer_name, NULL,
    c.staff_id, c.staff_name, '2026-10-21 09:15:00+07', '2026-11-20 23:59:59+07',
    t.subtotal, 7, t.vat_amount, t.total_amount,
    'SQL seed: bill related directly to delivery order', c.staff_id
FROM flow_context c CROSS JOIN flow_totals t;

INSERT INTO bill_delivery_order_links (bill_document_id, delivery_order_id)
VALUES ('seed-flow-bill-9001', 'seed-flow-do-9001');

-- 5) Receipt, sourced from the bill and settled against both BILL and DO
INSERT INTO documents (
    id, store_id, document_no, document_no_full, type, status, payment_status,
    customer_id, customer_name, customer_tax_id, staff_id, staff_name,
    document_date, due_date, subtotal, vat_rate, vat_amount, total_amount,
    source_document_id, notes, created_by
)
SELECT
    'seed-flow-rct-9001', c.store_id, 'RCT256910-9001', 'RCT256910-9001',
    'RECEIPT', 'COMPLETED', 'PAID', c.customer_id, c.customer_name, NULL,
    c.staff_id, c.staff_name, '2026-10-21 09:20:00+07', NULL,
    t.subtotal, 7, t.vat_amount, t.total_amount,
    'seed-flow-bill-9001', 'SQL seed: receipt from bill', c.staff_id
FROM flow_context c CROSS JOIN flow_totals t;

INSERT INTO receipt_settlements (
    id, receipt_document_id, billing_document_id, delivery_order_id,
    applied_amount, sort_order
)
SELECT
    'seed-settle-bill-9001', 'seed-flow-rct-9001', 'seed-flow-bill-9001', NULL,
    t.total_amount, 1
FROM flow_totals t;

-- Keep the DO relation explicit in the receipt settlement graph as well.
INSERT INTO receipt_settlements (
    id, receipt_document_id, billing_document_id, delivery_order_id,
    applied_amount, sort_order
)
SELECT
    'seed-settle-do-9001', 'seed-flow-rct-9001', NULL, 'seed-flow-do-9001',
    t.total_amount, 2
FROM flow_totals t;

-- Copy the same 40-product list into every document in the flow.
INSERT INTO document_items (
    id, document_id, product_id, description, quantity,
    unit_price, discount_type, discount_value, amount
)
SELECT
    'seed-' || d.code || '-' || LPAD(fp.row_no::TEXT, 2, '0'),
    d.document_id,
    fp.product_id,
    fp.description,
    1,
    fp.unit_price,
    '',
    0,
    fp.unit_price
FROM (
    VALUES
        ('qt',   'seed-flow-qt-9001'),
        ('inv',  'seed-flow-inv-9001'),
        ('do',   'seed-flow-do-9001'),
        ('bill', 'seed-flow-bill-9001'),
        ('rct',  'seed-flow-rct-9001')
) AS d(code, document_id)
CROSS JOIN flow_products fp;

COMMIT;

-- Expected result: 5 documents x 40 items, with relations:
-- QT -> INV -> DO, BILL -> DO, BILL -> RCT, and DO -> RCT settlement.

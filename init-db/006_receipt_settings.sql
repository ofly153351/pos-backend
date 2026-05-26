CREATE TABLE IF NOT EXISTS store_receipt_settings (
    id                      TEXT        PRIMARY KEY,
    store_id                TEXT        NOT NULL UNIQUE REFERENCES stores(id) ON DELETE CASCADE,

    -- Receipt template
    template_key            TEXT        NOT NULL DEFAULT 'modern_classic',
    paper_size              TEXT        NOT NULL DEFAULT '58mm',
    paper_length            TEXT        NOT NULL DEFAULT 'auto',

    -- Tax
    tax_mode                TEXT        NOT NULL DEFAULT 'exclusive',
    vat_rate                NUMERIC(5,2) NOT NULL DEFAULT 7.00,
    tax_label               TEXT        NOT NULL DEFAULT 'ภาษีมูลค่าเพิ่ม (VAT 7%)',

    -- Logo
    show_logo               BOOLEAN     NOT NULL DEFAULT TRUE,
    logo_position           TEXT        NOT NULL DEFAULT 'top_center',

    -- Store info display
    show_store_name         BOOLEAN     NOT NULL DEFAULT TRUE,
    show_address            BOOLEAN     NOT NULL DEFAULT TRUE,
    show_phone              BOOLEAN     NOT NULL DEFAULT TRUE,
    show_tax_id             BOOLEAN     NOT NULL DEFAULT TRUE,

    -- Footer
    footer_text             TEXT        NOT NULL DEFAULT 'ขอบคุณที่ใช้บริการ',

    -- Payment channels (JSONB array of {key, enabled})
    payment_channels        JSONB       NOT NULL DEFAULT '[
        {"key":"cash","enabled":true},
        {"key":"card","enabled":true},
        {"key":"qr","enabled":true},
        {"key":"promptpay","enabled":true},
        {"key":"truemoney","enabled":false},
        {"key":"shopeepay","enabled":false}
    ]'::jsonb,

    -- Printer
    printer_type            TEXT        NOT NULL DEFAULT 'thermal',
    printer_name            TEXT        NOT NULL DEFAULT '',
    auto_print              BOOLEAN     NOT NULL DEFAULT FALSE,
    copies                  INTEGER     NOT NULL DEFAULT 1 CHECK (copies BETWEEN 1 AND 5),

    -- PromptPay QR on receipt
    show_qr                 BOOLEAN     NOT NULL DEFAULT TRUE,
    qr_size                 TEXT        NOT NULL DEFAULT 'medium',

    -- Display
    show_customer_display   BOOLEAN     NOT NULL DEFAULT FALSE,
    show_product_images     BOOLEAN     NOT NULL DEFAULT FALSE,
    date_format             TEXT        NOT NULL DEFAULT 'DD/MM/YYYY',
    time_format             TEXT        NOT NULL DEFAULT '24h',
    currency_position       TEXT        NOT NULL DEFAULT 'before',

    created_at              TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at              TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

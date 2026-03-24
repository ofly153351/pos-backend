ALTER TABLE sales
ADD COLUMN IF NOT EXISTS vat_included BOOLEAN NOT NULL DEFAULT TRUE;

ALTER TABLE sales
ADD COLUMN IF NOT EXISTS vat_percent NUMERIC(5,2) NOT NULL DEFAULT 7;

ALTER TABLE sales
ADD COLUMN IF NOT EXISTS vat_amount NUMERIC(12,2) NOT NULL DEFAULT 0;

UPDATE sales
SET vat_included = TRUE,
    vat_percent = 7,
    vat_amount = ROUND((total_amount * 7 / 107)::numeric, 2)
WHERE vat_amount = 0;

ALTER TABLE sales
DROP CONSTRAINT IF EXISTS sales_vat_percent_check;

ALTER TABLE sales
ADD CONSTRAINT sales_vat_percent_check CHECK (vat_percent >= 0 AND vat_percent <= 100);

ALTER TABLE sales
DROP CONSTRAINT IF EXISTS sales_vat_amount_check;

ALTER TABLE sales
ADD CONSTRAINT sales_vat_amount_check CHECK (vat_amount >= 0);

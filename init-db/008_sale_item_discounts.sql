DO $$
BEGIN
    IF EXISTS (
        SELECT 1
        FROM information_schema.columns
        WHERE table_name = 'sales' AND column_name = 'subtotal'
    ) AND NOT EXISTS (
        SELECT 1
        FROM information_schema.columns
        WHERE table_name = 'sales' AND column_name = 'subtotal_amount'
    ) THEN
        EXECUTE 'ALTER TABLE sales RENAME COLUMN subtotal TO subtotal_amount';
    END IF;
END $$;

ALTER TABLE sales
ADD COLUMN IF NOT EXISTS subtotal_amount NUMERIC(12,2) NOT NULL DEFAULT 0;

ALTER TABLE sales
ADD COLUMN IF NOT EXISTS discount_amount NUMERIC(12,2) NOT NULL DEFAULT 0;

ALTER TABLE sales
ADD COLUMN IF NOT EXISTS bill_discount_amount NUMERIC(12,2) NOT NULL DEFAULT 0;

UPDATE sales
SET subtotal_amount = total_amount,
    discount_amount = 0
WHERE subtotal_amount = 0 AND total_amount >= 0;

ALTER TABLE sale_items
ADD COLUMN IF NOT EXISTS discount_type TEXT;

ALTER TABLE sale_items
ADD COLUMN IF NOT EXISTS discount_value NUMERIC(12,4);

ALTER TABLE sale_items
ADD COLUMN IF NOT EXISTS discount_amount_per_unit NUMERIC(12,2) NOT NULL DEFAULT 0;

ALTER TABLE sale_items
ADD COLUMN IF NOT EXISTS line_subtotal NUMERIC(12,2) NOT NULL DEFAULT 0;

ALTER TABLE sale_items
ADD COLUMN IF NOT EXISTS line_discount_total NUMERIC(12,2) NOT NULL DEFAULT 0;

UPDATE sale_items
SET line_subtotal = line_total,
    line_discount_total = 0,
    discount_amount_per_unit = 0
WHERE line_subtotal = 0;

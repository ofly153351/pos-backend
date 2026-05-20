-- Allow deleting products even if they have sale / invoice history
-- sale_items and invoice_items already store product_name and sku snapshots,
-- so setting product_id to NULL on product deletion preserves transaction history.

ALTER TABLE sale_items
    DROP CONSTRAINT IF EXISTS sale_items_product_id_fkey;

ALTER TABLE sale_items
    ALTER COLUMN product_id DROP NOT NULL;

ALTER TABLE sale_items
    ADD CONSTRAINT sale_items_product_id_fkey
        FOREIGN KEY (product_id) REFERENCES products(id) ON DELETE SET NULL;

ALTER TABLE invoice_items
    DROP CONSTRAINT IF EXISTS invoice_items_product_id_fkey;

ALTER TABLE invoice_items
    ALTER COLUMN product_id DROP NOT NULL;

ALTER TABLE invoice_items
    ADD CONSTRAINT invoice_items_product_id_fkey
        FOREIGN KEY (product_id) REFERENCES products(id) ON DELETE SET NULL;

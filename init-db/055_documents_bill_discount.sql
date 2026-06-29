-- 054_documents_bill_discount.sql
-- A persisted document (e.g. a TAX_INVOICE issued from a POS sale) must be able to
-- represent a whole-bill discount so its summary matches the originating receipt.
-- The document money model previously carried only per-item discount_value, which
-- silently dropped a sale's bill-level discount when converting sale → tax invoice.
-- bill_discount is added on top of the per-item discounts when the document summary
-- is rendered (see document.toDocData). Default 0 keeps every existing/manual document
-- unchanged.
ALTER TABLE documents ADD COLUMN IF NOT EXISTS bill_discount NUMERIC(14,2) NOT NULL DEFAULT 0;

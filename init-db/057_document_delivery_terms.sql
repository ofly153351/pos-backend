-- Delivery and quotation term fields. Nullable for backward compatibility.
ALTER TABLE documents
    ADD COLUMN IF NOT EXISTS price_validity_days INTEGER,
    ADD COLUMN IF NOT EXISTS delivery_lead_time_days INTEGER,
    ADD COLUMN IF NOT EXISTS po_received_date TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS expected_delivery_date TIMESTAMPTZ;
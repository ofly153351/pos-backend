-- Direct many-to-many relation between billing notices and delivery orders.
-- A BILL may settle multiple DOs, so source_document_id alone is insufficient.
CREATE TABLE IF NOT EXISTS bill_delivery_order_links (
    bill_document_id    VARCHAR(30) NOT NULL REFERENCES documents(id) ON DELETE CASCADE,
    delivery_order_id   VARCHAR(30) NOT NULL REFERENCES documents(id) ON DELETE RESTRICT,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (bill_document_id, delivery_order_id)
);

CREATE INDEX IF NOT EXISTS idx_bill_delivery_order_links_do
    ON bill_delivery_order_links(delivery_order_id);

-- Backfill legacy BILL snapshot items that contain the DO number.
INSERT INTO bill_delivery_order_links (bill_document_id, delivery_order_id)
SELECT DISTINCT bill.id, delivery.id
FROM documents bill
JOIN document_items item ON item.document_id = bill.id
JOIN documents delivery
  ON delivery.store_id = bill.store_id
 AND delivery.type = 'DELIVERY_ORDER'
 AND (item.description = delivery.document_no
      OR item.description = delivery.document_no_full
      OR item.description ILIKE '%' || delivery.document_no || '%')
WHERE bill.type = 'BILL'
ON CONFLICT DO NOTHING;

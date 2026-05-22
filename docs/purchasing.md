# Purchasing API

API for managing suppliers, supplier product catalogs, and purchase orders (POs).

All endpoints require: `Authorization: Bearer <token>`

---

## Suppliers

### POST /stores/:storeID/suppliers

Creates a new supplier.

Request body:

```json
{
  "name": "Acme Wholesale",
  "contact_name": "Jane Smith",
  "phone": "0812345678",
  "email": "jane@acme.com",
  "address": "Bangkok",
  "note": "primary coffee bean supplier"
}
```

### GET /stores/:storeID/suppliers

Returns all suppliers for the store.

### GET /stores/:storeID/suppliers/:supplierID

Returns details of a single supplier.

### PUT /stores/:storeID/suppliers/:supplierID

Updates a supplier's information.

### DELETE /stores/:storeID/suppliers/:supplierID

Deletes a supplier.

---

## Supplier Products

These endpoints manage the catalog of products available from a specific supplier.

### GET /stores/:storeID/suppliers/:supplierID/products

Returns all products linked to this supplier.

Response includes supplier-specific fields such as cost price and supplier SKU alongside the linked store product.

### POST /stores/:storeID/suppliers/:supplierID/products

Links an **existing** store product to this supplier.

Request body:

```json
{
  "product_id": "prod_001",
  "supplier_sku": "ACM-BEAN-001",
  "cost_price": 250.00,
  "note": "standard purchase cost"
}
```

### POST /stores/:storeID/suppliers/:supplierID/products/create

**Creates a new store product and links it to this supplier in one call** (no need to create the product separately first).

Request body:

```json
{
  "name": "Arabica Bean 1kg",
  "unit_id": "unit_xxx",
  "product_type_id": "type_xxx",
  "base_price": 350.00,
  "supplier_sku": "ACM-BEAN-001",
  "cost_price": 250.00
}
```

### PUT /stores/:storeID/suppliers/:supplierID/products/:productID

Updates the supplier-product link (e.g. cost price or supplier SKU).

### DELETE /stores/:storeID/suppliers/:supplierID/products/:productID

Removes a product from the supplier's catalog.

---

## Purchase Orders

### POST /stores/:storeID/purchase-orders

Creates a new purchase order.

The entire creation is wrapped in a `db.Transaction` — if any part fails, no orphan PO records are created.

Request body:

```json
{
  "supplier_id": "sup_001",
  "expected_at": "2026-06-01T00:00:00Z",
  "note": "monthly restock",
  "items": [
    {
      "product_id": "prod_001",
      "quantity": 100,
      "cost_price": 250.00
    },
    {
      "product_id": "prod_002",
      "quantity": 50,
      "cost_price": 180.00
    }
  ]
}
```

### GET /stores/:storeID/purchase-orders

Returns the list of purchase orders for the store.

Optional query params:
- `status` — filter by status (`pending`, `partial`, `completed`, `cancelled`)
- `supplier_id` — filter by supplier
- `from` / `to` — date range filter

### GET /stores/:storeID/purchase-orders/:poID

Returns full details of a single purchase order, including all line items.

### PUT /stores/:storeID/purchase-orders/:poID

Updates a purchase order (only allowed while in `pending` status).

### POST /stores/:storeID/purchase-orders/:poID/receive

Marks items as received, updates stock quantities, and transitions the PO status.

Request body:

```json
{
  "items": [
    {
      "product_id": "prod_001",
      "received_quantity": 80
    }
  ],
  "note": "partial delivery — 20 units back-ordered"
}
```

Status transition logic:
- If all items are fully received → `completed`
- If only some items are received → `partial`

### POST /stores/:storeID/purchase-orders/:poID/cancel

Cancels a purchase order. Only orders in `pending` or `partial` status can be cancelled.

Request body:

```json
{
  "reason": "supplier unable to fulfill order"
}
```

---

## PO Status Flow

```
pending → partial → completed
       ↘         ↘
        cancelled  cancelled
```

- `pending` — order created, no items received yet
- `partial` — some items have been received
- `completed` — all items received
- `cancelled` — order was cancelled before full receipt

---

## Key Behavior

- **CreatePO is transactional** — the entire PO and its line items are created in a single `db.Transaction`. If any item fails validation, no records are committed.
- **ReceiveStock updates stock** — when receiving items, the system increments the product's quantity in the store's stock.
- **Supplier product linking** — use `POST .../products` to link an existing product, or `POST .../products/create` to create the product and link it in a single request.

---

## Notes

- Deleting a supplier does not automatically remove linked products or POs; handle cascades carefully
- Cost price on the PO line item is the actual purchase cost recorded at the time of the order, independent of the product's `base_price`

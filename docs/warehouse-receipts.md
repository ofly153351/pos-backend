# Warehouse Receipts API

API for creating and managing goods-received notes (warehouse receipts). A receipt goes through a draft → confirmed lifecycle and automatically updates stock on confirmation.

All endpoints require: `Authorization: Bearer <token>`

---

## Lifecycle

```
draft → confirmed
draft → cancelled
```

- Only **draft** receipts can be edited (items, header, attachment).
- Only users with **manage** access can confirm or cancel.
- Confirming a receipt commits stock movements and makes the receipt immutable.

---

## Endpoints

### GET /warehouse/receipts

Lists warehouse receipts for the authenticated user's store.

Query params:

| Param | Type | Default | Description |
|---|---|---|---|
| `store_id` | string | (from token) | Filter by store. Falls back to user's primary store. |
| `status` | `draft` \| `confirmed` \| `cancelled` | — | Filter by status |
| `page` | int | 1 | Page number |
| `limit` | int | 50 | Items per page (max 200) |

---

### POST /warehouse/receipts

Creates a new receipt in **draft** status. `document_no` is auto-generated if omitted.

Request body:

```json
{
  "store_id": "str-00000001",
  "warehouse_id": "whr-00000001",
  "supplier_id": "sup-00000001",
  "purchase_order_id": "po-00000001",
  "document_no": "RCV-240101-001",
  "received_at": "2026-05-24T10:00:00Z",
  "reference_no": "INV-2026-001",
  "note": "First shipment",
  "vat_included": true,
  "vat_percent": 7
}
```

- `warehouse_id` — required. Must belong to the store.
- `supplier_id`, `purchase_order_id`, `reference_no`, `note` — optional.
- `vat_included` — defaults to `true`.
- `vat_percent` — defaults to `7`. Must be 0–100.
- If `purchase_order_id` is set, item quantities are validated against PO outstanding quantities on each `AddItems` call.

---

### GET /warehouse/receipts/:id

Returns a single receipt with its items.

---

### PUT /warehouse/receipts/:id

Updates the header of a draft receipt. All fields are optional (patch semantics).

Request body:

```json
{
  "warehouse_id": "whr-00000001",
  "supplier_id": "sup-00000001",
  "purchase_order_id": "po-00000001",
  "document_no": "RCV-240101-001",
  "received_at": "2026-05-24T10:00:00Z",
  "reference_no": "INV-2026-001",
  "note": "Updated note",
  "vat_included": true,
  "vat_percent": 7
}
```

---

### POST /warehouse/receipts/:id/items

Saves items on a draft receipt.

Request body:

```json
{
  "replace_existing": true,
  "items": [
    {
      "product_id": "pd-00000200",
      "location_id": "loc-00000001",
      "quantity": 7,
      "unit_price": 91,
      "discount_type": "percent",
      "discount_value": 5
    }
  ]
}
```

#### `replace_existing`

| Value | Behaviour |
|---|---|
| `true` | Deletes all existing items on the receipt, then inserts the new list. Use for full saves (step-2 autosave). |
| `false` | Appends new items to the existing list, then validates uniqueness. Use for incremental add. |

#### Item fields

| Field | Type | Required | Notes |
|---|---|---|---|
| `product_id` | string | yes | Must be an active product in the store. |
| `location_id` | string | yes | Must be an active, non-sale-point location in the **same warehouse** as the receipt. |
| `quantity` | int | yes | Must be ≥ 1. |
| `unit_price` | float | no | Defaults to product's cost price. Must be ≥ 0. |
| `discount_type` | `"amount"` \| `"percent"` \| `""` | no | Leave empty (or omit) for no discount. |
| `discount_value` | float \| null | no | Required when `discount_type` is set. Ignored when `discount_type` is empty. `0` is treated as no discount. |

#### Constraints

- `(product_id, location_id)` must be unique within a single `AddItems` call.
- If `replace_existing: false`, the merged list (existing + new) must also have no duplicate `(product_id, location_id)` pairs.
- If the receipt has a linked `purchase_order_id`, each item's quantity must not exceed the PO's outstanding quantity for that product.

---

### PUT /warehouse/receipts/:id/items/:item_id

Updates a single item on a draft receipt. All fields are optional (patch semantics).

Request body:

```json
{
  "product_id": "pd-00000200",
  "location_id": "loc-00000001",
  "quantity": 10,
  "unit_price": 91,
  "discount_type": "amount",
  "discount_value": 5
}
```

---

### DELETE /warehouse/receipts/:id/items/:item_id

Removes a single item from a draft receipt.

---

### POST /warehouse/receipts/:id/confirm

Confirms a draft receipt. Requires **manage** permission.

- Commits stock movements: increases stock at each item's location.
- Receipt becomes immutable.

---

### POST /warehouse/receipts/:id/cancel

Cancels a draft receipt. Requires **manage** permission.

---

### POST /warehouse/receipts/:id/attachment

Uploads a PDF, JPG, or PNG attachment (delivery note, invoice scan, etc.).

- Content-Type: `multipart/form-data`
- Form field: `file`
- Max size: 10 MB

---

### GET /warehouse/receipts/:id/stock-impact

Returns a preview of the stock changes that would result from confirming the receipt.

Response includes one row per item:

```json
[
  {
    "item_id": "wri-00000001",
    "product_id": "pd-00000200",
    "product_name": "Coffee Beans 1kg",
    "location_id": "loc-00000001",
    "location_name": "Shelf A-1",
    "quantity": 7,
    "before_quantity": 10,
    "after_quantity": 17
  }
]
```

---

### GET /warehouse/receipts/:id/print

Returns the printable receipt. Behaviour depends on `Accept` header:

| Accept | Response |
|---|---|
| `text/html` | Raw HTML string (for browser print) |
| `application/json` | JSON object with `html`, `file_name`, `document_no` |

---

### POST /warehouse/receipts/generate-document-no

Generates the next document number for today's date (Thai Buddhist calendar format: `RCV-DDMMYYYY-NNN`).

Request body:

```json
{ "store_id": "str-00000001" }
```

---

## Error reference

| HTTP | Message | Cause |
|---|---|---|
| 400 | `warehouse_id is required` | `warehouse_id` missing on create |
| 400 | `at least one receipt item is required` | `items` array is empty |
| 400 | `quantity must be greater than or equal to 1` | Item `quantity` < 1 |
| 400 | `unit_price must be greater than or equal to zero` | `unit_price` < 0, or `discount_type` is empty but `discount_value` is non-zero, or discount value is invalid |
| 400 | `duplicate product and location combination is not allowed` | Same `(product_id, location_id)` appears more than once |
| 400 | `location does not belong to receipt warehouse` | `location_id` belongs to a different warehouse |
| 400 | `sale point locations cannot be used for warehouse receipts` | `location_id` is a sale-point (POS till) location |
| 400 | `receipt quantity exceeds outstanding purchase order quantity` | Item quantity > PO outstanding qty, or product not on PO |
| 400 | `only draft warehouse receipt can be modified` | Attempting to edit a confirmed or cancelled receipt |
| 403 | `user cannot confirm this warehouse receipt` | User lacks manage permission |
| 404 | `warehouse receipt not found` | Receipt ID does not exist or belongs to another store |

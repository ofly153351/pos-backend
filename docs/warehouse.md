# Warehouse API

API for managing warehouses and warehouse stock within a store. Also supports cross-store stock transfers.

All endpoints require: `Authorization: Bearer <token>`

## Warehouse CRUD

### GET /stores/:storeID/warehouses

Returns all warehouses for the store.

### GET /stores/:storeID/warehouses/:warehouseID

Returns details of a single warehouse.

### POST /stores/:storeID/warehouses

Creates a new warehouse for the store.

Request body:

```json
{
  "name": "Main Warehouse",
  "description": "Primary storage location",
  "is_active": true
}
```

### PUT /stores/:storeID/warehouses/:warehouseID

Updates a warehouse.

### DELETE /stores/:storeID/warehouses/:warehouseID

Deletes a warehouse.

---

## Warehouse Products

### GET /stores/:storeID/warehouses/:warehouseID/products

Returns the list of products associated with this warehouse.

### POST /stores/:storeID/warehouses/:warehouseID/products

Adds a product to the warehouse inventory.

Request body:

```json
{
  "product_id": "prod_001",
  "quantity": 100
}
```

### PUT /stores/:storeID/warehouses/:warehouseID/products/:productID

Updates the product record within the warehouse (e.g. quantity adjustment).

### DELETE /stores/:storeID/warehouses/:warehouseID/products/:productID

Removes a product from the warehouse.

---

## Warehouse Transfer

### POST /stores/:storeID/warehouses/:warehouseID/transfer

Transfers stock out of this warehouse (Store A source) to a destination.

Two modes are supported based on whether the destination is in the same store or a different store.

---

### Same-Store Transfer

Transfer to another warehouse or a stock location within the same store.

Request body:

```json
{
  "destination_type": "warehouse",
  "destination_id": "<warehouseID or locationID>",
  "product_id": "prod_001",
  "quantity": 10
}
```

- `destination_type`: `"warehouse"` or `"stock"`
- `destination_id`: the target warehouse ID or stock location ID within the same store

---

### Cross-Store Transfer

Transfer stock to a warehouse in a different store (Store B).

Request body:

```json
{
  "destination_type": "warehouse",
  "destination_store_id": "<storeB_id>",
  "product_id": "prod_001",
  "quantity": 10
}
```

**Cross-Store Transfer Behavior:**

1. **Deduct from Store A** — finds the stock location in Store A that has actual quantity `>= qty`, ordered by `is_sale_point ASC` (non-sale-point locations are preferred as the deduction source)
2. **Clone product data in Store B** — product type, unit, brand, and the product itself are cloned by name in Store B (idempotent: if the product already exists by name, the existing record is used)
3. **Auto-create warehouse in Store B** — a new warehouse is automatically created in Store B if one does not already exist; the warehouse records `source_store_id` to track its origin
4. **Insert directly to Store B stocks** — the quantity is inserted directly into Store B's stocks at a non-sale-point location; there is no staging step via `warehouse_inventory`

---

## Warehouse Inventory (Cross-Store Staging)

These endpoints manage cross-store transfers that are pending allocation.

### GET /stores/:storeID/warehouses/:warehouseID/inventory

Returns pending inventory items awaiting allocation for this warehouse.

### POST /stores/:storeID/warehouses/:warehouseID/inventory/:productID/allocate

Allocates a pending inventory item into actual stock.

---

## Notes

- Cross-store transfer does **not** use the `warehouse_inventory` staging table; it writes directly to stocks in Store B
- Same-store transfer uses `destination_type` + `destination_id` to resolve the target
- Cross-store transfer uses `destination_type` + `destination_store_id` to resolve the target store
- The auto-created warehouse in Store B tracks `source_store_id` for traceability

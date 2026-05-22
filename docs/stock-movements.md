# Stock Movements API

API for recording stock movements — adding, removing, transferring, and adjusting inventory quantities.

All endpoints require: `Authorization: Bearer <token>`

## Endpoints

### POST /stores/:storeID/stock-movements/in

**Add Stock** — increases quantity at a location.

Request body:

```json
{
  "product_id": "prod_001",
  "quantity": 50,
  "location_id": "loc_001",
  "note": "restocked from supplier"
}
```

**Auto-resolve behavior:** If `location_id` is not provided, the system automatically resolves to the first active `is_sale_point=TRUE` location. If no sale-point location exists, it falls back to any active location.

---

### POST /stores/:storeID/stock-movements/out

**Remove Stock** — decreases quantity at a location.

Request body:

```json
{
  "product_id": "prod_001",
  "quantity": 5,
  "location_id": "loc_001",
  "note": "damaged goods removed"
}
```

---

### POST /stores/:storeID/stock-movements/transfer

**Transfer Between Locations** — moves quantity from one location to another within the same store.

Request body:

```json
{
  "product_id": "prod_001",
  "quantity": 20,
  "from_location_id": "loc_001",
  "to_location_id": "loc_002",
  "note": "moved to back storage"
}
```

---

### POST /stores/:storeID/stock-movements/adjust

**Adjust Stock** — sets an absolute quantity at a location (not a delta).

Request body:

```json
{
  "product_id": "prod_001",
  "quantity": 100,
  "location_id": "loc_001",
  "note": "physical count correction"
}
```

The system calculates the difference between the provided `quantity` and the current quantity on hand and records the delta as a movement.

---

### GET /stores/:storeID/stock-movements

**List Movements** — returns the movement history for the store.

Optional query params:
- `product_id` — filter by product
- `location_id` — filter by location
- `from` — start date (RFC3339 or `YYYY-MM-DD`)
- `to` — end date (RFC3339 or `YYYY-MM-DD`)
- `page` — default `1`
- `limit` — default `50`

---

## Key Behavior

### Movement Records

Every movement creates a record in the `stock_movements` table with:
- `created_by` = the authenticated user's ID (FK to `users` table)
- `created_by` is **never** set to a string like `"system"` — all movements must be attributed to a real user

### Frontend BFF Proxy

The frontend Backend-For-Frontend (BFF) proxy maps:

```
POST /api/stores/:storeId/stock/add  →  POST /stock-movements/in
```

When calling the add-stock flow from the frontend, the BFF proxies the request to the stock movements endpoint transparently.

---

## Notes

- All movement types are audit-logged with the acting user's ID
- The `adjust` endpoint is intended for physical inventory count corrections, not for incremental updates
- Use `in` / `out` for normal restocking and wastage flows
- Use `transfer` for relocating stock between locations without changing total store quantity

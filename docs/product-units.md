# Product Unit API

Each store can register its own unit labels (e.g., `Piece`, `Set`, `Bottle`) and the frontend can list them for selectors.

All endpoints require: `Authorization: Bearer <token>`

## POST /api/v1/stores/:storeID/product-units

Create a unit:

```json
{
  "name": "Piece",
  "description": "Sold individually",
  "is_active": true
}
```

## GET /api/v1/stores/:storeID/product-units

Returns all units for the store, used to populate dropdowns.

## PATCH /api/v1/stores/:storeID/product-units/:unitID

Update name, description, or active status.

## DELETE /api/v1/stores/:storeID/product-units/:unitID

Remove an unused unit.

## Notes

- Units belong to a store; `store_members` controls access.
- The frontend should cache units per store and display the `name` field in selectors.

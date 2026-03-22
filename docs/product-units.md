# Product Unit API

Each store can register its own unit labels (e.g., `Piece`, `Set`, `Bottle`) and the frontend can list them for selectors.

Base: `Authorization: Bearer <token>`

## POST /api/v1/stores/:storeID/product-units
Create unit:

```json
{
  "name": "Piece",
  "description": "Sold individually",
  "is_active": true
}
```

## GET /api/v1/stores/:storeID/product-units
Returns all units for dropdowns.

## PATCH /api/v1/stores/:storeID/product-units/:unitID
Update name/description/status.

## DELETE /api/v1/stores/:storeID/product-units/:unitID
Remove unused unit.

## Notes
- Units belong to a store; `store_members` controls access.
- Frontend should cache units per store and show the `name` field in selectors.

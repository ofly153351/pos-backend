# Product Structure Guide

## Purpose
Each store defines its own:

- **Product types** (categories such as `coffee`, `bakery`, `accessories`)
- **Product units** (measurement units like `piece`, `set`, `bottle`)

These tables allow frontend to build selectors without hardcoding values.

## Product Types (per store)

Fields:
- `name` identifies the category
- `description` for UI hints
- `is_active` to soft hide unused types

API:
1. `POST /api/v1/stores/:storeID/product-types` → create
2. `GET /api/v1/stores/:storeID/product-types` → list
3. `PATCH .../:productTypeID` → update
4. `DELETE .../:productTypeID` → remove

Frontend workflow:
1. Fetch types after selecting a store.
2. Populate dropdown for product’s `product_type_id`.

## Product Units (per store)

Fields:
- `name` (display label)
- `description` optional text
- `is_active` show/hide

API:
1. `POST /api/v1/stores/:storeID/product-units`
2. `GET /api/v1/stores/:storeID/product-units`
3. `PATCH /api/v1/stores/:storeID/product-units/:unitID`
4. `DELETE .../:unitID`

Frontend workflow:
1. After store selection, fetch units once.
2. Show unit `name` in selectors or admin screens.

## Product Creation

When creating/updating a product, send both:

- `product_type_id` (optional but recommended)
- `unit_type` (required string; ถ้าไม่ส่งจะ default เป็น `piece`)
- `quantity` (optional integer, default `0`)

Also submit optional `special_price` window to calculate `effective_price`.

## Notes

- Types/units belong to store; permissions enforced via `store_members`.
- Use the provided APIs instead of inline constants to keep frontend synced.

# Frontend Prompt

Use this summary to build the POS frontend interactions. Every request requires `Authorization: Bearer <token>` after login.

## Auth
1. `POST /api/v1/auth/register` — send `{name,email,password}`; stores user and returns JWT.
2. `POST /api/v1/auth/login` — send `{email,password}`; receives `{user,store_id,access_token}` when the user already belongs to a store.

## Store onboarding
1. `POST /api/v1/stores` (multipart/form-data): fields `name`, optionally `logo`, `phone`, `address`, `currency_code`. **Required** `subscription_plan_code`. Response includes `id`.
2. Frontend should store returned `store_id` for subsequent requests.

## Product type management
- `POST /api/v1/stores/:storeID/product-types` — fields `name`, `description`, `is_active`.
- `GET /api/v1/stores/:storeID/product-types` — list types for selector controls.
- `PATCH` / `DELETE` same path to update/remove a type.

## Product units
- `POST /api/v1/stores/:storeID/product-units` — fields `name`, `description`, `is_active`.
- `GET /api/v1/stores/:storeID/product-units` — list store-defined unit labels.
- `PATCH` / `DELETE` same path to update/remove a unit.

## Products
- `POST /api/v1/stores/:storeID/products` (multipart/form-data): `name`, `sku`, `product_type_id`, `unit_type` (free-text string, default `piece`), optional `quantity` (integer, default `0`), `base_price`, optional `special_price`, `special_price_start_at`, `special_price_end_at`, `is_active`, `image`.
- `GET /api/v1/stores/:storeID/products` — list including `effective_price`.
- `GET/PATCH/DELETE /api/v1/stores/:storeID/products/:productID` for individual records.

## Subscription
- `GET /api/v1/subscriptions/plans` — receive available plan codes (e.g., `starter`, `growth`, `pro`).
- `GET /api/v1/stores/:storeID/subscription` — show current plan/status.
- `PUT /api/v1/stores/:storeID/subscription` — change plan via `{plan_code}`.

## Admin-only hooks
- `GET /api/v1/admin/subscriptions` — list every store and owner.
- `GET/PUT/PATCH /api/v1/admin/stores/:storeID/subscription*` — admin can change plan or status without membership.

## Notes
- Always trigger subscription plan selection before adding products.
- `unit_type` defaults to `piece`; frontend can offer any store-defined label.
- All timestamps are RFC3339; special price windows impact `effective_price`.

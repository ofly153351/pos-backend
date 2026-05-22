# Customer Network API

This API allows stores to create and manage customers by assigning a `level` directly (no parent-customer hierarchy).

All endpoints require: `Authorization: Bearer <token>`

## Base URL

```bash
http://localhost:8080
```

Supports both:

- `/api/v1/*` (primary)
- `/api/*` (compatibility)

## Authorization

Roles that can access:

- `platform_admin`
- `store_members` with role `owner`, `manager`, or `cashier`

Note:
- Endpoints that manage level discount rules (`customer-level-discounts`) are restricted to `owner`, `manager`, and `platform_admin`

## Data Model (Summary)

- Each customer belongs to a `store_id`
- Each customer has a `level`, e.g. `1`, `2`, `3`
- The discount system maps from `level -> discount_percent`

## Endpoints

## POST /api/v1/stores/:storeID/customers

Creates a customer.

Request body:

```json
{
  "level": 2,
  "full_name": "Acme Retail Partner",
  "phone": "0812345678",
  "email": "acme@example.com",
  "address": "Bangkok",
  "note": "tier B",
  "is_active": true
}
```

`level` is optional; defaults to `1` if not provided.

## GET /api/v1/stores/:storeID/customers

Returns all customers for the store.

## GET /api/v1/stores/:storeID/customers/:customerID

Returns a single customer.

## PATCH /api/v1/stores/:storeID/customers/:customerID

Updates a customer. Send only the fields to be changed.

```json
{
  "level": 3,
  "full_name": "Acme Retail Partner Updated",
  "email": "new@example.com",
  "is_active": true
}
```

## DELETE /api/v1/stores/:storeID/customers/:customerID

Deletes a customer.

## GET /api/v1/stores/:storeID/customer-level-discounts

Returns the store's configured discount rules per customer level.

Example response:

```json
[
  {
    "store_id": "store_001",
    "level": 1,
    "discount_percent": 5
  },
  {
    "store_id": "store_001",
    "level": 2,
    "discount_percent": 10
  }
]
```

## PUT /api/v1/stores/:storeID/customer-level-discounts/:level

Sets or updates the discount for a given level (upsert).

Request body:

```json
{
  "discount_percent": 7.5
}
```

Constraints:

- `level` must be greater than `0`
- `discount_percent` must be between `0` and `100`

## DELETE /api/v1/stores/:storeID/customer-level-discounts/:level

Removes the discount rule for the specified level.

## Integration with Sales

When creating a sale and providing a `customer_id`, the system will:

1. Read the customer's `level`
2. Look up the discount `%` for that level from the store's `customer_level_discounts`
3. Automatically apply the network discount to the bill

## Common Error Status Codes

- `400` — invalid input, e.g. empty `full_name`, invalid email format, `level <= 0`
- `403` — no permission for this store
- `404` — customer not found
- `500` — internal server error

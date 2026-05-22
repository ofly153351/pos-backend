# Admin Subscription API

This API is for `platform_admin` to manage subscriptions for all stores in the system.

All endpoints require a Bearer token for a user with role `platform_admin`:

```http
Authorization: Bearer <access_token>
```

## GET /api/v1/admin/subscriptions

Returns subscriptions for all stores, including store and owner information.

## GET /api/v1/admin/stores/:storeID/subscription

Returns the current subscription for any store, accessed as admin.

## PUT /api/v1/admin/stores/:storeID/subscription

Changes the plan for a store.

Request body:

```json
{
  "plan_code": "growth"
}
```

## PATCH /api/v1/admin/stores/:storeID/subscription/status

Changes the status of the store's current subscription.

Request body:

```json
{
  "status": "past_due"
}
```

Supported statuses:
- `trialing`
- `active`
- `past_due`
- `cancelled`
- `expired`

## Notes

- These routes do not require membership in `store_members`
- Used for support, billing, or back-office admin operations

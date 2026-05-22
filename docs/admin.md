# Admin Guide

This document summarizes the admin-side usage of the POS backend system.

## Admin Role

The admin side uses the role:

- `platform_admin`

Users with this role can access routes under `/api/v1/admin/*` without needing to be a member in `store_members` for the target store.

## Authentication

All admin endpoints require a Bearer token:

```http
Authorization: Bearer <access_token>
```

If the token does not belong to a `platform_admin`, the system returns `403 Forbidden`.

## Current Admin APIs

The system currently exposes admin APIs for managing store subscriptions.

### GET /api/v1/admin/subscriptions

Returns the subscription list for all stores in the system, including store and owner details.

### GET /api/v1/admin/stores/:storeID/subscription

Returns the current subscription for the specified store.

### PUT /api/v1/admin/stores/:storeID/subscription

Changes a store's plan, e.g. from `starter` to `growth`.

Request body:

```json
{
  "plan_code": "growth"
}
```

### PATCH /api/v1/admin/stores/:storeID/subscription/status

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

## Recommended Admin Flow

1. Log in with an account that has the `platform_admin` role
2. Call `GET /api/v1/admin/subscriptions` to get an overview
3. Select the `storeID` to manage
4. Call `GET /api/v1/admin/stores/:storeID/subscription` to check the current status
5. To change the plan, use `PUT`
6. To change the billing status, use `PATCH .../status`

## Related Docs

- [Admin Subscription API](admin-subscription.md)
- [Subscription API](subscription.md)
- [System Flow](flow.md)

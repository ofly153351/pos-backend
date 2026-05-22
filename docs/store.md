# Store API

This document describes the API for managing store data.

All endpoints require: `Authorization: Bearer <token>`

## Base URL

For local development:

```bash
http://localhost:8080
```

The system supports two prefixes:

- `/api/v1` (primary)
- `/api` (compatibility path)

The examples below use `/api/v1`. Calls via `/api` work identically.

## GET /api/v1/me/stores

Returns the list of stores belonging to the current user.

Note: Also aliased as `GET /api/v1/stores` for compatibility with certain frontend flows.

### Example

```bash
curl http://localhost:8080/api/v1/me/stores \
  -H "Authorization: Bearer <token>"
```

### Success Response

Status: `200 OK`

```json
{
  "success": true,
  "message": "stores fetched",
  "data": [
    {
      "id": "65b493e98058f410a890859f",
      "owner_user_id": "65b4927f5d58d4a7ed8a4bf1",
      "name": "Main Branch",
      "logo_url": "http://localhost:9000/pos-assets/logos/xxx.png",
      "phone": "021234567",
      "address": "Bangkok",
      "promptpay_id": "0812345678",
      "currency_code": "THB",
      "subscription_plan_code": "growth",
      "subscription_status": "active",
      "subscription_period_end": "2026-04-22T10:00:00Z",
      "created_at": "2026-03-23T10:00:00Z"
    }
  ]
}
```

### Access Rules

- `owner` and `manager` see only stores they are members of
- `platform_admin` sees all stores

## POST /api/v1/stores

Creates a new store (multipart/form-data).

### Request Fields

- `name` required
- `phone` optional
- `address` optional
- `currency_code` optional (default `THB`)
- `promptpay_id` optional
- `subscription_plan_code` required (`starter`, `growth`, `pro`)
- `logo` optional (file)

### Example

```bash
curl -X POST http://localhost:8080/api/v1/stores \
  -H "Authorization: Bearer <token>" \
  -F "name=Main Branch" \
  -F "phone=021234567" \
  -F "address=Bangkok" \
  -F "promptpay_id=0812345678" \
  -F "currency_code=THB" \
  -F "subscription_plan_code=growth" \
  -F "logo=@/path/to/logo.png"
```

### Success Response

Status: `201 Created`

```json
{
  "success": true,
  "message": "store created",
  "data": {
    "id": "65b493e98058f410a890859f",
    "owner_user_id": "65b4927f5d58d4a7ed8a4bf1",
    "name": "Main Branch",
    "logo_url": "http://localhost:9000/pos-assets/stores/logo-xxx.png",
    "phone": "021234567",
    "address": "Bangkok",
    "promptpay_id": "0812345678",
    "currency_code": "THB",
    "subscription_plan_code": "growth",
    "subscription_status": "active",
    "subscription_period_end": "2026-04-22T10:00:00Z",
    "created_at": "2026-03-23T10:00:00Z"
  }
}
```

### Error Status

- `400 Bad Request` — invalid input
- `401 Unauthorized` — missing or invalid token
- `500 Internal Server Error`

## GET /api/v1/stores/:storeID

Returns store data by `storeID`, including the store's latest subscription details.

### Example

```bash
curl http://localhost:8080/api/v1/stores/65b493e98058f410a890859f \
  -H "Authorization: Bearer <token>"
```

Compatibility path:

```bash
curl http://localhost:8080/api/stores/65b493e98058f410a890859f \
  -H "Authorization: Bearer <token>"
```

### Success Response

Status: `200 OK`

```json
{
  "success": true,
  "message": "store fetched",
  "data": {
    "id": "65b493e98058f410a890859f",
    "owner_user_id": "65b4927f5d58d4a7ed8a4bf1",
    "name": "Main Branch",
    "logo_url": "http://localhost:9000/pos-assets/stores/logo-xxx.png",
    "phone": "021234567",
    "address": "Bangkok",
    "promptpay_id": "0812345678",
    "currency_code": "THB",
    "subscription_plan_code": "growth",
    "subscription_status": "active",
    "subscription_period_end": "2026-04-22T10:00:00Z",
    "created_at": "2026-03-23T10:00:00Z"
  }
}
```

### Access Rules

- `platform_admin` can access any store
- `owner` and `manager` can only access stores they are members of
- Other roles or non-members receive `403 Forbidden`

### Error Status

- `401 Unauthorized`
- `403 Forbidden`
- `404 Not Found` — store not found
- `500 Internal Server Error`

## PUT /api/v1/stores/:storeID

Updates store data. Supports both `application/json` and `multipart/form-data` (for logo upload).

### Request Fields

- `name` optional
- `phone` optional
- `address` optional
- `promptpay_id` optional
- `currency_code` optional
- `logo` optional (file)

### Example (JSON)

```bash
curl -X PUT http://localhost:8080/api/v1/stores/65b493e98058f410a890859f \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <token>" \
  -d '{
    "promptpay_id": "0812345678"
  }'
```

### Example (multipart/form-data)

```bash
curl -X PUT http://localhost:8080/api/v1/stores/65b493e98058f410a890859f \
  -H "Authorization: Bearer <token>" \
  -F "name=Main Branch (Renamed)" \
  -F "phone=020000000" \
  -F "promptpay_id=0812345678" \
  -F "logo=@/path/to/new-logo.png"
```

### Success Response

Status: `200 OK`

```json
{
  "success": true,
  "message": "store updated",
  "data": {
    "id": "65b493e98058f410a890859f",
    "owner_user_id": "65b4927f5d58d4a7ed8a4bf1",
    "name": "Main Branch (Renamed)",
    "logo_url": "http://localhost:9000/pos-assets/logos/xxx.png",
    "phone": "020000000",
    "address": "Bangkok",
    "promptpay_id": "0812345678",
    "currency_code": "THB",
    "subscription_plan_code": "growth",
    "subscription_status": "active",
    "subscription_period_end": "2026-04-22T10:00:00Z",
    "created_at": "2026-03-23T10:00:00Z"
  }
}
```

### Access Rules

- `platform_admin` can access any store
- `owner` and `manager` can only edit stores they are members of
- Other roles or non-members receive `403 Forbidden`

### Error Status

- `400 Bad Request` — invalid input
- `401 Unauthorized`
- `403 Forbidden`
- `404 Not Found`
- `500 Internal Server Error`

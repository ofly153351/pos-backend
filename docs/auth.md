# Auth API

This document describes the Auth API for the POS backend system, built with Go and Fiber.

## Base URL

For local development:

```bash
http://localhost:8080
```

If `APP_PORT` is changed, use the port from that environment variable.

## Common Response Format

All endpoints respond in this format:

```json
{
  "success": true,
  "message": "register success",
  "data": {}
}
```

Error response:

```json
{
  "success": false,
  "message": "invalid email or password",
  "error": null
}
```

## POST /api/v1/auth/register

Registers a new user.

### Request Body

```json
{
  "name": "POS Admin",
  "email": "admin@example.com",
  "password": "password123"
}
```

### Validation

- `name` must not be empty
- `email` must be a valid email address
- `password` must be at least 8 characters

### Example

```bash
curl -X POST http://localhost:8080/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{
    "name": "POS Admin",
    "email": "admin@example.com",
    "password": "password123"
  }'
```

### Success Response

Status: `201 Created`

```json
{
  "success": true,
  "message": "register success",
  "data": {
    "user": {
      "id": "generated-user-id",
      "name": "POS Admin",
      "email": "admin@example.com",
      "created_at": "2026-03-22T13:00:00Z"
    },
    "store_id": "generated-store-id",
    "access_token": "jwt-token",
    "token_type": "Bearer"
  }
}
```

### Error Status

- `400 Bad Request` — invalid input
- `409 Conflict` — email already in use
- `500 Internal Server Error`

Notes:
- `store_id` is returned when this user already has a store in `store_members`

## POST /api/v1/auth/login

Authenticates a user.

### Request Body

```json
{
  "email": "admin@example.com",
  "password": "password123"
}
```

### Example

```bash
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "email": "admin@example.com",
    "password": "password123"
  }'
```

### Success Response

Status: `200 OK`

```json
{
  "success": true,
  "message": "login success",
  "data": {
    "user": {
      "id": "generated-user-id",
      "name": "POS Admin",
      "email": "admin@example.com",
      "created_at": "2026-03-22T13:00:00Z"
    },
    "store_id": "generated-store-id",
    "access_token": "jwt-token",
    "token_type": "Bearer"
  }
}
```

### Error Status

- `400 Bad Request` — invalid request body
- `401 Unauthorized` — incorrect email or password
- `500 Internal Server Error`

## POST /api/v1/auth/logout

Performs a full logout (revokes the token).

This endpoint requires a valid Bearer token:

```bash
curl -X POST http://localhost:8080/api/v1/auth/logout \
  -H "Authorization: Bearer <token>"
```

Success:

- Status `200 OK`
- The current token and all previous tokens for the same user are immediately revoked

Notes:

- After logout, the user must log in again to receive a new token version
- Both `/api/v1/auth/logout` and `/api/auth/logout` are supported

## Health Check

Used to verify the API is running.

### GET /health

```bash
curl http://localhost:8080/health
```

Response:

```json
{
  "status": "ok"
}
```

## Notes

- The token is returned in the `access_token` field
- After login, the system attempts to return the user's first `store_id` if a membership already exists
- The current token type is `Bearer`
- Auth data is currently stored in-memory and will be lost on service restart

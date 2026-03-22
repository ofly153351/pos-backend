# Auth API

เอกสารนี้อธิบายการใช้งาน Auth API ของระบบ POS backend ปัจจุบันที่พัฒนาด้วย Go และ Fiber

## Base URL

สำหรับ local:

```bash
http://localhost:8080
```

ถ้าเปลี่ยน `APP_PORT` ให้ใช้ port ตาม env นั้น

## Common Response Format

ทุก endpoint จะตอบกลับในรูปแบบนี้:

```json
{
  "success": true,
  "message": "register success",
  "data": {}
}
```

error response:

```json
{
  "success": false,
  "message": "invalid email or password",
  "error": null
}
```

## POST /api/v1/auth/register

ใช้สำหรับสมัครผู้ใช้งานใหม่

### Request Body

```json
{
  "name": "POS Admin",
  "email": "admin@example.com",
  "password": "password123"
}
```

### Validation

- `name` ต้องไม่ว่าง
- `email` ต้องเป็น email ที่ถูกต้อง
- `password` ต้องยาวอย่างน้อย 8 ตัวอักษร

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
    "access_token": "jwt-token",
    "token_type": "Bearer"
  }
}
```

### Error Status

- `400 Bad Request` ข้อมูลไม่ถูกต้อง
- `409 Conflict` email ถูกใช้งานแล้ว
- `500 Internal Server Error` server error

## POST /api/v1/auth/login

ใช้สำหรับเข้าสู่ระบบ

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
    "access_token": "jwt-token",
    "token_type": "Bearer"
  }
}
```

### Error Status

- `400 Bad Request` request body ไม่ถูกต้อง
- `401 Unauthorized` email หรือ password ไม่ถูกต้อง
- `500 Internal Server Error` server error

## Health Check

ใช้ตรวจสอบว่า API ยังทำงานอยู่

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

- token ที่ส่งกลับอยู่ใน field `access_token`
- token type ปัจจุบันคือ `Bearer`
- auth data ตอนนี้เก็บแบบ in-memory จึงจะหายเมื่อ restart service

# Store API

เอกสารนี้อธิบายการใช้งาน API สำหรับจัดการข้อมูลร้านค้า

## Base URL

สำหรับ local:

```bash
http://localhost:8080
```

ระบบรองรับ 2 prefix:

- `/api/v1` (หลัก)
- `/api` (compatibility path)

ตัวอย่าง endpoint ต่อไปนี้จะใช้ `/api/v1` เป็นหลัก ถ้าฝั่ง frontend เรียกผ่าน `/api` ก็ใช้งานได้เหมือนกัน

## Authentication

ทุก endpoint ในเอกสารนี้ต้องส่ง Bearer token:

```http
Authorization: Bearer <access_token>
```

## GET /api/v1/me/stores

ดึงรายการร้านของ user ปัจจุบัน

หมายเหตุ: มี alias เป็น `GET /api/v1/stores` เช่นกัน เพื่อความเข้ากันได้กับ frontend บาง flow

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

- `owner` และ `manager` จะเห็นเฉพาะร้านที่ตัวเองเป็นสมาชิก
- `platform_admin` จะเห็นทุกร้าน

## POST /api/v1/stores

สร้างร้านใหม่ (multipart/form-data)

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

- `400 Bad Request` ข้อมูลไม่ถูกต้อง
- `401 Unauthorized` ไม่มี/token ไม่ถูกต้อง
- `500 Internal Server Error`

## GET /api/v1/stores/:storeID

ดึงข้อมูลร้านตาม `storeID` (รวมข้อมูล subscription ล่าสุดของร้าน)

### Example

```bash
curl http://localhost:8080/api/v1/stores/65b493e98058f410a890859f \
  -H "Authorization: Bearer <token>"
```

compatibility path:

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

- `platform_admin` เข้าถึงได้ทุกร้าน
- `owner` และ `manager` เข้าถึงเฉพาะร้านที่ตนเป็นสมาชิก
- role อื่นหรือไม่ใช่สมาชิกจะได้ `403 Forbidden`

### Error Status

- `401 Unauthorized`
- `403 Forbidden`
- `404 Not Found` ไม่พบร้าน
- `500 Internal Server Error`

## PUT /api/v1/stores/:storeID

แก้ไขข้อมูลร้าน รองรับทั้ง `application/json` และ `multipart/form-data` (สำหรับอัปโหลดโลโก้)

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

- `platform_admin` เข้าถึงได้ทุกร้าน
- `owner` และ `manager` แก้ไขได้เฉพาะร้านที่ตนเป็นสมาชิก
- role อื่นหรือไม่ใช่สมาชิกจะได้ `403 Forbidden`

### Error Status

- `400 Bad Request` ข้อมูลไม่ถูกต้อง
- `401 Unauthorized`
- `403 Forbidden`
- `404 Not Found`
- `500 Internal Server Error`

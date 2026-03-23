# Customer Network API

API ชุดนี้ใช้สำหรับให้ร้านค้าสร้างและจัดการลูกค้าโดยกำหนด `level` ตรง (ไม่ใช้ parent customer)

## Base URL

```bash
http://localhost:8080
```

รองรับทั้ง:

- `/api/v1/*` (หลัก)
- `/api/*` (compatibility)

## Authorization

ทุก endpoint ต้องส่ง Bearer token:

```http
Authorization: Bearer <access_token>
```

role ที่เข้าถึงได้:

- `platform_admin`
- สมาชิก `store_members` ที่เป็น `owner`, `manager`, `cashier`

หมายเหตุ:
- endpoint จัดการกฎส่วนลดระดับ (`customer-level-discounts`) จำกัดที่ `owner`, `manager`, `platform_admin`

## Data Model (ย่อ)

- ลูกค้าแต่ละคนอยู่ใต้ `store_id`
- ลูกค้าแต่ละคนมี `level` เช่น `1`, `2`, `3`
- ระบบส่วนลดจะ map จาก `level -> discount_percent`

## Endpoints

## POST /api/v1/stores/:storeID/customers

สร้างลูกค้า

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

`level` ไม่ส่งได้ จะ default เป็น `1`

## GET /api/v1/stores/:storeID/customers

ดึงรายการลูกค้าทั้งหมดของร้าน

## GET /api/v1/stores/:storeID/customers/:customerID

ดึงลูกค้ารายเดียว

## PATCH /api/v1/stores/:storeID/customers/:customerID

อัปเดตลูกค้า

Request body (ส่งเฉพาะที่ต้องแก้):

```json
{
  "level": 3,
  "full_name": "Acme Retail Partner Updated",
  "email": "new@example.com",
  "is_active": true
}
```

## DELETE /api/v1/stores/:storeID/customers/:customerID

ลบลูกค้า

## GET /api/v1/stores/:storeID/customer-level-discounts

ดึงค่าตั้งส่วนลดตามระดับลูกค้าของร้าน

response ตัวอย่าง:

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

ตั้งค่าหรือแก้ไขส่วนลดของระดับนั้น (upsert)

Request body:

```json
{
  "discount_percent": 7.5
}
```

เงื่อนไข:

- `level` ต้องมากกว่า 0
- `discount_percent` ต้องอยู่ระหว่าง `0-100`

## DELETE /api/v1/stores/:storeID/customer-level-discounts/:level

ลบกฎส่วนลดของระดับนั้น

## การผูกกับ Sales

เมื่อสร้าง sale และส่ง `customer_id`, ระบบจะ:

1. อ่าน `level` ของลูกค้า
2. อ่าน `% ส่วนลด` ตาม level ของร้าน
3. คำนวณส่วนลดเครือข่ายอัตโนมัติในบิล

## Error Status ที่พบบ่อย

- `400` ข้อมูลไม่ถูกต้อง เช่น `full_name` ว่าง, email format ไม่ถูกต้อง, `level <= 0`
- `403` ไม่มีสิทธิ์ในร้านนี้
- `404` ไม่พบลูกค้า
- `500` internal server error

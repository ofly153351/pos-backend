# Admin Subscription API

API ชุดนี้สำหรับ `platform_admin` ใช้จัดการ subscription ของร้านทั้งหมดในระบบ

## Base

ทุกเส้นต้องส่ง Bearer token ของ user ที่มี role `platform_admin`

```http
Authorization: Bearer <access_token>
```

## GET /api/v1/admin/subscriptions

ดึง subscription ของทุกร้าน พร้อมข้อมูลร้านและ owner

## GET /api/v1/admin/stores/:storeID/subscription

ดึง subscription ปัจจุบันของร้านใดก็ได้ในฐานะ admin

## PUT /api/v1/admin/stores/:storeID/subscription

เปลี่ยน plan ของร้าน

Request body:

```json
{
  "plan_code": "growth"
}
```

## PATCH /api/v1/admin/stores/:storeID/subscription/status

เปลี่ยน status ของ subscription ปัจจุบันของร้าน

Request body:

```json
{
  "status": "past_due"
}
```

status ที่รองรับ:
- `trialing`
- `active`
- `past_due`
- `cancelled`
- `expired`

## Notes

- route ชุดนี้ไม่ต้องอาศัย `store_members`
- ใช้สำหรับงาน support, billing, หรือ backoffice admin

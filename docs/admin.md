# Admin Guide

เอกสารนี้สรุปการใช้งานฝั่ง admin ของระบบ POS backend

## Admin Role

ฝั่ง admin ใช้ role:

- `platform_admin`

user ที่มี role นี้จะเข้าถึง route ใต้ `/api/v1/admin/*` ได้ โดยไม่ต้องเป็นสมาชิกใน `store_members` ของร้านนั้น

## Authentication

ทุกเส้น admin ต้องส่ง Bearer token:

```http
Authorization: Bearer <access_token>
```

ถ้า token ไม่ใช่ของ `platform_admin` ระบบจะตอบกลับ `403 Forbidden`

## Current Admin APIs

ตอนนี้ระบบเปิด admin API สำหรับจัดการ subscription ของร้าน

### GET /api/v1/admin/subscriptions

ดึงรายการ subscription ของทุกร้านในระบบ พร้อมข้อมูลร้านและ owner

### GET /api/v1/admin/stores/:storeID/subscription

ดึง subscription ปัจจุบันของร้านที่ระบุ

### PUT /api/v1/admin/stores/:storeID/subscription

เปลี่ยน plan ของร้าน เช่นจาก `starter` ไป `growth`

Request body:

```json
{
  "plan_code": "growth"
}
```

### PATCH /api/v1/admin/stores/:storeID/subscription/status

เปลี่ยนสถานะ subscription ปัจจุบันของร้าน

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

## Recommended Admin Flow

1. login ด้วย account ที่มี role `platform_admin`
2. เรียก `GET /api/v1/admin/subscriptions` เพื่อดูภาพรวม
3. เลือก `storeID` ที่ต้องการจัดการ
4. เรียก `GET /api/v1/admin/stores/:storeID/subscription` เพื่อตรวจสถานะล่าสุด
5. ถ้าต้องการเปลี่ยน plan ใช้ `PUT`
6. ถ้าต้องการเปลี่ยนสถานะ billing ใช้ `PATCH .../status`

## Related Docs

- [Admin Subscription API](/Users/obx/projects/pos-backend/docs/admin-subscription.md)
- [Subscription API](/Users/obx/projects/pos-backend/docs/subscription.md)
- [System Flow](/Users/obx/projects/pos-backend/docs/flow.md)

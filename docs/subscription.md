# Subscription API

API นี้ใช้ดูแผน subscription และเปลี่ยนแผนของร้าน

## Base

ทุกเส้นต้องส่ง Bearer token:

```http
Authorization: Bearer <access_token>
```

## GET /api/v1/subscriptions/plans

ดึงรายการแผนทั้งหมดที่เปิดใช้งาน

Example:

```bash
curl http://localhost:8080/api/v1/subscriptions/plans \
  -H "Authorization: Bearer <token>"
```

## GET /api/v1/stores/:storeID/subscription

ดึง subscription ปัจจุบันของร้าน

## PUT /api/v1/stores/:storeID/subscription

เปลี่ยนแผนของร้าน

Request body:

```json
{
  "plan_code": "growth"
}
```

Example:

```bash
curl -X PUT http://localhost:8080/api/v1/stores/ad65bd37a3b3da7828ec9111/subscription \
  -H "Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJlbWFpbCI6InBlZWwyYXB1dEBnbWFpbC5jb20iLCJleHAiOjE3NzQyNTI4MTEsIm5hbWUiOiJwaGlyYXBoYXQga2xpbnRhbiIsInJvbGUiOiJvd25lciIsInN1YiI6ImFkNjViZDM3YTNiM2RhNzgyOGVjOTExMSJ9.srukC9ttNmpMjlqzVUpTc_RHBp6A2zHGRDD1NuIhUwE" \
  -H "Content-Type: application/json" \
  -d '{"plan_code":"growth"}'
```

## Notes

- การเปลี่ยนแผนจะปิด subscription เดิมที่ยัง active อยู่ก่อน แล้วสร้าง row ใหม่
- สิทธิ์เปลี่ยนแผนต้องเป็น `owner`, `manager` หรือ `platform_admin`

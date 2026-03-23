# Invoice API

API สำหรับออกบิลค้างชำระให้ลูกค้าในเครือ และออกรายงาน PDF

## Base

- `Authorization: Bearer <access_token>`
- รองรับทั้ง `/api/v1/*` และ `/api/*`

## POST /api/v1/stores/:storeID/invoices

สร้างบิลค้างชำระ (สถานะเริ่มต้น `unpaid`)

```json
{
  "customer_id": "7e9d5ccccf930438e6ba0d0f",
  "due_at": "2026-04-30T00:00:00Z",
  "note": "เครดิต 30 วัน",
  "items": [
    {
      "product_id": "prod_001",
      "quantity": 3,
      "discount_type": "percent",
      "discount_value": 10
    }
  ]
}
```

Behavior:
- ต้องเป็นลูกค้าในเครือ (`customer_id`) เท่านั้น
- คำนวณส่วนลดเครือข่ายอัตโนมัติตาม `customer_level_discounts`
- ตัดสต็อกสินค้าใน transaction เดียวกับการสร้าง invoice

## GET /api/v1/stores/:storeID/invoices

ดึงรายการบิลค้างชำระของร้าน

## GET /api/v1/stores/:storeID/invoices/:invoiceID

ดึงรายละเอียดบิล 1 ใบ พร้อม `items` และ `payments`

## POST /api/v1/stores/:storeID/invoices/:invoiceID/payments

บันทึกการชำระบางส่วนหรือปิดบิล

```json
{
  "paid_amount": 500,
  "payment_method": "bank_transfer",
  "note": "โอนงวดแรก"
}
```

Status transition:
- `unpaid` -> `partially_paid` -> `paid`

## GET /api/v1/stores/:storeID/invoices/:invoiceID/pdf

สร้างเอกสาร PDF ของ invoice แล้วส่งกลับเป็น `application/pdf`

สามารถเปิดตรงใน browser หรือดาวน์โหลดได้ทันที

## Error ที่พบบ่อย

- `400` ข้อมูลไม่ถูกต้อง, จ่ายเกินยอดคงเหลือ, บิลปิดแล้ว
- `403` ไม่มีสิทธิ์ในร้าน
- `404` ไม่พบ invoice / customer / product
- `500` internal server error

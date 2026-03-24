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

บันทึกการชำระบางส่วนหรือปิดบิล พร้อมแนบหลักฐานการชำระเงินได้

รองรับ 2 รูปแบบ:
- `application/json` (เดิม)
- `multipart/form-data` (แนะนำเมื่อมีไฟล์หลักฐาน)

```json
{
  "paid_amount": 500,
  "payment_method": "bank_transfer",
  "note": "โอนงวดแรก"
}
```

ตัวอย่างแนบหลักฐาน (รูปหรือ PDF):

```bash
curl -X POST http://localhost:8080/api/v1/stores/{storeID}/invoices/{invoiceID}/payments \
  -H "Authorization: Bearer <token>" \
  -F "paid_amount=500" \
  -F "payment_method=bank_transfer" \
  -F "note=โอนงวดแรก" \
  -F "proof=@/path/to/slip.pdf"
```

ข้อกำหนดไฟล์ `proof`:
- รองรับ `image/jpeg`, `image/png`, `image/webp`, `application/pdf`
- ขนาดไฟล์สูงสุด `10MB`
- จัดเก็บใน MinIO และจะได้ URL ในข้อมูล payment (`proof_url`)

ตัวอย่างฟิลด์ที่เพิ่มใน `payments[]`:

```json
{
  "id": "pay_001",
  "paid_amount": 500,
  "payment_method": "bank_transfer",
  "proof_url": "http://127.0.0.1:9000/pos-assets/invoice-payments/abc123.pdf",
  "proof_mime_type": "application/pdf",
  "proof_file_name": "slip.pdf"
}
```

Status transition:
- `unpaid` -> `partially_paid` -> `paid`

## GET /api/v1/stores/:storeID/invoices/:invoiceID/payments/:paymentID/proof

เปิดดูหลักฐานการชำระเงินของ payment รายการนั้น (redirect ไปไฟล์ใน MinIO)

```bash
curl -L http://localhost:8080/api/v1/stores/{storeID}/invoices/{invoiceID}/payments/{paymentID}/proof \
  -H "Authorization: Bearer <token>"
```

หมายเหตุ:
- `paymentID` เอาจาก `payments[].id` ใน `GET /invoices/:invoiceID`
- ถ้า payment นั้นไม่มีหลักฐาน จะได้ `404 payment proof not found`

## POST /api/v1/stores/:storeID/invoices/:invoiceID/unpay

ใช้เมื่อชำระผิดพลาดจาก user error แล้วต้องการย้อนสถานะกลับเป็น `unpaid`

```json
{
  "reason": "บันทึกชำระผิดใบแจ้งหนี้"
}
```

Behavior:
- ต้องส่ง `reason` ทุกครั้ง (เก็บ audit)
- ระบบจะ mark payment ที่เคยบันทึกไว้ทั้งหมดเป็น `is_voided=true` (ไม่ลบทิ้ง)
- รีเซ็ต invoice เป็น:
  - `status = unpaid`
  - `paid_amount = 0`
  - `remaining_amount = total_amount`
  - `payment_method = null`

หมายเหตุ:
- ใน `payments[]` จะเห็นข้อมูล audit เพิ่ม เช่น `is_voided`, `voided_at`, `voided_by_user_id`, `void_reason`

## GET /api/v1/stores/:storeID/invoices/:invoiceID/pdf

สร้างเอกสาร PDF ของ invoice แล้วส่งกลับเป็น `application/pdf`

สามารถเปิดตรงใน browser หรือดาวน์โหลดได้ทันที

## Error ที่พบบ่อย

- `400` ข้อมูลไม่ถูกต้อง, จ่ายเกินยอดคงเหลือ, บิลปิดแล้ว
- `403` ไม่มีสิทธิ์ในร้าน
- `404` ไม่พบ invoice / customer / product
- `500` internal server error

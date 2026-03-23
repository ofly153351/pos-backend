# Product API

API นี้ใช้จัดการสินค้าในร้าน โดยแยก 2 แนวคิด:

- `product_type` ของร้าน: หมวดสินค้า เช่น `กาแฟ`, `เบเกอรี่`, `อุปกรณ์`
- `unit_type` ของสินค้า: หน่วยขาย เช่น `piece`, `pair`, `box`

สินค้าแต่ละตัวจะอ้าง `product_type_id` ของร้าน รองรับรูปสินค้า ราคาพิเศษ และจำนวนคงเหลือ (`quantity`)

## Base

ทุกเส้นต้องส่ง Bearer token:

```http
Authorization: Bearer <access_token>
```

## Unit Type

- `piece`: ขายเป็นชิ้น
- `pair`: ขายเป็นคู่

## Product Type APIs

### POST /api/v1/stores/:storeID/product-types

```json
{
  "name": "Coffee",
  "description": "Hot and iced coffee menu",
  "is_active": true
}
```

### GET /api/v1/stores/:storeID/product-types

### PATCH /api/v1/stores/:storeID/product-types/:productTypeID

### DELETE /api/v1/stores/:storeID/product-types/:productTypeID

## POST /api/v1/stores/:storeID/products

สร้างสินค้าใหม่ด้วย `multipart/form-data`

Fields:
- `name` required
- `sku` optional
- `product_type_id` optional, ต้องเป็น type ของร้านนั้น
- `unit_type` optional, default `piece`, รับค่า string ตามที่ร้านต้องการ
- `quantity` optional, default `0`, ต้องเป็นจำนวนเต็มตั้งแต่ `0` ขึ้นไป
- `base_price` required
- `special_price` optional
- `special_price_start_at` optional, RFC3339
- `special_price_end_at` optional, RFC3339
- `is_active` optional, default `true`
- `image` optional, file

Example:

```bash
curl -X POST http://localhost:8080/api/v1/stores/{storeID}/products \
  -H "Authorization: Bearer <token>" \
  -F "name=Coffee Mug" \
  -F "sku=MUG-001" \
  -F "product_type_id=type_xxx" \
  -F "unit_type=piece" \
  -F "quantity=24" \
  -F "base_price=120" \
  -F "special_price=99" \
  -F "image=@/path/to/product.jpg"
```

## GET /api/v1/stores/:storeID/products

ดึงรายการสินค้าทั้งหมดของร้าน

## GET /api/v1/stores/:storeID/products/:productID

ดึงข้อมูลสินค้า 1 รายการ

## PATCH /api/v1/stores/:storeID/products/:productID

อัปเดตสินค้าแบบ `multipart/form-data`

Fields ที่รองรับ:
- `name`
- `sku`
- `product_type_id`
- `unit_type`
- `quantity`
- `base_price`
- `special_price`
- `clear_special_price=true`
- `special_price_start_at`
- `special_price_end_at`
- `clear_special_window=true`
- `is_active`
- `image`

## DELETE /api/v1/stores/:storeID/products/:productID

ลบสินค้าออกจากระบบ

## Notes

- ราคาที่ตอบกลับจะมี `effective_price` คำนวณจาก special price window
- response ของสินค้าแต่ละรายการจะมี `quantity` เป็นจำนวนคงเหลือปัจจุบัน
- รูปสินค้าจะถูกอัปโหลดไปที่ MinIO และระบบจะเก็บ URL ที่ได้ไว้ในฐานข้อมูล
- แนะนำให้สร้าง `product_type` ของร้านก่อนแล้วค่อยสร้างสินค้า
- สำหรับ `unit_type` เฉพาะร้านอาจสร้างชุดของหน่วยไว้ก่อนด้วย `POST /api/v1/stores/:storeID/product-units` แล้ว fetch list เพื่อผูกกับ dropdown

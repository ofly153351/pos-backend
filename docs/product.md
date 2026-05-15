# Product API

API นี้ใช้จัดการสินค้าในร้าน โดยแยก 2 แนวคิด:

- `product_type` ของร้าน: หมวดสินค้า เช่น `กาแฟ`, `เบเกอรี่`, `อุปกรณ์`
- `product_unit` ของร้าน: หน่วยขาย เช่น `ชิ้น`, `คู่`, `กล่อง`

สินค้าแต่ละตัวจะอ้าง `product_type_id` ของร้าน รองรับการอ้างแบรนด์ผ่าน `brand_id`, รูปสินค้า, ราคาพิเศษ และจำนวนคงเหลือ (`quantity`)

## Base

ทุกเส้นต้องส่ง Bearer token:

```http
Authorization: Bearer <access_token>
```

## Product Unit

- สินค้าจะอ้างอิงหน่วยผ่าน `unit_id` (FK ไป `product_units.id`)
- ต้องสร้างหน่วยด้วย `POST /api/v1/stores/:storeID/product-units` ก่อน แล้วค่อยผูกกับสินค้า

## Product Brand

- สินค้าจะอ้างอิงแบรนด์ผ่าน `brand_id` (FK ไป `product_brands.id`)
- ตาราง `product_brands` เป็นข้อมูลแยกตามร้าน (`store_id`)
- response สินค้าจะคืนทั้ง `brand_id` และ `brand_name`

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
- `brand_id` optional (ต้องเป็นแบรนด์ของร้านนั้น)
- `sku` optional (ถ้าไม่ส่ง ระบบจะ generate barcode ให้เป็น EAN-13 อัตโนมัติ)
- `product_type_id` optional, ต้องเป็น type ของร้านนั้น
- `unit_id` required, ต้องเป็น unit ของร้านนั้น
- `quantity` optional, default `0`, ต้องเป็นจำนวนเต็มตั้งแต่ `0` ขึ้นไป
- `base_price` required
- `special_price` optional
- `special_price_start_at` optional, RFC3339
- `special_price_end_at` optional, RFC3339
- `is_active` optional, default `true`
- `image` optional, file

ตัวอย่าง (อัปโหลดรูปสินค้า):

```bash
curl -X POST http://localhost:8080/api/v1/stores/{storeID}/products \
  -H "Authorization: Bearer <token>" \
  -F "name=Coffee Mug" \
  -F "brand_id=brand_xxx" \
  -F "sku=MUG-001" \
  -F "product_type_id=type_xxx" \
  -F "unit_id=unit_xxx" \
  -F "quantity=24" \
  -F "base_price=120" \
  -F "special_price=99" \
  -F "image=@/path/to/product.jpg"
```

Success Response (`201 Created`)

```json
{
  "success": true,
  "message": "product created",
  "data": {
    "id": "3f7cbf2b6f9415d4d31811af",
    "store_id": "65b493e98058f410a890859f",
    "name": "Coffee Mug",
    "brand_id": "brand_xxx",
    "brand_name": "Acme",
    "sku": "MUG-001",
    "product_unit_id": "unit_xxx",
    "product_unit_name": "piece",
    "image_url": "http://127.0.0.1:9000/pos-assets/products/3f7cbf2b6f9415d4d31811af.jpg",
    "quantity": 24,
    "base_price": 120,
    "special_price": 99,
    "effective_price": 99,
    "is_active": true,
    "created_at": "2026-03-23T09:00:00Z",
    "updated_at": "2026-03-23T09:00:00Z"
  }
}
```

## GET /api/v1/stores/:storeID/products

ดึงรายการสินค้าแบบ pagination เพื่อลด response size

Query params:
- `page` optional, default `1`
- `limit` optional, default `50`, max `200`

ตัวอย่าง:

```bash
curl "http://localhost:8080/api/v1/stores/{storeID}/products?page=1&limit=50" \
  -H "Authorization: Bearer <token>"
```

Success Response (`200 OK`)

```json
{
  "success": true,
  "message": "products fetched",
  "data": {
    "items": [
      {
        "id": "prod_001",
        "name": "Coffee Mug",
        "quantity": 24,
        "base_price": 120,
        "effective_price": 120
      }
    ],
    "page": 1,
    "limit": 50,
    "total": 1000000,
    "total_pages": 20000,
    "has_next": true,
    "has_prev": false
  }
}
```

## GET /api/v1/stores/:storeID/products/:productID

ดึงข้อมูลสินค้า 1 รายการ

## PATCH /api/v1/stores/:storeID/products/:productID

อัปเดตสินค้าแบบ `multipart/form-data`

Fields ที่รองรับ:
- `name`
- `brand_id`
- `sku`
- `product_type_id`
- `unit_id`
- `quantity`
- `base_price`
- `special_price`
- `clear_special_price=true`
- `special_price_start_at`
- `special_price_end_at`
- `clear_special_window=true`
- `is_active`
- `image`

ตัวอย่าง (เปลี่ยนรูปสินค้า):

```bash
curl -X PATCH http://localhost:8080/api/v1/stores/{storeID}/products/{productID} \
  -H "Authorization: Bearer <token>" \
  -F "name=Coffee Mug 2026" \
  -F "brand_id=brand_yyy" \
  -F "base_price=129" \
  -F "image=@/path/to/new-product.jpg"
```

Success Response (`200 OK`)

```json
{
  "success": true,
  "message": "product updated",
  "data": {
    "id": "3f7cbf2b6f9415d4d31811af",
    "store_id": "65b493e98058f410a890859f",
    "name": "Coffee Mug 2026",
    "brand_id": "brand_yyy",
    "brand_name": "Acme Pro",
    "product_unit_id": "unit_xxx",
    "product_unit_name": "piece",
    "image_url": "http://127.0.0.1:9000/pos-assets/products/88b9a45a623ad97f0a7f2121.jpg",
    "quantity": 24,
    "base_price": 129,
    "effective_price": 129,
    "is_active": true,
    "created_at": "2026-03-23T09:00:00Z",
    "updated_at": "2026-03-23T10:15:00Z"
  }
}
```

## DELETE /api/v1/stores/:storeID/products/:productID

ลบสินค้าออกจากระบบ

## POST /api/v1/stores/:storeID/products/generate-missing-barcodes

ใช้ generate barcode (เก็บใน `sku`) ให้สินค้าที่ `sku` ว่างทั้งหมดในร้าน

Success Response (`200 OK`)

```json
{
  "success": true,
  "message": "missing product barcodes generated",
  "data": {
    "updated_count": 12
  }
}
```

## Notes

- ราคาที่ตอบกลับจะมี `effective_price` คำนวณจาก special price window
- response ของสินค้าแต่ละรายการจะมี `quantity` เป็นจำนวนคงเหลือปัจจุบัน
- response ของสินค้าแต่ละรายการจะมี `brand_id` และ `brand_name` (ถ้าไม่ตั้งค่า จะเป็นค่าว่าง)
- list endpoint ใช้ pagination เสมอเพื่อลด payload (`page/limit`)
- รูปสินค้าจะถูกอัปโหลดไปที่ MinIO path `products/<generated-filename>` และระบบจะคืนค่าในฟิลด์ `image_url`
- ถ้าไม่ส่ง `image` ตอน `PATCH` ระบบจะคงรูปเดิมไว้
- ถ้า `image_url` ว่างจะไม่ถูกส่งกลับใน JSON (เพราะ `omitempty`)
- ถ้าไม่ส่ง `sku` ตอนสร้างสินค้า ระบบจะ generate barcode แบบ EAN-13 ให้อัตโนมัติ
- ถ้าต้องการเติม barcode ให้ข้อมูลเดิมที่ยังว่าง ใช้ endpoint `generate-missing-barcodes`
- แนะนำให้สร้าง `product_type` ของร้านก่อนแล้วค่อยสร้างสินค้า
- ต้องสร้างหน่วยใน `product-units` ก่อน แล้วส่ง `unit_id` ตอนสร้าง/แก้สินค้า

# Product Brand Integration (Current Status)

เอกสารนี้สรุปเฉพาะ endpoint ที่เกี่ยวข้องกับแบรนด์สินค้า (`brand`) ตามโค้ดที่ implement แล้ว ณ ปัจจุบัน

## Data Model

- ใช้ตาราง `product_brands` แยกจาก `products`
- `products.brand_id` เป็น FK ไป `product_brands.id`
- response ของ product จะคืน:
  - `brand_id`
  - `brand_name` (มาจาก join `product_view`)

Migration ที่เกี่ยวข้อง:
- `init-db/026_product_brands_refactor.sql`

## Endpoints ที่ Implement แล้ว (เกี่ยวกับ Brand)

### 1) POST /api/v1/stores/:storeID/products

สร้างสินค้าใหม่ และสามารถแนบ `brand_id` ได้

Field ที่เกี่ยวกับแบรนด์:
- `brand_id` optional
- ถ้าส่งมา ต้องเป็น `brand_id` ของร้านเดียวกัน ไม่งั้นได้ `400` (`brand does not belong to this store`)

ตัวอย่าง:

```bash
curl -X POST http://localhost:8080/api/v1/stores/{storeID}/products \
  -H "Authorization: Bearer <token>" \
  -F "name=Running Shoes" \
  -F "brand_id=br_xxxxx" \
  -F "unit_id=unit_xxx" \
  -F "base_price=2590"
```

### 2) GET /api/v1/stores/:storeID/products

ดึงรายการสินค้า โดยแต่ละ item จะมี `brand_id` และ `brand_name`

### 3) GET /api/v1/stores/:storeID/products/:productID

ดึงสินค้า 1 รายการ พร้อม `brand_id` และ `brand_name`

### 4) PATCH /api/v1/stores/:storeID/products/:productID

อัปเดตสินค้า และเปลี่ยน `brand_id` ได้

Field ที่เกี่ยวกับแบรนด์:
- `brand_id` optional

ตัวอย่าง:

```bash
curl -X PATCH http://localhost:8080/api/v1/stores/{storeID}/products/{productID} \
  -H "Authorization: Bearer <token>" \
  -F "brand_id=br_newbrand"
```

### 5) POST /api/v1/stores/:storeID/products/generate-missing-barcodes

endpoint นี้ไม่แก้ brand โดยตรง แต่ยังอยู่ใน product module เดียวกันและทำงานร่วมกับสินค้าที่มี/ไม่มี `brand_id` ได้ตามปกติ

## Endpoints ที่ยังไม่ Implement

ตอนนี้ **ยังไม่มี** route สำหรับจัดการตาราง `product_brands` โดยตรง เช่น:

- `POST /api/v1/stores/:storeID/product-brands`
- `GET /api/v1/stores/:storeID/product-brands`
- `PATCH /api/v1/stores/:storeID/product-brands/:brandID`
- `DELETE /api/v1/stores/:storeID/product-brands/:brandID`

ดังนั้นการยิง `GET /api/stores/:storeID/product-brands` จะได้ `404` ตามที่พบ

## Notes

- ถ้าต้องการใช้งานหน้าจอเลือกแบรนด์จาก master data จำเป็นต้อง implement product-brand CRUD/list API เพิ่ม
- ในสภาพปัจจุบัน backend รองรับการอ้าง `brand_id` แล้ว แต่ยังต้องมีวิธีเตรียม `product_brands` data แยกต่างหาก (เช่น seed SQL/manual insert)

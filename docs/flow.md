# System Flow

เอกสารนี้อธิบาย flow หลักของระบบ POS จากมุมมองการใช้งานจริงของร้านค้า

## 1. สมัครสมาชิกและเข้าสู่ระบบ

เริ่มจากสร้าง user account แล้ว login เพื่อรับ `access_token`

ลำดับ:
1. `POST /api/v1/auth/register`
2. `POST /api/v1/auth/login`
3. เก็บ `access_token` แล้วส่งผ่าน `Authorization: Bearer <token>`

หมายเหตุ:
- user ที่สมัครใหม่จะใช้ token นี้ไปสร้างร้านและจัดการข้อมูลของร้านตัวเอง

## 2. สร้างร้าน

หลัง login แล้ว owner ต้องสร้างร้านก่อน เพราะข้อมูลอื่นทั้งหมดผูกกับ `store_id`

ลำดับ:
1. `POST /api/v1/stores`
2. ระบบจะสร้าง:
   - row ใน `stores`
   - row owner ใน `store_members`
   - row แรกใน `store_subscriptions`

ผลลัพธ์:
- จะได้ `store_id` ของร้าน
- ถ้ามี logo ร้าน ระบบจะเก็บไฟล์และบันทึก path ลง `logo_url`

## 3. ดูและเลือกแผน subscription

ร้านมี subscription ระดับร้าน ไม่ใช่ระดับ user

ลำดับ:
1. `GET /api/v1/subscriptions/plans`
2. `GET /api/v1/stores/:storeID/subscription`
3. ถ้าต้องการเปลี่ยนแผน ใช้ `PUT /api/v1/stores/:storeID/subscription`

หมายเหตุ:
- ผู้ที่เปลี่ยนแผนได้ต้องเป็น `owner`, `manager` หรือ `platform_admin`

## 4. สร้าง product type ของร้าน

แต่ละร้านมีหมวดสินค้าของตัวเอง เช่น `กาแฟ`, `ขนม`, `ของใช้`

ลำดับ:
1. `POST /api/v1/stores/:storeID/product-types`
2. `GET /api/v1/stores/:storeID/product-types`
3. แก้ไขด้วย `PATCH`
4. ลบด้วย `DELETE`

ผลลัพธ์:
- จะได้ `product_type_id` สำหรับนำไปผูกกับสินค้า

## 5. สร้างสินค้า

สินค้าแต่ละตัวอยู่ใต้ร้าน และอาจอ้าง `product_type_id` ของร้านนั้น

ลำดับ:
1. `POST /api/v1/stores/:storeID/products`
2. ส่งข้อมูลแบบ `multipart/form-data`
3. แนบไฟล์รูปผ่าน field `image` ได้

โครงสร้างสำคัญ:
- `product_type_id` = หมวดสินค้า
- `unit_type` = หน่วยขาย เช่น `piece` หรือ `pair`
- `quantity` = จำนวนสินค้าคงเหลือ
- `base_price` = ราคาปกติ
- `special_price` = ราคาพิเศษ

## 6. จัดการสินค้า

หลังสร้างสินค้าแล้ว สามารถจัดการต่อได้

ลำดับ:
1. `GET /api/v1/stores/:storeID/products`
2. `GET /api/v1/stores/:storeID/products/:productID`
3. `PATCH /api/v1/stores/:storeID/products/:productID`
4. `DELETE /api/v1/stores/:storeID/products/:productID`

หมายเหตุ:
- `PATCH` รองรับการเปลี่ยนรูปสินค้า
- ระบบจะคำนวณ `effective_price` จาก special price window ให้

## 7. สิทธิ์การเข้าถึง

ระบบตรวจสิทธิ์จาก `store_members`

- `owner`: จัดการร้าน, subscription, product types, products ได้
- `manager`: จัดการข้อมูลร้านในระดับปฏิบัติการได้
- `cashier`: ไม่ควรใช้จัดการ config ของร้าน
- `platform_admin`: เข้าถึงได้ทุก store

ถ้าขึ้น error:

```json
{
  "success": false,
  "message": "user cannot manage this store"
}
```

แปลว่า user จาก token ไม่มีสิทธิ์ใน `store_id` นั้น หรือไม่ได้อยู่ใน `store_members`

## 8. หน้าขาย POS

เมื่อมีสินค้าในระบบแล้ว `cashier`, `manager`, `owner` สามารถเปิดหน้าขายเพื่อสร้างบิล

ลำดับ:
1. `GET /api/v1/stores/:storeID/products`
2. เลือกสินค้าและจำนวนที่ต้องขาย
3. `POST /api/v1/stores/:storeID/sales`
4. ถ้าต้องการเปิดใบเสร็จย้อนหลังใช้ `GET /api/v1/stores/:storeID/sales/:saleID`

ข้อมูลสำคัญ:
- ระบบจะหัก `product.quantity` ทันทีเมื่อขายสำเร็จ
- ระบบจะเก็บ snapshot ของชื่อสินค้า, ราคา, จำนวน ที่ขายใน `sale_items`
- หน้าขายสามารถโหลดประวัติด้วย `GET /api/v1/stores/:storeID/sales`

## Recommended Flow

สำหรับร้านใหม่:

1. register
2. login
3. create store
4. get current subscription
5. create product types
6. create products
7. update subscription เมื่อร้านต้องการเปลี่ยนแผน

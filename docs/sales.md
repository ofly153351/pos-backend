# Sales API

API ชุดนี้ใช้สำหรับหน้าขาย POS เพื่อสร้างบิลขาย ดูประวัติการขาย และเปิดรายละเอียดใบเสร็จ

Base: `Authorization: Bearer <token>`

สิทธิ์ที่ใช้งานได้:
- `owner`
- `manager`
- `cashier`
- `platform_admin`

## POST /api/v1/stores/:storeID/sales

สร้างบิลขายและตัดจำนวนสินค้า (`product.quantity`) ใน transaction เดียว โดยรองรับ discount รายการสินค้าแบบต่อหน่วย

```json
{
  "payment_method": "cash",
  "paid_amount": 500,
  "note": "walk-in customer",
  "customer_id": "cust_lv2_001",
  "items": [
    {
      "product_id": "prod_coffee_001",
      "quantity": 2,
      "discount_type": "amount",
      "discount_value": 10
    },
    {
      "product_id": "prod_bakery_002",
      "quantity": 1,
      "discount_type": "percent",
      "discount_value": 15
    }
  ]
}
```

Behavior:
- ใช้ราคาปัจจุบันของสินค้า ณ เวลาขาย (`effective price`)
- `discount_type` รองรับ `amount` และ `percent`
- ถ้าไม่ส่ง `discount_type` และ `discount_value` จะถือว่าไม่มีส่วนลด
- `discount_value` ของ `amount` คือส่วนลดต่อหน่วย
- `discount_value` ของ `percent` ต้องอยู่ในช่วง `0-100`
- ถ้าส่ง `customer_id` ระบบจะคำนวณส่วนลดเครือข่ายเพิ่มตาม level ของลูกค้า (LV1/LV2/...) โดยอิงจาก `customer_level_discounts`
- ส่วนลดเครือข่ายคำนวณต่อหน่วยบนยอดหลังหักส่วนลด manual ของรายการนั้น
- ถ้าสต็อกไม่พอจะไม่สร้างบิล
- ถ้าสินค้า inactive จะไม่ขาย
- ถ้า `paid_amount < total_amount` จะ reject
- บันทึก `subtotal_amount`, `discount_amount`, `total_amount`, `change_amount`

Response shape หลัก:

```json
{
  "success": true,
  "message": "sale created",
  "data": {
    "id": "sale_xxx",
    "payment_method": "cash",
    "customer_id": "cust_lv2_001",
    "customer_level": 2,
    "network_discount_percent": 10,
    "paid_amount": 500,
    "subtotal_amount": 255,
    "discount_amount": 35,
    "total_amount": 220,
    "change_amount": 280,
    "items": [
      {
        "product_id": "prod_coffee_001",
        "product_name": "Coffee Mug",
        "quantity": 2,
        "unit_price": 120,
        "discount_type": "amount",
        "discount_value": 10,
        "discount_amount_per_unit": 10,
        "line_subtotal": 240,
        "line_discount_total": 20,
        "line_total": 220
      }
    ]
  }
}
```

## GET /api/v1/stores/:storeID/sales

ดึงประวัติการขายของร้าน เรียงล่าสุดก่อน พร้อมยอด:
- `subtotal_amount`
- `discount_amount`
- `total_amount`
- `paid_amount`
- `change_amount`
- `note`
- `created_at`

## GET /api/v1/stores/:storeID/sales/:saleID

ดึงรายละเอียดบิลขาย 1 รายการ พร้อม `items` ที่เก็บ snapshot ของ:
- `unit_price`
- `discount_type`
- `discount_value`
- `discount_amount_per_unit`
- `line_subtotal`
- `line_discount_total`
- `line_total`

## Notes

- หน้าขายควรใช้ `GET /api/v1/stores/:storeID/products` เพื่อดึงสินค้าที่เหลือก่อนเริ่มขาย
- หลังสร้าง sale สำเร็จ ควร refresh product list เพราะ `quantity` ถูกหักแล้ว
- response จาก backend ควรใช้เป็น source of truth สำหรับ order summary และ receipt

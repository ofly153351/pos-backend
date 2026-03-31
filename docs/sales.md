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
  "discount_bill": 20,
  "vat_included": false,
  "vat_percent": 7,
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
- `discount_bill` คือส่วนลดท้ายบิล (บาท) หลังหักส่วนลดรายรายการแล้ว
- `discount_bill` ห้ามน้อยกว่า `0` และห้ามมากกว่ายอดที่ต้องจ่ายก่อน VAT
- ถ้าส่ง `customer_id` ระบบจะคำนวณส่วนลดเครือข่ายเพิ่มตาม level ของลูกค้า (LV1/LV2/...) โดยอิงจาก `customer_level_discounts`
- ส่วนลดเครือข่ายคำนวณต่อหน่วยบนยอดหลังหักส่วนลด manual ของรายการนั้น
- ถ้าสต็อกไม่พอจะไม่สร้างบิล
- ถ้าสินค้า inactive จะไม่ขาย
- ถ้า `paid_amount < total_amount` จะ reject
- รองรับ VAT ด้วย `vat_included` และ `vat_percent` (default `true` และ `7`)
- บันทึก `subtotal_amount`, `discount_amount`, `vat_amount`, `total_amount`, `change_amount`
- บันทึก `bill_discount_amount` แยกจาก `discount_amount` (ซึ่งเป็นส่วนลดรวม)
- ถ้า `vat_included=true`: `total_amount` คือยอดรวมที่มี VAT อยู่แล้ว
- ถ้า `vat_included=false`: `total_amount` คือยอดหลังหักส่วนลด + VAT เพิ่ม

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
    "discount_amount": 55,
    "bill_discount_amount": 20,
    "vat_included": false,
    "vat_percent": 7,
    "vat_amount": 15.4,
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
- `vat_included`
- `vat_percent`
- `vat_amount`
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

## GET /api/v1/stores/:storeID/sales/:saleID/receipt

สร้างใบเสร็จแบบ HTML สำหรับพิมพ์ (thermal style) ตาม template:
- `store` (name/address/tax_id/vat_included)
- `order` (order_no/staff/datetime)
- `customer`
- `items`
- `summary` (`subtotal`, `discount_bill`, `vat_amount`, `grand_total`)
- `payment`
- `footer`

PromptPay QR:
- ถ้าร้านมี `promptpay_id` ระบบจะแสดงเลขพร้อมเพย์และ QR ที่ท้ายบิลอัตโนมัติ
- QR จะ encode ยอด `grand_total` ของบิลนั้นให้พร้อมสแกนจ่าย
- ถ้าร้านยังไม่มี `promptpay_id` จะไม่แสดง block QR

Response:
- `200 OK`
- `Content-Type: text/html; charset=utf-8`

ตัวอย่าง:

```bash
curl http://localhost:8080/api/v1/stores/{storeID}/sales/{saleID}/receipt \
  -H "Authorization: Bearer <token>"
```

compatibility path:

```bash
curl http://localhost:8080/api/stores/{storeID}/sales/{saleID}/receipt \
  -H "Authorization: Bearer <token>"
```

## GET /api/v1/stores/:storeID/sales/:saleID/receipt/preview

คืนหน้า HTML preview พร้อมปุ่ม `Print` ในตัว (เหมาะกับ browser flow)

Response:
- `200 OK`
- `Content-Type: text/html; charset=utf-8`

```bash
curl http://localhost:8080/api/v1/stores/{storeID}/sales/{saleID}/receipt/preview \
  -H "Authorization: Bearer <token>"
```

compatibility path:

```bash
curl http://localhost:8080/api/stores/{storeID}/sales/{saleID}/receipt/preview \
  -H "Authorization: Bearer <token>"
```

แนวทาง frontend ที่ไม่เจอ `about:blank`:
- เรียก endpoint ด้วย `fetch` และแนบ `Authorization` header
- เอา HTML response ไป `document.write()` ลงหน้าต่างใหม่ แล้วค่อยกด `Print`

ตัวอย่าง:

```javascript
const res = await fetch(`/api/v1/stores/${storeID}/sales/${saleID}/receipt/preview`, {
  headers: { Authorization: `Bearer ${token}` }
});
const html = await res.text();
const popup = window.open("", "_blank");
if (popup) {
  popup.document.open();
  popup.document.write(html);
  popup.document.close();
}
```

หมายเหตุ VAT:
- ใบเสร็จอ่านค่า `vat_included`, `vat_percent`, `vat_amount` จากข้อมูล sale ที่บันทึกจริง

## Notes

- หน้าขายควรใช้ `GET /api/v1/stores/:storeID/products` เพื่อดึงสินค้าที่เหลือก่อนเริ่มขาย
- หลังสร้าง sale สำเร็จ ควร refresh product list เพราะ `quantity` ถูกหักแล้ว
- response จาก backend ควรใช้เป็น source of truth สำหรับ order summary และ receipt
- ถ้าต้องการ PromptPay QR ในใบเสร็จ ให้ตั้งค่า `promptpay_id` ของร้านก่อนที่ `PUT /api/v1/stores/:storeID` (ดู `docs/store.md`)

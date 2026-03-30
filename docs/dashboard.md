# Dashboard API

API นี้ใช้สำหรับหน้า dashboard ของร้าน เพื่อดึงภาพรวมยอดขายและสถานะสินค้าในคำขอเดียว

Base: `Authorization: Bearer <token>`

สิทธิ์ที่ใช้งานได้:
- `owner`
- `manager`
- `cashier`
- `platform_admin`

## GET /api/v1/stores/:storeID/dashboard

ดึงข้อมูล dashboard ของร้านในช่วงเวลาที่กำหนด

รองรับ query params:
- `period` optional: `today` | `7d` | `30d` (default: `today`)
- `from` optional: RFC3339 หรือ `YYYY-MM-DD` (ต้องใช้คู่กับ `to`)
- `to` optional: RFC3339 หรือ `YYYY-MM-DD` (ถ้าเป็นรูปแบบวัน จะตีความเป็นปลายวัน)
- `top_limit` optional: จำนวนสินค้า top selling (default `5`, max `20`)
- `recent_limit` optional: จำนวนบิลล่าสุด (default `10`, max `50`)
- `low_stock_limit` optional: จำนวนสินค้าสต็อกต่ำที่ต้องการแสดง (default `10`, max `50`)
- `low_stock_threshold` optional: เกณฑ์สต็อกต่ำ (default `10`)

Behavior:
- ถ้าส่ง `from` หรือ `to` มาอย่างใดอย่างหนึ่ง ต้องส่งทั้งคู่
- ช่วงเวลา custom ต้องเป็น `from < to`
- ช่วงเวลา custom ยาวได้ไม่เกิน 366 วัน
- ถ้าไม่ส่ง custom range จะใช้ `period`

ตัวอย่าง request:

```bash
curl "http://localhost:8080/api/v1/stores/{storeID}/dashboard?period=7d&top_limit=5&recent_limit=10&low_stock_threshold=10" \
  -H "Authorization: Bearer <token>"
```

Success Response (`200 OK`)

```json
{
  "success": true,
  "message": "dashboard fetched",
  "data": {
    "range": {
      "period": "7d",
      "from": "2026-03-23T09:00:00Z",
      "to": "2026-03-30T09:00:00Z"
    },
    "summary": {
      "sales_count": 42,
      "revenue": 18560.5,
      "total_items": 133,
      "average_ticket": 441.92,
      "discount_amount": 860,
      "vat_amount": 1213.34
    },
    "payment_breakdown": [
      {
        "payment_method": "cash",
        "sales_count": 25,
        "amount": 9800
      },
      {
        "payment_method": "promptpay",
        "sales_count": 17,
        "amount": 8760.5
      }
    ],
    "top_products": [
      {
        "product_id": "prod_001",
        "product_name": "Americano",
        "quantity_sold": 31,
        "amount": 2480
      }
    ],
    "low_stock_products": [
      {
        "product_id": "prod_002",
        "name": "Arabica Bean 1kg",
        "sku": "BEAN-1KG",
        "unit_type": "piece",
        "quantity": 4
      }
    ],
    "recent_sales": [
      {
        "id": "sale_001",
        "sale_number": "S20260330-120001",
        "total_items": 3,
        "total_amount": 245,
        "payment_method": "cash",
        "cashier_name": "Alice",
        "customer_name": "Bob",
        "sold_at": "2026-03-30T08:58:00Z"
      }
    ]
  }
}
```

## Error Cases

- `400 Bad Request`
  - `invalid period, allowed values: today, 7d, 30d`
  - `invalid time range`
  - `invalid limit`
- `403 Forbidden`
  - `user cannot operate pos for this store`
- `500 Internal Server Error`
  - `internal server error`

## Notes

- Endpoint นี้ออกแบบให้หน้า dashboard โหลดได้ใน request เดียว
- `summary` และ `payment_breakdown` คิดจากข้อมูลในตาราง `sales`
- `top_products` คิดจาก `sale_items` join กับ `sales`
- `low_stock_products` อ่านจาก `products` ที่ `is_active = true` และ `quantity <= low_stock_threshold`
- `recent_sales` จะคืนเฉพาะบิลในช่วงเวลาที่เลือก

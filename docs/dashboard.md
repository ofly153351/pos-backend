# Dashboard API

This API is used for the store's dashboard screen to retrieve a sales overview and product status in a single request.

All endpoints require: `Authorization: Bearer <token>`

Roles that can access:
- `owner`
- `manager`
- `cashier`
- `platform_admin`

## GET /api/v1/stores/:storeID/dashboard

Returns dashboard data for the store over a specified time period.

Supported query params:
- `period` optional: `today` | `7d` | `30d` (default: `today`)
- `from` optional: RFC3339 or `YYYY-MM-DD` (must be used together with `to`)
- `to` optional: RFC3339 or `YYYY-MM-DD` (if date-only format, interpreted as end of day)
- `top_limit` optional: number of top-selling products to return (default `5`, max `20`)
- `recent_limit` optional: number of recent bills to return (default `10`, max `50`)
- `low_stock_limit` optional: number of low-stock products to show (default `10`, max `50`)
- `low_stock_threshold` optional: low-stock threshold (default `10`)

Behavior:
- If either `from` or `to` is provided, both must be provided
- A custom date range must satisfy `from < to`
- A custom range cannot exceed 366 days
- If no custom range is provided, `period` is used

Example request:

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

- This endpoint is designed to load the dashboard in a single request
- `summary` and `payment_breakdown` are calculated from data in the `sales` table
- `top_products` is calculated from `sale_items` joined with `sales`
- `low_stock_products` reads from `products` where `is_active = true` and `quantity <= low_stock_threshold`
- `recent_sales` returns only bills within the selected time range

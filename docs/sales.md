# Sales API

This API is used for the POS sales screen to create sale bills, view sales history, and open receipt details.

All endpoints require: `Authorization: Bearer <token>`

Roles that can access:
- `owner`
- `manager`
- `cashier`
- `platform_admin`

## POST /api/v1/stores/:storeID/sales

Creates a sale bill and deducts product quantities (`product.quantity`) in a single transaction. Supports per-item discounts on a per-unit basis.

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
- Uses the product's current price at the time of sale (`effective price`)
- `discount_type` supports `amount` and `percent`
- If `discount_type` and `discount_value` are not sent, no discount is applied
- `discount_value` for `amount` is the discount per unit
- `discount_value` for `percent` must be in the range `0-100`
- `discount_bill` is the bill-level discount (in currency) applied after per-item discounts
- `discount_bill` must not be less than `0` and must not exceed the amount due before VAT
- If `customer_id` is provided, the system automatically calculates a network discount based on the customer's level (LV1/LV2/...) using `customer_level_discounts`
- Network discount is calculated per unit on the amount after the manual per-item discount
- If stock is insufficient, the bill will not be created
- If a product is inactive, it will not be sold
- If `paid_amount < total_amount`, the request is rejected
- Supports VAT via `vat_included` and `vat_percent` (defaults: `true` and `7`)
- Records `subtotal_amount`, `discount_amount`, `vat_amount`, `total_amount`, `change_amount`
- Records `bill_discount_amount` separately from `discount_amount` (which is the total combined discount)
- If `vat_included=true`: `total_amount` is the grand total with VAT already included
- If `vat_included=false`: `total_amount` is the amount after discounts plus VAT added on top

Response shape:

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

Retrieves the store's sales history, ordered by most recent first. Each record includes:
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

Retrieves details of a single sale bill, including `items` which store a snapshot of:
- `unit_price`
- `discount_type`
- `discount_value`
- `discount_amount_per_unit`
- `line_subtotal`
- `line_discount_total`
- `line_total`

## GET /api/v1/stores/:storeID/sales/:saleID/receipt

Generates an HTML receipt for printing (thermal style) based on the template:
- `store` (name/address/tax_id/vat_included)
- `order` (order_no/staff/datetime)
- `customer`
- `items`
- `summary` (`subtotal`, `discount_bill`, `vat_amount`, `grand_total`)
- `payment`
- `footer`

PromptPay QR:
- If the store has a `promptpay_id`, the system automatically displays the PromptPay number and a QR code at the bottom of the receipt
- The QR encodes the `grand_total` of that bill, ready to scan and pay
- If the store has no `promptpay_id`, the QR block is not shown

Response:
- `200 OK`
- `Content-Type: text/html; charset=utf-8`

Example:

```bash
curl http://localhost:8080/api/v1/stores/{storeID}/sales/{saleID}/receipt \
  -H "Authorization: Bearer <token>"
```

Compatibility path:

```bash
curl http://localhost:8080/api/stores/{storeID}/sales/{saleID}/receipt \
  -H "Authorization: Bearer <token>"
```

## GET /api/v1/stores/:storeID/sales/:saleID/receipt/preview

Returns an HTML preview page with a built-in `Print` button (suitable for browser flow).

Response:
- `200 OK`
- `Content-Type: text/html; charset=utf-8`

```bash
curl http://localhost:8080/api/v1/stores/{storeID}/sales/{saleID}/receipt/preview \
  -H "Authorization: Bearer <token>"
```

Compatibility path:

```bash
curl http://localhost:8080/api/stores/{storeID}/sales/{saleID}/receipt/preview \
  -H "Authorization: Bearer <token>"
```

Frontend approach to avoid `about:blank` issues:
- Call the endpoint using `fetch` and attach the `Authorization` header
- Take the HTML response and use `document.write()` into a new window, then trigger `Print`

Example:

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

VAT note:
- The receipt reads `vat_included`, `vat_percent`, and `vat_amount` from the actual saved sale record.

## Notes

- The sales screen should use `GET /api/v1/stores/:storeID/products` to fetch available products before starting a sale
- After a sale is successfully created, refresh the product list because `quantity` has already been deducted
- The backend response should be used as the source of truth for the order summary and receipt
- To show PromptPay QR in the receipt, configure the store's `promptpay_id` first via `PUT /api/v1/stores/:storeID` (see `docs/store.md`)

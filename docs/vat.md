# VAT Calculation API

Endpoints in this module help the frontend display VAT-inclusive amounts while keeping the backend as the source of truth for tax math.

## Base

- `Authorization: Bearer <token>` (owner/manager/cashier/platform_admin)
- base path: `/api/v1` (compatibility `/api`)

## POST /api/v1/stores/:storeID/vat/calculate

Request body:

```json
{
  "items": [
    {
      "code": "A S000001",
      "name": "ส้มโอตก 10kg",
      "qty": 1,
      "price": 20.00,
      "discount_per_unit": 1.00
    }
  ],
  "discount_bill": 5.00,
  "vat_percent": 7,
  "vat_included": true
}
```

- `items`: array of line items (code/name optional metadata)  
- `qty`, `price`, `discount_per_unit` describe the per-line totals  
- `discount_bill`: additional bill-level discount  
- `vat_percent`: optional override, default `7`  
- `vat_included`: defaults to `true`; set `false` when supplying net amounts to add VAT

Response:

- Status `200 OK`
- `summary` payload follows the template from the receipt:
  - `subtotal`, `discount_item`, `discount_bill`, `after_discount`
  - `vat_percent`, `vat_amount`, `grand_total`, `vat_included`

Example success response:

```json
{
  "success": true,
  "message": "vat calculated",
  "data": {
    "summary": {
      "subtotal": 20,
      "discount_item": 1,
      "discount_bill": 5,
      "after_discount": 14,
      "vat_percent": 7,
      "vat_amount": 0.92,
      "grand_total": 14,
      "vat_included": true
    }
  }
}
```

### Notes

- When `vat_included` is `true`, the backend treats `after_discount` as the VAT-inclusive `grand_total` and derives `vat_amount = grand_total * vat_percent / (100 + vat_percent)`  
- When `vat_included` is `false`, the API adds VAT on top of the provided `after_discount`  
- Use this endpoint before showing the receipt so the frontend and receipt generator share the same numbers
- For sale creation, send the same VAT mode to `POST /api/v1/stores/:storeID/sales` via `vat_included` and `vat_percent`

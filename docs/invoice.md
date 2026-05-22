# Invoice API

API for issuing credit invoices to network customers and generating PDF reports.

All endpoints require: `Authorization: Bearer <token>`

Supports both `/api/v1/*` and `/api/*`.

## POST /api/v1/stores/:storeID/invoices

Creates a credit invoice (initial status: `unpaid`).

```json
{
  "customer_id": "7e9d5ccccf930438e6ba0d0f",
  "due_at": "2026-04-30T00:00:00Z",
  "note": "30-day credit",
  "vat_included": false,
  "vat_percent": 7,
  "items": [
    {
      "product_id": "prod_001",
      "quantity": 3,
      "discount_type": "percent",
      "discount_value": 10
    }
  ]
}
```

Behavior:
- Must be a network customer (`customer_id`)
- Network discount is automatically calculated based on `customer_level_discounts`
- Stock is deducted in the same transaction as invoice creation
- Supports VAT via `vat_included` and `vat_percent` (defaults: `true` and `7`)
- Records `vat_amount` on the invoice; this value is used for both payment and PDF generation
- If `vat_included=true`: `total_amount` is the grand total with VAT already included
- If `vat_included=false`: `total_amount` is the amount after discounts plus VAT added on top

## GET /api/v1/stores/:storeID/invoices

Returns the list of credit invoices for the store.

## GET /api/v1/stores/:storeID/invoices/:invoiceID

Returns details of a single invoice, including `items` and `payments`.

## POST /api/v1/stores/:storeID/invoices/:invoiceID/payments

Records a partial or full payment on the invoice. Optionally attach proof of payment.

Supports two formats:
- `application/json` (legacy)
- `multipart/form-data` (recommended when attaching a proof file)

```json
{
  "paid_amount": 500,
  "payment_method": "bank_transfer",
  "note": "first installment"
}
```

Example with proof attachment (image or PDF):

```bash
curl -X POST http://localhost:8080/api/v1/stores/{storeID}/invoices/{invoiceID}/payments \
  -H "Authorization: Bearer <token>" \
  -F "paid_amount=500" \
  -F "payment_method=bank_transfer" \
  -F "note=first installment" \
  -F "proof=@/path/to/slip.pdf"
```

`proof` file requirements:
- Accepted types: `image/jpeg`, `image/png`, `image/webp`, `application/pdf`
- Maximum file size: `10MB`
- Stored in MinIO; URL is returned in the payment record as `proof_url`

Example fields added to `payments[]`:

```json
{
  "id": "pay_001",
  "paid_amount": 500,
  "payment_method": "bank_transfer",
  "proof_url": "http://127.0.0.1:9000/pos-assets/invoice-payments/abc123.pdf",
  "proof_mime_type": "application/pdf",
  "proof_file_name": "slip.pdf"
}
```

Status transitions:
- `unpaid` -> `partially_paid` -> `paid`

## GET /api/v1/stores/:storeID/invoices/:invoiceID/payments/:paymentID/proof

Opens the proof of payment for a specific payment record (redirects to the file in MinIO).

```bash
curl -L http://localhost:8080/api/v1/stores/{storeID}/invoices/{invoiceID}/payments/{paymentID}/proof \
  -H "Authorization: Bearer <token>"
```

Notes:
- `paymentID` comes from `payments[].id` in `GET /invoices/:invoiceID`
- If the payment has no proof attached, returns `404 payment proof not found`

## POST /api/v1/stores/:storeID/invoices/:invoiceID/unpay

Use when a payment was recorded in error and needs to be reversed back to `unpaid`.

```json
{
  "reason": "recorded payment on wrong invoice"
}
```

Behavior:
- `reason` is required (stored for audit purposes)
- All previously recorded payments are marked as `is_voided=true` (not deleted)
- The invoice is reset to:
  - `status = unpaid`
  - `paid_amount = 0`
  - `remaining_amount = total_amount`
  - `payment_method = null`

Notes:
- In `payments[]`, the audit fields are visible: `is_voided`, `voided_at`, `voided_by_user_id`, `void_reason`

## GET /api/v1/stores/:storeID/invoices/:invoiceID/pdf

Generates a PDF document of the invoice and returns it as `application/pdf`.

Can be opened directly in a browser or downloaded immediately.

VAT notes in PDF:
- Displays `VAT %`, `VAT amount`, and `VAT mode (Included/Excluded)` from the values actually saved on the invoice.

## Common Errors

- `400` — invalid input, payment exceeds remaining balance, invoice already closed
- `403` — no permission for this store
- `404` — invoice / customer / product not found
- `500` — internal server error

# Parked Bills API

API for saving the current cart state to be restored later ("hold bill" / "park bill").

All endpoints require: `Authorization: Bearer <token>`

## Overview

A parked bill captures a snapshot of an in-progress sale — items, customer, payment method, discounts, and notes — so that the cashier can set it aside and return to it later. This is useful when a customer needs time to decide, another customer needs to be served immediately, or the cart must be held between shifts.

Parked bills do **not** deduct stock. Stock is only deducted when the sale is actually completed via `POST /stores/:storeID/sales`.

---

## POST /stores/:storeID/parked-bills

Parks the current cart state.

Request body:

```json
{
  "customer_id": "cust_001",
  "payment_method": "cash",
  "discount_bill": 20,
  "note": "customer will return in 10 minutes",
  "items": [
    {
      "product_id": "prod_001",
      "quantity": 2,
      "discount_type": "amount",
      "discount_value": 10
    },
    {
      "product_id": "prod_002",
      "quantity": 1,
      "discount_type": "percent",
      "discount_value": 5
    }
  ]
}
```

Fields:
- `customer_id` optional — attached customer
- `payment_method` optional — intended payment method
- `discount_bill` optional — bill-level discount to restore
- `note` optional — free-text note
- `items` required — array of cart line items

Success Response (`201 Created`):

```json
{
  "success": true,
  "message": "parked bill created",
  "data": {
    "id": "park_001",
    "store_id": "store_001",
    "customer_id": "cust_001",
    "payment_method": "cash",
    "discount_bill": 20,
    "note": "customer will return in 10 minutes",
    "items": [
      {
        "product_id": "prod_001",
        "quantity": 2,
        "discount_type": "amount",
        "discount_value": 10
      }
    ],
    "created_at": "2026-05-22T10:00:00Z"
  }
}
```

---

## GET /stores/:storeID/parked-bills

Returns all currently parked bills for the store, ordered by most recent first.

---

## GET /stores/:storeID/parked-bills/:parkedBillID

Returns details of a single parked bill, including all stored items and cart metadata.

---

## DELETE /stores/:storeID/parked-bills/:parkedBillID

Deletes a parked bill. Use this after successfully restoring and completing the sale, or when the bill is no longer needed.

---

## Typical Workflow

1. Cashier builds a cart on the POS screen
2. Cashier calls `POST /stores/:storeID/parked-bills` to park the cart
3. Cashier handles other customers or takes a break
4. Cashier calls `GET /stores/:storeID/parked-bills` to see all parked bills
5. Cashier selects and restores the parked bill from `GET /stores/:storeID/parked-bills/:parkedBillID`
6. Cashier re-creates the cart in the UI from the parked bill data and completes the sale via `POST /stores/:storeID/sales`
7. Cashier calls `DELETE /stores/:storeID/parked-bills/:parkedBillID` to clean up the parked bill

---

## Notes

- Parked bills do not affect stock — they are purely a cart snapshot
- There is no automatic expiry; parked bills persist until explicitly deleted
- Products may change price between parking and restoring — validate prices on restore if needed
- The response snapshot stores the cart as-entered; `effective_price` and discount calculations are recomputed by the backend at actual sale time

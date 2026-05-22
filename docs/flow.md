# System Flow

This document describes the main flow of the POS system from the perspective of real store usage.

## 1. Registration and Login

Start by creating a user account and then logging in to receive an `access_token`.

Steps:
1. `POST /api/v1/auth/register`
2. `POST /api/v1/auth/login`
3. Store the `access_token` and send it via `Authorization: Bearer <token>`

Notes:
- Newly registered users use this token to create a store and manage their own store data

## 2. Create a Store

After logging in, the owner must create a store first because all other data is tied to a `store_id`.

Steps:
1. `POST /api/v1/stores`
2. The system creates:
   - A row in `stores`
   - An owner row in `store_members`
   - The first row in `store_subscriptions`

Result:
- The store's `store_id` is returned
- If a logo is provided, the file is stored and its path is saved in `logo_url`

## 3. View and Select a Subscription Plan

Subscriptions are at the store level, not the user level.

Steps:
1. `GET /api/v1/subscriptions/plans`
2. `GET /api/v1/stores/:storeID/subscription`
3. To change the plan, use `PUT /api/v1/stores/:storeID/subscription`

Notes:
- Only `owner`, `manager`, or `platform_admin` can change the plan

## 4. Create Product Types for the Store

Each store has its own product categories, e.g. `Coffee`, `Bakery`, `Supplies`.

Steps:
1. `POST /api/v1/stores/:storeID/product-types`
2. `GET /api/v1/stores/:storeID/product-types`
3. Update with `PATCH`
4. Remove with `DELETE`

Result:
- A `product_type_id` is returned for linking to products

## 5. Create Products

Each product belongs to a store and may reference a `product_type_id` from that store.

Steps:
1. `POST /api/v1/stores/:storeID/products`
2. Send data as `multipart/form-data`
3. Optionally attach an image via the `image` field

Key fields:
- `product_type_id` — product category
- `unit_type` — selling unit, e.g. `piece` or `pair`
- `quantity` — current stock on hand
- `base_price` — regular price
- `special_price` — promotional price

## 6. Manage Products

After creating products, further management is available.

Steps:
1. `GET /api/v1/stores/:storeID/products`
2. `GET /api/v1/stores/:storeID/products/:productID`
3. `PATCH /api/v1/stores/:storeID/products/:productID`
4. `DELETE /api/v1/stores/:storeID/products/:productID`

Notes:
- `PATCH` supports changing the product image
- The system calculates `effective_price` from the special price window automatically

## 7. Access Permissions

The system checks permissions from `store_members`:

- `owner`: can manage store, subscription, product types, and products
- `manager`: can manage store data at the operational level
- `cashier`: should not manage store configuration
- `platform_admin`: can access all stores

If this error appears:

```json
{
  "success": false,
  "message": "user cannot manage this store"
}
```

It means the user from the token has no permission for that `store_id`, or is not in `store_members`.

## 8. POS Sales Screen

Once products are in the system, `cashier`, `manager`, and `owner` can open the sales screen to create bills.

Steps:
1. `GET /api/v1/stores/:storeID/products`
2. Select the products and quantities to sell
3. `POST /api/v1/stores/:storeID/sales`
4. To view a past receipt, use `GET /api/v1/stores/:storeID/sales/:saleID`

Key details:
- The system immediately deducts `product.quantity` upon a successful sale
- A snapshot of product name, price, and quantity sold is stored in `sale_items`
- The sales screen can load history via `GET /api/v1/stores/:storeID/sales`

## Recommended Flow for a New Store

1. register
2. login
3. create store
4. get current subscription
5. create product types
6. create products
7. update subscription when the store needs to change plans

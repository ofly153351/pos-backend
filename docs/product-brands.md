# Product Brand Integration

This document summarizes the endpoints related to product brands (`brand`) as currently implemented.

All endpoints require: `Authorization: Bearer <token>`

## Data Model

- Uses a `product_brands` table separate from `products`
- `products.brand_id` is a FK to `product_brands.id`
- Product responses return:
  - `brand_id`
  - `brand_name` (from a join on `product_view`)

Related migration:
- `init-db/026_product_brands_refactor.sql`

## Implemented Endpoints (Brand-related)

### 1) POST /api/v1/stores/:storeID/products

Creates a new product and optionally attaches a `brand_id`.

Brand-related field:
- `brand_id` optional
- If provided, it must belong to the same store; otherwise returns `400` (`brand does not belong to this store`)

Example:

```bash
curl -X POST http://localhost:8080/api/v1/stores/{storeID}/products \
  -H "Authorization: Bearer <token>" \
  -F "name=Running Shoes" \
  -F "brand_id=br_xxxxx" \
  -F "unit_id=unit_xxx" \
  -F "base_price=2590"
```

### 2) GET /api/v1/stores/:storeID/products

Returns the product list. Each item includes `brand_id` and `brand_name`.

### 3) GET /api/v1/stores/:storeID/products/:productID

Returns a single product with `brand_id` and `brand_name`.

### 4) PATCH /api/v1/stores/:storeID/products/:productID

Updates a product and can change the `brand_id`.

Brand-related field:
- `brand_id` optional

Example:

```bash
curl -X PATCH http://localhost:8080/api/v1/stores/{storeID}/products/{productID} \
  -H "Authorization: Bearer <token>" \
  -F "brand_id=br_newbrand"
```

### 5) POST /api/v1/stores/:storeID/products/generate-missing-barcodes

This endpoint does not directly modify brands, but lives in the same product module and works normally with products that have or do not have a `brand_id`.

## Not Yet Implemented

There are currently **no** routes for managing the `product_brands` table directly, such as:

- `POST /api/v1/stores/:storeID/product-brands`
- `GET /api/v1/stores/:storeID/product-brands`
- `PATCH /api/v1/stores/:storeID/product-brands/:brandID`
- `DELETE /api/v1/stores/:storeID/product-brands/:brandID`

Calling `GET /api/stores/:storeID/product-brands` will therefore return `404`.

## Notes

- If a brand selection screen from master data is needed, the product-brand CRUD/list API must be implemented first
- The backend already supports referencing a `brand_id`, but populating the `product_brands` data must be done separately (e.g. via seed SQL or manual insert)

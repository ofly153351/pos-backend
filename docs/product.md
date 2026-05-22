# Product API

All endpoints require: `Authorization: Bearer <token>`

This API manages products within a store. It separates two concepts:

- `product_type` per store: product categories, e.g. `Coffee`, `Bakery`, `Accessories`
- `product_unit` per store: selling units, e.g. `Piece`, `Pair`, `Box`

Each product references its store's `product_type_id`, supports a brand via `brand_id`, a product image, a special price, and a stock quantity (`quantity`).

## Product Unit

- Products reference a unit via `unit_id` (FK to `product_units.id`)
- Create a unit with `POST /api/v1/stores/:storeID/product-units` before linking it to a product

## Product Brand

- Products reference a brand via `brand_id` (FK to `product_brands.id`)
- The `product_brands` table is store-scoped (`store_id`)
- Product responses include both `brand_id` and `brand_name`

## Product Type APIs

### POST /api/v1/stores/:storeID/product-types

```json
{
  "name": "Coffee",
  "description": "Hot and iced coffee menu",
  "is_active": true
}
```

### GET /api/v1/stores/:storeID/product-types

### PATCH /api/v1/stores/:storeID/product-types/:productTypeID

### DELETE /api/v1/stores/:storeID/product-types/:productTypeID

## POST /api/v1/stores/:storeID/products

Creates a new product using `multipart/form-data`.

Fields:
- `name` required
- `brand_id` optional (must belong to the same store)
- `sku` optional (if not provided, the system auto-generates an EAN-13 barcode)
- `product_type_id` optional, must be a type belonging to the same store
- `unit_id` required, must be a unit belonging to the same store
- `quantity` optional, default `0`, must be a non-negative integer
- `base_price` required
- `special_price` optional
- `special_price_start_at` optional, RFC3339
- `special_price_end_at` optional, RFC3339
- `is_active` optional, default `true`
- `image` optional, file

Example (with product image upload):

```bash
curl -X POST http://localhost:8080/api/v1/stores/{storeID}/products \
  -H "Authorization: Bearer <token>" \
  -F "name=Coffee Mug" \
  -F "brand_id=brand_xxx" \
  -F "sku=MUG-001" \
  -F "product_type_id=type_xxx" \
  -F "unit_id=unit_xxx" \
  -F "quantity=24" \
  -F "base_price=120" \
  -F "special_price=99" \
  -F "image=@/path/to/product.jpg"
```

Success Response (`201 Created`)

```json
{
  "success": true,
  "message": "product created",
  "data": {
    "id": "3f7cbf2b6f9415d4d31811af",
    "store_id": "65b493e98058f410a890859f",
    "name": "Coffee Mug",
    "brand_id": "brand_xxx",
    "brand_name": "Acme",
    "sku": "MUG-001",
    "product_unit_id": "unit_xxx",
    "product_unit_name": "piece",
    "image_url": "http://127.0.0.1:9000/pos-assets/products/3f7cbf2b6f9415d4d31811af.jpg",
    "quantity": 24,
    "base_price": 120,
    "special_price": 99,
    "effective_price": 99,
    "is_active": true,
    "created_at": "2026-03-23T09:00:00Z",
    "updated_at": "2026-03-23T09:00:00Z"
  }
}
```

## GET /api/v1/stores/:storeID/products

Returns a paginated product list to reduce response size.

Query params:
- `page` optional, default `1`
- `limit` optional, default `50`, max `200`

Example:

```bash
curl "http://localhost:8080/api/v1/stores/{storeID}/products?page=1&limit=50" \
  -H "Authorization: Bearer <token>"
```

Success Response (`200 OK`)

```json
{
  "success": true,
  "message": "products fetched",
  "data": {
    "items": [
      {
        "id": "prod_001",
        "name": "Coffee Mug",
        "quantity": 24,
        "base_price": 120,
        "effective_price": 120
      }
    ],
    "page": 1,
    "limit": 50,
    "total": 1000000,
    "total_pages": 20000,
    "has_next": true,
    "has_prev": false
  }
}
```

## GET /api/v1/stores/:storeID/products/:productID

Returns a single product's details.

## PATCH /api/v1/stores/:storeID/products/:productID

Updates a product using `multipart/form-data`.

Supported fields:
- `name`
- `brand_id`
- `sku`
- `product_type_id`
- `unit_id`
- `quantity`
- `base_price`
- `special_price`
- `clear_special_price=true`
- `special_price_start_at`
- `special_price_end_at`
- `clear_special_window=true`
- `is_active`
- `image`

Example (changing the product image):

```bash
curl -X PATCH http://localhost:8080/api/v1/stores/{storeID}/products/{productID} \
  -H "Authorization: Bearer <token>" \
  -F "name=Coffee Mug 2026" \
  -F "brand_id=brand_yyy" \
  -F "base_price=129" \
  -F "image=@/path/to/new-product.jpg"
```

Success Response (`200 OK`)

```json
{
  "success": true,
  "message": "product updated",
  "data": {
    "id": "3f7cbf2b6f9415d4d31811af",
    "store_id": "65b493e98058f410a890859f",
    "name": "Coffee Mug 2026",
    "brand_id": "brand_yyy",
    "brand_name": "Acme Pro",
    "product_unit_id": "unit_xxx",
    "product_unit_name": "piece",
    "image_url": "http://127.0.0.1:9000/pos-assets/products/88b9a45a623ad97f0a7f2121.jpg",
    "quantity": 24,
    "base_price": 129,
    "effective_price": 129,
    "is_active": true,
    "created_at": "2026-03-23T09:00:00Z",
    "updated_at": "2026-03-23T10:15:00Z"
  }
}
```

## DELETE /api/v1/stores/:storeID/products/:productID

Removes a product from the system.

## POST /api/v1/stores/:storeID/products/generate-missing-barcodes

Generates barcodes (stored in `sku`) for all products in the store that currently have an empty `sku`.

Success Response (`200 OK`)

```json
{
  "success": true,
  "message": "missing product barcodes generated",
  "data": {
    "updated_count": 12
  }
}
```

## Notes

- The returned price includes `effective_price` calculated from the special price window
- Each product response includes `quantity` as the current stock on hand
- Each product response includes `brand_id` and `brand_name` (empty if not set)
- The list endpoint always uses pagination to reduce payload (`page/limit`)
- Product images are uploaded to MinIO at path `products/<generated-filename>` and the URL is returned in `image_url`
- If `image` is not sent during `PATCH`, the existing image is preserved
- If `image_url` is empty it is not included in the JSON response (due to `omitempty`)
- If `sku` is not provided when creating a product, an EAN-13 barcode is auto-generated
- To backfill barcodes for existing products with empty SKUs, use the `generate-missing-barcodes` endpoint
- It is recommended to create store `product_type` entries before creating products
- Units must be created in `product-units` first, then the `unit_id` is sent when creating or updating a product

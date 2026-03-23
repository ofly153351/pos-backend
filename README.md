# POS Backend

Go + Fiber backend for a POS system with PostgreSQL, store subscriptions, store logo upload, and product special pricing.

## Run

```bash
docker compose up -d postgres
GOCACHE=$(pwd)/.cache/go-build GOMODCACHE=$(pwd)/.cache/go-mod go run ./cmd/api.go
```

## Core Endpoints

```bash
POST /api/v1/auth/register
POST /api/v1/auth/login
POST /api/v1/stores
POST /api/v1/stores/:storeID/product-types
GET  /api/v1/stores/:storeID/product-types
PATCH /api/v1/stores/:storeID/product-types/:productTypeID
DELETE /api/v1/stores/:storeID/product-types/:productTypeID
POST /api/v1/stores/:storeID/product-units
GET  /api/v1/stores/:storeID/product-units
PATCH /api/v1/stores/:storeID/product-units/:unitID
DELETE /api/v1/stores/:storeID/product-units/:unitID
POST /api/v1/stores/:storeID/products
GET  /api/v1/stores/:storeID/products
GET  /api/v1/stores/:storeID/products/:productID
PATCH /api/v1/stores/:storeID/products/:productID
DELETE /api/v1/stores/:storeID/products/:productID
POST /api/v1/stores/:storeID/sales
GET  /api/v1/stores/:storeID/sales
GET  /api/v1/stores/:storeID/sales/:saleID
GET  /api/v1/subscriptions/plans
GET  /api/v1/stores/:storeID/subscription
PUT  /api/v1/stores/:storeID/subscription
GET  /api/v1/admin/subscriptions
GET  /api/v1/admin/stores/:storeID/subscription
PUT  /api/v1/admin/stores/:storeID/subscription
PATCH /api/v1/admin/stores/:storeID/subscription/status
```

## Notes

- PostgreSQL schema is defined in `init-db/001_schema.sql`
- Incremental schema changes are in `init-db/002_product_type.sql`
- System flow guide: `docs/flow.md`
- Admin guide: `docs/admin.md`
- Product API guide: `docs/product.md`
- Sales API guide: `docs/sales.md`
- Subscription API guide: `docs/subscription.md`
- Product Unit API guide: `docs/product-units.md`
- Product structure guide (types & units): `docs/product-structure.md`
- Admin subscription API guide: `docs/admin-subscription.md`
- Uploaded store logos are served from `/uploads/logos/*`
- Uploaded product images are served from `/uploads/products/*`
- Store creation also creates the first active subscription row

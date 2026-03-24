# POS Backend

Go + Fiber backend for a POS system with PostgreSQL, store subscriptions, store logo upload, and product special pricing.

## Run

```bash
docker compose up -d postgres
GOCACHE=$(pwd)/.cache/go-build GOMODCACHE=$(pwd)/.cache/go-mod go run ./cmd/api.go
```

## Core Endpoints

```bash
# Auth
POST /api/v1/auth/register
POST /api/v1/auth/login
POST /api/v1/auth/logout

# Store
POST /api/v1/stores
GET  /api/v1/stores/:storeID

# Product Type
POST /api/v1/stores/:storeID/product-types
GET  /api/v1/stores/:storeID/product-types
PATCH /api/v1/stores/:storeID/product-types/:productTypeID
DELETE /api/v1/stores/:storeID/product-types/:productTypeID

# Product Unit
POST /api/v1/stores/:storeID/product-units
GET  /api/v1/stores/:storeID/product-units
PATCH /api/v1/stores/:storeID/product-units/:unitID
DELETE /api/v1/stores/:storeID/product-units/:unitID

# Product
POST /api/v1/stores/:storeID/products
GET  /api/v1/stores/:storeID/products
GET  /api/v1/stores/:storeID/products/:productID
PATCH /api/v1/stores/:storeID/products/:productID
DELETE /api/v1/stores/:storeID/products/:productID

# Customer Network
POST /api/v1/stores/:storeID/customers
GET  /api/v1/stores/:storeID/customers
GET  /api/v1/stores/:storeID/customers/:customerID
PATCH /api/v1/stores/:storeID/customers/:customerID
DELETE /api/v1/stores/:storeID/customers/:customerID
GET  /api/v1/stores/:storeID/customer-level-discounts
PUT  /api/v1/stores/:storeID/customer-level-discounts/:level
DELETE /api/v1/stores/:storeID/customer-level-discounts/:level

# Sales
POST /api/v1/stores/:storeID/sales
GET  /api/v1/stores/:storeID/sales
GET  /api/v1/stores/:storeID/sales/:saleID
GET  /api/v1/stores/:storeID/sales/:saleID/receipt
GET  /api/v1/stores/:storeID/sales/:saleID/receipt/preview
# VAT
POST /api/v1/stores/:storeID/vat/calculate

# Outstanding Invoices
POST /api/v1/stores/:storeID/invoices
GET  /api/v1/stores/:storeID/invoices
GET  /api/v1/stores/:storeID/invoices/:invoiceID
POST /api/v1/stores/:storeID/invoices/:invoiceID/payments
GET  /api/v1/stores/:storeID/invoices/:invoiceID/pdf

# Subscription
GET  /api/v1/subscriptions/plans
GET  /api/v1/stores/:storeID/subscription
PUT  /api/v1/stores/:storeID/subscription

# Admin
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
- Store API guide: `docs/store.md`
- Product API guide: `docs/product.md`
- Sales API guide: `docs/sales.md`
- Invoice API guide: `docs/invoice.md`
- Subscription API guide: `docs/subscription.md`
- Customer network status guide: `docs/customer-network.md`
- Product Unit API guide: `docs/product-units.md`
- Product structure guide (types & units): `docs/product-structure.md`
- Admin subscription API guide: `docs/admin-subscription.md`
- Uploaded store logos are stored in MinIO
- Uploaded product images are stored in MinIO
- Store creation also creates the first active subscription row
- API routes support both `/api/v1/*` and compatibility path `/api/*`

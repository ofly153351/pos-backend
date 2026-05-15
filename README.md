# POS Backend

Go + Fiber backend for a POS system with PostgreSQL, store subscriptions, store logo upload, and product special pricing.

## Run

```bash
docker compose up -d postgres
GOCACHE=$(pwd)/.cache/go-build GOMODCACHE=$(pwd)/.cache/go-mod go run ./cmd/api.go
```

## Project Structure

```text
.
├── cmd/
│   └── api.go                  # API entrypoint
├── docs/                       # API and feature guides
├── init-db/                    # Forward-only SQL schema/migrations
├── internal/
│   ├── app/                    # HTTP bootstrap and route wiring
│   ├── config/                 # Environment-backed configuration
│   ├── database/               # PostgreSQL connection and migration runner
│   ├── middleware/             # JWT auth and request guards
│   ├── modules/                # Bounded business features
│   │   ├── auth/
│   │   ├── customer/
│   │   ├── dashboard/
│   │   ├── invoice/
│   │   ├── product/
│   │   ├── sale/
│   │   ├── store/
│   │   ├── subscription/
│   │   └── vat/
│   └── platform/httpx/         # Shared HTTP response helpers
├── docker-compose.yml
└── README.md
```

## โครงสร้างระบบ (สำหรับเขียน Agent.md)

โปรเจกต์นี้ใช้แนวทางแยกชั้น (layered architecture) ชัดเจนเพื่อให้แก้ไขได้ปลอดภัย:

- `cmd/`: จุดเริ่มรันแอป (entrypoint)
- `internal/app/`: ประกอบ server, dependencies, และผูก route ของแต่ละโมดูล
- `internal/middleware/`: middleware กลาง เช่น JWT auth
- `internal/modules/<feature>/handler.go`: แปลง HTTP request/response
- `internal/modules/<feature>/service.go`: กฎธุรกิจ, สิทธิ์การเข้าถึง, transaction
- `internal/modules/<feature>/repository.go`: SQL และการคุย PostgreSQL
- `internal/database/`: connection และ migration runner
- `internal/platform/httpx/`: helper สำหรับ response/request ที่ใช้ซ้ำ
- `init-db/`: schema + migration SQL แบบ forward-only

Request flow มาตรฐาน:

1. Route ถูกประกาศใน `internal/app/*.go`
2. Handler รับ request แล้วเรียก Service
3. Service ตรวจสิทธิ์/validate/ทำธุรกรรม
4. Repository ยิง SQL กับ PostgreSQL
5. ส่งผลลัพธ์กลับผ่าน `httpx`

โมดูลธุรกิจหลักใน `internal/modules/`:

- `auth`: สมัคร/ล็อกอิน/ออกจากระบบ, token และ claims
- `store`: จัดการร้านและสมาชิกในร้าน
- `subscription`: แผนใช้งานและสถานะ subscription ของร้าน
- `producttype`: หมวดสินค้าแบบผูกกับ store
- `productunit`: หน่วยสินค้า
- `product`: สินค้า, ราคา, รูปสินค้า
- `sale`: ขายสินค้าและใบเสร็จ
- `invoice`: ใบวางบิล, ชำระเงิน, PDF/proof
- `customer`: เครือข่ายลูกค้าและส่วนลดตามระดับ
- `dashboard`: ตัวเลขสรุปภาพรวมร้าน
- `vat`: คำนวณภาษีมูลค่าเพิ่ม

## Core Endpoints

```bash
# Auth
POST /api/v1/auth/register
POST /api/v1/auth/login
POST /api/v1/auth/logout

# Store
GET  /api/v1/me/stores
POST /api/v1/stores
GET  /api/v1/stores
GET  /api/v1/stores/:storeID
PUT  /api/v1/stores/:storeID

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
POST /api/v1/stores/:storeID/products/generate-missing-barcodes
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

# Dashboard
GET  /api/v1/stores/:storeID/dashboard

# VAT
POST /api/v1/stores/:storeID/vat/calculate

# Outstanding Invoices
POST /api/v1/stores/:storeID/invoices
GET  /api/v1/stores/:storeID/invoices
GET  /api/v1/stores/:storeID/invoices/:invoiceID
POST /api/v1/stores/:storeID/invoices/:invoiceID/payments
GET  /api/v1/stores/:storeID/invoices/:invoiceID/payments/:paymentID/proof
POST /api/v1/stores/:storeID/invoices/:invoiceID/unpay
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
- Incremental schema changes are in `init-db/` (latest: `026_product_brands_refactor.sql`)
- System flow guide: `docs/flow.md`
- Admin guide: `docs/admin.md`
- Store API guide: `docs/store.md`
- API development guide: `docs/api-development.md`
- Product API guide: `docs/product.md`
- Product brand integration guide: `docs/product-brands.md`
- Sales API guide: `docs/sales.md`
- Dashboard API guide: `docs/dashboard.md`
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

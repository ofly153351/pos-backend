# Repository Guidelines

## Architecture
The backend is a layered Go + Fiber service:

- `cmd/` entrypoints
- `internal/app/` HTTP bootstrap and route wiring
- `internal/config/` environment-backed config
- `internal/database/` PostgreSQL connection and SQL migrations
- `internal/middleware/` JWT auth and request guards
- `internal/modules/*` bounded features such as `auth`, `store`, and `product`
- `internal/platform/httpx/` shared JSON response helpers
- `init-db/` idempotent SQL schema for PostgreSQL

## Request Lifecycle
Use this flow when reasoning about where to add or change logic:

1. Route registration lives in `internal/app/*.go` and wires handlers by feature.
2. Handlers in `internal/modules/<feature>/handler.go` parse params/body and call services.
3. Services in `internal/modules/<feature>/service.go` enforce business rules, validation, transactions, and authorization checks.
4. Repositories in `internal/modules/<feature>/repository.go` run SQL against PostgreSQL.
5. Shared response/request helpers are in `internal/platform/httpx/`.
6. JWT guard and identity context are enforced in `internal/middleware/auth.go`.

## Module Map
Current bounded modules under `internal/modules/`:

- `auth`: login/register/logout, token security, identity claims
- `store`: store profile, membership, store-level access
- `subscription`: plan catalog and per-store subscription state
- `producttype`: store-scoped product categories
- `productunit`: unit definitions used by products
- `product`: catalog data, price rules, optional image upload
- `sale`: POS sale creation, totals, receipt output
- `invoice`: outstanding invoices, payment proof and PDF generation
- `customer`: customer network and level-based discount relationships
- `dashboard`: store-level summary metrics
- `vat`: VAT calculation endpoint for checkout flows

## POS Data Model
This project uses PostgreSQL as the source of truth.

- `users`: account identity, password hash, global role (`platform_admin`, `owner`, `manager`, `cashier`)
- `stores`: merchant storefront metadata, contact data, and `logo_url`
- `store_members`: store-level role mapping so one user can belong to multiple stores
- `product_types`: store-owned product categories such as coffee, bakery, or retail items
- `products`: catalog items with `product_type_id`, `unit_type`, `base_price`, and optional special pricing window
- `subscription_plans`: billable plans such as `starter`, `growth`, `pro`
- `store_subscriptions`: active plan assignment per store with billing period boundaries

Subscription is modeled at the store level because POS access is tenant-oriented. A store owner creates a store, selects a plan, and that subscription governs feature access for that tenant.

## Backend Rules
- Auth issues bearer tokens with user identity and role claims.
- Store creation must create the store, owner membership, and first subscription in one transaction.
- Store logos and product images are uploaded as multipart files to MinIO; only the object URL/path is persisted in PostgreSQL.
- Product pricing must support `base_price` and optional `special_price`, `special_price_start_at`, and `special_price_end_at`.
- Product type ownership is store-scoped. Do not attach product categories to `user_id`; use `store_id`.
- New write flows should be transaction-safe and should validate role access before mutating store data.

## Local Development
- Run the API with `go run ./cmd/api.go`
- Run tests with `GOCACHE=$(pwd)/.cache/go-build GOMODCACHE=$(pwd)/.cache/go-mod go test ./...`
- PostgreSQL bootstrap SQL lives in `init-db/001_schema.sql`

## Contributor Notes
- Keep business rules in services, not handlers.
- Keep SQL in repositories and prefer explicit queries over hidden ORM behavior.
- New schema changes should be added as forward-only SQL files in `init-db/`.

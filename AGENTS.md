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

## Known Issues

### DATABASE ENCODING — Thai text garbled in expense_categories (and any table seeded with Thai names)

**Status:** Active bug. Do NOT attempt to fix by wiping the volume without a full backup/export first.

**Root cause:** The `postgres_data` Docker volume was initialised without `POSTGRES_INITDB_ARGS: "--encoding=UTF8"`.
PostgreSQL used the container's default locale encoding (not UTF-8), so Thai characters written to the database
are stored as the wrong byte sequence and returned as `à¸à¸...` mojibake.

**Scope:**
- Affects: any row with Thai text written before the volume is recreated (e.g. `expense_categories.name` seeds).
- Does NOT affect: financial amounts, IDs, dates, or ASCII strings — these are all correct.
- The `/finance/pnl` expense-breakdown widget shows garbled category names for this reason.

**Workaround applied:**
- `config.go` DSN already has `&client_encoding=UTF8` — new rows written in Thai will be correct if the
  underlying column encoding supports it (limited mitigation only).
- `docker-compose.yml` already has `POSTGRES_INITDB_ARGS: "--encoding=UTF8 ..."` — applies to FRESH volumes only.

**Correct fix (when ready):**
1. Export all data: `pg_dump -U postgres pos_db > pos_db_backup.sql`
2. Recreate the volume: `docker compose down -v && docker compose up -d postgres`
3. Restore: `psql -U postgres -d pos_db < pos_db_backup.sql`
4. Re-seed if categories come out wrong (run the app once to trigger lazy-seed).

**⚠ NEVER run `docker compose down -v` without a verified backup. This deletes all data.**

### Go server must be restarted after adding new modules

`go run cmd/api/main.go` compiles once at startup and does NOT hot-reload.
After adding a new module and wiring its routes in `internal/app/`, the process must be restarted
for the new routes to be registered. Symptom: route returns `Cannot GET /api/v1/stores/.../new-route`
even though the source file is correct.

### Activity Center — change capture (extend to a new module)

The activity log derives **severity + category** from `(action, module)` in
`internal/modules/activity_log/taxonomy.go` (single source — the list filters
translate back through the same functions, so they can never drift; unit-tested).

To capture a `{before, after}` field diff for a module's update (drives the
Activity Center timeline + forward-only restore):
1. Add `internal/modules/<mod>/activity.go` with `activitySnapshot(entity)` →
   `map[string]any` keyed by the **API field names** the update endpoint accepts.
2. In the service `Update`, snapshot `before` right after loading current state,
   then after a successful write call
   `activitycapture.Record(ctx, "<kind>", beforeSnapshot, activitySnapshot(updated))`.
   The handler must pass `c.UserContext()` to the service (all do).
3. `<kind>` is the restore dispatch key (frontend) — distinct from `module`
   because the middleware canonicalises member/store/receipt to `module=settings`.

`changes` is a nullable JSONB column (migration 054); the Go field is `JSONText`
(its `Value()` returns a string so pgx stores real jsonb, not bytea). Historical
rows stay NULL → the UI shows "not captured before upgrade", never a fake diff.

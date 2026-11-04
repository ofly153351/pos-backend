# P0 PostgreSQL integration verification

`p0_postgres_test.go` uses real Fiber handlers in process, real services and GORM
repositories, and real PostgreSQL. It does not open an HTTP listener or use SQL
mocks. Ordinary `go test ./...` skips this package's integration tests unless
`POS_P0_TEST_DSN` is explicitly supplied.

## Safety contract

The DSN must be a `postgres://` URL targeting `127.0.0.1`, an explicit port other
than 5432, and a database named `pos_p0_disposable_*`. Its only query parameter
must be `sslmode=disable`. Before any DDL the test verifies that the database
comment is exactly `POS P0 DISPOSABLE TEST DATABASE`.

Each test creates a unique schema, applies the repository's actual `init-db/*.sql`
files in filename order (excluding `002_seed.sql`), inserts its own synthetic
fixtures, and drops only that schema through `t.Cleanup`. All queries/transactions
use a single-session pool with that schema's search path. No demo seed, local
business data, `.env`, or production connection is read. Use a genuinely throwaway
container/database, not a renamed or marked business database.

## Reproduce with one disposable container

Run from the backend repository. This uses tmpfs for PostgreSQL data, no existing
volume, and a loopback ephemeral published port. The trap removes only the
uniquely named container created by this invocation.

```sh
name="pos-p0-verification-$(date +%s)-$$"
docker run --detach --name "$name" \
  --publish 127.0.0.1::5432 \
  --env POSTGRES_HOST_AUTH_METHOD=trust \
  --env POSTGRES_DB=pos_p0_disposable_verification \
  --tmpfs /var/lib/postgresql/data postgres:15-alpine
trap 'docker rm --force --volumes "$name"' EXIT
ready=0
for i in $(seq 1 30); do
  if docker exec "$name" pg_isready -U postgres -d pos_p0_disposable_verification; then ready=1; break; fi
  sleep 1
done
[ "$ready" = 1 ] || { printf 'Disposable PostgreSQL did not become ready\n'; exit 1; }
docker exec "$name" psql -U postgres -d pos_p0_disposable_verification \
  -v ON_ERROR_STOP=1 \
  -c "COMMENT ON DATABASE pos_p0_disposable_verification IS 'POS P0 DISPOSABLE TEST DATABASE';"
port=$(docker port "$name" 5432/tcp)
port=${port##*:}
export POS_P0_TEST_DSN="postgres://postgres@127.0.0.1:${port}/pos_p0_disposable_verification?sslmode=disable"
go test ./internal/tests/p0postgres -count=1 -v
go test -race ./internal/tests/p0postgres -count=1 -v
```

The reproduction uses passwordless trust authentication only inside a new,
loopback-published disposable container. Never use this configuration for a
business database or a public interface.

## Verified behavior

- **B10:** JSON, URL-encoded form and multipart set/omit/explicit-empty tax ID
  transitions. Assertions check PUT response, a fresh handler GET, raw database
  contents, and unchanged tax ID in a second store.
- **B04:** Two units at 100, valid 25% campaign, inflated requested discount 200,
  duplicate campaign IDs and foreign/invalid campaigns yield a verified discount
  of 50, payable 150, stock 18. Fresh GET verifies persisted header and items.
  One sale, one stock movement and one campaign ledger row/counter increment
  survive an identical idempotent replay. Changed intent with the same key is
  409; the same key in another store is independent; a foreign store's sale GET
  is 404. Cross-store and invalid campaigns receive no usage/counter increments.
- **Legacy campaigns:** Negative/over-100 percentages, numeric strings, null,
  missing parameters, invalid JSON, negative bundle/fixed prices and invalid
  dates cannot create promotional discounts or usage entries.
- **Atomicity:** Negative discount (422) and underpayment (422) leave exact JSON
  snapshots unchanged across `sales`, `sale_items`, `stocks`, `stock_movements`,
  `promotions`, `promotion_usages`. A real PostgreSQL trigger deliberately raises
  `P0 forced final promo counter failure` at the final counter UPDATE, after the
  preceding sale/stock/usage writes. The handler returns 500 and the same six
  snapshots remain byte-identical, including timestamps and aggregate counters.
  Removing the trigger and retrying the same key commits exactly one complete sale.

## Recorded execution

On 2026-11-04 the uniquely named container was
`pos-p0-verification-20261104-subagent`, image `postgres:15-alpine`, published as
`127.0.0.1:32768`. Docker inspection reported no volume mounts; data was tmpfs.
Each test applied **67 schema migration files**, excluding the demo seed.

Recorded targeted command (throwaway DSN supplied through the environment):

```sh
go test -race ./internal/tests/p0postgres -count=1 -v
```

Result: `ok pos-backend/internal/tests/p0postgres 3.036s`, all four top-level tests
and their subtests passed. The forced PostgreSQL error in output is intentional.
The full `go test ./... -count=1` also passed with this DSN (integration package
`1.976s`); `go vet ./...` and `git diff --check` passed.

Cleanup read-back before container removal returned `remaining_test_schemas=0`
and `public_tables=0`; inspection confirmed
`tmpfs={"/var/lib/postgresql/data":""} mounts=[]`. Then
`docker rm --force --volumes pos-p0-verification-20261104-subagent` succeeded and
`docker ps -a --filter name=pos-p0-verification-20261104-subagent --format '{{.Names}}'`
returned no rows. No disposable container or persistent volume was left behind.

Regression sensitivity was checked without modifying the dirty source files:
Go's `-overlay` substituted HEAD versions of `store/handler.go` and
`sale/promo_discount.go` only at compilation. Against that pre-fix overlay,
URL-form/multipart empty tax IDs failed (old IDs remained), and over-100
percentage/negative bundle/negative fixed-price campaigns incorrectly reduced a
200 bill to zero. The same tests passed with the existing working-tree P0 fixes.
No additional production-code fix was needed. The first harness run exposed a
fixture assumption (default VAT is exclusive 7%, not zero); the fixture now
persists explicit `tax_mode=none` and uses the real DB settings adapter.

## Independent parent verification

The parent reran `go test -race ./internal/tests/p0postgres -count=1 -v` on a
second independently created disposable, loopback-published PostgreSQL container
using the passwordless reproduction configuration above. All four top-level
tests passed (`2.916s`), including raw DB read-back and forced late rollback.
Cleanup read-back reported zero remaining test schemas, zero public tables,
`mounts=[]`, and tmpfs data. Removal succeeded and a filtered container listing
returned no rows. No application servers or existing data were used.

Final verification aligned the store update harness with the actual PUT route.
The full backend suite passed with the disposable DSN (integration package
`1.988s`), followed by race-enabled integration (`2.873s`) and promotion/sale/store
module tests, `go vet ./...`, and diff checks. Final container cleanup again
returned zero remaining test schemas/public tables and an empty filtered listing.

## Limitations / historical audit

Authentication/store-membership middleware and browser/proxy behavior are outside
this package: it injects an owner and tests actual handler/service/repository seams.
The test pool is single-session; this is not a concurrent idempotency stress test.
Promotion evaluation remains the application's declared upper-bound contract, not
a complete reimplementation of client line/scope matching.

No historic user/business database was inspected. Consequently no claim can be
made about existing affected campaigns, historical losses, or remediation totals.
A read-only audit is possible only after explicit authorization and provision of
an appropriate snapshot/DSN; do not point these mutating tests at it. Migration
056 documents that pre-snapshot usage history lacks exact historical campaign
values, so even an authorized audit cannot always reconstruct original rules.

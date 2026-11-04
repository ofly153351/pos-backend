# B32: standalone operational health checks (local implementation)

`scripts/health_check.sh` is a one-shot, read-only Bash/curl probe, independent
of `golive.sh`. It does not deploy, start/restart services, load `.env`, install
cron, or repair anything. No production URLs are built in. An operator can run
it manually or invoke it from an independently configured scheduler.

## Usage

With **already running, authorized** local endpoints:

```bash
POS_HEALTH_FRONTEND_URL=http://127.0.0.1:3000/ \
POS_HEALTH_BACKEND_URL=http://127.0.0.1:8080/health \
POS_HEALTH_TIMEOUT=5 \
bash scripts/health_check.sh
```

Both URLs are required; HTTP(S) only. Set the exact canonical endpoint: **all
redirects fail**, including same-host redirects. This avoids treating a login
page or unexpected redirect host as healthy. Every HTTP 2xx succeeds; other
statuses, TLS/DNS/connect errors and timeouts fail. Both endpoints are always
checked after configuration validation, even if the first fails.

`POS_HEALTH_TIMEOUT` defaults to 5 seconds and accepts integers 1–300. Each probe
has that total timeout and connection timeout, with no retries. Two probes run
sequentially. curl's user config is disabled (`-q` first); response bodies,
headers and raw curl errors are not logged. Output identifies endpoint, HTTP
status (`000` if unavailable) and curl exit code (for example, 28 = timeout).
Never put credentials or secrets in URLs, including paths. Credentials, queries,
fragments and whitespace/control characters in URLs are rejected without
printing their values. TLS certificate verification remains enabled.

### Optional alert executable

Set `POS_HEALTH_ALERT_HOOK` to an absolute path to a trusted executable file.
It is executed directly, with no arguments or `eval`, **once per failed run**,
receiving the two endpoint diagnostic lines on stdin. It is not invoked when
healthy. It must return 0 only after successful delivery; its own stdout/stderr
are discarded to avoid leaking credentials or provider responses. The probe
reports `ALERT delivered` (hook exit 0) or `ALERT failed exit=N` (hook failure).
The hook must implement its own bounded delivery timeout and protect its
credentials; the checker does not supervise a hanging hook. No real delivery
hook is supplied or invoked by the tests.

Without a hook, failures remain visible on output and exit status, with
`ALERT not configured`. A scheduler must retain output/exit status and provide
its own execution environment, interval, log retention and notification policy.
There is no suppression, state/history, recovery notification or deduplication
between runs; repeated failures can notify every run. Nothing schedules itself.

| Exit | Meaning |
| --- | --- |
| 0 | Both endpoints returned HTTP 2xx; no alert sent |
| 1 | At least one endpoint failed; hook absent or returned success |
| 2 | Invalid configuration, missing curl or unusable configured hook; no probes |
| 3 | Endpoint failure and alert hook returned nonzero |

## Deterministic local evidence

```bash
bash -n scripts/health_check.sh scripts/test_health_check.sh
bash scripts/test_health_check.sh
```

The test suite uses temporary fake `curl` and alert executables, `.invalid`
URLs and a PATH override. It makes **no network requests**, starts no HTTP
server and sends no real alerts. It verifies healthy responses, 502/503,
timeout, checking both endpoints on failure, rejecting 302 rather than
following it, curl timeout/config/security flags, invalid URL/timeout inputs,
secret suppression, no alert on success, one combined alert on failure,
visible alert delivery failure and invalid hook paths. Observed final result:

```text
PASS healthy endpoints
PASS HTTP failure, timeout, both probes, redirect rejection and curl safety
PASS reject unsafe URL without exposing secrets
PASS invalid URLs and timeout configuration
PASS alert delivery, one summary, suppressed hook output and visible failures
```

RED → GREEN was exercised in slices: missing script (healthy-probe assertion
failed), then unsafe URL rejection (configuration assertion failed), then absent
alert delivery (hook-invocation assertion failed); each slice was implemented
and rerun to green before proceeding. Shell syntax checks also passed.

## Independent local network check

The parent also ran the checker against the existing user-started frontend and
an absent local backend, with a 3-second timeout. Actual output was frontend
HTTP 200 / curl exit 0, backend HTTP 000 / curl exit 7, `ALERT not configured`,
and checker exit 1. This verifies connection-failure detection without starting
any application server or sending a notification. It is not a production check
or proof of real alert delivery.

## Scope and outstanding B32 acceptance

Inspection found `golive.sh` checks health only during startup and warns on
failures; it is unchanged. `internal/app/app.go` returns `{"status":"ok"}`
from `/health` without checking the database. This script adds an operational
probe/alert capability, **not database readiness or business-flow validation**.
A 2xx frontend response alone does not prove rendering/auth/sales correctness.

The historical Cloudflare 502 root cause is **unknown**: no production tunnel,
proxy, application or DB logs were accessed or correlated. No production probe,
deployment, real scheduler installation or real notification delivery was
performed. B32 remains partially open until the operator correlates outage
logs, verifies actual website/API responses, and exercises an authorized origin
failure plus real alert delivery. Compare a public endpoint with a separately
run direct-origin check to distinguish edge reachability from origin liveness;
a failing HTTP status alone does not identify the responsible component.

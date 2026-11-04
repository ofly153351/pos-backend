#!/usr/bin/env bash
# Deterministic offline tests: fake curl only; never contact a real endpoint.
set -euo pipefail
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
TMP="$(mktemp -d "${TMPDIR:-/tmp}/pos-health-test.XXXXXX")"
trap 'rm -rf "$TMP"' EXIT
mkdir "$TMP/bin"
cat > "$TMP/bin/curl" <<'FAKE'
#!/usr/bin/env bash
printf '%s\n' "$*" >> "$CALLS"
case "${*: -1}" in
  http://frontend.invalid/) printf '%s' "${FE_STATUS:-200}"; exit "${FE_RC:-0}" ;;
  http://backend.invalid/health) printf '%s' "${BE_STATUS:-200}"; exit "${BE_RC:-0}" ;;
  *) echo 'unexpected endpoint' >&2; exit 99 ;;
esac
FAKE
chmod +x "$TMP/bin/curl"
export PATH="$TMP/bin:$PATH" CALLS="$TMP/calls"
export POS_HEALTH_FRONTEND_URL=http://frontend.invalid/
export POS_HEALTH_BACKEND_URL=http://backend.invalid/health
export POS_HEALTH_TIMEOUT=4
unset POS_HEALTH_ALERT_HOOK FE_STATUS FE_RC BE_STATUS BE_RC
fail() { printf 'FAIL: %s\n%s\n' "$1" "${OUT:-}" >&2; exit 1; }
run() {
  : > "$CALLS"
  RC=0
  OUT=$(bash "$ROOT/scripts/health_check.sh" 2>&1) || RC=$?
}
contains() { [[ "$OUT" == *"$1"* ]] || fail "missing diagnostic: $1"; }
run
[[ $RC == 0 ]] || fail "healthy probes must succeed (exit=$RC)"
contains 'OK frontend endpoint=http://frontend.invalid/ status=200 curl_exit=0'
contains 'OK backend endpoint=http://backend.invalid/health status=200 curl_exit=0'
printf 'PASS healthy endpoints\n'
# One failed endpoint must not prevent the other from being checked.
export FE_STATUS=502
run
[[ $RC == 1 ]] || fail '502 must fail'
contains 'FAIL frontend endpoint=http://frontend.invalid/ status=502 curl_exit=0'
contains 'OK backend'
export FE_STATUS=000 FE_RC=28 BE_STATUS=503
run
[[ $RC == 1 ]] || fail 'timeout must fail'
contains 'status=000 curl_exit=28'
contains 'FAIL backend endpoint=http://backend.invalid/health status=503'
unset FE_STATUS FE_RC BE_STATUS
export FE_STATUS=302
run
[[ $RC == 1 ]] || fail 'redirect must not count as healthy'
# Audit fake-curl argv: no curlrc, redirects, retry, body/header or verbose logs.
while IFS= read -r args; do
  [[ "$args" == '-q '* ]] || fail 'curl must disable curlrc first'
  [[ "$args" == *'--connect-timeout 4 --max-time 4'* ]] || fail 'bounded request required'
  [[ "$args" == *'--output /dev/null'* && "$args" == *'--proto =http,https'* ]] || fail 'safe HTTP-only probe required'
  [[ "$args" != *'--location'* && "$args" != *' -L'* && "$args" != *'--verbose'* && "$args" != *'--insecure'* ]] || fail 'unsafe curl flags'
done < "$CALLS"
unset FE_STATUS
printf 'PASS HTTP failure, timeout, both probes, redirect rejection and curl safety\n'
export POS_HEALTH_FRONTEND_URL='https://user:SECRET@example.invalid/?token=SECRET'
run
[[ $RC == 2 && ! -s "$CALLS" ]] || fail 'unsafe URL must fail configuration before any probe'
[[ "$OUT" != *SECRET* ]] || fail 'configuration secret leaked'
contains 'CONFIG frontend URL'
printf 'PASS reject unsafe URL without exposing secrets\n'
for bad in '' 'file:///etc/passwd' 'https://host.invalid/?SECRET' 'https://host.invalid/#SECRET' $'http://host.invalid/\nSECRET' 'https://SECRET@host.invalid/'; do
  export POS_HEALTH_FRONTEND_URL="$bad"
  run
  [[ $RC == 2 && ! -s "$CALLS" && "$OUT" != *SECRET* ]] || fail 'invalid URL not rejected safely'
done
export POS_HEALTH_FRONTEND_URL=http://frontend.invalid/
for bad in 0 -1 301 0005 1.5 SECRET; do
  export POS_HEALTH_TIMEOUT="$bad"
  run
  [[ $RC == 2 && ! -s "$CALLS" ]] || fail 'invalid timeout must fail before probes'
done
export POS_HEALTH_TIMEOUT=4
printf 'PASS invalid URLs and timeout configuration\n'
cat > "$TMP/bin/alert" <<'FAKE'
#!/usr/bin/env bash
printf 'called\n' >> "$ALERT_CALLS"
while IFS= read -r line; do printf '%s\n' "$line"; done > "$ALERT_INPUT"
printf 'SECRET hook stdout\n'
printf 'SECRET hook stderr\n' >&2
exit "${ALERT_RC:-0}"
FAKE
chmod +x "$TMP/bin/alert"
export POS_HEALTH_ALERT_HOOK="$TMP/bin/alert" ALERT_CALLS="$TMP/alerts" ALERT_INPUT="$TMP/input"
run
[[ $RC == 0 && ! -e "$ALERT_CALLS" ]] || fail 'healthy checks must not alert'
export FE_STATUS=502 BE_STATUS=503
run
[[ $RC == 1 && -s "$ALERT_CALLS" ]] || fail 'failure must invoke optional hook'
contains 'ALERT delivered'
[[ "$OUT" != *SECRET* ]] || fail 'hook output must not leak'
input=$(< "$ALERT_INPUT")
[[ "$input" == *'FAIL frontend'* && "$input" == *'FAIL backend'* ]] || fail 'hook needs both endpoint results'
count=0; while IFS= read -r line; do count=$((count + 1)); done < "$ALERT_CALLS"
[[ $count == 1 ]] || fail 'only one alert per failed run'
export ALERT_RC=7
run
[[ $RC == 3 ]] || fail 'alert delivery failure must have distinct nonzero exit'
contains 'ALERT failed exit=7'
[[ "$OUT" != *SECRET* ]] || fail 'failed hook output must not leak'
unset ALERT_RC FE_STATUS BE_STATUS
export POS_HEALTH_ALERT_HOOK="$TMP/missing"
run
[[ $RC == 2 && ! -s "$CALLS" ]] || fail 'missing hook must fail before probes'
export POS_HEALTH_ALERT_HOOK='echo SECRET'
run
[[ $RC == 2 && "$OUT" != *SECRET* ]] || fail 'hook is executable path, not shell text'
printf 'PASS alert delivery, one summary, suppressed hook output and visible failures\n'

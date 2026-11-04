#!/usr/bin/env bash
# One-shot, read-only HTTP probes. No service management or scheduler changes.
set -euo pipefail

config_error() { printf 'CONFIG %s\n' "$1" >&2; exit 2; }
validate_url() {
  local name="$1" url="$2" authority
  # Deliberately reject credentials, queries, fragments and log-control characters.
  # No raw rejected value is ever printed.
  [[ "$url" =~ ^https?://[^/]+(/.*)?$ && "$url" != *'@'* && "$url" != *'?'* &&
     "$url" != *'#'* && "$url" != *'\\'* && ! "$url" =~ [[:space:][:cntrl:]] ]] ||
    config_error "$name URL must be explicit HTTP(S), without credentials, query, fragment or whitespace"
  authority="${url#*://}"; authority="${authority%%/*}"
  [[ -n "$authority" ]] || config_error "$name URL requires a host"
}
validate_url frontend "${POS_HEALTH_FRONTEND_URL:-}"
validate_url backend "${POS_HEALTH_BACKEND_URL:-}"
timeout="${POS_HEALTH_TIMEOUT:-5}"
[[ "$timeout" =~ ^[1-9][0-9]{0,2}$ ]] && (( timeout <= 300 )) ||
  config_error 'POS_HEALTH_TIMEOUT must be an integer from 1 to 300 seconds'
command -v curl >/dev/null 2>&1 || config_error 'curl executable not found'
hook="${POS_HEALTH_ALERT_HOOK:-}"
if [[ -n "$hook" ]]; then
  [[ "$hook" == /* && -f "$hook" && -x "$hook" ]] ||
    config_error 'POS_HEALTH_ALERT_HOOK must be an absolute executable file path'
fi

probe() {
  local name="$1" url="$2" status rc=0
  status=$(curl -q --silent --globoff --output /dev/null \
    --connect-timeout "${POS_HEALTH_TIMEOUT:-5}" --max-time "${POS_HEALTH_TIMEOUT:-5}" \
    --retry 0 --proto '=http,https' --write-out '%{http_code}' \
    --url "$url" 2>/dev/null) || rc=$?
  if [[ "$rc" == 0 && "$status" =~ ^2[0-9][0-9]$ ]]; then
    printf 'OK %s endpoint=%s status=%s curl_exit=%s\n' "$name" "$url" "$status" "$rc"
  else
    printf 'FAIL %s endpoint=%s status=%s curl_exit=%s\n' "$name" "$url" "${status:-000}" "$rc"
    return 1
  fi
}

failed=0
frontend_result=$(probe frontend "$POS_HEALTH_FRONTEND_URL") || failed=1
backend_result=$(probe backend "$POS_HEALTH_BACKEND_URL") || failed=1
printf '%s\n%s\n' "$frontend_result" "$backend_result"
if (( failed )); then
  if [[ -n "$hook" ]]; then
    alert_rc=0
    # No eval, shell arguments, headers or response bodies. The trusted hook owns
    # delivery credentials and its own delivery timeout. Never relay its output.
    "$hook" <<< "$frontend_result"$'\n'"$backend_result" >/dev/null 2>&1 || alert_rc=$?
    if (( alert_rc != 0 )); then
      printf 'ALERT failed exit=%s\n' "$alert_rc" >&2
      exit 3
    fi
    printf 'ALERT delivered\n'
  else
    printf 'ALERT not configured; endpoint failure reported by output and exit=1\n' >&2
  fi
fi
exit "$failed"

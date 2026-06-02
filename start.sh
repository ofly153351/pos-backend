#!/usr/bin/env bash
# POS System — start all services
# Usage: ./start.sh [--tunnel] [--tailscale]
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
if [[ -f "$SCRIPT_DIR/go.mod" ]]; then
  BACKEND_DIR="$SCRIPT_DIR"
  FRONTEND_DIR="$(dirname "$SCRIPT_DIR")/pos-frontend"
else
  BACKEND_DIR="$SCRIPT_DIR/pos-backend"
  FRONTEND_DIR="$SCRIPT_DIR/pos-frontend"
fi

WITH_TUNNEL=false
WITH_TAILSCALE=false
for arg in "$@"; do
  [[ "$arg" == "--tunnel" ]]    && WITH_TUNNEL=true
  [[ "$arg" == "--tailscale" ]] && WITH_TAILSCALE=true
done

# load nvm
export NVM_DIR="${NVM_DIR:-$HOME/.nvm}"
[[ -s "$NVM_DIR/nvm.sh" ]] && source "$NVM_DIR/nvm.sh"

# load env values for ports
set -a; source "$BACKEND_DIR/.env" 2>/dev/null || true; set +a
BE_PORT="${APP_PORT:-8080}"
FE_PORT=$(grep "^PORT=" "$FRONTEND_DIR/.env.local" 2>/dev/null | cut -d= -f2 || echo "3000")

echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "  POS System — starting"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

# ── 1. Docker services ──────────────────────────────────────────────
echo "→  Docker (PostgreSQL + MinIO)..."
cd "$BACKEND_DIR"
docker compose up -d postgres minio minio-client
until docker compose exec -T postgres \
  pg_isready -U "${POSTGRES_USER:-postgres}" -d "${POSTGRES_DB:-pos_db}" &>/dev/null; do
  sleep 1
done
echo "✔  PostgreSQL ready"

# ── 2. Backend ──────────────────────────────────────────────────────
echo "→  Go backend..."
cd "$BACKEND_DIR"
go run cmd/api.go > /tmp/pos-backend.log 2>&1 &
BE_PID=$!
sleep 4
if ! kill -0 "$BE_PID" 2>/dev/null; then
  echo "✖  Backend failed — tail /tmp/pos-backend.log"
  tail -20 /tmp/pos-backend.log
  exit 1
fi
echo "✔  Backend  PID=$BE_PID  →  http://localhost:$BE_PORT"

# ── 3. Frontend ─────────────────────────────────────────────────────
echo "→  Next.js frontend..."
cd "$FRONTEND_DIR"
npm run dev > /tmp/pos-frontend.log 2>&1 &
FE_PID=$!
sleep 4
echo "✔  Frontend PID=$FE_PID  →  http://localhost:$FE_PORT"

# ── 4. Cloudflare Tunnel (optional) ─────────────────────────────────
if [[ "$WITH_TUNNEL" == true ]]; then
  TUNNEL_CFG="$SCRIPT_DIR/.tunnel_config"
  if [[ ! -f "$TUNNEL_CFG" ]]; then
    echo "⚠  No tunnel config found. Run ./setup.sh --tunnel first."
  else
    CF_CONFIG=$(head -1 "$TUNNEL_CFG")
    FE_URL=$(sed -n '2p' "$TUNNEL_CFG")
    BE_URL=$(sed -n '3p' "$TUNNEL_CFG")
    echo "→  Cloudflare Tunnel..."
    cloudflared tunnel --config "$CF_CONFIG" run > /tmp/pos-tunnel.log 2>&1 &
    CF_PID=$!
    sleep 2
    echo "✔  Tunnel PID=$CF_PID"
    echo "    Frontend → https://$FE_URL"
    echo "    Backend  → https://$BE_URL"
  fi
fi

# ── 5. Tailscale status (optional) ──────────────────────────────────
TS_IP=""
if [[ "$WITH_TAILSCALE" == true ]]; then
  echo "→  Tailscale..."
  if ! command -v tailscale &>/dev/null; then
    echo "⚠  tailscale not found. Run ./setup.sh --tailscale first."
  else
    # ensure daemon is up (Linux); macOS uses GUI app
    if [[ "$(uname)" != "Darwin" ]]; then
      sudo systemctl start tailscaled 2>/dev/null || true
    fi
    TS_IP=$(tailscale ip -4 2>/dev/null || echo "")
    if [[ -z "$TS_IP" ]]; then
      echo "⚠  Tailscale not connected. Run: sudo tailscale up"
    else
      echo "✔  Tailscale IP: $TS_IP"
    fi
  fi
fi

# ─────────────────────────────────────────────────────────────────────
echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "  All services running"
echo ""
echo "  Local:"
echo "    Frontend : http://localhost:$FE_PORT"
echo "    Backend  : http://localhost:$BE_PORT"
echo "    MinIO    : http://localhost:9001"
if [[ -n "$TS_IP" ]]; then
  echo ""
  echo "  Tailscale (accessible by team):"
  echo "    Frontend : http://$TS_IP:$FE_PORT"
  echo "    Backend  : http://$TS_IP:$BE_PORT"
fi
echo ""
echo "  Logs:"
echo "    Backend  → /tmp/pos-backend.log"
echo "    Frontend → /tmp/pos-frontend.log"
echo ""
echo "  Stop → ./stop.sh   |   Ctrl+C to detach"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

wait

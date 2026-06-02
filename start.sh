#!/usr/bin/env bash
# POS System — start all services via PM2
# Usage: ./start.sh [--tunnel] [--tailscale] [--dev]
#   --dev        use Next.js dev server instead of production build
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
DEV_MODE=false
for arg in "$@"; do
  [[ "$arg" == "--tunnel" ]]    && WITH_TUNNEL=true
  [[ "$arg" == "--tailscale" ]] && WITH_TAILSCALE=true
  [[ "$arg" == "--dev" ]]       && DEV_MODE=true
done

# load nvm
export NVM_DIR="${NVM_DIR:-$HOME/.nvm}"
[[ -s "$NVM_DIR/nvm.sh" ]] && source "$NVM_DIR/nvm.sh"

# load env
set -a; source "$BACKEND_DIR/.env" 2>/dev/null || true; set +a
BE_PORT="${APP_PORT:-8080}"
FE_PORT=$(grep "^PORT=" "$FRONTEND_DIR/.env.local" 2>/dev/null | cut -d= -f2 || echo "3000")

echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "  POS System — starting (PM2)"
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

# ── 2. Build Go binary ──────────────────────────────────────────────
echo "→  Building Go backend..."
cd "$BACKEND_DIR"
mkdir -p bin
go build -o bin/api cmd/api.go
echo "✔  Backend binary: bin/api"

# ── 3. Build / prepare Next.js ──────────────────────────────────────
cd "$FRONTEND_DIR"
if [[ "$DEV_MODE" == true ]]; then
  echo "→  Frontend: dev mode (skip build)"
  # patch ecosystem to use dev server
  pm2 start npm --name pos-frontend -- run dev 2>/dev/null || true
  FRONTEND_STARTED_VIA_PM2_DEV=true
else
  echo "→  Building Next.js frontend..."
  npm run build
  echo "✔  Frontend build complete"
  FRONTEND_STARTED_VIA_PM2_DEV=false
fi

# ── 4. Create PM2 log dir ───────────────────────────────────────────
sudo mkdir -p /var/log/pm2
sudo chown "$USER":"$USER" /var/log/pm2 2>/dev/null || true

# ── 5. Start with PM2 ───────────────────────────────────────────────
echo "→  Starting PM2 processes..."
cd "$BACKEND_DIR"

if [[ "$FRONTEND_STARTED_VIA_PM2_DEV" == true ]]; then
  # only start backend via ecosystem; frontend already running
  pm2 start ecosystem.config.js --only pos-backend
else
  pm2 start ecosystem.config.js
fi

# save process list so PM2 restarts on reboot
pm2 save

echo "✔  PM2 processes started"

# ── 6. Cloudflare Tunnel (optional) ─────────────────────────────────
if [[ "$WITH_TUNNEL" == true ]]; then
  TUNNEL_CFG="$SCRIPT_DIR/.tunnel_config"
  if [[ ! -f "$TUNNEL_CFG" ]]; then
    echo "⚠  No tunnel config. Run: bash setup.sh --tunnel"
  else
    CF_CONFIG=$(head -1 "$TUNNEL_CFG")
    FE_URL=$(sed -n '2p' "$TUNNEL_CFG")
    BE_URL=$(sed -n '3p' "$TUNNEL_CFG")
    pm2 start cloudflared \
      --name pos-tunnel \
      --interpreter none \
      -- tunnel --config "$CF_CONFIG" run
    pm2 save
    echo "✔  Tunnel started"
    echo "    Frontend → https://$FE_URL"
    echo "    Backend  → https://$BE_URL"
  fi
fi

# ── 7. Tailscale status (optional) ──────────────────────────────────
TS_IP=""
if [[ "$WITH_TAILSCALE" == true ]]; then
  if command -v tailscale &>/dev/null; then
    [[ "$(uname)" != "Darwin" ]] && sudo systemctl start tailscaled 2>/dev/null || true
    TS_IP=$(tailscale ip -4 2>/dev/null || echo "")
    [[ -n "$TS_IP" ]] && echo "✔  Tailscale IP: $TS_IP" || echo "⚠  Tailscale not connected — run: sudo tailscale up"
  else
    echo "⚠  tailscale not found. Run: bash setup.sh --tailscale"
  fi
fi

# ── Summary ─────────────────────────────────────────────────────────
echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "  All services running via PM2"
echo ""
echo "  Local:"
echo "    Frontend : http://localhost:$FE_PORT"
echo "    Backend  : http://localhost:$BE_PORT"
echo "    MinIO    : http://localhost:9001"
if [[ -n "$TS_IP" ]]; then
  echo ""
  echo "  Tailscale:"
  echo "    Frontend : http://$TS_IP:$FE_PORT"
  echo "    Backend  : http://$TS_IP:$BE_PORT"
fi
echo ""
echo "  PM2 commands:"
echo "    pm2 status          — process list"
echo "    pm2 logs            — stream all logs"
echo "    pm2 logs pos-backend — backend logs only"
echo "    pm2 restart all     — restart everything"
echo "    pm2 monit           — live CPU/RAM monitor"
echo ""
echo "  Stop → ./stop.sh"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

pm2 status

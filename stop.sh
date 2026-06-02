#!/usr/bin/env bash
# POS System — stop all services
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BACKEND_DIR="$([[ -f "$SCRIPT_DIR/go.mod" ]] && echo "$SCRIPT_DIR" || echo "$SCRIPT_DIR/pos-backend")"

# load nvm
export NVM_DIR="${NVM_DIR:-$HOME/.nvm}"
[[ -s "$NVM_DIR/nvm.sh" ]] && source "$NVM_DIR/nvm.sh"

echo "Stopping POS services..."

# stop PM2 processes
if command -v pm2 &>/dev/null; then
  pm2 stop pos-backend  2>/dev/null && echo "✔  pos-backend stopped"  || echo "–  pos-backend not running"
  pm2 stop pos-frontend 2>/dev/null && echo "✔  pos-frontend stopped" || echo "–  pos-frontend not running"
  pm2 stop pos-tunnel   2>/dev/null && echo "✔  pos-tunnel stopped"   || echo "–  pos-tunnel not running"
  pm2 save
else
  # fallback: pkill
  pkill -f "bin/api"        2>/dev/null && echo "✔  Backend stopped"  || echo "–  Backend not running"
  pkill -f "next"           2>/dev/null && echo "✔  Frontend stopped" || echo "–  Frontend not running"
  pkill -f "cloudflared"    2>/dev/null && echo "✔  Tunnel stopped"   || echo "–  Tunnel not running"
fi

# stop Docker
cd "$BACKEND_DIR" 2>/dev/null && \
  docker compose down 2>/dev/null && echo "✔  Docker stopped" || echo "–  Docker not running"

echo "Done."

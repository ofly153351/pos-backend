#!/usr/bin/env bash
# POS System — stop all services
set -euo pipefail
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BACKEND_DIR="$([[ -f "$SCRIPT_DIR/go.mod" ]] && echo "$SCRIPT_DIR" || echo "$SCRIPT_DIR/pos-backend")"

echo "Stopping POS services..."

pkill -f "go run cmd/api.go" 2>/dev/null && echo "✔  Backend stopped"    || echo "–  Backend not running"
pkill -f "next dev"           2>/dev/null && echo "✔  Frontend stopped"   || echo "–  Frontend not running"
pkill -f "cloudflared tunnel" 2>/dev/null && echo "✔  Tunnel stopped"     || echo "–  Tunnel not running"

cd "$BACKEND_DIR" 2>/dev/null && \
  docker compose down        2>/dev/null && echo "✔  Docker stopped"      || echo "–  Docker not running"

echo "Done."

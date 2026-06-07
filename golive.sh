#!/usr/bin/env bash
# =============================================================================
# POS System — Go Live Script
# รัน script นี้ครั้งเดียวเพื่อ startup ทุกอย่างและพร้อมใช้งาน
#
# Usage:
#   bash golive.sh
# =============================================================================
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BACKEND_DIR="$SCRIPT_DIR"
FRONTEND_DIR="$(dirname "$SCRIPT_DIR")/pos-frontend"
TUNNEL_CONFIG="$HOME/.cloudflared/config.yml"

# ── Git credentials (token stored in ~/.git-credentials via credential.helper store)
GITHUB_USER="ofly153351"
GITHUB_TOKEN_FILE="$HOME/.github_token"   # token เก็บแยกไฟล์ ไม่ hard-code ใน script

GREEN='\033[0;32m'; YELLOW='\033[1;33m'; RED='\033[0;31m'
CYAN='\033[0;36m'; BOLD='\033[1m'; RESET='\033[0m'

ok()   { echo -e "${GREEN}✔${RESET}  $*"; }
info() { echo -e "${CYAN}→${RESET}  $*"; }
warn() { echo -e "${YELLOW}⚠${RESET}  $*"; }
die()  { echo -e "${RED}✖${RESET}  $*" >&2; exit 1; }
step() { echo -e "\n${BOLD}━━━  $*${RESET}"; }

# load nvm
export NVM_DIR="${NVM_DIR:-$HOME/.nvm}"
[[ -s "$NVM_DIR/nvm.sh" ]] && source "$NVM_DIR/nvm.sh"

# load env
set -a; source "$BACKEND_DIR/.env" 2>/dev/null || die "ไม่พบ .env — สร้างก่อนด้วย setup.sh"; set +a
BE_PORT="${APP_PORT:-8080}"
FE_PORT=$(grep "^PORT=" "$FRONTEND_DIR/.env.local" 2>/dev/null | cut -d= -f2 || echo "3000")
CORS="${APP_CORS_ALLOW_ORIGINS:-}"

# ─────────────────────────────────────────────────────────────────────────────
echo ""
echo -e "${BOLD}  ██████╗  ██████╗ ███████╗"
echo    "  ██╔══██╗██╔═══██╗██╔════╝"
echo    "  ██████╔╝██║   ██║███████╗"
echo    "  ██╔═══╝ ██║   ██║╚════██║"
echo    "  ██║     ╚██████╔╝███████║"
echo -e "  ╚═╝      ╚═════╝ ╚══════╝  Go Live${RESET}"
echo ""

# ── 0. Pre-flight checks ─────────────────────────────────────────────────────
step "Pre-flight checks"

[[ -d "$BACKEND_DIR" ]]  || die "ไม่พบ pos-backend ที่ $BACKEND_DIR"
[[ -d "$FRONTEND_DIR" ]] || die "ไม่พบ pos-frontend ที่ $FRONTEND_DIR"
command -v docker      &>/dev/null || die "docker ไม่ได้ติดตั้ง — รัน setup.sh ก่อน"
command -v go          &>/dev/null || die "go ไม่ได้ติดตั้ง — รัน setup.sh ก่อน"
command -v node        &>/dev/null || die "node ไม่ได้ติดตั้ง — รัน setup.sh ก่อน"
command -v pm2         &>/dev/null || die "pm2 ไม่ได้ติดตั้ง — รัน: npm install -g pm2"
[[ -f "$GITHUB_TOKEN_FILE" ]] || die "ไม่พบ token file ที่ $GITHUB_TOKEN_FILE — รัน: echo 'ghp_xxx' > $GITHUB_TOKEN_FILE && chmod 600 $GITHUB_TOKEN_FILE"

ok "ทุกอย่างพร้อม"

# ── 1. Git pull latest code ──────────────────────────────────────────────────
step "Git pull (backend + frontend)"

GITHUB_TOKEN="$(cat "$GITHUB_TOKEN_FILE" | tr -d '[:space:]')"

# ตั้ง remote URL พร้อม token ชั่วคราว (ไม่เขียนลง config ถาวร)
_be_remote=$(git -C "$BACKEND_DIR"  remote get-url origin 2>/dev/null || echo "")
_fe_remote=$(git -C "$FRONTEND_DIR" remote get-url origin 2>/dev/null || echo "")

_inject_token() {
  local url="$1"
  # แทน https://github.com/ → https://user:token@github.com/
  echo "$url" | sed "s|https://|https://${GITHUB_USER}:${GITHUB_TOKEN}@|"
}

git -C "$BACKEND_DIR"  remote set-url origin "$(_inject_token "$_be_remote")"
git -C "$FRONTEND_DIR" remote set-url origin "$(_inject_token "$_fe_remote")"

# Force-sync a repo to origin/<BRANCH> regardless of current branch / local state.
# Production deploys from main. Always restore the token-free remote URL afterwards.
BRANCH="main"
sync_repo() {
  local dir="$1" name="$2"
  info "Syncing $name → origin/$BRANCH..."
  git -C "$dir" fetch origin "$BRANCH" || die "$name: git fetch failed"
  git -C "$dir" checkout -B "$BRANCH" "origin/$BRANCH" || die "$name: checkout failed"
  git -C "$dir" reset --hard "origin/$BRANCH" || die "$name: reset failed"
  ok "$name updated → $(git -C "$dir" rev-parse --short HEAD)"
}

sync_repo "$BACKEND_DIR"  "pos-backend"
sync_repo "$FRONTEND_DIR" "pos-frontend"

# คืน remote URL กลับเป็นแบบไม่มี token (ปลอดภัย)
git -C "$BACKEND_DIR"  remote set-url origin "$_be_remote"
git -C "$FRONTEND_DIR" remote set-url origin "$_fe_remote"
command -v cloudflared &>/dev/null || die "cloudflared ไม่ได้ติดตั้ง — รัน setup.sh ก่อน"
[[ -f "$TUNNEL_CONFIG" ]] || die "ไม่พบ tunnel config ที่ $TUNNEL_CONFIG — รัน setup.sh --tunnel ก่อน"

ok "ทุกอย่างพร้อม"

# ── 1. Stop existing processes ───────────────────────────────────────────────
step "หยุด process เดิม (ถ้ามี)"
pm2 stop all 2>/dev/null && pm2 delete all 2>/dev/null || true
ok "ล้าง PM2 แล้ว"

# ── 2. Docker — PostgreSQL + MinIO ──────────────────────────────────────────
step "Docker (PostgreSQL + MinIO)"
cd "$BACKEND_DIR"
info "Starting containers..."
docker compose up -d postgres minio minio-client

info "รอ PostgreSQL ready..."
local_attempts=0
until docker compose exec -T postgres \
  pg_isready -U "${POSTGRES_USER:-postgres}" -d "${POSTGRES_DB:-pos_db}" &>/dev/null; do
  local_attempts=$((local_attempts + 1))
  [[ $local_attempts -gt 30 ]] && die "PostgreSQL ไม่ตอบสนอง — ดู: docker compose logs postgres"
  sleep 1
done
ok "PostgreSQL ready"
ok "MinIO ready"

# ── 3. Build Go backend ──────────────────────────────────────────────────────
step "Build Go backend"
cd "$BACKEND_DIR"
info "Building binary..."
mkdir -p bin
go build -o bin/api cmd/api.go
ok "Binary: $BACKEND_DIR/bin/api"

# ── 4. Build Next.js frontend ────────────────────────────────────────────────
step "Build Next.js frontend"
cd "$FRONTEND_DIR"
info "npm install..."
npm install --prefer-offline 2>/dev/null || npm install
info "ล้าง .next cache เก่า..."
rm -rf "$FRONTEND_DIR/.next"
info "npm run build..."
npm run build
ok "Frontend build complete"

# ── 5. PM2 log directory ─────────────────────────────────────────────────────
mkdir -p /var/log/pm2 2>/dev/null || sudo mkdir -p /var/log/pm2
chown "$USER":"$USER" /var/log/pm2 2>/dev/null || true

# ── 6. Start via PM2 ────────────────────────────────────────────────────────
step "Start services via PM2"
cd "$BACKEND_DIR"
pm2 start ecosystem.config.js
ok "pos-backend และ pos-frontend เริ่มแล้ว"

# ── 7. Cloudflare Tunnel ─────────────────────────────────────────────────────
step "Cloudflare Tunnel"
info "Starting tunnel..."
pm2 start cloudflared \
  --name pos-tunnel \
  --interpreter none \
  -- tunnel --config "$TUNNEL_CONFIG" run

# รอสักครู่ให้ tunnel connect
sleep 4

if pm2 show pos-tunnel 2>/dev/null | grep -q "online"; then
  ok "Tunnel online"
else
  warn "Tunnel อาจยังไม่ connect — ดู: pm2 logs pos-tunnel"
fi

# ── 8. Save PM2 process list ─────────────────────────────────────────────────
step "Save PM2 + Startup"
pm2 save
ok "pm2 save แล้ว (reboot จะ restart อัตโนมัติ)"

# ── 9. Health checks ─────────────────────────────────────────────────────────
step "Health checks"
sleep 3

# backend
if curl -sf "http://localhost:${BE_PORT}/health" &>/dev/null; then
  ok "Backend  http://localhost:${BE_PORT}/health"
else
  warn "Backend ยังไม่ตอบ — ดู: pm2 logs pos-backend"
fi

# frontend
if curl -sf "http://localhost:${FE_PORT}" &>/dev/null; then
  ok "Frontend http://localhost:${FE_PORT}"
else
  warn "Frontend ยังไม่ตอบ — ดู: pm2 logs pos-frontend"
fi

# minio
if curl -sf "http://localhost:9000/minio/health/live" &>/dev/null; then
  ok "MinIO    http://localhost:9000"
else
  warn "MinIO ยังไม่ตอบ"
fi

# public URLs จาก tunnel config
FE_HOST=$(grep "hostname:" "$TUNNEL_CONFIG" | awk '{print $3}' | head -1)
BE_HOST=$(grep "hostname:" "$TUNNEL_CONFIG" | awk '{print $3}' | sed -n '2p')
MEDIA_HOST=$(grep "hostname:" "$TUNNEL_CONFIG" | awk '{print $3}' | sed -n '3p')

sleep 2
if curl -sf "https://${BE_HOST}/health" &>/dev/null; then
  ok "Public backend  https://${BE_HOST}/health"
else
  warn "Public backend ยังไม่ตอบ (tunnel อาจยังกำลัง connect)"
fi

# ── Summary ──────────────────────────────────────────────────────────────────
echo ""
echo -e "${GREEN}${BOLD}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${RESET}"
echo -e "${GREEN}${BOLD}  🚀 POS System is LIVE!${RESET}"
echo -e "${GREEN}${BOLD}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${RESET}"
echo ""
echo "  Public URLs:"
echo "    Frontend  →  https://${FE_HOST}"
echo "    Backend   →  https://${BE_HOST}"
echo "    MinIO     →  https://${MEDIA_HOST}"
echo ""
echo "  Default login:"
echo "    owner@pos.dev   / Owner1234!"
echo "    manager@pos.dev / Manager1234!"
echo "    cashier@pos.dev / Cashier1234!"
echo ""
echo "  PM2 commands:"
echo "    pm2 status              — ดู process ทั้งหมด"
echo "    pm2 logs                — ดู log realtime"
echo "    pm2 logs pos-backend    — log backend"
echo "    pm2 logs pos-frontend   — log frontend"
echo "    pm2 logs pos-tunnel     — log tunnel"
echo "    pm2 restart pos-backend — restart backend"
echo "    pm2 monit               — CPU/RAM realtime"
echo ""
echo "  Stop → bash stop.sh"
echo -e "${GREEN}${BOLD}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${RESET}"
echo ""

pm2 status

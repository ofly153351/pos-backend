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

# Production deploys from main.
BRANCH="main"

# Build a tokenized fetch URL from origin WITHOUT persisting the token in git config.
# Strips any existing creds first → idempotent (no double-injection even if a prior
# run left a token baked into the remote URL).
tokened_url() {
  local clean
  clean=$(git -C "$1" remote get-url origin | sed -E 's#https://[^@/]+@#https://#')
  echo "$clean" | sed "s#https://#https://${GITHUB_USER}:${GITHUB_TOKEN}@#"
}

# Force-sync a repo to origin/<BRANCH> regardless of current branch / local state.
sync_repo() {
  local dir="$1" name="$2" url
  url=$(tokened_url "$dir")
  info "Syncing $name → origin/$BRANCH..."
  git -C "$dir" fetch "$url" "$BRANCH" || die "$name: git fetch failed (token in $GITHUB_TOKEN_FILE?)"
  git -C "$dir" checkout -B "$BRANCH" 2>/dev/null || die "$name: checkout failed"
  git -C "$dir" reset --hard FETCH_HEAD || die "$name: reset failed"
  ok "$name updated → $(git -C "$dir" rev-parse --short HEAD)"
}

sync_repo "$BACKEND_DIR"  "pos-backend"
sync_repo "$FRONTEND_DIR" "pos-frontend"
[[ -f "$TUNNEL_CONFIG" ]] || die "ไม่พบ tunnel config ที่ $TUNNEL_CONFIG — รัน setup.sh --tunnel ก่อน"
command -v systemctl &>/dev/null || die "systemd/systemctl ไม่พร้อมใช้งาน — ตรวจสอบเครื่อง Ubuntu ก่อน"
systemctl cat cloudflared.service &>/dev/null || die "ไม่พบ systemd service cloudflared — ติดตั้ง cloudflared service ก่อน"
systemctl is-enabled --quiet cloudflared || die "cloudflared ยังไม่ enabled — รัน: sudo systemctl enable --now cloudflared"

ok "ทุกอย่างพร้อม"

# ── 1. Stop existing processes ───────────────────────────────────────────────
step "หยุด process เดิม (ถ้ามี)"
pm2 stop all 2>/dev/null && pm2 delete all 2>/dev/null || true
ok "ล้าง PM2 แล้ว"

# ── 2. Docker — PostgreSQL + MinIO ──────────────────────────────────────────
step "Docker (PostgreSQL + MinIO)"
cd "$BACKEND_DIR"

docker_failure() {
  local message="$1"
  echo ""
  warn "$message"
  info "docker compose ps postgres minio"
  docker compose ps postgres minio
  info "docker compose logs --tail=100 postgres minio"
  docker compose logs --tail=100 postgres minio
  die "$message"
}

info "Pulling PostgreSQL and MinIO images..."
if ! docker compose pull postgres minio; then
  docker_failure "ไม่สามารถ pull image ของ PostgreSQL หรือ MinIO ได้"
fi

info "Starting PostgreSQL and MinIO..."
if ! docker compose up -d postgres minio; then
  docker_failure "ไม่สามารถ start PostgreSQL หรือ MinIO ได้"
fi

info "ตรวจสถานะ containers หลัง startup..."
docker compose ps postgres minio

info "รอ PostgreSQL ready..."
local_attempts=0
until docker compose exec -T postgres \
  pg_isready -U "${POSTGRES_USER:-postgres}" -d "${POSTGRES_DB:-pos_db}" &>/dev/null; do
  local_attempts=$((local_attempts + 1))
  if [[ $local_attempts -gt 30 ]]; then
    docker_failure "PostgreSQL ไม่ตอบสนอง"
  fi
  sleep 1
done
ok "PostgreSQL ready"

info "รอ MinIO healthy..."
local_attempts=0
until curl -sf "http://localhost:${MINIO_API_PORT:-9000}/minio/health/live" &>/dev/null; do
  local_attempts=$((local_attempts + 1))
  if [[ $local_attempts -gt 30 ]]; then
    docker_failure "MinIO ไม่ตอบสนองหรือยังไม่ healthy"
  fi
  sleep 1
done
ok "MinIO ready"

docker compose ps postgres minio

# minio-client only initializes the bucket; it is not a runtime dependency.
# Do not let an unavailable/broken mc image take down PostgreSQL or MinIO.
MC_IMAGE=$(docker compose config --images | awk '/\/mc:/ { print; exit }')
if [[ -n "$MC_IMAGE" ]] && docker compose pull minio-client; then
  info "Starting optional MinIO bucket initializer (mc)..."
  if docker compose up -d minio-client; then
    ok "MinIO bucket initializer started"
  else
    warn "ข้าม MinIO bucket initializer (mc) — PostgreSQL และ MinIO ยังทำงานต่อ"
    warn "ตรวจสอบภายหลังด้วย: docker compose logs --tail=100 minio-client"
  fi
else
  warn "ไม่สามารถ pull image ${MC_IMAGE:-minio/mc} — ข้าม bucket initializer (mc)"
  warn "PostgreSQL และ MinIO ยังทำงานต่อ; ตรวจสอบ bucket initialization ตาม deployment policy"
fi

# ── 2.5 Restored-DB guard: empty schema_migrations ledger ─────────────────
# A DB restored from a dump has an EMPTY schema_migrations ledger, so the
# backend's auto-migrate re-runs init-db/*.sql in order and dies on 001_schema.sql
# ("cannot drop columns from view" — later migrations already extended the views).
# Fill the ledger for exactly the migrations the DB already reflects; genuinely
# new migration files stay unrecorded so the backend applies them on boot.
# No-op on a fresh DB (schema_migrations doesn't exist yet) or when the ledger
# already has entries.
# No-op on a fresh DB with no schema yet; on a DB whose Docker init scripts
# already created the core schema, bootstrap the ledger before preparing it.
if ! LEDGER_STATE=$(docker compose exec -T postgres \
  psql -U "${POSTGRES_USER:-postgres}" -d "${POSTGRES_DB:-pos_db}" \
  -tAc "SELECT CASE WHEN to_regclass('public.schema_migrations') IS NULL THEN 'missing' ELSE (SELECT count(*)::text FROM schema_migrations) END" 2>/dev/null | tr -d '[:space:]'); then
  die "อ่านสถานะ schema_migrations ไม่ได้ — ตรวจสอบ: docker compose logs --tail=100 postgres"
fi

if [ "$LEDGER_STATE" = "missing" ]; then
  if ! CORE_SCHEMA_READY=$(docker compose exec -T postgres \
    psql -U "${POSTGRES_USER:-postgres}" -d "${POSTGRES_DB:-pos_db}" \
    -tAc "SELECT CASE WHEN to_regclass('public.stores') IS NOT NULL AND to_regclass('public.documents') IS NOT NULL THEN 'yes' ELSE 'no' END" 2>/dev/null | tr -d '[:space:]'); then
    die "ตรวจสอบ core schema ไม่ได้ — ตรวจสอบ: docker compose logs --tail=100 postgres"
  fi
  if [ "$CORE_SCHEMA_READY" = "yes" ]; then
    info "พบ core schema แต่ไม่มี migration ledger — สร้าง schema_migrations..."
    docker compose exec -T postgres psql -U "${POSTGRES_USER:-postgres}" -d "${POSTGRES_DB:-pos_db}" \
      -c "CREATE TABLE IF NOT EXISTS schema_migrations (id SERIAL PRIMARY KEY, filename TEXT NOT NULL UNIQUE, applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW())" >/dev/null
    LEDGER_COUNT="0"
  else
    info "ยังไม่มี core schema — ให้ backend bootstrap migration เอง"
    LEDGER_COUNT=""
  fi
else
  LEDGER_COUNT="$LEDGER_STATE"
fi

LEDGER_HAS_BASE="no"
if [ "$LEDGER_STATE" != "missing" ]; then
  if ! LEDGER_HAS_BASE=$(docker compose exec -T postgres \
    psql -U "${POSTGRES_USER:-postgres}" -d "${POSTGRES_DB:-pos_db}" \
    -tAc "SELECT CASE WHEN EXISTS (SELECT 1 FROM schema_migrations WHERE filename = '001_schema.sql') THEN 'yes' ELSE 'no' END" 2>/dev/null | tr -d '[:space:]'); then
    die "อ่าน migration ledger ไม่ได้ — ตรวจสอบ: docker compose logs --tail=100 postgres"
  fi
fi

if [ "${LEDGER_COUNT:-x}" = "0" ] || [ "$LEDGER_HAS_BASE" != "yes" ]; then
  info "schema_migrations ว่าง (DB ถูก restore) — เตรียม ledger..."
  if command -v uv >/dev/null 2>&1; then
    uv run scripts/mark_migrations_applied.py
  else
    warn "ไม่พบ uv — ใช้ python3 แทน (สคริปต์เป็น stdlib)"
    python3 scripts/mark_migrations_applied.py
  fi
  if ! LEDGER_HAS_BASE=$(docker compose exec -T postgres \
    psql -U "${POSTGRES_USER:-postgres}" -d "${POSTGRES_DB:-pos_db}" \
    -tAc "SELECT CASE WHEN EXISTS (SELECT 1 FROM schema_migrations WHERE filename = '001_schema.sql') THEN 'yes' ELSE 'no' END" 2>/dev/null | tr -d '[:space:]'); then
    die "ตรวจสอบ migration ledger หลัง repair ไม่ได้"
  fi
  [ "$LEDGER_HAS_BASE" = "yes" ] || die "migration ledger repair ไม่สำเร็จ — 001_schema.sql ยังไม่ถูก mark; ไม่ start PM2"
  ok "migration ledger พร้อม — backend จะ apply เฉพาะ migration ใหม่"
fi

# ── 3. Build Go backend ──────────────────────────────────────────────────────
step "Build Go backend"
cd "$BACKEND_DIR"
info "Building binary..."
mkdir -p bin
go build -o bin/api cmd/api.go
ok "Binary: $BACKEND_DIR/bin/api"

# Document "Download PDF" renders via headless Chrome (chromedp) → the host needs a
# Chrome/Chromium/Edge binary. Without it, only that one button fails (print still works).
info "ตรวจ Chrome/Chromium สำหรับสร้าง PDF เอกสาร..."
CHROME_FOUND=""
if [ -n "${CHROME_PATH:-}" ] && [ -x "$CHROME_PATH" ]; then
  CHROME_FOUND="$CHROME_PATH (CHROME_PATH)"
else
  for c in google-chrome google-chrome-stable chromium chromium-browser chrome msedge; do
    if command -v "$c" >/dev/null 2>&1; then CHROME_FOUND="$(command -v "$c")"; break; fi
  done
  for p in \
    "/usr/bin/google-chrome" "/usr/bin/chromium" "/usr/bin/chromium-browser" \
    "/c/Program Files/Google/Chrome/Application/chrome.exe" \
    "/c/Program Files (x86)/Microsoft/Edge/Application/msedge.exe"; do
    [ -z "$CHROME_FOUND" ] && [ -x "$p" ] && CHROME_FOUND="$p"
  done
fi
if [ -n "$CHROME_FOUND" ]; then
  ok "พบ Chrome: $CHROME_FOUND"
else
  warn "ไม่พบ Chrome/Chromium บนเครื่องนี้ — ปุ่ม 'ดาวน์โหลด PDF' จะใช้ไม่ได้"
  warn "  แก้: ติดตั้ง chromium (Linux: sudo apt install -y chromium) หรือ set CHROME_PATH=/path/to/chrome"
  warn "  (ปุ่ม 'พิมพ์' ยังใช้งานได้ปกติ — มี fallback ฝั่งหน้าเว็บให้อัตโนมัติ)"
fi

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
if ! pm2 start ecosystem.config.js; then
  pm2 status
  pm2 logs pos-backend --lines 100 --nostream 2>/dev/null || true
  pm2 logs pos-frontend --lines 100 --nostream 2>/dev/null || true
  die "PM2 start failed — ตรวจสอบ logs ด้านบน"
fi
sleep 2
for app in pos-backend pos-frontend; do
  if ! pm2 show "$app" 2>/dev/null | grep -q "status.*online"; then
    pm2 status
    pm2 logs "$app" --lines 100 --nostream 2>/dev/null || true
    die "$app ไม่ได้อยู่ในสถานะ online — ตรวจสอบ logs ด้านบน"
  fi
done
ok "pos-backend และ pos-frontend เริ่มแล้ว"

# ── 7. Cloudflare Tunnel ─────────────────────────────────────────────────────
step "Cloudflare Tunnel"
info "Starting cloudflared via systemd..."
sudo systemctl start cloudflared

# รอสักครู่ให้ tunnel connect
sleep 4

if systemctl is-active --quiet cloudflared; then
  ok "Tunnel online"
else
  warn "Tunnel ยังไม่ทำงาน — ตรวจสอบด้วย: sudo journalctl -u cloudflared -f"
fi

# ── 8. Save PM2 process list ─────────────────────────────────────────────────
step "Save PM2 + Startup"
pm2 save
ok "pm2 save แล้วสำหรับ application services (pos-backend, pos-frontend)"

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
echo "    pm2 restart pos-backend — restart backend"
echo "    pm2 monit               — CPU/RAM realtime"
echo ""
echo "  Cloudflare Tunnel (systemd):"
echo "    systemctl status cloudflared"
echo "    sudo journalctl -u cloudflared -f"
echo ""
echo "  Stop → bash stop.sh"
echo -e "${GREEN}${BOLD}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${RESET}"
echo ""

pm2 status

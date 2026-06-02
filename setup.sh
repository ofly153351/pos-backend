#!/usr/bin/env bash
# =============================================================================
# POS System — Full Setup Script
# Supports: macOS (Homebrew) | Ubuntu/Debian | Raspberry Pi (arm64)
# Usage:
#   chmod +x setup.sh
#   ./setup.sh              # interactive — prompts for all secrets
#   ./setup.sh --tunnel     # also start Cloudflare tunnel at the end
#   ./setup.sh --tailscale  # also install & connect Tailscale VPN
#   ./setup.sh --help
# =============================================================================
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# support running from inside pos-backend/ OR from the parent directory
if [[ -f "$SCRIPT_DIR/go.mod" ]]; then
  BACKEND_DIR="$SCRIPT_DIR"
  FRONTEND_DIR="$(dirname "$SCRIPT_DIR")/pos-frontend"
else
  BACKEND_DIR="$SCRIPT_DIR/pos-backend"
  FRONTEND_DIR="$SCRIPT_DIR/pos-frontend"
fi
LOG_FILE="$SCRIPT_DIR/setup.log"

# ── versions required ────────────────────────────────────────────────────────
REQUIRED_GO_MAJOR=1
REQUIRED_GO_MINOR=21
REQUIRED_NODE_MAJOR=20
# ─────────────────────────────────────────────────────────────────────────────

RED='\033[0;31m'; GREEN='\033[0;32m'; YELLOW='\033[1;33m'
CYAN='\033[0;36m'; BOLD='\033[1m'; RESET='\033[0m'

ok()   { echo -e "${GREEN}✔${RESET}  $*"; }
info() { echo -e "${CYAN}→${RESET}  $*"; }
warn() { echo -e "${YELLOW}⚠${RESET}  $*"; }
die()  { echo -e "${RED}✖${RESET}  $*" >&2; exit 1; }
step() { echo -e "\n${BOLD}━━━  $* ${RESET}"; }

WITH_TUNNEL=false
WITH_TAILSCALE=false
for arg in "$@"; do
  case $arg in
    --tunnel)    WITH_TUNNEL=true ;;
    --tailscale) WITH_TAILSCALE=true ;;
    --help)
      echo "Usage: $0 [--tunnel] [--tailscale] [--help]"
      echo "  --tunnel     Also configure & launch Cloudflare Tunnel"
      echo "  --tailscale  Also install & connect Tailscale VPN"
      exit 0 ;;
  esac
done

# ─────────────────────────────────────────────────────────────────────────────
# detect OS
# ─────────────────────────────────────────────────────────────────────────────
detect_os() {
  if [[ "$OSTYPE" == "darwin"* ]]; then
    OS=mac
  elif grep -qi "ubuntu\|debian\|raspbian" /etc/os-release 2>/dev/null; then
    OS=debian
  elif grep -qi "centos\|rhel\|fedora" /etc/os-release 2>/dev/null; then
    OS=rhel
  else
    warn "Unknown OS — will attempt generic install"
    OS=generic
  fi
  ok "OS detected: $OS"
}

# ─────────────────────────────────────────────────────────────────────────────
# package manager helpers
# ─────────────────────────────────────────────────────────────────────────────
brew_install() {
  if ! brew list "$1" &>/dev/null; then
    info "Installing $1 via Homebrew..."
    brew install "$1" >> "$LOG_FILE" 2>&1
    ok "Installed $1"
  else
    ok "$1 already installed"
  fi
}

apt_install() {
  if ! dpkg -l "$1" &>/dev/null 2>&1; then
    info "Installing $1..."
    sudo apt-get install -y "$1" >> "$LOG_FILE" 2>&1
    ok "Installed $1"
  else
    ok "$1 already installed"
  fi
}

# ─────────────────────────────────────────────────────────────────────────────
# install Homebrew (macOS)
# ─────────────────────────────────────────────────────────────────────────────
install_homebrew() {
  if [[ "$OS" != "mac" ]]; then return; fi
  if command -v brew &>/dev/null; then
    ok "Homebrew already installed"
    return
  fi
  info "Installing Homebrew..."
  /bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)"
  # add to PATH for Apple Silicon
  if [[ -f /opt/homebrew/bin/brew ]]; then
    eval "$(/opt/homebrew/bin/brew shellenv)"
  fi
  ok "Homebrew installed"
}

# ─────────────────────────────────────────────────────────────────────────────
# install Docker
# ─────────────────────────────────────────────────────────────────────────────
install_docker() {
  step "Docker"
  if command -v docker &>/dev/null && docker info &>/dev/null 2>&1; then
    ok "Docker already running"
    return
  fi

  if [[ "$OS" == "mac" ]]; then
    if ! command -v docker &>/dev/null; then
      warn "Docker Desktop not found. Installing via Homebrew cask..."
      brew install --cask docker >> "$LOG_FILE" 2>&1
      ok "Docker Desktop installed — please open it once to finish setup, then re-run this script."
      open /Applications/Docker.app
      echo -e "\n${YELLOW}Press Enter after Docker Desktop has fully started...${RESET}"
      read -r
    fi
  elif [[ "$OS" == "debian" ]]; then
    if ! command -v docker &>/dev/null; then
      info "Installing Docker Engine..."
      curl -fsSL https://get.docker.com | sudo sh >> "$LOG_FILE" 2>&1
      sudo usermod -aG docker "$USER"
      ok "Docker installed — you may need to log out and back in for group changes"
    fi
    sudo systemctl enable --now docker >> "$LOG_FILE" 2>&1
  fi

  if command -v docker &>/dev/null; then
    ok "Docker $(docker --version | awk '{print $3}' | tr -d ',')"
  else
    die "Docker installation failed. Check $LOG_FILE"
  fi
}

# ─────────────────────────────────────────────────────────────────────────────
# install Go
# ─────────────────────────────────────────────────────────────────────────────
install_go() {
  step "Go"
  local need_install=false

  if command -v go &>/dev/null; then
    local ver
    ver=$(go version | awk '{print $3}' | sed 's/go//')
    local major minor
    major=$(echo "$ver" | cut -d. -f1)
    minor=$(echo "$ver" | cut -d. -f2)
    if (( major > REQUIRED_GO_MAJOR || (major == REQUIRED_GO_MAJOR && minor >= REQUIRED_GO_MINOR) )); then
      ok "Go $ver (requirement ≥ $REQUIRED_GO_MAJOR.$REQUIRED_GO_MINOR satisfied)"
      return
    else
      warn "Go $ver is too old — installing newer version"
      need_install=true
    fi
  else
    need_install=true
  fi

  if [[ "$need_install" == true ]]; then
    if [[ "$OS" == "mac" ]]; then
      brew_install go
    elif [[ "$OS" == "debian" ]]; then
      # determine arch
      local arch
      arch=$(dpkg --print-architecture)
      local go_arch
      case $arch in
        amd64) go_arch="amd64" ;;
        arm64|aarch64) go_arch="arm64" ;;
        armhf) go_arch="armv6l" ;;
        *) die "Unsupported architecture: $arch" ;;
      esac
      local GO_VERSION="1.23.5"
      local tarball="go${GO_VERSION}.linux-${go_arch}.tar.gz"
      info "Downloading Go $GO_VERSION ($go_arch)..."
      curl -fsSL "https://go.dev/dl/$tarball" -o "/tmp/$tarball"
      sudo rm -rf /usr/local/go
      sudo tar -C /usr/local -xzf "/tmp/$tarball"
      rm "/tmp/$tarball"
      # add to PATH if not already
      if ! grep -q "/usr/local/go/bin" ~/.bashrc 2>/dev/null; then
        echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.bashrc
      fi
      export PATH=$PATH:/usr/local/go/bin
    fi
    ok "Go $(go version | awk '{print $3}')"
  fi
}

# ─────────────────────────────────────────────────────────────────────────────
# install Node.js (via nvm)
# ─────────────────────────────────────────────────────────────────────────────
install_node() {
  step "Node.js"

  # load nvm if exists
  export NVM_DIR="${NVM_DIR:-$HOME/.nvm}"
  [[ -s "$NVM_DIR/nvm.sh" ]] && source "$NVM_DIR/nvm.sh"

  if command -v node &>/dev/null; then
    local ver
    ver=$(node -e "process.stdout.write(process.versions.node)")
    local major
    major=$(echo "$ver" | cut -d. -f1)
    if (( major >= REQUIRED_NODE_MAJOR )); then
      ok "Node.js $ver (requirement ≥ $REQUIRED_NODE_MAJOR satisfied)"
      return
    fi
    warn "Node.js $ver is too old"
  fi

  if ! command -v nvm &>/dev/null; then
    info "Installing nvm..."
    curl -o- https://raw.githubusercontent.com/nvm-sh/nvm/v0.39.7/install.sh | bash >> "$LOG_FILE" 2>&1
    export NVM_DIR="$HOME/.nvm"
    source "$NVM_DIR/nvm.sh"
    ok "nvm installed"
  fi

  info "Installing Node.js LTS..."
  nvm install --lts >> "$LOG_FILE" 2>&1
  nvm use --lts >> "$LOG_FILE" 2>&1
  nvm alias default "lts/*" >> "$LOG_FILE" 2>&1
  ok "Node.js $(node --version) / npm $(npm --version)"
}

# ─────────────────────────────────────────────────────────────────────────────
# install Tailscale
# ─────────────────────────────────────────────────────────────────────────────
install_tailscale() {
  step "Tailscale"
  if command -v tailscale &>/dev/null; then
    ok "Tailscale $(tailscale version | head -1)"
    return
  fi

  if [[ "$OS" == "mac" ]]; then
    # prefer App Store build; fall back to Homebrew cask
    if ! brew list --cask tailscale &>/dev/null; then
      info "Installing Tailscale via Homebrew cask..."
      brew install --cask tailscale >> "$LOG_FILE" 2>&1
    fi
    ok "Tailscale installed — open the menu-bar app to sign in"
    return
  fi

  if [[ "$OS" == "debian" ]]; then
    info "Installing Tailscale via official script..."
    curl -fsSL https://tailscale.com/install.sh | sudo sh >> "$LOG_FILE" 2>&1
    sudo systemctl enable --now tailscaled >> "$LOG_FILE" 2>&1
    ok "Tailscale daemon started"
  elif [[ "$OS" == "rhel" ]]; then
    # Fedora / RHEL / CentOS
    local rel
    rel=$(rpm -E '%{rhel}' 2>/dev/null || echo "8")
    sudo dnf config-manager --add-repo \
      "https://pkgs.tailscale.com/stable/rhel/${rel}/tailscale.repo" >> "$LOG_FILE" 2>&1
    sudo dnf install -y tailscale >> "$LOG_FILE" 2>&1
    sudo systemctl enable --now tailscaled >> "$LOG_FILE" 2>&1
    ok "Tailscale daemon started"
  fi
}

connect_tailscale() {
  step "Tailscale — connect"

  # macOS: the GUI app handles auth; guide the user
  if [[ "$OS" == "mac" ]]; then
    warn "On macOS, open the Tailscale menu-bar icon and click 'Log in' to connect."
    warn "After connecting, your Tailscale IP will be shown in the app."
    return
  fi

  echo ""
  echo -e "${BOLD}Tailscale login options:${RESET}"
  echo "  1) Interactive (opens browser)"
  echo "  2) Auth key   (headless / server)"
  read -rp "  Choose [1/2]: " ts_mode
  ts_mode="${ts_mode:-1}"

  if [[ "$ts_mode" == "2" ]]; then
    read -rp "  Paste your Tailscale auth key (tskey-auth-...): " ts_key
    if [[ -n "$ts_key" ]]; then
      sudo tailscale up --authkey="$ts_key" --accept-routes >> "$LOG_FILE" 2>&1
    else
      warn "No key provided — running interactive login instead"
      sudo tailscale up
    fi
  else
    sudo tailscale up
  fi

  # print Tailscale IP
  local ts_ip
  ts_ip=$(tailscale ip -4 2>/dev/null || echo "unknown")
  ok "Tailscale connected — IP: $ts_ip"

  # save Tailscale IP for start.sh summary
  echo "$ts_ip" > "$SCRIPT_DIR/.tailscale_ip"
}

# ─────────────────────────────────────────────────────────────────────────────
# install cloudflared
# ─────────────────────────────────────────────────────────────────────────────
install_cloudflared() {
  step "cloudflared"
  if command -v cloudflared &>/dev/null; then
    ok "cloudflared $(cloudflared --version 2>&1 | head -1)"
    return
  fi

  if [[ "$OS" == "mac" ]]; then
    brew_install cloudflared
  elif [[ "$OS" == "debian" ]]; then
    local arch
    arch=$(dpkg --print-architecture)
    local pkg_arch
    case $arch in
      amd64)  pkg_arch="amd64" ;;
      arm64)  pkg_arch="arm64" ;;
      armhf)  pkg_arch="arm" ;;
      *) die "Unsupported arch for cloudflared: $arch" ;;
    esac
    curl -fsSL "https://github.com/cloudflare/cloudflared/releases/latest/download/cloudflared-linux-${pkg_arch}.deb" \
      -o /tmp/cloudflared.deb
    sudo dpkg -i /tmp/cloudflared.deb >> "$LOG_FILE" 2>&1
    rm /tmp/cloudflared.deb
  fi
  ok "cloudflared installed"
}

# ─────────────────────────────────────────────────────────────────────────────
# install miscellaneous tools
# ─────────────────────────────────────────────────────────────────────────────
install_misc() {
  step "Misc tools (git, curl, jq, make)"
  if [[ "$OS" == "mac" ]]; then
    brew_install git
    brew_install curl
    brew_install jq
  elif [[ "$OS" == "debian" ]]; then
    sudo apt-get update -qq >> "$LOG_FILE" 2>&1
    for pkg in git curl jq make build-essential; do apt_install "$pkg"; done
  fi
}

# ─────────────────────────────────────────────────────────────────────────────
# configure backend .env
# ─────────────────────────────────────────────────────────────────────────────
setup_backend_env() {
  step "Backend .env"
  local env_file="$BACKEND_DIR/.env"

  if [[ -f "$env_file" ]]; then
    warn ".env already exists — skipping (delete it to reconfigure)"
    return
  fi

  echo ""
  echo -e "${BOLD}Configure backend secrets (press Enter to keep default):${RESET}"

  read -rp "  DB password        [change-me]      : " db_pass;    db_pass="${db_pass:-change-me}"
  read -rp "  JWT secret key     [change-this-secret]: " jwt_key; jwt_key="${jwt_key:-change-this-secret}"
  read -rp "  MinIO password     [change-me]      : " minio_pass; minio_pass="${minio_pass:-change-me}"
  read -rp "  Backend port       [8080]            : " be_port;   be_port="${be_port:-8080}"
  read -rp "  CORS origin (frontend URL) [http://localhost:3000]: " cors; cors="${cors:-http://localhost:3000}"

  cat > "$env_file" <<EOF
APP_HOST=0.0.0.0
APP_PORT=${be_port}
APP_NAME=pos-backend
APP_TOKEN_KEY=${jwt_key}
APP_CORS_ALLOW_ORIGINS=${cors}
APP_AUTO_MIGRATE=true
APP_MIGRATIONS_DIR=init-db
APP_UPLOAD_DIR=storage

POSTGRES_HOST=127.0.0.1
POSTGRES_DB=pos_db
POSTGRES_USER=postgres
POSTGRES_PASSWORD=${db_pass}
POSTGRES_PORT=5432
POSTGRES_SSLMODE=disable

MINIO_ROOT_USER=minioadmin
MINIO_ROOT_PASSWORD=${minio_pass}
MINIO_ACCESS_KEY=
MINIO_SECRET_KEY=
MINIO_BUCKET_NAME=pos-assets
MINIO_API_PORT=9000
MINIO_CONSOLE_PORT=9001
MINIO_BROWSER_REDIRECT_URL=
MINIO_SERVER_URL=

PDF_TTF_PATH=""
EOF

  ok "Backend .env created"
}

# ─────────────────────────────────────────────────────────────────────────────
# configure frontend .env.local
# ─────────────────────────────────────────────────────────────────────────────
setup_frontend_env() {
  step "Frontend .env.local"
  local env_file="$FRONTEND_DIR/.env.local"

  if [[ -f "$env_file" ]]; then
    warn ".env.local already exists — skipping"
    return
  fi

  # read backend port from backend .env
  local be_port
  be_port=$(grep "^APP_PORT=" "$BACKEND_DIR/.env" 2>/dev/null | cut -d= -f2 || echo "8080")

  read -rp "  Frontend port [3000]: " fe_port; fe_port="${fe_port:-3000}"

  cat > "$env_file" <<EOF
PORT=${fe_port}
NEXT_PUBLIC_API_BASE_URL=http://localhost:${be_port}
API_BASE_URL=http://localhost:${be_port}
EOF

  ok "Frontend .env.local created"
}

# ─────────────────────────────────────────────────────────────────────────────
# start Docker services (postgres + minio)
# ─────────────────────────────────────────────────────────────────────────────
start_docker_services() {
  step "Docker services (PostgreSQL + MinIO)"
  cd "$BACKEND_DIR"

  # export vars so docker compose can read them
  set -a; source .env; set +a

  info "Starting containers..."
  docker compose up -d postgres minio minio-client >> "$LOG_FILE" 2>&1

  info "Waiting for PostgreSQL to be healthy..."
  local attempts=0
  until docker compose exec -T postgres pg_isready -U "$POSTGRES_USER" -d "$POSTGRES_DB" &>/dev/null; do
    attempts=$((attempts+1))
    if (( attempts > 30 )); then
      die "PostgreSQL did not become healthy after 30 s. Check: docker compose logs postgres"
    fi
    sleep 1
  done

  ok "PostgreSQL ready"
  ok "MinIO ready"
}

# ─────────────────────────────────────────────────────────────────────────────
# install backend dependencies & build
# ─────────────────────────────────────────────────────────────────────────────
build_backend() {
  step "Backend (Go modules + build check)"
  cd "$BACKEND_DIR"
  info "Downloading Go modules..."
  go mod download >> "$LOG_FILE" 2>&1
  info "Verifying build..."
  go build ./... >> "$LOG_FILE" 2>&1
  ok "Backend builds clean"
}

# ─────────────────────────────────────────────────────────────────────────────
# install frontend dependencies
# ─────────────────────────────────────────────────────────────────────────────
install_frontend_deps() {
  step "Frontend (npm install)"
  cd "$FRONTEND_DIR"
  info "Installing npm packages..."
  npm install >> "$LOG_FILE" 2>&1
  ok "npm packages installed"
}

# ─────────────────────────────────────────────────────────────────────────────
# Cloudflare Tunnel setup
# ─────────────────────────────────────────────────────────────────────────────
setup_cloudflare_tunnel() {
  step "Cloudflare Tunnel"

  # check login
  if ! cloudflared tunnel list &>/dev/null 2>&1; then
    info "Logging into Cloudflare (browser will open)..."
    cloudflared tunnel login
  fi

  local tunnel_name="pos-system"

  # create tunnel if it doesn't exist
  if ! cloudflared tunnel list 2>/dev/null | grep -q "$tunnel_name"; then
    info "Creating tunnel '$tunnel_name'..."
    cloudflared tunnel create "$tunnel_name"
  else
    ok "Tunnel '$tunnel_name' already exists"
  fi

  # get tunnel ID
  local tunnel_id
  tunnel_id=$(cloudflared tunnel list 2>/dev/null | awk "/$tunnel_name/ {print \$1}")

  # read frontend port
  local fe_port
  fe_port=$(grep "^PORT=" "$FRONTEND_DIR/.env.local" 2>/dev/null | cut -d= -f2 || echo "3000")
  local be_port
  be_port=$(grep "^APP_PORT=" "$BACKEND_DIR/.env" 2>/dev/null | cut -d= -f2 || echo "8080")

  echo ""
  echo -e "${BOLD}Cloudflare Tunnel DNS setup:${RESET}"
  read -rp "  Your Cloudflare zone (domain), e.g. example.com : " cf_zone
  read -rp "  Subdomain for frontend (e.g. pos)              : " cf_fe_sub
  read -rp "  Subdomain for backend API (e.g. pos-api)       : " cf_be_sub

  local cf_dir="$HOME/.cloudflared"
  local config_file="$cf_dir/pos-config.yml"
  mkdir -p "$cf_dir"

  cat > "$config_file" <<EOF
tunnel: ${tunnel_id}
credentials-file: ${cf_dir}/${tunnel_id}.json

ingress:
  - hostname: ${cf_fe_sub}.${cf_zone}
    service: http://localhost:${fe_port}
  - hostname: ${cf_be_sub}.${cf_zone}
    service: http://localhost:${be_port}
  - service: http_status:404
EOF

  ok "Tunnel config written to $config_file"

  # create DNS records
  info "Creating DNS CNAME records..."
  cloudflared tunnel route dns "$tunnel_name" "${cf_fe_sub}.${cf_zone}" 2>/dev/null || warn "DNS record may already exist for ${cf_fe_sub}.${cf_zone}"
  cloudflared tunnel route dns "$tunnel_name" "${cf_be_sub}.${cf_zone}" 2>/dev/null || warn "DNS record may already exist for ${cf_be_sub}.${cf_zone}"

  # update frontend .env.local to point to public backend URL
  local env_file="$FRONTEND_DIR/.env.local"
  sed -i.bak "s|NEXT_PUBLIC_API_BASE_URL=.*|NEXT_PUBLIC_API_BASE_URL=https://${cf_be_sub}.${cf_zone}|" "$env_file"
  sed -i.bak "s|API_BASE_URL=.*|API_BASE_URL=http://localhost:${be_port}|" "$env_file"
  rm -f "${env_file}.bak"

  # update backend CORS
  local be_env="$BACKEND_DIR/.env"
  local existing_cors
  existing_cors=$(grep "^APP_CORS_ALLOW_ORIGINS=" "$be_env" | cut -d= -f2)
  sed -i.bak "s|^APP_CORS_ALLOW_ORIGINS=.*|APP_CORS_ALLOW_ORIGINS=${existing_cors},https://${cf_fe_sub}.${cf_zone}|" "$be_env"
  rm -f "${be_env}.bak"

  ok "CORS updated to allow https://${cf_fe_sub}.${cf_zone}"

  # save tunnel config path for start script
  echo "$config_file" > "$SCRIPT_DIR/.tunnel_config"
  echo "${cf_fe_sub}.${cf_zone}" >> "$SCRIPT_DIR/.tunnel_config"
  echo "${cf_be_sub}.${cf_zone}" >> "$SCRIPT_DIR/.tunnel_config"
}

# ─────────────────────────────────────────────────────────────────────────────
# write start.sh helper
# ─────────────────────────────────────────────────────────────────────────────
write_start_script() {
  step "Writing start.sh"
  local fe_port be_port
  fe_port=$(grep "^PORT=" "$FRONTEND_DIR/.env.local" 2>/dev/null | cut -d= -f2 || echo "3000")
  be_port=$(grep "^APP_PORT=" "$BACKEND_DIR/.env" 2>/dev/null | cut -d= -f2 || echo "8080")

  local tunnel_section=""
  if [[ "$WITH_TUNNEL" == true && -f "$SCRIPT_DIR/.tunnel_config" ]]; then
    local tunnel_cfg
    tunnel_cfg=$(head -1 "$SCRIPT_DIR/.tunnel_config")
    local fe_url be_url
    fe_url=$(sed -n '2p' "$SCRIPT_DIR/.tunnel_config")
    be_url=$(sed -n '3p' "$SCRIPT_DIR/.tunnel_config")
    tunnel_section="
# ── Cloudflare Tunnel ───────────────────────────────────────────────
echo '→  Starting Cloudflare Tunnel...'
cloudflared tunnel --config ${tunnel_cfg} run &
CF_PID=\$!
echo \"✔  Tunnel started (PID \$CF_PID)\"
echo \"    Frontend : https://${fe_url}\"
echo \"    Backend  : https://${be_url}\"
"
  fi

  cat > "$SCRIPT_DIR/start.sh" <<STARTEOF
#!/usr/bin/env bash
# POS System — start all services
set -euo pipefail
SCRIPT_DIR="\$(cd "\$(dirname "\${BASH_SOURCE[0]}")" && pwd)"
BACKEND_DIR="\$SCRIPT_DIR/pos-backend"
FRONTEND_DIR="\$SCRIPT_DIR/pos-frontend"

# load nvm if available
export NVM_DIR="\${NVM_DIR:-\$HOME/.nvm}"
[[ -s "\$NVM_DIR/nvm.sh" ]] && source "\$NVM_DIR/nvm.sh"

echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "  POS System — starting up"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""

# ── Docker services ──────────────────────────────────────────────────
echo '→  Starting Docker services (PostgreSQL + MinIO)...'
cd "\$BACKEND_DIR"
set -a; source .env; set +a
docker compose up -d postgres minio minio-client
until docker compose exec -T postgres pg_isready -U "\$POSTGRES_USER" -d "\$POSTGRES_DB" &>/dev/null; do sleep 1; done
echo '✔  PostgreSQL ready'

# ── Backend ──────────────────────────────────────────────────────────
echo '→  Starting Go backend...'
cd "\$BACKEND_DIR"
go run cmd/api.go > /tmp/pos-backend.log 2>&1 &
BE_PID=\$!
sleep 3
if ! kill -0 "\$BE_PID" 2>/dev/null; then
  echo '✖  Backend failed to start. See /tmp/pos-backend.log'
  exit 1
fi
echo "✔  Backend running (PID \$BE_PID) — http://localhost:${be_port}"

# ── Frontend ─────────────────────────────────────────────────────────
echo '→  Starting Next.js frontend...'
cd "\$FRONTEND_DIR"
npm run dev > /tmp/pos-frontend.log 2>&1 &
FE_PID=\$!
sleep 5
echo "✔  Frontend running (PID \$FE_PID) — http://localhost:${fe_port}"
${tunnel_section}
echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "  All services started!"
echo "  Backend  : http://localhost:${be_port}"
echo "  Frontend : http://localhost:${fe_port}"
echo ""
echo "  Logs:"
echo "    Backend  → /tmp/pos-backend.log"
echo "    Frontend → /tmp/pos-frontend.log"
echo ""
echo "  To stop all: ./stop.sh"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

# keep script alive so Ctrl+C stops everything
wait
STARTEOF

  cat > "$SCRIPT_DIR/stop.sh" <<'STOPEOF'
#!/usr/bin/env bash
echo "Stopping POS services..."
pkill -f "go run cmd/api.go" 2>/dev/null && echo "✔  Backend stopped" || true
pkill -f "next dev"           2>/dev/null && echo "✔  Frontend stopped" || true
pkill -f "cloudflared tunnel" 2>/dev/null && echo "✔  Tunnel stopped" || true
cd "$(dirname "$0")/pos-backend" && docker compose down 2>/dev/null && echo "✔  Docker services stopped" || true
STOPEOF

  chmod +x "$SCRIPT_DIR/start.sh" "$SCRIPT_DIR/stop.sh"
  ok "start.sh and stop.sh created"
}

# ─────────────────────────────────────────────────────────────────────────────
# summary
# ─────────────────────────────────────────────────────────────────────────────
print_summary() {
  local fe_port be_port
  fe_port=$(grep "^PORT=" "$FRONTEND_DIR/.env.local" 2>/dev/null | cut -d= -f2 || echo "3000")
  be_port=$(grep "^APP_PORT=" "$BACKEND_DIR/.env" 2>/dev/null | cut -d= -f2 || echo "8080")

  echo ""
  echo -e "${GREEN}${BOLD}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${RESET}"
  echo -e "${GREEN}${BOLD}  Setup complete!${RESET}"
  echo -e "${GREEN}${BOLD}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${RESET}"
  echo ""
  echo "  Default login:"
  echo "    owner@pos.dev   / Owner1234!"
  echo "    manager@pos.dev / Manager1234!"
  echo "    cashier@pos.dev / Cashier1234!"
  echo ""
  echo "  Start everything:  ./start.sh"
  echo "  Stop everything:   ./stop.sh"
  echo ""
  echo "  Local URLs:"
  echo "    Frontend  →  http://localhost:${fe_port}"
  echo "    Backend   →  http://localhost:${be_port}"
  echo "    MinIO     →  http://localhost:9001"
  echo ""
  if [[ "$WITH_TUNNEL" == true && -f "$SCRIPT_DIR/.tunnel_config" ]]; then
    local fe_url be_url
    fe_url=$(sed -n '2p' "$SCRIPT_DIR/.tunnel_config")
    be_url=$(sed -n '3p' "$SCRIPT_DIR/.tunnel_config")
    echo "  Public URLs (Cloudflare Tunnel):"
    echo "    Frontend  →  https://${fe_url}"
    echo "    Backend   →  https://${be_url}"
    echo ""
  fi
  if [[ "$WITH_TAILSCALE" == true && -f "$SCRIPT_DIR/.tailscale_ip" ]]; then
    local ts_ip
    ts_ip=$(cat "$SCRIPT_DIR/.tailscale_ip")
    echo "  Tailscale VPN:"
    echo "    This machine IP  →  ${ts_ip}"
    echo "    Frontend         →  http://${ts_ip}:${fe_port}"
    echo "    Backend          →  http://${ts_ip}:${be_port}"
    echo ""
  fi
  echo "  Logs: $LOG_FILE"
  echo ""
}

# ─────────────────────────────────────────────────────────────────────────────
# main
# ─────────────────────────────────────────────────────────────────────────────
main() {
  echo -e "${BOLD}"
  echo "  ██████╗  ██████╗ ███████╗"
  echo "  ██╔══██╗██╔═══██╗██╔════╝"
  echo "  ██████╔╝██║   ██║███████╗"
  echo "  ██╔═══╝ ██║   ██║╚════██║"
  echo "  ██║     ╚██████╔╝███████║"
  echo "  ╚═╝      ╚═════╝ ╚══════╝  Setup"
  echo -e "${RESET}"
  echo "  Log file: $LOG_FILE"
  echo "" > "$LOG_FILE"

  # verify project directories exist
  [[ -d "$BACKEND_DIR" ]]  || die "pos-backend directory not found at $BACKEND_DIR"
  [[ -d "$FRONTEND_DIR" ]] || die "pos-frontend directory not found at $FRONTEND_DIR"

  detect_os
  install_homebrew
  install_misc
  install_docker
  install_go
  install_node
  [[ "$WITH_TUNNEL" == true ]]    && install_cloudflared
  [[ "$WITH_TAILSCALE" == true ]] && install_tailscale
  setup_backend_env
  setup_frontend_env
  start_docker_services
  build_backend
  install_frontend_deps
  [[ "$WITH_TUNNEL" == true ]]    && setup_cloudflare_tunnel
  [[ "$WITH_TAILSCALE" == true ]] && connect_tailscale
  write_start_script
  print_summary
}

main

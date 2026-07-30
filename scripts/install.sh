#!/usr/bin/env bash
# One-command installer for a Debian/Ubuntu-based host (e.g. a Proxmox LXC).
#
# Usage (run as root inside the LXC):
#   curl -fsSL https://raw.githubusercontent.com/eiyanproject/kakeibo/master/scripts/install.sh | bash
#
# Re-running this script updates an existing install (git pull, rebuild, restart).
# Override the DB password with: KAKEIBO_DB_PASSWORD=... before the command above.
set -euo pipefail

REPO_URL="https://github.com/eiyanproject/kakeibo.git"
INSTALL_DIR="/opt/kakeibo"
GO_VERSION="1.25.7"
SERVICE_USER="kakeibo"
DB_NAME="kakeibo"
DB_USER="kakeibo"
DB_PASS="${KAKEIBO_DB_PASSWORD:-kakeibo}"

if [ "$(id -u)" -ne 0 ]; then
  echo "This installer must be run as root (e.g. inside the LXC as root, or with sudo)." >&2
  exit 1
fi

echo "==> Installing system dependencies (git, postgresql, poppler-utils)"
export DEBIAN_FRONTEND=noninteractive
apt-get update -qq
apt-get install -y -qq git curl ca-certificates sudo postgresql poppler-utils >/dev/null

case "$(dpkg --print-architecture)" in
  amd64) GOARCH=amd64 ;;
  arm64) GOARCH=arm64 ;;
  *) echo "Unsupported architecture: $(dpkg --print-architecture)" >&2; exit 1 ;;
esac

if ! command -v go >/dev/null 2>&1 || [ "$(go version | awk '{print $3}')" != "go${GO_VERSION}" ]; then
  echo "==> Installing Go ${GO_VERSION}"
  curl -fsSL "https://go.dev/dl/go${GO_VERSION}.linux-${GOARCH}.tar.gz" -o /tmp/go.tar.gz
  rm -rf /usr/local/go
  tar -C /usr/local -xzf /tmp/go.tar.gz
  rm /tmp/go.tar.gz
  ln -sf /usr/local/go/bin/go /usr/local/bin/go
  ln -sf /usr/local/go/bin/gofmt /usr/local/bin/gofmt
fi

echo "==> Fetching kakeibo"
if [ -d "$INSTALL_DIR/.git" ]; then
  git -C "$INSTALL_DIR" pull --ff-only
else
  git clone --depth 1 "$REPO_URL" "$INSTALL_DIR"
fi

echo "==> Setting up PostgreSQL role and database"
sudo -u postgres psql -v ON_ERROR_STOP=1 -q <<SQL
DO \$\$
BEGIN
   IF NOT EXISTS (SELECT FROM pg_roles WHERE rolname = '${DB_USER}') THEN
      CREATE ROLE ${DB_USER} LOGIN PASSWORD '${DB_PASS}';
   END IF;
END
\$\$;
SQL
sudo -u postgres psql -tAc "SELECT 1 FROM pg_database WHERE datname='${DB_NAME}'" | grep -q 1 \
  || sudo -u postgres createdb -O "${DB_USER}" "${DB_NAME}"

echo "==> Building kakeibo"
cd "$INSTALL_DIR"
/usr/local/go/bin/go build -o kakeibo ./cmd/server

if [ ! -f "$INSTALL_DIR/.env" ]; then
  echo "==> Writing .env"
  cat > "$INSTALL_DIR/.env" <<EOF
DATABASE_URL=postgres://${DB_USER}:${DB_PASS}@localhost:5432/${DB_NAME}?sslmode=disable
ADDR=:8080
SESSION_SECURE=false
EOF
fi

echo "==> Installing systemd service"
id -u "$SERVICE_USER" >/dev/null 2>&1 || useradd --system --no-create-home --shell /usr/sbin/nologin "$SERVICE_USER"
chown -R "$SERVICE_USER":"$SERVICE_USER" "$INSTALL_DIR"
cp "$INSTALL_DIR/deploy/kakeibo.service" /etc/systemd/system/kakeibo.service
systemctl daemon-reload
systemctl enable --now kakeibo
systemctl restart kakeibo

IP=$(hostname -I | awk '{print $1}')
echo ""
echo "==> Done. Kakeibo is running at http://${IP}:8080"
echo "    First visit will prompt you to create an admin account."
echo "    Put a reverse proxy in front for HTTPS and set SESSION_SECURE=true in ${INSTALL_DIR}/.env if you expose this beyond your LAN."

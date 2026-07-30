# Kakeibo (家計簿)

A self-hosted personal finance tracker: import bank CSV/PDF statements and
auto-categorize spending. Go backend, PostgreSQL, server-rendered HTML with htmx —
no JS build step, works fine from a phone browser.

## Local development

1. Start Postgres:
   ```
   docker compose up -d
   ```
2. Copy `.env.example` to `.env` and adjust if needed.
3. Run the server (it applies DB migrations automatically on startup):
   ```
   go run ./cmd/server
   ```
4. Open http://localhost:8080 — first visit prompts you to create an admin account.

## Importing a statement

Go to **Import**, pick the account, upload a CSV or PDF statement, then map which
columns hold the date / description / amount. The mapping is remembered per account,
so re-importing next month's statement is a one-click "Preview" → "Commit".
Re-uploading the same file is safe — duplicate rows (same account, date, merchant,
amount) are skipped.

PDF import currently targets Yucho Bank ("ゆうちょ銀行") meisai-style credit card
statements (`ご利用明細書`) and requires `pdftotext` (from `poppler-utils`) to be
installed on the host — `pacman -S poppler` / `apt install poppler-utils`. Files are
read strictly by content (a real `%PDF-` header), not by filename.

## Categorization

**Categories** page lets you add categories and "rules": if a transaction's merchant
name contains some text (case-insensitive), it's auto-assigned to that category on
import. You can also fix a category inline from the Transactions page — check
"create rule from this" (via the merchant field) to turn that fix into a future rule.

## Accounts

Each account is `checking`, `credit`, `cash`, `investment`, or `other` (a display
label). Accounts fed by CSV/PDF imports get their balance from opening balance + all
transactions. Accounts without a transaction feed (e.g. investments) can instead get
periodic manual **balance snapshots** from the account detail page.

## Deploying to a Proxmox LXC

One command, run as root inside a Debian/Ubuntu-based LXC (installs Go, PostgreSQL,
poppler-utils, builds the binary, sets up the database, and installs+starts the
systemd service):

```bash
curl -fsSL https://raw.githubusercontent.com/eiyanproject/kakeibo/master/scripts/install.sh | bash
```

Re-running the same command later updates an existing install (`git pull`, rebuild,
restart). Override the generated DB password with `KAKEIBO_DB_PASSWORD=...` prefixed
before the command above.

To reach it from your phone: either connect via Tailscale/WireGuard to the LXC's LAN,
or put a reverse proxy (Caddy is simplest — automatic HTTPS) in front and set
`SESSION_SECURE=true` in `/opt/kakeibo/.env` so session cookies require HTTPS.

<details>
<summary>Manual install (if you'd rather not pipe a script into bash)</summary>

1. Build a Linux binary (cross-compile from Windows):
   ```
   $env:GOOS="linux"; $env:GOARCH="amd64"; go build -o kakeibo ./cmd/server
   ```
2. Copy `kakeibo` (the binary — templates/static assets are embedded in it) and `.env`
   to `/opt/kakeibo` on the LXC.
3. Run Postgres in the LXC (or a separate LXC/container) — `apt install postgresql`
   or run the same `docker-compose.yml`.
4. Install the systemd unit:
   ```
   sudo cp deploy/kakeibo.service /etc/systemd/system/
   sudo systemctl daemon-reload
   sudo systemctl enable --now kakeibo
   ```
</details>

## Backup & migration

Postgres's own tools handle this — nothing app-specific needed:

```bash
# Backup
DATABASE_URL=postgres://kakeibo:kakeibo@localhost:5432/kakeibo pg_dump "$DATABASE_URL" -F c -f kakeibo.dump
# or: ./scripts/backup.sh

# Restore (into a fresh empty database)
pg_restore -d "$DATABASE_URL" --clean --if-exists kakeibo.dump
```

To migrate to the homelab: `pg_dump` on the source, copy the `.dump` file over, then
`pg_restore` into the LXC's Postgres. The app's schema migrations run automatically
on next startup if the schema version there is behind.

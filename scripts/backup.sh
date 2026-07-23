#!/usr/bin/env bash
# Usage: DATABASE_URL=postgres://... ./scripts/backup.sh [output_dir]
set -euo pipefail
STAMP=$(date +%Y%m%d_%H%M%S)
OUT_DIR="${1:-./backups}"
mkdir -p "$OUT_DIR"
pg_dump "$DATABASE_URL" -F c -f "$OUT_DIR/kakeibo_${STAMP}.dump"
echo "Backup written to $OUT_DIR/kakeibo_${STAMP}.dump"

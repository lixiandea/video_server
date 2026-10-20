#!/usr/bin/env bash
set -euo pipefail

APP_DIR="${APP_DIR:-/opt/video-server}"
ENV_FILE="${ENV_FILE:-$APP_DIR/.env.production}"
SECRETS_DIR="${SECRETS_DIR:-$APP_DIR/secrets}"
BACKUP_DIR="${BACKUP_DIR:-/var/backups/video-server}"
RETENTION_DAYS="${BACKUP_RETENTION_DAYS:-30}"

set -a
source "$ENV_FILE"
set +a

mkdir -p "$BACKUP_DIR"
timestamp="$(date -u +%Y%m%dT%H%M%SZ)"
target="$BACKUP_DIR/video-server-$timestamp.sql.gz"

MYSQL_PWD="$(cat "$SECRETS_DIR/db_password")" mysqldump \
  --single-transaction \
  --quick \
  --routines \
  --triggers \
  --ssl-ca="$SECRETS_DIR/db_ca.pem" \
  -h "$DB_HOST" -P "$DB_PORT" -u "$DB_USER" "$DB_NAME" \
  | gzip -9 > "$target"

find "$BACKUP_DIR" -type f -name 'video-server-*.sql.gz' -mtime "+$RETENTION_DAYS" -delete

if [[ -n "${RCLONE_REMOTE:-}" ]] && command -v rclone >/dev/null 2>&1; then
  rclone copy "$target" "$RCLONE_REMOTE" --immutable
fi

echo "$target"

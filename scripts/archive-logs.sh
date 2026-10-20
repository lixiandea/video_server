#!/usr/bin/env bash
set -euo pipefail

APP_DIR="${APP_DIR:-/opt/video-server}"
ENV_FILE="${ENV_FILE:-$APP_DIR/.env.production}"
OUTPUT_DIR="${LOG_ARCHIVE_DIR:-/var/log/video-server/archive}"
RETENTION_DAYS="${LOG_RETENTION_DAYS:-30}"

set -a
source "$ENV_FILE"
set +a

mkdir -p "$OUTPUT_DIR"
stamp="$(date -u +%Y%m%dT%H%M%SZ)"
archive="$OUTPUT_DIR/video-server-logs-$stamp.tar.gz"
snapshot="$(mktemp -d)"
trap 'rm -rf "$snapshot"' EXIT

for service in api-server scheduler transcode-worker frontend caddy clamav prometheus alertmanager grafana; do
  docker compose --env-file "$ENV_FILE" -f "$APP_DIR/docker-compose.prod.yml" \
    logs --no-color --timestamps --since 24h "$service" > "$snapshot/$service.log" 2>&1 || true
done

tar -C "$snapshot" -czf "$archive" .
find "$OUTPUT_DIR" -type f -name 'video-server-logs-*.tar.gz' -mtime "+$RETENTION_DAYS" -delete

if [[ -n "${RCLONE_LOG_REMOTE:-}" ]]; then
  rclone copy "$archive" "$RCLONE_LOG_REMOTE" --immutable
fi

echo "$archive"

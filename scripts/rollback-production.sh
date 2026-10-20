#!/usr/bin/env bash
set -euo pipefail

APP_DIR="${APP_DIR:-/opt/video-server}"
ENV_FILE="${ENV_FILE:-$APP_DIR/.env.production}"
COMPOSE_FILE="$APP_DIR/docker-compose.prod.yml"
IMAGE_TAG="${1:?usage: rollback-production.sh <image-tag>}"
export IMAGE_TAG

cd "$APP_DIR"
docker compose --env-file "$ENV_FILE" -f "$COMPOSE_FILE" pull \
  api-server scheduler transcode-worker frontend
docker compose --env-file "$ENV_FILE" -f "$COMPOSE_FILE" up -d --remove-orphans
printf '%s\n' "$IMAGE_TAG" > "$APP_DIR/.current_image_tag"
echo "Rolled application images back to $IMAGE_TAG; database migrations were not reversed."

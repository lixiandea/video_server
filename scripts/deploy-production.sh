#!/usr/bin/env bash
set -euo pipefail

APP_DIR="${APP_DIR:-/opt/video-server}"
ENV_FILE="${ENV_FILE:-$APP_DIR/.env.production}"
COMPOSE_FILE="$APP_DIR/docker-compose.prod.yml"
TAG_FILE="$APP_DIR/.current_image_tag"

cd "$APP_DIR"
requested_image_tag="${IMAGE_TAG:-}"
set -a
source "$ENV_FILE"
set +a

if [[ -f "${SECRETS_DIR:-$APP_DIR/secrets}/ghcr_token" && -n "${GHCR_USERNAME:-}" ]]; then
  cat "${SECRETS_DIR:-$APP_DIR/secrets}/ghcr_token" | \
    docker login ghcr.io -u "$GHCR_USERNAME" --password-stdin
fi

previous_tag="$(cat "$TAG_FILE" 2>/dev/null || true)"
export IMAGE_TAG="${requested_image_tag:-${IMAGE_TAG:?IMAGE_TAG is required}}"

compose() {
  docker compose --env-file "$ENV_FILE" -f "$COMPOSE_FILE" "$@"
}

rollback() {
  trap - ERR
  if [[ -n "$previous_tag" ]]; then
    echo "Deployment failed; rolling back to $previous_tag" >&2
    IMAGE_TAG="$previous_tag" compose up -d --remove-orphans api-server scheduler transcode-worker frontend
  fi
}

mkdir -p /var/backups/video-server
APP_DIR="$APP_DIR" ENV_FILE="$ENV_FILE" SECRETS_DIR="${SECRETS_DIR:-$APP_DIR/secrets}" "$APP_DIR/scripts/backup-database.sh"

compose --profile tools pull
compose run --rm migrate up

trap rollback ERR
compose up -d --remove-orphans

for _ in $(seq 1 30); do
  if compose exec -T api-server wget -q -O- http://127.0.0.1:8080/ready | grep -q READY; then
    break
  fi
  sleep 2
done

compose exec -T api-server wget -q -O- http://127.0.0.1:8080/ready | grep -q READY
for _ in $(seq 1 15); do
  if curl -fsS "https://$APP_DOMAIN/api/v1/recommendations?limit=1" >/dev/null; then
    break
  fi
  sleep 2
done
curl -fsS "https://$APP_DOMAIN/api/v1/recommendations?limit=1" >/dev/null

printf '%s\n' "$IMAGE_TAG" > "$TAG_FILE"
trap - ERR
echo "Deployed $IMAGE_TAG to https://$APP_DOMAIN"

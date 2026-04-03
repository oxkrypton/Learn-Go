#!/usr/bin/env bash

set -euo pipefail

APP_DIR="/opt/go-redis-deploy"
APP_NAME="hmdp-server"
APP_PATH="${APP_DIR}/${APP_NAME}"
BACKUP_DIR="${APP_DIR}/backup"
UPLOAD_PATH="${1:-${APP_DIR}/hmdp-server.new}"
SERVICE_NAME="hmdp-go"

mkdir -p "${BACKUP_DIR}"

if [ ! -f "${UPLOAD_PATH}" ]; then
  echo "uploaded binary not found: ${UPLOAD_PATH}"
  exit 1
fi

timestamp="$(date +%Y%m%d-%H%M%S)"
backup_path="${BACKUP_DIR}/${APP_NAME}.${timestamp}"

if [ -f "${APP_PATH}" ]; then
  cp "${APP_PATH}" "${backup_path}"
fi

chmod +x "${UPLOAD_PATH}"
mv "${UPLOAD_PATH}" "${APP_PATH}"
chmod +x "${APP_PATH}"

cd "${APP_DIR}"
if docker compose version >/dev/null 2>&1; then
  compose_cmd=(docker compose)
elif command -v docker-compose >/dev/null 2>&1; then
  compose_cmd=(docker-compose)
else
  echo "docker compose command not found"
  exit 1
fi

"${compose_cmd[@]}" restart "${SERVICE_NAME}"
sleep 5

if "${compose_cmd[@]}" ps "${SERVICE_NAME}" | grep -q "Up"; then
  echo "deploy success: ${SERVICE_NAME} restarted with new binary"
  exit 0
fi

echo "new version failed to start, rolling back"

if [ -f "${backup_path}" ]; then
  cp "${backup_path}" "${APP_PATH}"
  chmod +x "${APP_PATH}"
  "${compose_cmd[@]}" restart "${SERVICE_NAME}"
  sleep 5
fi

if "${compose_cmd[@]}" ps "${SERVICE_NAME}" | grep -q "Up"; then
  echo "rollback success: previous version restored"
  exit 1
fi

echo "rollback failed: ${SERVICE_NAME} is not running"
exit 1

#!/usr/bin/env bash

set -euo pipefail

APP_DIR="/opt/go-redis-deploy"
APP_NAME="hmdp-server"
APP_PATH="${APP_DIR}/${APP_NAME}"
BACKUP_DIR="${APP_DIR}/backup"
LOG_FILE="${APP_DIR}/server.log"
UPLOAD_PATH="${1:-/tmp/hmdp-server}"

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

pkill -f "${APP_PATH}" || true
sleep 2

cd "${APP_DIR}"
nohup "${APP_PATH}" > "${LOG_FILE}" 2>&1 &
sleep 5

if pgrep -f "${APP_PATH}" > /dev/null; then
  echo "deploy success: ${APP_PATH} is running"
  exit 0
fi

echo "new version failed to start, rolling back"

if [ -f "${backup_path}" ]; then
  cp "${backup_path}" "${APP_PATH}"
  chmod +x "${APP_PATH}"
  nohup "${APP_PATH}" > "${LOG_FILE}" 2>&1 &
  sleep 5
fi

if pgrep -f "${APP_PATH}" > /dev/null; then
  echo "rollback success: previous version restored"
  exit 1
fi

echo "rollback failed: service is not running"
exit 1

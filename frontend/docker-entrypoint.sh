#!/bin/sh
set -eu

PORT="${PORT:-80}"
BACKEND_HOST="${BACKEND_HOST:-backend}"
BACKEND_PORT="${BACKEND_PORT:-8080}"
BACKEND_UPSTREAM="${BACKEND_UPSTREAM:-${BACKEND_HOST}:${BACKEND_PORT}}"

export PORT BACKEND_HOST BACKEND_PORT BACKEND_UPSTREAM

echo "[nginx] PORT=${PORT} BACKEND_UPSTREAM=${BACKEND_UPSTREAM}"

envsubst '${PORT} ${BACKEND_HOST} ${BACKEND_PORT} ${BACKEND_UPSTREAM}' \
  < /etc/nginx/templates/app.conf.template \
  > /etc/nginx/conf.d/app.conf

exec nginx -g 'daemon off;'
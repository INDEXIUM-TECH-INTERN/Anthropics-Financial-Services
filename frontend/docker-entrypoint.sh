#!/bin/sh
set -eu

PORT="${PORT:-10000}"
BACKEND_HOST="${BACKEND_HOST:-testai-backend}"
BACKEND_PORT="${BACKEND_PORT:-10000}"
BACKEND_UPSTREAM="${BACKEND_UPSTREAM:-${BACKEND_HOST}:${BACKEND_PORT}}"

export PORT BACKEND_HOST BACKEND_PORT BACKEND_UPSTREAM

echo "[nginx] PORT=${PORT}"
echo "[nginx] BACKEND_UPSTREAM=${BACKEND_UPSTREAM}"

envsubst '$PORT $BACKEND_HOST $BACKEND_PORT $BACKEND_UPSTREAM' \
  < /etc/nginx/templates/app.conf.template \
  > /etc/nginx/conf.d/app.conf

if ! nginx -t 2>&1; then
  echo "[nginx] Generated config:"
  cat /etc/nginx/conf.d/app.conf
  exit 1
fi

exec nginx -g 'daemon off;'
#!/bin/sh
set -eu

PORT="${PORT:-10000}"
BACKEND_HOST="${BACKEND_HOST:-testai-backend}"
BACKEND_PORT="${BACKEND_PORT:-10000}"
DNS_RESOLVER="${DNS_RESOLVER:-10.220.0.1}"

export PORT BACKEND_HOST BACKEND_PORT DNS_RESOLVER

echo "[nginx] PORT=${PORT}"
echo "[nginx] BACKEND=${BACKEND_HOST}:${BACKEND_PORT}"

envsubst '$PORT $BACKEND_HOST $BACKEND_PORT $DNS_RESOLVER' \
  < /etc/nginx/templates/app.conf.template \
  > /etc/nginx/conf.d/app.conf

if ! nginx -t 2>&1; then
  echo "[nginx] Generated config:"
  cat /etc/nginx/conf.d/app.conf
  exit 1
fi

exec nginx -g 'daemon off;'
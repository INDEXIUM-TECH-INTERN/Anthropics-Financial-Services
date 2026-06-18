#!/bin/sh
set -eu

BACKEND_HOST="${BACKEND_HOST:-backend}"
BACKEND_PORT="${BACKEND_PORT:-8080}"

export BACKEND_HOST BACKEND_PORT

envsubst '${BACKEND_HOST} ${BACKEND_PORT}' \
  < /etc/nginx/templates/app.conf.template \
  > /etc/nginx/conf.d/app.conf

exec nginx -g 'daemon off;'
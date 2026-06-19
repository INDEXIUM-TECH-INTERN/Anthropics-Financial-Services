#!/bin/sh
set -eu

PORT="${PORT:-10000}"
API_BASE="${BACKEND_PUBLIC_URL:-}"

if [ -z "$API_BASE" ]; then
  API_BASE="http://${BACKEND_HOST:-testai-backend}:${BACKEND_PORT:-10000}"
fi

export PORT

echo "[nginx] PORT=${PORT}"
echo "[nginx] API_BASE=${API_BASE}"

envsubst '$PORT' \
  < /etc/nginx/templates/app.conf.template \
  > /etc/nginx/conf.d/app.conf

cat > /usr/share/nginx/html/config.js <<EOF
window.__INDEXIUM_API_BASE__ = "${API_BASE}";
EOF

if ! nginx -t 2>&1; then
  echo "[nginx] Generated config:"
  cat /etc/nginx/conf.d/app.conf
  exit 1
fi

exec nginx -g 'daemon off;'
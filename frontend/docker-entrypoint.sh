#!/bin/sh
set -eu

PORT="${PORT:-10000}"
BACKEND_UPSTREAM="${BACKEND_PUBLIC_URL:-}"

if [ -z "$BACKEND_UPSTREAM" ]; then
  BACKEND_UPSTREAM="http://${BACKEND_HOST:-testai-backend}:${BACKEND_PORT:-10000}"
fi

# Strip trailing slash
BACKEND_UPSTREAM="${BACKEND_UPSTREAM%/}"

# Host header for upstream (Render backend hostname)
case "$BACKEND_UPSTREAM" in
  http://*|https://*)
    PROXY_BACKEND_HOST="$(printf '%s' "$BACKEND_UPSTREAM" | sed -E 's#^[a-zA-Z]+://([^/:]+).*#\1#')"
    ;;
  *)
    PROXY_BACKEND_HOST="${BACKEND_HOST:-testai-backend}"
    ;;
esac

export PORT BACKEND_UPSTREAM PROXY_BACKEND_HOST

echo "[nginx] PORT=${PORT}"
echo "[nginx] BACKEND_UPSTREAM=${BACKEND_UPSTREAM}"
echo "[nginx] PROXY_BACKEND_HOST=${PROXY_BACKEND_HOST}"

envsubst '$PORT $BACKEND_UPSTREAM $PROXY_BACKEND_HOST' \
  < /etc/nginx/templates/app.conf.template \
  > /etc/nginx/conf.d/app.conf

# Same-origin API: nginx proxies /api/* → backend (see nginx.app.conf.template)
cat > /usr/share/nginx/html/config.js <<EOF
window.__INDEXIUM_API_BASE__ = "";
EOF

if ! nginx -t 2>&1; then
  echo "[nginx] Generated config:"
  cat /etc/nginx/conf.d/app.conf
  exit 1
fi

exec nginx -g 'daemon off;'
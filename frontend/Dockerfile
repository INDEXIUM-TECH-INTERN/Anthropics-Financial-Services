# ---- Build stage ----
FROM node:22-alpine AS builder

WORKDIR /app

# Copy package files first (layer caching)
COPY package.json package-lock.json* ./

# Use npm ci if lockfile exists, else npm install
RUN npm ci --ignore-scripts 2>/dev/null || npm install --ignore-scripts

# Copy source
COPY . .

# Build production bundle
RUN npm run build

# ---- Runtime stage (serve static files) ----
FROM nginx:1.27-alpine

# Remove default nginx config
RUN rm /etc/nginx/conf.d/default.conf

# Custom nginx config for SPA + API proxy
COPY <<'NGCONF' /etc/nginx/conf.d/app.conf
server {
    listen 80;
    server_name localhost;
    root /usr/share/nginx/html;
    index index.html;

    # Gzip
    gzip on;
    gzip_types text/plain text/css application/json application/javascript text/xml;
    gzip_min_length 256;

    # Frontend SPA — serve static files
    location / {
        try_files $uri $uri/ /index.html;
    }

    # Proxy API requests to backend
    location /api/ {
        proxy_pass http://backend:8080/api/;
        proxy_http_version 1.1;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;

        # SSE support (long-lived connections)
        proxy_set_header Connection '';
        proxy_buffering off;
        proxy_cache off;
        chunked_transfer_encoding off;
        proxy_read_timeout 3600s;
    }

    # WebSocket proxy (nếu cần)
    location /ws {
        proxy_pass http://backend:8080;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
        proxy_read_timeout 3600s;
        proxy_buffering off;
    }
}
NGCONF

# Copy built assets from builder
COPY --from=builder /app/dist /usr/share/nginx/html

EXPOSE 80

CMD ["nginx", "-g", "daemon off;"]

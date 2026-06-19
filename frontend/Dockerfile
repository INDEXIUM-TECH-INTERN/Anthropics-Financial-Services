# ---- Build stage ----
FROM node:22-alpine AS builder

WORKDIR /app

COPY package.json package-lock.json ./
RUN npm ci --ignore-scripts --no-audit --no-fund

COPY . .

# Vite-only build: faster and lower memory use on Render Free tier
ENV NODE_OPTIONS=--max-old-space-size=460
RUN npm run build:docker

# ---- Runtime stage ----
FROM nginx:1.27-alpine

RUN apk add --no-cache gettext \
  && rm -f /etc/nginx/conf.d/default.conf

COPY nginx.app.conf.template /etc/nginx/templates/app.conf.template
COPY docker-entrypoint.sh /docker-entrypoint.sh
RUN chmod +x /docker-entrypoint.sh

COPY --from=builder /app/dist /usr/share/nginx/html

ENV PORT=10000
EXPOSE 10000

ENTRYPOINT ["/docker-entrypoint.sh"]
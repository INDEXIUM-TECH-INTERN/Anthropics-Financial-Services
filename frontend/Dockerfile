# ---- Build stage ----
FROM node:22-alpine AS builder

WORKDIR /app

COPY package.json package-lock.json* ./

RUN npm ci --ignore-scripts 2>/dev/null || npm install --ignore-scripts

COPY . .

RUN npm run build

# ---- Runtime stage (serve static files) ----
FROM nginx:1.27-alpine

RUN apk add --no-cache gettext \
  && rm /etc/nginx/conf.d/default.conf

COPY nginx.app.conf.template /etc/nginx/templates/app.conf.template
COPY docker-entrypoint.sh /docker-entrypoint.sh
RUN chmod +x /docker-entrypoint.sh

COPY --from=builder /app/dist /usr/share/nginx/html

EXPOSE 80

ENTRYPOINT ["/docker-entrypoint.sh"]
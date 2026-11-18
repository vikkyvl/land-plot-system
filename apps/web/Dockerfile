FROM node:26.12.1-alpine AS build

WORKDIR /app

COPY package.json package-lock.json ./
RUN npm ci --no-audit --no-fund

COPY . .

ARG VITE_GATEWAY_URL=http://localhost:8080
ARG VITE_OIDC_AUTHORITY=http://localhost:8180/realms/land-plot
ARG VITE_OIDC_CLIENT_ID=land-plot-web
ARG VITE_GEOCODING_URL=https://nominatim.openstreetmap.org
ARG VITE_MAP_TILES_URL=https://tile.openstreetmap.org/{z}/{x}/{y}.png
ARG VITE_SATELLITE_TILES_URL=https://server.arcgisonline.com/ArcGIS/rest/services/World_Imagery/MapServer/tile/{z}/{y}/{x}

ENV VITE_GATEWAY_URL=$VITE_GATEWAY_URL \
    VITE_OIDC_AUTHORITY=$VITE_OIDC_AUTHORITY \
    VITE_OIDC_CLIENT_ID=$VITE_OIDC_CLIENT_ID \
    VITE_GEOCODING_URL=$VITE_GEOCODING_URL \
    VITE_MAP_TILES_URL=$VITE_MAP_TILES_URL \
    VITE_SATELLITE_TILES_URL=$VITE_SATELLITE_TILES_URL

RUN npm run build

FROM nginx:1.30.5-alpine

RUN apk add --no-cache libcap \
    && setcap 'cap_net_bind_service=+ep' /usr/sbin/nginx \
    && apk del libcap \
    && rm -f /etc/nginx/conf.d/default.conf \
    && chown -R nginx:nginx /etc/nginx/conf.d /var/cache/nginx /usr/share/nginx/html

COPY deploy/nginx/nginx.conf /etc/nginx/nginx.conf
COPY deploy/nginx/web.conf.template /etc/nginx/templates/web.conf.template
COPY --from=build --chown=nginx:nginx /app/dist /usr/share/nginx/html

ENV CSP_GATEWAY_ORIGIN=http://localhost:8080 \
    CSP_KEYCLOAK_ORIGIN=http://localhost:8180 \
    CSP_GEOCODING_ORIGIN=https://nominatim.openstreetmap.org \
    CSP_TILE_ORIGINS="https://tile.openstreetmap.org https://server.arcgisonline.com" \
    NGINX_ENVSUBST_FILTER=^CSP_

USER nginx

EXPOSE 80

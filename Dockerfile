# syntax=docker/dockerfile:1

# ---- Stage 1: build the Svelte SPA ----------------------------------------
FROM node:24-alpine AS frontend
WORKDIR /app/frontend
COPY frontend/package.json frontend/package-lock.json* ./
RUN npm ci
COPY frontend/ ./
RUN npm run build

# ---- Stage 2: build the Go binary (embeds the SPA) ------------------------
FROM golang:1.25-alpine AS backend
WORKDIR /app/backend
COPY backend/go.mod backend/go.sum ./
RUN go mod download
COPY backend/ ./
# Replace the embed placeholder with the freshly built frontend.
RUN rm -rf cmd/server/frontend_dist && mkdir -p cmd/server/frontend_dist
COPY --from=frontend /app/frontend/build/ cmd/server/frontend_dist/
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/nimlypanel ./cmd/server

# ---- Stage 3: minimal runtime --------------------------------------------
# Alpine gives us a real entrypoint so we can chown /data before dropping
# to nonroot — distroless has no shell and can't fix volume permissions at
# startup.
FROM alpine:3.22
RUN apk add --no-cache ca-certificates su-exec \
 && addgroup -S nonroot \
 && adduser -S -G nonroot nonroot \
 && mkdir -p /data && chown nonroot:nonroot /data
COPY --from=backend /out/nimlypanel /nimlypanel
COPY deploy/docker-entrypoint.sh /docker-entrypoint.sh
EXPOSE 8080
ENTRYPOINT ["/docker-entrypoint.sh"]
CMD ["/nimlypanel"]

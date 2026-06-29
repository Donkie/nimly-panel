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
# Pre-create /data owned by distroless nonroot (UID 65532) so that Docker
# initialises a mounted volume with the correct permissions on first run.
RUN mkdir /data && chown 65532:65532 /data

# ---- Stage 3: minimal runtime --------------------------------------------
# distroless/static ships CA certificates (needed for OIDC discovery / MQTT TLS)
# and runs as a non-root user.
FROM gcr.io/distroless/static-debian12:nonroot
WORKDIR /
COPY --from=backend /out/nimlypanel /nimlypanel
COPY --from=backend /data /data
EXPOSE 8080
USER nonroot:nonroot
ENTRYPOINT ["/nimlypanel"]

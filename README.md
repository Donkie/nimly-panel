# nimly-panel

A self-hosted web admin panel for a **Nimly Touch Pro** (Onesti / `easyCodeTouch`)
smart lock exposed over **Zigbee2MQTT**. Manage PIN codes, lock/unlock, lock
settings and view live activity — from your phone first.

- **Backend:** Go 1.25 (stdlib `net/http`), talks directly to your MQTT broker.
- **Frontend:** SvelteKit (SPA), mobile-first.
- **Auth:** OIDC (Authorization Code + PKCE) via **PocketID**.
- **Deploy:** single Docker image (Go binary with the SPA embedded).

## Features

| Area | What you can do |
|------|-----------------|
| Lock | Lock / unlock, live state, battery, voltage, signal |
| PIN codes | List, add, edit, enable/disable, delete; access types (unrestricted, master, schedules, no-access); validated against the lock's min/max length and max users |
| Settings | Sound volume, auto-relock, refresh-from-lock |
| Activity | Live unlock/lock events with source (keypad/fingerprint/rfid/zigbee) and user |

PIN digits are never displayed, logged, or stored — only metadata.

> **Why the panel stores PINs:** this lock's firmware does not report its stored
> PIN table over Zigbee (`getPinCode` times out), so the panel is the source of
> truth. It persists slot metadata (name, type, enabled, created date — never the
> digits) to `PIN_STORE_PATH` (a file on a mounted volume) and pushes set/clear
> to the lock. Editing a slot with a blank code just renames it. Set
> `PIN_STORE_KEY` to encrypt the store at rest (AES-256-GCM).

## How it talks to the lock

The lock is controlled entirely over MQTT on its Zigbee2MQTT base topic
(`MQTT_LOCK_TOPIC`, e.g. `zigbee2mqtt/Front Door Lock`):

- **Publish** to `<topic>/set`, e.g. `{"state":"UNLOCK"}` or
  `{"pin_code":{"user":3,"user_type":"unrestricted","user_enabled":true,"pin_code":1234}}`
  (clear a slot with `"pin_code":null`).
- **Read** via `<topic>/get`, e.g. `{"pin_code":""}` and the constraints.
- **Subscribe** to `<topic>` for live state, which the backend caches and pushes
  to the browser over Server-Sent Events (`/api/stream`).

> **Firmware note:** the exact shape of the `pin_code` read-back varies by
> firmware. The parser in `backend/internal/lock/parse.go` handles a single
> object, an array of objects, and null/empty. If your lock reports a different
> shape, capture a raw message (`mosquitto_sub -t 'zigbee2mqtt/Front Door Lock'`)
> and adjust `parsePins`.

## Project layout

```
backend/    Go service (config, mqtt, lock domain, auth, api) + embedded SPA
frontend/   SvelteKit SPA (mobile-first)
Dockerfile  multi-stage: build SPA → build Go (embeds SPA) → distroless runtime
docker-compose.yml  local dev stack (app + test mosquitto)
```

## Configuration

All config is via environment variables — see [`.env.example`](.env.example).
Required: `APP_BASE_URL`, `MQTT_BROKER_URL`, `MQTT_LOCK_TOPIC`, `OIDC_ISSUER`,
`OIDC_CLIENT_ID`, `OIDC_CLIENT_SECRET`, `SESSION_SECRET` (≥32 bytes).

Optional allowlist: set `ALLOWED_GROUPS` and/or `ALLOWED_SUBS` to restrict access
to specific PocketID groups/users. If both are empty, any successfully
authenticated PocketID user is allowed.

## Local development

Two terminals, hot-reloading frontend proxied to the Go backend:

```bash
# 1. backend (reads .env)
cd backend
set -a && source ../.env && set +a
go run ./cmd/server      # serves API on :8080

# 2. frontend (proxies /api → :8080)
cd frontend
npm install
npm run dev              # http://localhost:5173
```

For local auth set `DEV_MODE=true` (relaxes the Secure-cookie flag over http) and
register `http://localhost:5173/api/auth/callback` as a redirect URI in PocketID.

To exercise the whole thing in containers with a throwaway broker:

```bash
docker compose up --build
# then publish a fake lock state:
mosquitto_pub -h localhost -t 'zigbee2mqtt/Front Door Lock' \
  -m '{"state":"LOCK","battery":82,"max_pin_users":20,"min_pin_length":4,"max_pin_length":8}'
```

## PocketID setup

1. In PocketID create an **OIDC client**.
2. Redirect URI: `https://<your-domain>/api/auth/callback`.
3. Scopes: `openid profile email groups`.
4. Copy the client id/secret into `OIDC_CLIENT_ID` / `OIDC_CLIENT_SECRET` and set
   `OIDC_ISSUER` to your PocketID base URL.

## Deploy on Dokploy

Deploy as a Dokploy **Application** using the Dockerfile build (one container, one
port — the Go binary serves both the API and the embedded SPA).

1. **Create app:** Dokploy → Project → *Create Service* → *Application*. Point it
   at this git repo (branch `main`).
2. **Build type:** Dockerfile, path `./Dockerfile`. Build context `.`.
3. **Environment:** add the variables from `.env.example` (mark
   `OIDC_CLIENT_SECRET`, `MQTT_PASSWORD`, `SESSION_SECRET` as secrets). Generate
   the session secret with `openssl rand -base64 48`.
4. **Port:** the app listens on `8080`.
5. **Domain:** add your domain in Dokploy → it provisions a Let's Encrypt cert via
   Traefik. Set `APP_BASE_URL` to `https://<that-domain>`.
6. **Networking:** ensure the Dokploy host can reach your MQTT broker
   (`MQTT_BROKER_URL` with the broker's LAN IP/host — the broker runs alongside
   your Zigbee2MQTT, it is not the HA add-on).
7. **Persistence volume:** mount a volume at `/data`, set
   `PIN_STORE_PATH=/data/pins.json` so the PIN inventory survives redeploys, and
   set `PIN_STORE_KEY` (a long random secret) to encrypt it at rest. Optionally
   also set `AUDIT_LOG_PATH=/data/audit.jsonl` for the audit trail.
8. **Deploy**, then open the domain — you'll be redirected to PocketID to log in.

## Security notes

- OIDC Authorization Code + **PKCE**; ID-token signature/issuer/audience/nonce
  validated; random `state`.
- Server-side sessions (cookie `HttpOnly`, `Secure`, `SameSite=Lax`), token
  rotation on login, idle + absolute timeouts.
- **CSRF**: state-changing requests require the `X-CSRF-Token` header matching the
  session token (served via `/api/me`).
- Broker credentials and the OIDC client secret stay server-side; the browser
  never receives them.
- Security headers (CSP, `X-Frame-Options: DENY`, `nosniff`, no-referrer); HSTS +
  TLS terminated by Traefik.
- Every lock/unlock and PIN change is audit-logged with the authenticated user.

## Tests

```bash
cd backend && go test ./...     # parser, event derivation, PIN validation
cd frontend && npm run check    # svelte-check
```

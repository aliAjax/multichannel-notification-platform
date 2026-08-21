#!/bin/sh
set -eu
ROOT=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
PORT=${NOTIFY_SMOKE_PORT:-18080}
DATA=$(mktemp -d)
cleanup(){ [ -n "${PID:-}" ] && kill "$PID" 2>/dev/null || true; rm -rf "$DATA"; }
trap cleanup EXIT INT TERM
(cd "$ROOT" && NOTIFY_HTTP_ADDR=:$PORT NOTIFY_DATA_FILE="$DATA/store.json" go run ./cmd/notification-api) >/tmp/notification-smoke.log 2>&1 &
PID=$!
for i in $(seq 1 30); do curl -fsS "http://127.0.0.1:$PORT/readyz" >/dev/null 2>&1 && break; sleep 0.2; done
BODY='{"tenant_id":"smoke","idempotency_key":"smoke-1","channel":"email","targets":[{"address":"smoke@example.com"}],"subject":"Smoke","body":"hello","priority":50}'
FIRST=$(curl -fsS -X POST "http://127.0.0.1:$PORT/api/v1/notifications" -H 'Content-Type: application/json' -H 'X-Tenant-ID: smoke' -d "$BODY")
ID=$(printf '%s' "$FIRST" | sed -n 's/.*"id":"\([^"]*\)".*/\1/p')
[ -n "$ID" ]
SECOND=$(curl -fsS -X POST "http://127.0.0.1:$PORT/api/v1/notifications" -H 'Content-Type: application/json' -H 'X-Tenant-ID: smoke' -d "$BODY")
[ "$(printf '%s' "$SECOND" | grep -o '"id":"' | wc -l | tr -d ' ')" -eq 1 ]
curl -fsS "http://127.0.0.1:$PORT/api/v1/notifications/$ID/timeline" >/dev/null
echo "smoke ok: $ID"


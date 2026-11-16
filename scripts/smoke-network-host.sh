#!/usr/bin/env bash
# Smoke: network policy dest_type=host (backend).
# Prereq: API on http://127.0.0.1:18080, migrate 00015 applied.
set -euo pipefail

API="${SG_API:-http://127.0.0.1:18080}"
EMAIL="${SG_ADMIN_EMAIL:-admin@sailguard.local}"
PASS="${SG_ADMIN_PASSWORD:-ChangeMe!SailGuard1}"

need() { command -v "$1" >/dev/null || { echo "missing $1"; exit 1; }; }
need curl
need jq

echo "== health =="
curl -sS -m 5 "$API/v1/health" | jq .

echo "== login =="
TOKEN=$(curl -sS -X POST "$API/v1/auth/login" \
  -H 'Content-Type: application/json' \
  -d "{\"email\":\"$EMAIL\",\"password\":\"$PASS\"}" \
  | jq -r '.data.access_token')
if [[ -z "$TOKEN" || "$TOKEN" == "null" ]]; then
  echo "login failed"
  exit 1
fi
AUTH="Authorization: Bearer $TOKEN"

echo "== create policy =="
POLICY_ID=$(curl -sS -X POST "$API/v1/network-policies" \
  -H "$AUTH" -H 'Content-Type: application/json' \
  -d '{"name":"smoke-host-activation","description":"backend host smoke","mode":"enforce","priority":100}' \
  | jq -r '.data.id')
echo "POLICY_ID=$POLICY_ID"
[[ -n "$POLICY_ID" && "$POLICY_ID" != "null" ]]

echo "== add host rule (expect OK, lowercase) =="
HOST_RULE=$(curl -sS -X POST "$API/v1/network-policies/$POLICY_ID/rules" \
  -H "$AUTH" -H 'Content-Type: application/json' \
  -d '{"action":"deny","dest_type":"host","dest_value":"Activation.Windows.com","label":"win-activation"}')
echo "$HOST_RULE" | jq .
echo "$HOST_RULE" | jq -e '.ok == true and .data.dest_type == "host" and .data.dest_value == "activation.windows.com"' >/dev/null

echo "== add host+IP (expect fail) =="
CODE=$(curl -sS -o /tmp/sg-host-ip.json -w '%{http_code}' -X POST "$API/v1/network-policies/$POLICY_ID/rules" \
  -H "$AUTH" -H 'Content-Type: application/json' \
  -d '{"action":"deny","dest_type":"host","dest_value":"1.1.1.1","label":"should-fail"}')
cat /tmp/sg-host-ip.json | jq . || true
[[ "$CODE" == "400" ]]

echo "== add ip rule (expect OK) =="
curl -sS -X POST "$API/v1/network-policies/$POLICY_ID/rules" \
  -H "$AUTH" -H 'Content-Type: application/json' \
  -d '{"action":"deny","dest_type":"ip","dest_value":"1.1.1.1","label":"ip-ok"}' \
  | jq -e '.ok == true' >/dev/null

echo "== assign org + publish =="
curl -sS -X POST "$API/v1/network-policies/$POLICY_ID/assignments" \
  -H "$AUTH" -H 'Content-Type: application/json' \
  -d '{"scope":"org"}' | jq -e '.ok == true' >/dev/null
curl -sS -X POST "$API/v1/network-policies/$POLICY_ID/publish" \
  -H "$AUTH" | jq -e '.ok == true' >/dev/null

DEVICE_ID=$(curl -sS "$API/v1/devices" -H "$AUTH" | jq -r '.data.items[0].id // .data[0].id // empty')
if [[ -n "$DEVICE_ID" && "$DEVICE_ID" != "null" ]]; then
  echo "== effective for device $DEVICE_ID =="
  EFF=$(curl -sS "$API/v1/devices/$DEVICE_ID/network-policy" -H "$AUTH")
  echo "$EFF" | jq .
  echo "$EFF" | jq -e '
    .ok == true and
    ([.data.rules[]? | select(.dest_type=="host" and .dest_value=="activation.windows.com")] | length) > 0
  ' >/dev/null
else
  echo "== skip effective (no devices enrolled) =="
fi

if [[ "${SG_SMOKE_KEEP:-}" != "1" ]]; then
  echo "== cleanup delete policy =="
  curl -sS -X DELETE "$API/v1/network-policies/$POLICY_ID" -H "$AUTH" | jq -e '.ok == true' >/dev/null
fi

echo "PASS network host smoke"

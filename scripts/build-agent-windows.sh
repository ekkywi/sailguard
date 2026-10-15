#!/usr/bin/env bash
# Build SailGuard agent for modern Windows (10/11+) with the current Go toolchain.
# Usage: ./scripts/build-agent-windows.sh
# Output: bin/sailguard-agent-windows.exe
# For Windows 7 SP1 / 8.1 use: ./scripts/build-agent-win7.sh (Go 1.20)
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
OUT="${ROOT}/bin/sailguard-agent-windows.exe"

mkdir -p "$ROOT/bin"
export GOWORK=off

(
  cd "$ROOT/agent"
  echo "using: $(go version)"
  GOOS=windows GOARCH=amd64 go build -o "$OUT" ./cmd/sailguard-agent
)

echo "built: $OUT"
file "$OUT"

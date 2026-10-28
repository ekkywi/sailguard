#!/usr/bin/env bash
# Build SailGuard agent for Linux amd64 (current Go toolchain).
# Usage: ./scripts/build-agent-linux.sh
# Output: bin/sailguard-agent-linux
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
OUT="${ROOT}/bin/sailguard-agent-linux"

mkdir -p "$ROOT/bin"
export GOWORK=off

(
    cd "$ROOT/agent"
    echo "using: $(go version)"
    GOOS=linux GOARCH=amd64 go build -o "$OUT" ./cmd/sailguard-agent
)

echo "build: $OUT"
file "$OUT"
#!/usr/bin/env bash
# Build SailGuard agent for Windows 7+ using Go 1.20 (separate from day-to-day toolchain).
# Usage: ./scripts/build-agent-win7.sh
# Output: bin/sailguard-agent-win7.exe
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
GO120="${GO120_BIN:-/usr/local/go1.20/bin/go}"
OUT="${ROOT}/bin/sailguard-agent-win7.exe"

if [[ ! -x "$GO120" ]]; then
  echo "error: Go 1.20 not found at $GO120" >&2
  echo "install: https://go.dev/dl/go1.20.14.linux-amd64.tar.gz → /usr/local/go1.20" >&2
  echo "or set GO120_BIN=/path/to/go1.20/bin/go" >&2
  exit 1
fi

echo "using: $($GO120 version)"

TMP="$(mktemp -d)"
cleanup() {
  cp -f "$TMP/agent.go.mod" "$ROOT/agent/go.mod"
  cp -f "$TMP/agent.go.sum" "$ROOT/agent/go.sum"
  cp -f "$TMP/pkgs.go.mod" "$ROOT/pkgs/go.mod"
  rm -rf "$TMP"
}
trap cleanup EXIT

cp "$ROOT/agent/go.mod" "$TMP/agent.go.mod"
cp "$ROOT/agent/go.sum" "$TMP/agent.go.sum"
cp "$ROOT/pkgs/go.mod" "$TMP/pkgs.go.mod"

# Temporary module metadata compatible with Go 1.20
cat >"$ROOT/pkgs/go.mod" <<'EOF'
module github.com/ekkywi/sailguard/pkgs

go 1.20
EOF

cat >"$ROOT/agent/go.mod" <<'EOF'
module github.com/ekkywi/sailguard/agent

go 1.20

require github.com/ekkywi/sailguard/pkgs v0.0.0

require golang.org/x/sys v0.15.0

replace github.com/ekkywi/sailguard/pkgs => ../pkgs
EOF

cat >"$ROOT/agent/go.sum" <<'EOF'
golang.org/x/sys v0.15.0 h1:h48lPFYpsTvQJZF4EKyI4aLHaev3CxivZmv7yZig9pc=
golang.org/x/sys v0.15.0/go.mod h1:/VUhepiaJMQUp4+oa/7Zr1D23ma6VTLIYjOOTFZPUcA=
EOF

mkdir -p "$ROOT/bin"
export GOWORK=off
export GOTOOLCHAIN=local

(
  cd "$ROOT/agent"
  GOOS=windows GOARCH=amd64 "$GO120" build -o "$OUT" ./cmd/sailguard-agent
)

echo "built: $OUT"
file "$OUT"

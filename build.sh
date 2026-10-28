#!/usr/bin/env bash
set -euo pipefail
VERSION=$(git describe --tags --always --dirty 2>/dev/null || echo dev)
go build -trimpath -ldflags "-s -w -X main.version=$VERSION" -o gocode ./cmd/gocode
echo "built ./gocode ($VERSION)"
if [[ "${1:-}" == "install" ]]; then
  install -m 755 gocode "${PREFIX:-/usr/local}/bin/gocode"
  echo "installed to ${PREFIX:-/usr/local}/bin/gocode"
fi

#!/usr/bin/env bash
set -euo pipefail

# Cross-compile edge-api for the phone (ARM64 / Alpine musl).
# CGO disabled => fully static, runs on Alpine without glibc.
# Go cross-compiles natively, so no emulator is needed to build.

cd "$(dirname "$0")/.."

OUT="${1:-bin/edge-api}"
mkdir -p "$(dirname "$OUT")"

CGO_ENABLED=0 GOOS=linux GOARCH=arm64 \
	go build -trimpath -ldflags="-s -w" -o "$OUT" ./cmd/api

echo "built $OUT"
file "$OUT" 2>/dev/null || true

.PHONY: run build build-arm64 tidy test fmt emu emu-shell emu-logs emu-down

# Run locally (x86 / WSL) against the remote Postgres in .env
run:
	go run ./cmd/api

# Build for the current host
build:
	go build -o bin/edge-api ./cmd/api

# Cross-compile a static ARM64 binary for the phone (Alpine/musl)
build-arm64:
	./scripts/build-arm64.sh

tidy:
	go mod tidy

test:
	go test ./...

fmt:
	gofmt -w .

# --- ARM64 emulator (CPU-level, via Docker/QEMU binfmt) ---
# Runs the actual aarch64 phone binary + Postgres + Redis.
EMU := docker compose -f docker-compose.yml -f docker-compose.arm64.yml

emu: build-arm64
	$(EMU) up -d
	@echo "edge-api (aarch64) on http://localhost:8080  — try: curl localhost:8080/db/time"

emu-shell:
	$(EMU) exec api sh

emu-logs:
	$(EMU) logs -f api

emu-down:
	$(EMU) down

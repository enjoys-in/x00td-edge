# x00td-edge

Headless Alpine (postmarketOS, `UI: none`) edge node for the **Asus ZenFone
Max Pro M1** (`asus-x00td`, Snapdragon 636, 6 GB RAM). The phone runs a Go API
+ Redis cache + nginx. **PostgreSQL lives on a real server**, not the phone —
eMMC fsync is the device's weak spot, so the one disk-heavy tier is offloaded.

```
        Client (LAN)
            │  HTTP
            ▼
 ┌──────── X00TD (phone) ─────────┐
 │  nginx (reverse proxy)         │
 │      │                         │
 │      ▼                         │
 │  Go API ──► Redis (cache)      │  ← RAM-speed
 │      │                         │
 └──────┼─────────────────────────┘
        │  network (LAN / WireGuard)
        ▼
   Real server: PostgreSQL (NVMe/UFS)   ← durable, disk-heavy tier
```

The Go API exposes a cache-miss (Postgres round-trip) vs cache-hit (Redis)
path so you can measure the DB round-trip cost described in the plan.

## Layout

```
cmd/api/            Go API entrypoint (graceful shutdown)
internal/config/    env configuration
internal/db/        pgxpool → remote Postgres
internal/cache/     Redis client
internal/httpapi/   router + handlers (/healthz, /db/time)
deploy/redis/       redis.conf (cache-only, persistence off)
deploy/nginx/       reverse proxy vhost
deploy/openrc/      OpenRC service for the Go API (auto-restart)
scripts/            ARM64 cross-build + QEMU rehearsal helpers
```

## Develop (x86, your machine / WSL)

```sh
cp .env.example .env     # point DATABASE_URL at your Postgres
make tidy
make run
# GET http://localhost:8080/healthz   -> {"status":"ok","db":"ok","redis":"ok"}
# GET http://localhost:8080/db/time   -> source:"db" first, source:"cache" after
```

You need a reachable Postgres and a local Redis (`redis-server`) for the full
path; `/healthz` will report which dependency is down otherwise.

## Local full-path testing (Docker)

Spin up stand-in Postgres + Redis (dev only — Postgres maps to host `5433` to
avoid clashing with a local 5432):

```sh
docker compose up -d
export DATABASE_URL='postgres://app:app@localhost:5433/app?sslmode=disable'
export REDIS_ADDR='localhost:6379'
make run

curl -s localhost:8080/healthz          # {"db":"ok","redis":"ok","status":"ok"}
curl -s localhost:8080/db/time          # source:"db"    (cache miss -> Postgres)
curl -s localhost:8080/db/time          # source:"cache" (cache hit  -> Redis)

docker compose down
```

Load test (plan §7) — needs [k6](https://k6.io):

```sh
k6 run scripts/loadtest.js
BASE_URL=http://localhost:8080 VUS=50 DURATION=30s k6 run scripts/loadtest.js
# CACHE_TTL_SECONDS=0 on the API forces every request down the DB round-trip path.
```

Avoid benchmarking with a `curl`-per-request loop on Windows — it measures
`curl.exe` spawn cost, not the API.

## Emulate the phone (ARM64) — two fidelity levels

You can test without touching (or bricking) the phone:

**1. CPU-level emulation (fast, runs the real aarch64 binary) — `make emu`**

Runs the actual cross-compiled `bin/edge-api` on aarch64 via Docker's QEMU/binfmt,
alongside Postgres + Redis:

```sh
make emu                 # build-arm64 + bring up the aarch64 api + db + cache
curl localhost:8080/db/time
make emu-shell           # drop into a shell inside the emulated aarch64 Alpine
make emu-logs
make emu-down
```

This proves the ARM binaries run and the full cache/DB path works. It is **not**
a full-kernel boot — it emulates the userland CPU, not the device.

**2. Full-system boot (faithful, slower) — `scripts/qemu-test.sh`**

Boots the real postmarketOS/Alpine kernel + init (`UI: none`) in QEMU via
`pmbootstrap` (run in WSL2). Needs `sudo` (your password) and, without KVM,
runs in software emulation. Validates headless boot + SSH + OpenRC services.

Neither emulates the Snapdragon/Wi-Fi — those remain phone-only (Phase 0).

## Build for the phone (ARM64 / musl)

```sh
make build-arm64         # -> bin/edge-api  (static, CGO disabled)
```

Go cross-compiles natively — no emulator needed to *produce* the binary.

## Rehearse the OS + services in QEMU (WSL2)

```sh
./scripts/qemu-test.sh
```

Boots the **real** pmOS/Alpine userland (`UI: none`) in QEMU to validate the
headless-boot + SSH + OpenRC-service workflow. **It does not emulate the
`asus-x00td` hardware** — Wi-Fi, USB-gadget networking, eMMC and stable boot
must still be verified on the real phone (Phase 0).

![enjoys-os booting in QEMU: ENJOYS banner + login and live system-info MOTD](docs/enjoys-os-emulator.png)

## Deploy on the phone (pmOS)

```sh
# as root on the phone, after copying bin/edge-api and deploy/ over
adduser -S -D -H edge
install -m 0755 edge-api /usr/local/bin/edge-api

install -d /etc/edge-api
cp deploy/edge-api.env.example /etc/edge-api/edge-api.env   # fill in secrets
cp deploy/redis/redis.conf     /etc/redis.conf
cp deploy/nginx/edge.conf      /etc/nginx/http.d/edge.conf
cp deploy/openrc/edge-api      /etc/init.d/edge-api
chmod +x /etc/init.d/edge-api

rc-update add redis    default
rc-update add edge-api default
rc-update add nginx    default
rc-service redis    start
rc-service edge-api start
rc-service nginx    start
```

## Status

Research / brainstorm. Current focus: headless install + SSH on the device;
services and benchmarking come after the OS install is solid.

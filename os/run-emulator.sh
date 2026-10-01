#!/usr/bin/env bash
set -euo pipefail

# Boot the built postmarketOS image in the QEMU emulator (run in WSL2).
# SSH is forwarded to localhost:2222. On Windows 11 WSLg the QEMU window shows
# on your desktop; pass --display none to stay headless and use SSH only.
#
#   os/run-emulator.sh                 # windowed (WSLg) or headless per your setup
#   os/run-emulator.sh --display none  # force headless

command -v pmbootstrap >/dev/null 2>&1 || {
	echo "pmbootstrap not found. Install: pipx install pmbootstrap" >&2
	exit 1
}

echo "Booting the phone OS in QEMU..."
echo "SSH in from another shell:  ssh -p 2222 <user>@127.0.0.1"
echo "Inside: rc-status ; curl localhost:8080/db/time"
echo

pmbootstrap qemu "$@"

#!/usr/bin/env bash
set -euo pipefail

# Rehearse the OS + services in QEMU via pmbootstrap (run inside WSL2).
#
# This boots the REAL pmOS/Alpine userland (UI: none) so you can validate the
# headless-boot + SSH + OpenRC-service workflow. It does NOT emulate the
# asus-x00td hardware — Wi-Fi / USB-gadget networking / eMMC / stable boot must
# still be verified on the real phone (Phase 0).
#
# One-time setup (interactive):
#   pipx install pmbootstrap
#   pmbootstrap init
#     device:  qemu-amd64   (fast, x86)  OR  qemu-aarch64 (matches the phone, slower)
#     UI:      none
#     extra packages: openssh
#
# Then run this script to (re)build and boot the VM.

if ! command -v pmbootstrap >/dev/null 2>&1; then
	echo "pmbootstrap not found. Install with: pipx install pmbootstrap" >&2
	exit 1
fi

pmbootstrap install

echo
echo "Booting QEMU. SSH forwarding is set up on localhost:2222."
echo "In another terminal:  ssh -p 2222 <user>@127.0.0.1"
echo "Add '--display none' below to run fully headless."
echo

# On Windows 11 WSL2, WSLg shows the QEMU window on the Windows desktop.
pmbootstrap qemu

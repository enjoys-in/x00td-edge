#!/usr/bin/env bash
set -euo pipefail

# Build the phone OS image (postmarketOS/Alpine): minimal enjoys-os = SSH + Docker.
# Run inside WSL2 Ubuntu. Uses pmbootstrap, which uses sudo — you type the
# password when prompted; this script never handles it.
#
#   TARGET=qemu-aarch64 os/build-os.sh   # image to boot in the emulator (default)
#   TARGET=asus-x00td   os/build-os.sh   # image to flash to the phone
#
# Assumes you already ran `pmbootstrap init` for $TARGET (UI: none, +openssh).

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
TARGET="${TARGET:-qemu-aarch64}"
PKG="enjoys-base"
PKGDIR="$ROOT/os/pmaports/$PKG"

command -v pmbootstrap >/dev/null 2>&1 || {
	echo "pmbootstrap not found. Install: pipx install pmbootstrap" >&2
	exit 1
}

echo ">> [1/4] normalizing package files (CRLF -> LF)"
for f in APKBUILD "$PKG.post-install" enjoys-os.plymouth enjoys-os.script plymouthd.conf Caddyfile; do
	[ -f "$PKGDIR/$f" ] && sed -i 's/\r$//' "$PKGDIR/$f"
done

echo ">> [2/4] linking package into pmbootstrap's pmaports checkout"
APORTS="$(pmbootstrap config aports 2>/dev/null | tail -1 || true)"
[ -n "${APORTS:-}" ] && [ -d "$APORTS" ] || APORTS="$HOME/.local/var/pmbootstrap/cache_git/pmaports"
mkdir -p "$APORTS/main"
ln -sfn "$PKGDIR" "$APORTS/main/$PKG"

echo ">> [3/4] building the $PKG apk"
pmbootstrap checksum "$PKG"
# --force + --arch: rebuild the TARGET-arch apk. This is a noarch pkg cached
# per-arch; without --arch the install step reuses a stale aarch64 apk. Both
# of our targets (qemu-aarch64, asus-x00td) are aarch64.
pmbootstrap build --force --arch aarch64 "$PKG"

echo ">> [4/4] building the rootfs image for $TARGET (with $PKG baked in)"
# --zap: wipe any previous rootfs so removed packages/services don't linger.
case "$TARGET" in
	# Emulator: throwaway password so the build is non-interactive.
	qemu-*) pmbootstrap install --add "$PKG" --password "${USER_PASSWORD:-enjoys}" --zap ;;
	# Phone: the 'user' account is locked on first boot (root login = enjoys),
	# so the user password is irrelevant — set it non-interactively too.
	*)      pmbootstrap install --add "$PKG" --password "${USER_PASSWORD:-enjoys}" --zap ;;
esac

echo
echo ">> done."
case "$TARGET" in
	qemu-*) echo "   boot the emulator:  os/run-emulator.sh" ;;
	*)      echo "   flash the phone:    pmbootstrap flasher flash_kernel && pmbootstrap flasher flash_rootfs" ;;
esac

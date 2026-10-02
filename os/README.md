# Building enjoys-os (postmarketOS / Alpine)

This folder builds **enjoys-os** — a minimal headless Linux (postmarketOS /
Alpine) for the Asus ZenFone Max Pro M1 (`asus-x00td`): **SSH only**, with a
boot splash, root login, and a branded MOTD. A close-to-mainline kernel carries
the phone's front (OV8856) + rear main (OV13855) camera device-tree.

The same build produces two images:

| Target         | Purpose                                   |
|----------------|-------------------------------------------|
| `qemu-aarch64` | boot in the **emulator** (QEMU) to test   |
| `asus-x00td`   | **flash** to the real phone               |

Both share the same Alpine userland + the `enjoys-base` package; only the
kernel/drivers differ.

## What's in the image
- `openssh` (sshd) + boot splash (`postmarketos-bootsplash`, enjoys-os theme)
- **Root login** (password `enjoys`); the non-root `user` account is locked
- Dynamic **MOTD**: ENJOYS banner + live system info (load, memory, disk, IP…)
- Rootfs **~175 MB**. Docker + Caddy were removed in **v0.2.0** to keep it small
  — add them back on-device with `apk add docker docker-cli-compose caddy`.

RAM and storage are **auto-detected** per device; the root filesystem auto-expands
to fill the phone's partition on first boot.

## Prerequisites (one-time, inside WSL2 Ubuntu)

`pmbootstrap` needs real Linux and uses `sudo` (you type the password — the
scripts never handle it). WSL2 has no `/dev/kvm`, so the emulator runs under TCG
(software emulation — slower, still works).

```sh
export PATH="$HOME/.local/bin:$PATH"   # where pmbootstrap is installed
pmbootstrap --version
```

## 1. Initialise pmbootstrap for a target (one-time per device)

```sh
pmbootstrap init
#   channel: edge
#   device:  qemu-aarch64   (emulator)   — repeat later for asus-x00td
#   UI:      none           (headless, pure shell + sshd)
#   extra packages: openssh
```

## 2. Build the image

```sh
TARGET=qemu-aarch64 bash os/build-os.sh   # emulator image
TARGET=asus-x00td   bash os/build-os.sh   # flashable phone image
```

The script CR-strips the package files, builds the `enjoys-base` apk, and bakes
it into the rootfs image (non-interactive; root login is `enjoys`).

## 3. Test in the emulator

See [../docs/qemu.md](../docs/qemu.md). Log in as **`root`** / **`enjoys`**.

```sh
os/run-emulator.sh          # boots the image in QEMU (pmbootstrap qemu)
```

## 4. Flash the phone

See [../flash.md](../flash.md). In short, from WSL with the phone in fastboot:

```sh
pmbootstrap flasher flash_vbmeta
pmbootstrap flasher flash_kernel
pmbootstrap flasher flash_rootfs
```
Then `ssh root@172.16.42.1` (password `enjoys`).

## Files in this folder

| Path | What |
|---|---|
| `pmaports/enjoys-base/` | the custom package — `APKBUILD`, `post-install` (root login, MOTD, services), plymouth splash theme |
| `camera/` | the front+rear camera device-tree + kernel integration (see [../docs/camera-port.md](../docs/camera-port.md)) |
| `build-os.sh` | build the `enjoys-base` apk + rootfs image for a target |
| `run-emulator.sh` | boot the emulator image in QEMU |

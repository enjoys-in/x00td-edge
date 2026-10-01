# Building the phone OS (postmarketOS / Alpine)

This folder builds the **actual operating system** for the Asus ZenFone Max
Pro M1 (`asus-x00td`) — a real Alpine Linux (postmarketOS), not a container.
Your services (Go API + Redis + nginx) are baked into the image as the
`edge-stack` package so they come up on boot.

The same build produces two images:

| Target         | Purpose                                   |
|----------------|-------------------------------------------|
| `qemu-aarch64` | boot in the **emulator** (QEMU) to test   |
| `asus-x00td`   | **flash** to the real phone               |

Both share the same Alpine userland + `edge-stack`; only the kernel/drivers
differ. So the emulator validates the OS + services — only Wi-Fi/boot remain
phone-only.

## Prerequisites (one-time, inside WSL2 Ubuntu)

`pmbootstrap` needs real Linux and uses `sudo` (you type your password — the
scripts never handle it). KVM is absent under WSL2, so the emulator runs in
software emulation (slower, still works).

```sh
sudo apt update && sudo apt install -y pipx qemu-system-arm
pipx install pmbootstrap && pipx ensurepath
pmbootstrap --version
```

## 1. Initialise pmbootstrap for a target (interactive, one-time per device)

```sh
pmbootstrap init
#   channel: edge
#   device:  qemu-aarch64   (for the emulator)  — repeat later for asus-x00td
#   UI:      none           (headless, pure shell + sshd)
#   extra packages: openssh
```

## 2. Build the OS image with the stack baked in

From the project root:

```sh
TARGET=qemu-aarch64 os/build-os.sh      # emulator image
# TARGET=asus-x00td  os/build-os.sh     # flashable phone image
```

This cross-builds the Go binary, assembles the `edge-stack` package, builds
it, and bakes it into the rootfs image.

## 3. Test it in the emulator

```sh
os/run-emulator.sh                      # boots the image in QEMU
# in another shell:
ssh -p 2222 <user>@127.0.0.1            # you're now in YOUR phone OS
rc-status                               # redis / edge-api / nginx running
curl localhost:8080/db/time
```

## 4. Flash to the phone (only once the emulator looks right)

```sh
# after: pmbootstrap init (device: asus-x00td) && TARGET=asus-x00td os/build-os.sh
pmbootstrap flasher flash_kernel
pmbootstrap flasher flash_rootfs
```

## Notes

- `edge-stack` keeps the phone lean per the plan: **Postgres stays remote**.
  To also run Postgres/Docker *on the device*, add `postgresql` / `docker` to
  the package `depends` — but that hits the slow eMMC (see the plan's rationale
  for keeping the DB off the phone).
- `pmbootstrap` / `sudo` steps can't be auto-run here (password entry), so run
  these in your WSL terminal; the scripts do everything except type the password.

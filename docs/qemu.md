# Running enjoys-os in QEMU (emulator)

How to build and boot the **enjoys-os** image in QEMU to validate the OS before
flashing a phone. This is the **emulator** target (`qemu-aarch64`) — a generic
virtual ARM64 machine. It shares the same Alpine/postmarketOS userland and the
`enjoys-base` package (minimal SSH + boot splash) as the phone image; only the
kernel/drivers differ.

> The phone camera kernel (`linux-postmarketos-qcom-sdm660`) is **phone-only** —
> QEMU cannot emulate the sdm660 CAMSS, so camera bring-up is not testable here.
> QEMU validates the OS, services, boot, login, and SSH.

All commands run inside **WSL2 Ubuntu**.

---

## 0. Prerequisites (one-time)

```sh
sudo apt update && sudo apt install -y qemu-system-arm qemu-utils
# pmbootstrap (installed from git in this project; PyPI releases are yanked):
#   ~/.local/bin/pmbootstrap must be on PATH
export PATH="$HOME/.local/bin:$PATH"
pmbootstrap --version
```

Notes:
- WSL2 has **no `/dev/kvm`**, so QEMU uses TCG (software emulation) → the VM is
  slow but works. Boot to the login prompt takes a minute or two.
- `--host-qemu` uses the `qemu-system-aarch64` you installed above.

## 1. Point pmbootstrap at the emulator target

```sh
pmbootstrap config device qemu-aarch64
pmbootstrap config ui none            # headless: pure shell + sshd
# verify:
pmbootstrap config device   # -> qemu-aarch64
pmbootstrap config ui       # -> none
```

(First time only, you may instead run the interactive `pmbootstrap init` and
choose: channel `edge`, device `qemu-aarch64`, UI `none`, add pkg `openssh`.)

## 2. Build the image (rootfs with enjoys-base baked in)

Either use the wrapper:

```sh
TARGET=qemu-aarch64 bash os/build-os.sh
```

…or run the two steps directly (what the wrapper does):

```sh
# build the custom package (SSH + boot splash)
pmbootstrap checksum enjoys-base
pmbootstrap build --force --arch aarch64 enjoys-base

# generate the bootable rootfs image (throwaway emulator password)
pmbootstrap -y install --add enjoys-base --password enjoys --zap
```

`--zap` wipes any previous rootfs so removed packages/services don't linger.
The install ends with `DONE!` and creates `qemu-aarch64.img`.

> If `pmbootstrap qemu` says *"The rootfs has not been generated yet"*, you
> skipped this step (or a `--force` build zapped it) — just re-run the install.

## 3. Boot it

**Headless (serial console in your terminal — best for logs/CI):**

```sh
pmbootstrap qemu --display none --host-qemu
```

**Windowed (GUI via WSLg):**

```sh
pmbootstrap qemu --host-qemu --display gtk
```

On start pmbootstrap prints the connection info:

```
Running postmarketOS in QEMU VM (aarch64)
WARNING: QEMU is not using KVM and will run slower!
Connect to the VM:
* (ssh) ssh -p 2222 user@localhost
* (serial) in this console (stdout/stdin)
```

## 4. The boot sequence you should see

```
UEFI firmware (version ... )
BdsDxe: loading Boot0002 "UEFI Misc Device" ...
BdsDxe: starting Boot0002 "UEFI Misc Device" ...
      ... kernel boot (dmesg) ...
      ... OpenRC starts sshd ...

   ENJOYS-OS  -  Cloud & Edge Node
   enjoys-os (/dev/ttyAMA0)

enjoys-os login:
```

The banner (`/etc/issue`), hostname `enjoys-os`, and the getty on
`/dev/ttyAMA0` confirm `enjoys-base` booted correctly. After login you get the
branded MOTD (ENJOYS banner + live system info).

> Under TCG the serial output is buffered — if the console looks stuck at UEFI,
> give it a minute; the kernel log and login prompt appear in a burst.

## 5. Log in

Log in as **`root`** / **`enjoys`** — on the serial console, the GUI window, or
over SSH. On login you get the branded MOTD (ENJOYS banner + live system info).

```sh
rc-status                            # sshd should be "started"
```

**SSH in the emulator needs a one-time network bring-up** (UI=none images don't
auto-configure networking). On the serial console as root:

```sh
ip link set eth0 up; udhcpc -b -i eth0    # gets 10.0.2.15 from QEMU
```
then from the host:
```sh
ssh -p 2222 root@localhost          # password: enjoys
```
(On the real phone, USB networking comes up automatically — no manual step.)

## 6. Shut down

- Serial: `sudo poweroff` inside the VM, or press `Ctrl-a` then `x` to kill QEMU.
- GUI: close the window, or `sudo poweroff` inside.

---

## Troubleshooting

| Symptom | Cause / fix |
|---|---|
| `rootfs has not been generated yet` | Run step 2 (`pmbootstrap install …`). A `--force` kernel build zaps the rootfs. |
| Console stuck at `UEFI ... BdsDxe` | TCG is slow + serial is buffered; wait ~1–2 min for the kernel/login burst. |
| `WARNING: QEMU is not using KVM` | Expected under WSL2 (no `/dev/kvm`); just slower. |
| GUI window fails (GL/EGL errors) | Drop `--no-gl`; use `--display gtk` (default `gl=on`) or go headless with `--display none`. |
| `network doesn't work automatically` (UI=none) | On the serial console (as root) run `ip link set eth0 up; udhcpc -b -i eth0`, then `ssh -p 2222 root@localhost`; see <https://postmarketos.org/qemu-network>. |
| Out of disk space in the VM | `pmbootstrap qemu --image-size 2G …` |

## Relationship to the phone image

Same userland + `enjoys-base`; the only difference is the target:

```sh
pmbootstrap config device asus-x00td   # phone (pulls the camera kernel)
pmbootstrap -y install --add enjoys-base --password enjoys --zap
pmbootstrap export <dir>               # boot.img + rootfs for fastboot
```

See [flash.md](../flash.md) for the full fastboot flashing procedure and
[docs/camera-port.md](camera-port.md) for the phone camera device-tree.

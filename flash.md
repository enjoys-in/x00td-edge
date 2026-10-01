# Flashing enjoys-os to the phone (Asus ZenFone Max Pro M1 — `asus-x00td`)

This flashes the **real** `enjoys-os` image (SSH + Docker + Caddy, OpenRC,
boot splash) onto the phone with `pmbootstrap` + `fastboot`. It is the same
userland you tested in the QEMU emulator; only the kernel/hardware differ.

> ⚠️ **Read first**
> - **This WIPES the phone** (all Android data) and **unlocking the bootloader
>   voids the warranty**.
> - `asus-x00td` is a **`testing`** port in postmarketOS — it boots, but some
>   peripherals (notably **Wi-Fi**) may not work yet. Verify on the
>   [device wiki page](https://wiki.postmarketos.org/wiki/ASUS_ZenFone_Max_Pro_M1_(asus-x00td))
>   before relying on it.
> - Back up anything you need off the phone now.
> - Everything below runs in **WSL2 Ubuntu** (where `pmbootstrap` lives), except
>   the `usbipd` and (optional) native-fastboot steps, which run on **Windows**.

---

## 1. Point pmbootstrap at the phone (not the emulator)

We built the emulator image with device `qemu-aarch64`. Switch to the phone:

```bash
pmbootstrap init
#   vendor:  asus
#   device:  asus-x00td
#   kernel:  stable   (or mainline)
#   UI:      none
#   service manager: openrc
#   extra packages:  openssh
```

(Re-select the same answers we used; only the device changes.)

## 2. Build the phone image

```bash
cd /mnt/f/private/ENJOYS/x00td-edge
TARGET=asus-x00td bash os/build-os.sh
```

Unlike the emulator build, this **prompts for a real login password** (no dummy
`enjoys`). Choose a strong one — it's your console + SSH password.

## 3. Unlock the bootloader (one-time, wipes the device)

On the phone:
1. Settings → About → tap **Build number** 7× to enable Developer options.
2. Developer options → enable **OEM unlocking** and **USB debugging**.

Then from a terminal with `adb`/`fastboot` (see step 4 for getting USB into WSL):
```bash
adb reboot bootloader
fastboot flashing unlock      # confirm on the phone with Volume/Power
# if rejected, try:
fastboot oem unlock
```
Asus has at times required their official unlock tool/APK for this model — if
`fastboot` unlock is refused, check the device wiki page for the current method.

## 4. Get the phone's USB into the flashing environment

WSL2 can't see USB devices by default. Two options:

### Option A — attach USB to WSL with `usbipd-win` (keeps the pmbootstrap flow)
On **Windows** (Admin PowerShell), one-time:
```powershell
winget install --exact dorssel.usbipd-win
```
With the phone in **fastboot/bootloader mode** and plugged in:
```powershell
usbipd list                      # note the phone's BUSID (e.g. 2-4)
usbipd bind   --busid <BUSID>    # one-time, Admin
usbipd attach --wsl --busid <BUSID>
```
In **WSL Ubuntu**, confirm it's visible:
```bash
sudo apt install -y android-tools-adb android-tools-fastboot
fastboot devices                 # should list the phone
```

### Option B — flash from native Windows `fastboot` (no usbipd)
Export the images from WSL, then flash with Windows `fastboot.exe`:
```bash
pmbootstrap export               # writes boot.img + rootfs to /tmp/postmarketOS-export/
```
Copy those files to Windows and use the manual `fastboot flash` commands in
step 5 (the `pmbootstrap flasher` convenience wrapper is WSL-only).

## 5. Flash

Put the phone in fastboot mode (`adb reboot bootloader`, or Power+Vol-Down),
then from **WSL**:

```bash
# Device uses AVB — flash a verity-disabled vbmeta first:
pmbootstrap flasher flash_vbmeta

# Kernel / boot image:
pmbootstrap flasher flash_kernel

# Root filesystem (sparse, goes to userdata):
pmbootstrap flasher flash_rootfs
```

<details>
<summary>Manual <code>fastboot</code> equivalent (Option B / Windows)</summary>

```bash
# from the pmbootstrap export dir:
fastboot --disable-verity --disable-verification flash vbmeta vbmeta.img
fastboot flash boot boot.img
fastboot flash userdata <rootfs-image>.img     # rootfs partition is 'userdata' on x00td
```
Partition names come from the deviceinfo (`flash_method=fastboot`,
`generate_bootimg=true`, vbmeta partition = `vbmeta`). Prefer `pmbootstrap
flasher` — it knows the offsets/partitions automatically.
</details>

## 6. First boot

```bash
fastboot reboot
```
First boot is **slow** (filesystem resize + initial setup). Give it a few
minutes. On the phone screen you should see the **enjoys-os Plymouth splash**,
then a console login — hostname `enjoys-os`.

## 7. Connect over SSH

With the phone booted and plugged into USB, a USB-network interface appears and
the phone is reachable at **`172.16.42.1`**:
```bash
ssh user@172.16.42.1        # password = the one you set in step 2
```
Then verify the stack:
```bash
rc-status                   # sshd, docker, caddy started
docker --version
caddy version
```

## 8. Post-flash setup

- **Wi-Fi / Bluetooth:** see section 9 — the drivers + firmware are already in
  the port, so joining WiFi is usually just `sudo nmtui`.
- **Let `user` run docker without sudo** (if it didn't take at build time):
  ```bash
  sudo addgroup user docker && sudo rc-service docker restart
  ```
- **Timezone:** `sudo setup-timezone`
- **Update:** `sudo apk update && sudo apk upgrade`

## 9. Wireless (WiFi / Bluetooth) & firmware

You do **not** need to build or write drivers — they are mainline and already
enabled in this port's kernel, and WiFi firmware is shipped:

| | Chip / driver | Status in this port |
|---|---|---|
| **WiFi** | Qualcomm **WCN3990** via **`ath10k_snoc`** (`CONFIG_ATH10K_SNOC=m`) | driver on; firmware `ath10k/WCN3990/*` shipped by `firmware-asus-x00td` |
| **Bluetooth** | Qualcomm **`hci_qca`** (`CONFIG_BT_QCA=m`) | driver on; BT firmware blobs may need the stock ones (see extraction) |
| **Camera** | mainline Qualcomm CAMSS | not supported |

The device also pulls in **`msm-firmware-loader`**, which loads firmware straight
from the phone's own stock firmware partition at runtime — so in many cases **no
manual extraction is needed**.

### Verify / bring up (over SSH)
```bash
dmesg | grep -iE 'ath10k|qca|bluetooth|firmware'   # driver + firmware load
rfkill list                                        # unblock if soft-blocked
ip link                                            # look for wlan0
```
Join WiFi (UI=none has no GUI; install NetworkManager if absent):
```bash
sudo apk add networkmanager networkmanager-tui
sudo rc-update add networkmanager default && sudo rc-service networkmanager start
sudo nmtui                                          # or: iwctl / wpa_supplicant
```
Bluetooth (install the userspace stack):
```bash
sudo apk add bluez
sudo rc-update add bluetooth default && sudo rc-service bluetooth start
bluetoothctl
```
If `ath10k` loads its firmware cleanly, `wlan0` appears and `nmtui` connects —
that's **working WiFi, no driver work**.

### If a firmware blob IS missing — extract from the stock ROM
Qualcomm firmware (especially **Bluetooth** `qca/crbtfw*.tlv` + `qca/crnv*.bin`,
or a board-specific WiFi `board-2.bin`/BDF) is non-free and sometimes absent.
Pull it from the stock **vendor/firmware** partition — you extract **firmware
blobs, never kernel drivers**:

```bash
# Option 1 - from a running stock Android (adb):
adb pull /vendor/firmware ./vendor-fw          # path varies by ROM

# Option 2 - from a stock ROM image:
#   A/B OTA payload.bin -> vendor.img:
payload-dumper-go payload.bin
#   sparse image -> raw -> mount:
simg2img vendor.img vendor.raw && sudo mount -o loop,ro vendor.raw /mnt
#   newer EROFS images:  fsck.erofs --extract=./out vendor.img
```
Then copy the blobs to the path the driver asks for (check the `dmesg`
"firmware: failed to load ..." line) and reload:
```bash
# e.g. Qualcomm BT firmware, copied to the phone via scp:
sudo install -Dm644 crbtfw*.tlv crnv*.bin -t /lib/firmware/qca/
sudo modprobe -r hci_qca && sudo modprobe hci_qca   # or reboot
# WiFi board data goes under /lib/firmware/ath10k/WCN3990/hw1.0/
```

## 10. Troubleshooting

| Symptom | Fix |
|---|---|
| `fastboot devices` empty in WSL | Re-run `usbipd attach --wsl --busid <BUSID>`; phone must be in **bootloader** mode |
| Unlock refused | Enable **OEM unlocking** first; try `fastboot oem unlock`; check device wiki for Asus tool |
| Boot loops / black screen | Re-flash `vbmeta` (verity must be disabled); confirm `flash_kernel` + `flash_rootfs` both succeeded |
| No `172.16.42.1` | USB networking profile is `developer` (default); try another cable/port; check `dmesg` on the host |
| `wlan0` missing / ath10k fw fail | `dmesg | grep ath10k`; supply the exact `board-2.bin`/BDF it names (§9 extraction) |
| Bluetooth missing | install `bluez`; `hci_qca` may need `qca/crbtfw*.tlv`+`crnv*.bin` from stock vendor (§9) |

## 11. Reference

- Generic install guide: <https://wiki.postmarketos.org/wiki/Installation_guide>
- Device page: <https://wiki.postmarketos.org/wiki/ASUS_ZenFone_Max_Pro_M1_(asus-x00td)>
- Device flash facts (from `deviceinfo`): `flash_method=fastboot`,
  `generate_bootimg=true`, `append_dtb=true`, dtb `qcom/sdm636-asus-x00td`,
  vbmeta partition `vbmeta`, sparse rootfs.

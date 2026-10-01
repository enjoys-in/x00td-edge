# Camera port notes — Asus ZenFone Max Pro M1 (`asus-x00td`)

Goal: get **raw capture** working on mainline Linux / postmarketOS (`qcom-camss`),
for the project's `enjoys-os` image. This file records the hardware facts
extracted from the **stock ROM** (not guesses) so the driver/DT work can start.

> Status: research. Drivers are mainline (no "building a driver" for some);
> remaining work is per-sensor driver port + device-tree + on-device bring-up.
> Camera bring-up **requires the physical phone** (QEMU can't emulate CAMSS).

---

## 1. Sensor binding — from stock `/vendor/etc/camera/camera_config.xml`

The device probes several **module-vendor variants** per slot and uses whichever
is physically fitted. All sensors are **OmniVision / Hynix / GalaxyCore**
(there is **no Sony IMX486** — a common myth for this phone).

| Slot (CameraId) | Position | CSID | Lanes (LaneMask) | Sensor candidates (module) | Actuator / EEPROM |
|---|---|---|---|---|---|
| **0 — rear main** | BACK, 90° | 0 | 4-lane (`0x1F`) | `ov13855` (chicony/holitech, 13MP), `hi1333` (ofilm, 13MP), `ov16880` (qtech, 16MP), `ov16885` (ofilm/holitech, 16MP) | AF: s2034/dw9714/fp5510; EEPROM fm24c64d |
| **1 — front** | FRONT, 270° | 2 | 4-lane (`0x1F`) | `ov8856` (chicony, 8MP), `hi846` (kingcome/tsp), `ov16880` (qtech, 16MP) | EEPROM only |
| **2 — rear aux / depth** | BACK_AUX, 90° | 1 | 2-lane (`0x7`) | `hi556` (holitech, 5MP), `gc5025` (ofilm, 5MP) | EEPROM fm24c64d |

I²C bus: all on **CCI** (QCOM camera control interface); master per the DT below.

## 2. Platform wiring — from the stock `boot.img` DTB (`qcom,camera@N`)

Carved from the appended DTB in `boot.img` and decompiled with `dtc`.
(`qcom,camera@N` platform index ↔ logical slot; CSID per §1.)

| DT node | Maps to | CSIPHY | CCI (I²C) master | MCLK | RESET | VANA-en | VDIG-en | MCLK clk |
|---|---|---|---|---|---|---|---|---|
| `qcom,camera@0` | rear main | 0 | 0 | TLMM **GPIO32** | TLMM **GPIO46** | TLMM **GPIO51** | PMIC gpio p4 | 24 MHz |
| `qcom,camera@1` | depth (BACK_AUX) | 1 | 1 | TLMM **GPIO34** | TLMM **GPIO48** | TLMM **GPIO51** | PMIC gpio p3 | 24 MHz |
| `qcom,camera@2` | front | 2 | 1 | TLMM **GPIO33** | TLMM **GPIO47** | TLMM **GPIO51** | PMIC gpio p3 | 24 MHz |

Shared regulators (from `qcom,cam-vreg-name`): **`cam_vio`** (1.8 V I/O),
**`cam_vana`** (2.8 V analog), **`cam_vdig`** (~1.1 V digital). GPIO labels in the
DTB: `CAMIF_MCLK*`, `CAM_RESET*`, `CAM_VDIG`, `CAM_VANA`.

## 3. Mainline driver availability (checked vs `torvalds/linux` master)

| Sensor | Role | Mainline `drivers/media/i2c/` | Porting effort |
|---|---|---|---|
| **OV8856** | **front** | ✅ `ov8856.c` exists | **low** — DT only |
| OV5670 | (depth tuning present, not bound) | ✅ `ov5670.c` | n/a |
| OV13858 | close relative of rear OV13855 | ✅ `ov13858.c` | adapt for OV13855 |
| **OV13855** | rear main (13MP) | ❌ none | port (base on ov13858) |
| **OV16885 / OV16880** | rear main (16MP) | ❌ none | write driver |
| **HI556 / GC5025** | depth (5MP) | ❌ none | write driver |
| HI1333 | rear main alt (13MP) | ❌ none | write driver |

**Easiest first win: the front camera (OV8856)** — mainline driver already
exists, so it's device-tree + bring-up only.

## 4. Port plan

1. **Front (OV8856)** — add `camss` + sensor DT nodes (CSIPHY2, CSID2, 4-lane,
   GPIO33/47, cam_vio/vana/vdig, 24 MHz), enable `CONFIG_VIDEO_OV8856`,
   rebuild kernel, test raw capture (`media-ctl`, `v4l2-ctl`).
2. **Rear main (OV13855)** — port a driver from the mainline `ov13858` as a base;
   DT: CSIPHY0, CSID0, 4-lane, GPIO32/46.
3. **Depth (HI556/GC5025)** — lowest priority; write a minimal V4L2 driver.
4. For each: `camss` pipeline in libcamera + a tuning file for usable frames.

CSIPHY/CSID/VFE platform nodes come from the mainline `sdm660.dtsi` (CAMSS is
already `CONFIG_VIDEO_QCOM_CAMSS=m` in the pmOS kernel).

## 5. Firmware blobs in the stock ROM (for WiFi/BT, not camera)

From the same firmware zip (`firmware-update/`): `BTFM.bin` (Bluetooth),
`NON-HLOS.bin` (modem/WCNSS incl. WiFi). Camera needs **no firmware blob** —
it's driver + DT only.

## 6. How this was extracted (reproducible)

```sh
# DTB (platform wiring) from boot.img:
unzip FW.zip boot.img
# carve appended DTBs by magic d00dfeed, then: dtc -I dtb -O dts dtb_N.dtb

# sensor binding from vendor.img:
unzip FW.zip vendor.new.dat.br vendor.transfer.list
brotli -d vendor.new.dat.br
python3 sdat2img.py vendor.transfer.list vendor.new.dat vendor.img
sudo mount -o ro,loop vendor.img /mnt
cat /mnt/etc/camera/camera_config.xml          # slot -> sensor binding
ls  /mnt/lib64 | grep libmmcamera_             # sensor driver libs
```

## 7. Mainline bring-up — front OV8856 (the first target)

Verified against current mainline (the pmOS edge kernel is 7.2.x):

- **CAMSS supports sdm660**: `qcom,sdm660-camss` is in the `camss` driver
  (`CAMSS_660`: 3 CSIPHYs, 4 CSIDs, 2 VFEs).
- **`sdm630.dtsi` already has the blocks**: `camss@ca00020` + `cci@ca0c000`
  (with `cci_i2c0`/`cci_i2c1` and an empty `ports` node), both `status =
  "disabled"`. Pinctrl `cci1_default` = GPIO38/39 (CCI master 1 = front).
- **OV8856 has a mainline driver** (`ovti,ov8856`). So the front camera is
  **device-tree only** — no driver to write.

Scaffold in this repo:
- `os/camera/sdm660-x00td-ov8856-front.dtsi` — enables `&camss`/`&cci`, adds the
  OV8856 node on `cci_i2c1` + the CSIPHY2 endpoint (TODOs: exact MCLK clock +
  pinctrl, PMIC regulator phandles, I2C addr, link-frequencies).
- `os/camera/ov8856.kconfig` — enables `VIDEO_QCOM_CAMSS` + `VIDEO_OV8856`.

Integrate + test (needs the phone):
```sh
# 1. add the .dtsi to the x00td board DTS (include it), apply the kconfig:
pmbootstrap kconfig edit linux-postmarketos-qcom-sdm660   # merge ov8856.kconfig
# 2. rebuild + flash:
pmbootstrap build --force linux-postmarketos-qcom-sdm660
# 3. on-device:
dmesg | grep -iE 'camss|csiphy|ov8856'    # did the sensor probe + link?
media-ctl -p                               # inspect the media graph
v4l2-ctl --list-devices
v4l2-ctl -d /dev/videoX --stream-mmap --stream-to=frame.raw --stream-count=1
```
Iterate on the TODOs (MCLK freq 19.2 vs 24 MHz, I2C addr, regulators) using the
`dmesg` errors until a raw frame is captured. Rear (OV13855) and depth
(HI556/GC5025) follow once the front path works.


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

## 7. Mainline bring-up — front OV8856 (implemented)

Verified against the actual pmOS kernel source
(`github.com/sdm660-mainline/linux`, tag `v7.0.14-sdm660`):

- **CAMSS supports sdm660**: `qcom,sdm660-camss` (`CAMSS_660`: 3 CSIPHYs,
  4 CSIDs, 2 VFEs). Config already ships `CONFIG_VIDEO_QCOM_CAMSS=m`.
- **Base blocks exist**: `camss@ca00020` + `cci@ca0c000` (`cci_i2c0`/`cci_i2c1`),
  both `status = "disabled"`; board DTS enables + wires them.
- **OV8856 has a mainline driver** (`ovti,ov8856`) → front camera is
  **device-tree only**, no driver to write. We flip `CONFIG_VIDEO_OV8856=m`.
- **Board DTS** is `sdm636-asus-x00td.dts` (→ `sdm636.dtsi` → `sdm660.dtsi`).

### Resolved wiring (no placeholders)

The downstream rails were traced through the stock DTB phandles to the real
mainline `pm660`/`pm660l` regulators defined in the board DTS:

| sensor pin | stock rail | mainline phandle |
|---|---|---|
| DOVDD 1.8 V | `pm660_l11` | `vreg_l11a_1p8` |
| DVDD 1.2 V | `pm660_s5` | `vreg_s5a_1p35` |
| AVDD 2.8 V | BoB + TLMM gpio51 enable | `regulator-fixed` (`vin = vreg_bob`) |
| MCLK (gpio33) | cam mclk1 | `&mmcc CAMSS_MCLK1_CLK` |
| CAMSS vdda | — | `vreg_l1a_1p225` |
| RESET | gpio47 (active low) | `&tlmm 47` |
| data path | CSIPHY2 / CCI master 1 / 4-lane D-PHY | `port@2` ↔ `camera@10` |

### Files in this repo

- `os/camera/sdm636-asus-x00td-camera.dtsi` — the complete, compilable fragment
  (`&cci`/`&cci_i2c1`/`&camss` enabled, OV8856 `camera@10`, CSIPHY2 endpoint,
  `cam_vana_front` fixed-regulator, `cam_front_mclk` pinctrl).
- `os/camera/integrate.py` — patches the kernel aport: adds the dtsi to
  `source=` and extends `prepare()` to `#include` it from the board DTS.
- `os/camera/ov8856.kconfig` — the config delta (`VIDEO_OV8856=m`).

### Build integration (what `integrate.py` automates)

```sh
A=~/.local/var/pmbootstrap/.../linux-postmarketos-qcom-sdm660
cp os/camera/sdm636-asus-x00td-camera.dtsi "$A"/
python3 os/camera/integrate.py "$A"                 # source= + prepare() #include
sed -i 's/^# CONFIG_VIDEO_OV8856 is not set/CONFIG_VIDEO_OV8856=m/' \
    "$A"/config-postmarketos-qcom-sdm660.aarch64
pmbootstrap checksum linux-postmarketos-qcom-sdm660
pmbootstrap build --force linux-postmarketos-qcom-sdm660   # compiles DTB + driver
```

The DTB compiling clean (dtc resolves every phandle) is the build-side proof.

### On-device bring-up (needs the phone)

```sh
dmesg | grep -iE 'camss|csiphy|ov8856'     # did the sensor probe + link come up?
media-ctl -p                                # inspect the media graph
v4l2-ctl --list-devices
v4l2-ctl -d /dev/videoX --stream-mmap --stream-to=frame.raw --stream-count=1
```

Remaining values that can only be confirmed on hardware (do **not** affect the
DTB compile): OV8856 I²C address (`0x10`), MCLK 19.2 vs 24 MHz, link-frequencies,
and the `gpio33` `cam_mclk` pinmux function name. Rear (OV13855, needs an
`ov13858`-based driver port) and depth (HI556/GC5025) follow the same pattern.

---

## 8. Build log — what we added, how, and what's missing

### 8.1 What we added

Three files under `os/camera/` make up the front-camera (OV8856, 8 MP) bring-up:

- `os/camera/sdm636-asus-x00td-camera.dtsi` — complete, compilable DT fragment
  appended to the board DTS `sdm636-asus-x00td.dts`. It enables `&cci`,
  `&cci_i2c1`, and `&camss`; adds the OV8856 sensor node `camera@10` on CCI
  master 1; defines a `cam_vana_front` fixed-regulator (2.8 V, enabled by TLMM
  `gpio51`, fed from `vreg_bob`); adds a `cam_front_mclk` pinctrl state for
  `gpio33`; and wires the CSIPHY2 endpoint (`port@2`) to the sensor's 4-lane
  D-PHY endpoint.
- `os/camera/integrate.py` — idempotent script that patches the pmaports kernel
  aport `linux-postmarketos-qcom-sdm660`: adds the dtsi to `source=` and extends
  `prepare()` to install the dtsi into the kernel tree and append a trailing
  `#include "sdm636-asus-x00td-camera.dtsi"` to the board DTS.
- `os/camera/ov8856.kconfig` — the kernel config delta; the key line is
  `CONFIG_VIDEO_OV8856=m` (CAMSS was already `CONFIG_VIDEO_QCOM_CAMSS=m` in the
  stock pmOS config).

### 8.2 How we added it (in order)

1. **Carved the authoritative facts from the stock images.** Extracted the
   appended DTBs from the stock `boot.img`, decompiled `dtb_0.dts` with `dtc`,
   and mounted the stock `vendor.img` to read `camera_config.xml`. This gave the
   per-slot sensor + pin + rail facts.
2. **Resolved the downstream rails to real mainline regulators** by following the
   stock DTB phandles:
   - `cam_vio` → phandle `0x1bb` = `pm660_l11` = mainline `vreg_l11a_1p8`
     (1.8 V DOVDD)
   - `cam_vdig` → phandle `0x1bc` = `pm660_s5` = `vreg_s5a_1p35` (1.2 V DVDD)
   - `cam_vana` → phandle `0x9f` = `pm660l_bob` = `vreg_bob` (the 2.8 V AVDD is
     an external LDO fed from BoB, enabled by TLMM `gpio51`)
3. **Identified the mainline board DTS** as `sdm636-asus-x00td.dts` (from
   `github.com/sdm660-mainline/linux` tag `v7.0.14-sdm660`, the exact kernel the
   pmOS aport builds). It includes `sdm636.dtsi` → `sdm660.dtsi` and defines the
   `pm660`/`pm660l` regulators (`vreg_l11a_1p8`, `vreg_s5a_1p35`, `vreg_bob`,
   `vreg_l1a_1p225`).
4. **Confirmed mainline CAMSS supports this SoC**: `qcom,sdm660-camss` (version
   `CAMSS_660`: 3 CSIPHYs, 4 CSIDs, 2 VFEs). The base `camss@ca00020` and
   `cci@ca0c000` nodes ship `status = "disabled"` with `cci_i2c0`/`cci_i2c1`
   buses — the board DTS just enables and wires them.
5. **Mapped the front MCLK** (TLMM `gpio33` = cam mclk1) to the clock macro
   `CAMSS_MCLK1_CLK` from `<dt-bindings/clock/qcom,mmcc-sdm660.h>`; set it to
   19.2 MHz (the rate the mainline ov8856 register tables expect; the stock ROM
   used 24 MHz).
6. **Wrote + integrated + built**: wrote the DT fragment, integrated it via
   `integrate.py`, flipped `CONFIG_VIDEO_OV8856=m`, then ran
   `pmbootstrap checksum` followed by
   `pmbootstrap build --force --arch aarch64 linux-postmarketos-qcom-sdm660`.

### 8.3 Resolved wiring table

| Signal | Value | Mainline source |
|---|---|---|
| DOVDD 1.8 V | `vreg_l11a_1p8` | `pm660_l11` |
| DVDD 1.2 V | `vreg_s5a_1p35` | `pm660_s5` |
| AVDD 2.8 V | `cam_vana_front` fixed-reg | `gpio51` enable, `vin = vreg_bob` |
| MCLK (`gpio33`) | `&mmcc CAMSS_MCLK1_CLK` @ 19.2 MHz | `qcom,mmcc-sdm660.h` |
| CAMSS vdda | `vreg_l1a_1p225` | `pm660` |
| RESET | `&tlmm 47`, active-low | board DTS |
| Data path | CSIPHY2 / CCI master 1 / 4-lane D-PHY | `port@2` ↔ `camera@10` |

### 8.4 Build proof (this session)

- `prepare()` correctly appended the `#include` to the board DTS.
- The DTB `sdm636-asus-x00td.dtb` compiled successfully — `dtc` resolved every
  phandle. Decompiling the built DTB confirmed:
  - `camss@ca00020` and `cci@ca0c000` enabled;
  - `camera@10 { compatible = "ovti,ov8856"; reg = <0x10>; }` present;
  - resolved `dovdd/avdd/dvdd-supply`;
  - `clock-frequency = 19200000` (`0x124f800`);
  - `reset-gpios = <&tlmm 47 GPIO_ACTIVE_LOW>`;
  - `cam_vana_front_2v8` regulator present.
- The full kernel image + `ov8856.ko` module build was **still finishing at
  doc-writing time**. The DTB compile is proven; the full-image/module build was
  in progress.

### 8.5 What is still missing

- **On-device verification** — a real phone is **required** (QEMU cannot emulate
  CAMSS). Unverified values that do **not** affect the DTB compile but may need
  tuning: OV8856 I²C address (currently `0x10`), MCLK 19.2 vs 24 MHz,
  `link-frequencies` (currently 360 MHz), and the `gpio33` `cam_mclk` pinmux
  function name.
- **No frame captured yet.** The on-device loop is:

  ```sh
  # flash the kernel, then:
  dmesg | grep -iE 'camss|csiphy|ov8856'     # did probe + link come up?
  media-ctl -p                                # inspect the media graph
  v4l2-ctl -d /dev/videoX --stream-mmap       # try to pull a frame
  ```

  Iterate on the dmesg errors.
- **Only the FRONT camera is done.** REAR main (OV13855) needs a driver port
  based on mainline `ov13858` plus its own DT (CSIPHY0 / CSID0 / 4-lane /
  `gpio32` / `gpio46`). DEPTH (HI556/GC5025) needs a minimal V4L2 driver (lowest
  priority).
- **libcamera pipeline + a camera tuning file** are needed for usable
  (non-raw) images.
- **No firmware blob needed** — the camera is driver + DT only, unlike WiFi/BT,
  which pull firmware from the stock ROM (documented in `flash.md`).


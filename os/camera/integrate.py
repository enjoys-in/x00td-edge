#!/usr/bin/env python3
"""Integrate the front OV8856 camera device-tree into the pmaports
linux-postmarketos-qcom-sdm660 kernel aport.

Usage: integrate.py <path-to-kernel-aport-dir>

It is idempotent: adds the dtsi to source=, extends prepare() to install the
dtsi and append a trailing #include to the board DTS, and leaves the config /
checksum steps to the caller (pmbootstrap checksum).
"""
import sys
import pathlib

aport = pathlib.Path(sys.argv[1])
apkbuild = aport / "APKBUILD"
s = apkbuild.read_text()

# 1) add the dtsi to source=
src_needle = "        config-$_flavor.aarch64\n\"\n"
if "sdm636-asus-x00td-camera.dtsi" not in s:
    assert src_needle in s, "source block not found"
    s = s.replace(
        src_needle,
        "        config-$_flavor.aarch64\n"
        "        sdm636-asus-x00td-camera.dtsi\n\"\n",
        1,
    )

# 2) extend prepare() to drop in the dtsi + append the include once
old = (
    'prepare() {\n'
    '        default_prepare\n'
    '        cp -v "$srcdir/config-$_flavor.$CARCH" "$builddir"/.config\n'
    '}'
)
inc = '#include "sdm636-asus-x00td-camera.dtsi"'
new = (
    'prepare() {\n'
    '        default_prepare\n'
    '        cp -v "$srcdir/config-$_flavor.$CARCH" "$builddir"/.config\n'
    '\n'
    '        # enjoys-os: front OV8856 camera device-tree\n'
    '        local _dts="$builddir/arch/arm64/boot/dts/qcom/sdm636-asus-x00td.dts"\n'
    '        install -Dm644 "$srcdir/sdm636-asus-x00td-camera.dtsi" \\\n'
    '                "$builddir/arch/arm64/boot/dts/qcom/sdm636-asus-x00td-camera.dtsi"\n'
    f'        grep -q "sdm636-asus-x00td-camera.dtsi" "$_dts" || \\\n'
    f'                echo \'{inc}\' >> "$_dts"\n'
    '}'
)
if "enjoys-os: front OV8856" not in s:
    assert old in s, "prepare() not matched"
    s = s.replace(old, new, 1)

apkbuild.write_text(s)
print("APKBUILD patched OK")

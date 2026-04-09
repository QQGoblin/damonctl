#!/usr/bin/env python3
"""
Scan /proc for qemu-system-x86_64 processes,
extract guest UUID from -name guest=<uuid>,... argument,
and generate procmon.json.

Compatible with Python 3.6+.
"""

import json
import os
import re

QEMU_BIN = "/usr/bin/qemu-system-x86_64"
GUEST_RE = re.compile(r"-name\x00guest=([0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12})")


def scan_qemu_processes():
    entries = []
    for pid_dir in os.listdir("/proc"):
        if not pid_dir.isdigit():
            continue
        cmdline_path = "/proc/{}/cmdline".format(pid_dir)
        try:
            with open(cmdline_path, "rb") as f:
                cmdline = f.read()
        except (IOError, OSError):
            continue

        # cmdline is null-byte separated
        if QEMU_BIN.encode() not in cmdline:
            continue

        cmdline_str = cmdline.decode("utf-8", errors="replace")
        m = GUEST_RE.search(cmdline_str)
        if m:
            entries.append({"pid": int(pid_dir), "name": m.group(1)})

    entries.sort(key=lambda e: e["pid"])
    return entries


def main():
    entries = scan_qemu_processes()
    print(json.dumps(entries, indent=4))


if __name__ == "__main__":
    main()

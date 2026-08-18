#!/bin/sh
# Reproduce issue #71 (EXDEV self-update) and verify replaceBinary copies instead.
set -eu

ROOT=$(CDPATH= cd -- "$(dirname "$0")/.." && pwd)
cd "$ROOT"

SRC=${CROSS_DEVICE_SRC:-}
DEST=${CROSS_DEVICE_DEST:-}

if [ -z "$SRC" ] || [ -z "$DEST" ]; then
	SRC=/mnt/redis-tui-exdev-src
	DEST=/var/tmp/redis-tui-exdev-dest
	if [ "$(id -u)" -eq 0 ]; then
		mkdir -p "$SRC"
		mount -t tmpfs -o size=64M,exec tmpfs "$SRC"
	elif command -v sudo >/dev/null 2>&1; then
		sudo mkdir -p "$SRC"
		sudo mount -t tmpfs -o size=64M,exec tmpfs "$SRC"
		sudo chmod 1777 "$SRC"
	else
		echo "set CROSS_DEVICE_SRC and CROSS_DEVICE_DEST, or run as root/sudo" >&2
		exit 1
	fi
fi

mkdir -p "$SRC" "$DEST"

export CROSS_DEVICE_SRC=$SRC
export CROSS_DEVICE_DEST=$DEST
export REQUIRE_CROSS_DEVICE=1

echo "src=$SRC dest=$DEST"

PROBE_SRC=$SRC/rename-probe
PROBE_DEST=$DEST/rename-probe
printf 'new-binary\n' > "$PROBE_SRC"
rm -f "$PROBE_DEST"

echo "=== Reproducing issue #71: os.Rename across filesystems ==="
set +e
LN_ERR=$(ln "$PROBE_SRC" "$PROBE_DEST" 2>&1)
LN_STATUS=$?
set -e
if [ "$LN_STATUS" -eq 0 ]; then
	echo "FAIL: hard link succeeded; $SRC and $DEST are on the same device" >&2
	exit 1
fi
echo "hard link failed as expected: $LN_ERR"

echo "=== Verifying replaceBinary copies on EXDEV ==="
echo "=== and that 1.0.42 can escape via same-device TMPDIR ==="
go test -count=1 -race -v -run 'TestReplaceBinary_RealCrossDevice|TestReplaceBinaryLegacy_SameDevice|TestUpdateTempDir' .

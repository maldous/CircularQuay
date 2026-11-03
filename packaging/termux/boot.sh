#!/data/data/com.termux/files/usr/bin/sh
set -eu

# Example for ~/.termux/boot/ after compatible Termux:Boot installation and one
# initial launch. Standard Termux cannot guarantee execution before first unlock.
# No key values belong in this file or shared /sdcard storage.
if command -v termux-wake-lock >/dev/null 2>&1; then
    termux-wake-lock
fi
if [ -f "$PREFIX/etc/profile.d/start-services.sh" ]; then
    . "$PREFIX/etc/profile.d/start-services.sh"
fi

#!/bin/bash
set -ex

function init_dbus() {
    if pgrep dbus-daemon; then
        echo "Dbus is running"
        return 0
    fi

    echo "Startup dbus"
    mkdir -p /var/lib/dbus
    cp -f /etc/machine-id /var/lib/dbus/machine-id

    # cleanups
    mkdir -p /run/dbus
    rm -f /run/dbus/pid

    # run
    dbus-daemon --system --print-address
}

init_dbus
cp contrib/io.hass.conf /etc/dbus-1/system.d/

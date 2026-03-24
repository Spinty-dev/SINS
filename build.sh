#!/bin/bash
# SINS - SINS Is Not Systemd
# Modular build script

set -e

# Support for non-interactive build (e.g. PKGBUILD) via env vars
if [[ -z "$SINS_CHOICE" ]]; then
    if [[ "$1" == "--verify" ]]; then
        echo "Verifying SINS compilation (all modules)..."
        go build -tags "dbus,notify,cgroups,timers,sockets" -o /dev/null ./...
        echo "Verification SUCCESS."
        exit 0
    fi

    echo "SINS - SINS Is Not Systemd: Modular Build"
    echo "-----------------------------------------"
    echo "Select modules to include (comma-separated, e.g. 1,3,5):"
    echo "1. D-Bus Bridge (org.freedesktop.systemd1)"
    echo "2. Socket Activation (socket binding)"
    echo "3. Timers (timer daemon)"
    echo "4. Notify Socket (/run/systemd/notify)"
    echo "5. Cgroups v2 (Resource Limits)"
    echo "0. ALL"
    echo "-----------------------------------------"
    read -p "Your choice: " SINS_CHOICE
fi

tags=""
mkdir -p build

if [[ "$SINS_CHOICE" == "0" ]]; then
    tags="dbus,notify,cgroups,timers,sockets"
else
    [[ "$SINS_CHOICE" == *"1"* ]] && tags+="dbus,"
    [[ "$SINS_CHOICE" == *"2"* ]] && tags+="sockets,"
    [[ "$SINS_CHOICE" == *"3"* ]] && tags+="timers,"
    [[ "$SINS_CHOICE" == *"4"* ]] && tags+="notify,"
    [[ "$SINS_CHOICE" == *"5"* ]] && tags+="cgroups,"
fi

tags=$(echo $tags | sed 's/,$//')

echo "Building with tags: $tags"

echo "Building systemctl..."
go build -tags "$tags" -o build/systemctl ./cmd/systemctl/main.go

if [[ "$tags" == *"dbus"* || "$tags" == *"notify"* ]]; then
    echo "Building sins-daemon..."
    go build -tags "$tags" -o build/sins-daemon ./cmd/sins-daemon/main.go
fi

if [[ "$tags" == *"sockets"* ]]; then
    echo "Building sins-sockets..."
    go build -tags "$tags" -o build/sins-sockets ./cmd/socket-activator/main.go
fi

if [[ "$tags" == *"timers"* ]]; then
    echo "Building sins-timers..."
    go build -tags "$tags" -o build/sins-timers ./cmd/timers/main.go
fi

echo "-----------------------------------------"
echo "Success! Binaries are in build/"
ls -lh build/

#!/usr/bin/env bash
# Session-bus smoke for sins-daemon (run via dbus-run-session …).
set -euo pipefail
DAEMON="${1:?path to sins-daemon}"
export SINS_SESSION=1
"$DAEMON" &
dpid=$!
cleanup() { kill -TERM "$dpid" 2>/dev/null || true; wait "$dpid" 2>/dev/null || true; }
trap cleanup EXIT
sleep 2
tmp="$(mktemp)"
dbus-send --session --print-reply \
  --dest=org.freedesktop.systemd1 \
  /org/freedesktop/systemd1 \
  org.freedesktop.DBus.Introspectable.Introspect >"$tmp"
grep -q 'interface name=' "$tmp" || grep -q '<node' "$tmp"
rm -f "$tmp"

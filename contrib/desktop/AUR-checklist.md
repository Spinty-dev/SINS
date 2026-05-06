# AUR / `PKGBUILD` expectations vs SINS

Typical friction points when a package assumes **real systemd**:

| Expectation | With SINS (`SINS_PROFILE=de` or `full`) |
|-------------|----------------------------------------|
| `systemctl` exists | Yes (`/usr/bin/systemctl` from this project) |
| `pkg-config --exists libsystemd` | Often yes if `libsystemd.so` is the SINS shim on `LD_LIBRARY_PATH` or `/usr/lib`; otherwise install/link the built `.so` and a `.pc` if the build insists |
| postinstall calls `systemctl daemon-reload` / `try-restart` | Shim implements common commands; logs improve with `systemctl status` (runit + logs) |
| user units (`systemctl --user`) | Partial: **read** `.config/systemd/user` for `status`/`cat`; **start/enable** need system runit paths — see main `README` |
| hard dependency on **systemd** PID1 | Not satisfied — this is intentional; use Artix wiki / IgnorePkg as documented |

Quick link test after `makepkg` or `./build.sh`:

```bash
ldd /usr/lib/libsystemd.so.0   # or build/libsystemd.so.0
pkg-config --libs libsystemd    # may need a .pc from libelogind or a one-liner -L/usr/lib -lsystemd
```

If an AUR build fails only on `systemd` version checks, patch or report upstream; SINS does not fake full `systemd-analyze` / `sd_booted()` semantics.

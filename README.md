# SINS - SINS Is Not Systemd

**SINS** is a modular, lightweight compatibility layer that bridges the gap between `runit` and `systemd`. It allows you to run `systemd`-dependent software (like Nginx, Docker, or GNOME services) on `runit`-based systems by providing a `systemctl` shim and background daemons for D-Bus, Notify signals, and Cgroups resource management.

---

[English](README.md) | [Русский](README.ru.md)

---

## 🚀 Key Features

- **Standard CLI**: Full `systemctl` command set (`start`, `stop`, `status`, `enable`, `disable`).
- **D-Bus Bridge**: Implements `org.freedesktop.systemd1` so external tools (like `busctl` or installers) see your services.
- **Socket Activation**: Support for `.socket` units and FD 3 passing.
- **Timer Support**: Native `.timer` unit scheduling via a dedicated daemon.
- **Notify Protocol**: Full support for `Type=notify` services via `/run/systemd/notify`.
- **Cgroups v2 Integration**: Automated resource limiting (`MemoryMax`, `CPUQuota`) via `/sys/fs/cgroup/sins/`.
- **Modular Build**: Choose exactly which modules to compile using Go Build Tags.

---

## 🐧 Supported Distributions

SINS is designed for **Linux** distributions running **runit** as the init system or service manager:
- **Artix Linux** (runit flavor)
- **Void Linux**
- **Arch Linux** (with custom runit setup)
- **Devuan** (runit flavor)

---

## 🛠️ Building & Installation

### Dependencies
- **Go** (1.25+ recommended)
- **Runit**
- **D-Bus** (for the D-Bus bridge module)

### 1. Interactive Build
Run the build script to select which modules you want to include:
```bash
chmod +x build.sh
./build.sh
```
Follow the menu prompts (e.g., enter `0` for everything or `1,5` for D-Bus + Cgroups).

### 2. Manual/Non-Interactive Build
Use the `SINS_CHOICE` environment variable:
```bash
# Build everything
SINS_CHOICE=0 ./build.sh
```

### 3. Installation (Arch/Artix)
For AUR-compatible distributions, use the provided `PKGBUILD`:
```bash
makepkg -si
```

---

## 🧬 Usage

### Managing Services
Use the `systemctl` binary as usual:
```bash
systemctl start nginx
systemctl status nginx -f
```

### SINS Daemon
If you enabled D-Bus or Notify support, ensure `sins-daemon` is running in the background. You can set it up as a runit service itself:
```bash
# Create runit service for sins-daemon
mkdir -p /etc/runit/sv/sins-daemon
echo -e "#!/bin/sh\nexec sins-daemon" > /etc/runit/sv/sins-daemon/run
chmod +x /etc/runit/sv/sins-daemon/run
ln -s /etc/runit/sv/sins-daemon /var/service/
```

---

## 🧩 Modularity System
SINS uses **Go Build Tags** to keep binaries slim.
- `dbus`: Includes D-Bus bridge.
- `notify`: Includes Notify socket support.
- `timers`: Includes timer daemon logic.
- `sockets`: Includes socket activator support.
- `cgroups`: Includes Cgroups v2 limit logic.

---

## 📜 License
MIT

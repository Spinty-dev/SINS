# Maintainer: spinty <spinty@example.com>
pkgname=sins-git
pkgver=0.1
pkgrel=2
pkgdesc="SINS Is Not Systemd - a modular systemd-to-runit shim with D-Bus and Cgroups support"
arch=('x86_64' 'aarch64')
url="https://github.com/Spinty-dev/SINS"
license=('MIT')
depends=('runit' 'dbus' 'libelogind')
makedepends=('go' 'git' 'gcc')
provides=(
  'systemd'
  'systemd-libs'
  'libsystemd'
  'systemd-sysvcompat'
  'systemctl-shim'
)
conflicts=(
  'systemd'
  'systemd-libs'
  'systemd-sysvcompat'
)
source=("git+file:///home/spinty/SINS")
md5sums=('SKIP')
install=sins.install

build() {
  cd "$srcdir/SINS"
  SINS_CHOICE=0 ./build.sh
}

package() {
  cd "$srcdir/SINS"
  
  # Binaries
  install -Dm755 build/systemctl "$pkgdir/usr/bin/systemctl"
  install -Dm755 build/sins-daemon "$pkgdir/usr/bin/sins-daemon"
  install -Dm755 build/sins-timers "$pkgdir/usr/bin/sins-timers"
  install -Dm755 build/sins-sockets "$pkgdir/usr/bin/sins-sockets"

  # Library stub
  install -Dm755 build/libsystemd.so.0 "$pkgdir/usr/lib/libsystemd.so.0"
  ln -s libsystemd.so.0 "$pkgdir/usr/lib/libsystemd.so"

  # D-Bus policy
  install -Dm644 org.freedesktop.systemd1.conf "$pkgdir/usr/share/dbus-1/system.d/org.freedesktop.systemd1.conf"

  # Runit service
  install -Dm755 sins-daemon.run "$pkgdir/etc/runit/sv/sins-daemon/run"
}

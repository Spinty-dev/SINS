# Maintainer: spinty <spinty@example.com>
pkgname=sins-git
pkgver=0.1
pkgrel=1
pkgdesc="SINS Is Not Systemd - a modular systemd-to-runit shim with D-Bus and Cgroups support"
arch=('x86_64' 'aarch64')
url="https://github.com/Spinty-dev/SINS"
license=('MIT')
depends=('runit' 'dbus')
makedepends=('go' 'git')
provides=('systemd-shim' 'systemctl-shim')
conflicts=('systemd-shim' 'systemctl-shim')
source=("git+https://github.com/Spinty-dev/SINS.git")
md5sums=('SKIP')

build() {
  cd "$srcdir/sins"
  # SINS_CHOICE=0 builds all modules (D-Bus, Timers, Sockets, Notify, Cgroups)
  export SINS_CHOICE=0
  ./build.sh
}

package() {
  cd "$srcdir/sins"
  install -Dm755 build/systemctl "$pkgdir/usr/bin/systemctl"
  
  if [ -f build/sins-daemon ]; then
    install -Dm755 build/sins-daemon "$pkgdir/usr/bin/sins-daemon"
  fi
  
  if [ -f build/sins-timers ]; then
    install -Dm755 build/sins-timers "$pkgdir/usr/bin/sins-timers"
  fi
  
  if [ -f build/sins-sockets ]; then
    install -Dm755 build/sins-sockets "$pkgdir/usr/bin/sins-sockets"
  fi
}

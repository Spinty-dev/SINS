# SINS - SINS Is Not Systemd

**SINS** — это модульный и легковесный слой совместимости, который устраняет разрыв между `runit` и `systemd`. Он позволяет запускать программное обеспечение, зависящее от `systemd` (например, Nginx, Docker или сервисы GNOME), в среде `runit`, предоставляя прослойку `systemctl` и фоновые демоны для D-Bus, уведомлений Notify и управления ресурсами Cgroups.

---

[English](README.md) | [Русский](README.ru.md)

---

## 🚀 Основные возможности

- **Стандартный CLI**: Полный набор команд `systemctl` (`start`, `stop`, `status`, `enable`, `disable`).
- **D-Bus Bridge**: Реализует интерфейс `org.freedesktop.systemd1`, чтобы внешние инструменты (например, `busctl` или установщики) видели ваши сервисы.
- **Socket Activation**: Поддержка юнитов `.socket` и передача FD 3.
- **Поддержка таймеров**: Нативное планирование юнитов `.timer` через выделенный демон.
- **Протокол Notify**: Полная поддержка сервисов `Type=notify` через `/run/systemd/notify`.
- **Интеграция с Cgroups v2**: Автоматическое ограничение ресурсов (`MemoryMax`, `CPUQuota`) через `/sys/fs/cgroup/sins/`.
- **Модульная сборка**: Выбирайте нужные модули при компиляции с помощью Go Build Tags.

---

## 🐧 Поддерживаемые дистрибутивы

SINS разработан для дистрибутивов **Linux**, использующих **runit** в качестве системы инициализации или менеджера сервисов:
- **Artix Linux** (версия runit)
- **Void Linux**
- **Arch Linux** (с кастомной настройкой runit)
- **Devuan** (версия runit)

---

## 🛠️ Сборка и установка

### Зависимости
- **Go** (рекомендуется 1.25+)
- **Runit**
- **D-Bus** (для модуля D-Bus bridge)

### 1. Интерактивная сборка
Запустите скрипт сборки, чтобы выбрать нужные модули:
```bash
chmod +x build.sh
./build.sh
```
Следуйте подсказкам меню (например, введите `0` для всего или `1,5` для D-Bus + Cgroups).

### 2. Ручная/Автоматическая сборка
Используйте переменную окружения `SINS_CHOICE`:
```bash
# Собрать все модули
SINS_CHOICE=0 ./build.sh
```

### 3. Установка (Arch/Artix)
Для дистрибутивов с поддержкой AUR используйте предоставленный `PKGBUILD`:
```bash
makepkg -si
```

---

## 🧬 Использование

### Управление сервисами
Используйте бинарный файл `systemctl` как обычно:
```bash
systemctl start nginx
systemctl status nginx -f
```

### Демон SINS (sins-daemon)
Если вы включили поддержку D-Bus или Notify, убедитесь, что `sins-daemon` запущен в фоновом режиме. Вы можете настроить его как сервис runit:
```bash
# Создание сервиса runit для sins-daemon
mkdir -p /etc/runit/sv/sins-daemon
echo -e "#!/bin/sh\nexec sins-daemon" > /etc/runit/sv/sins-daemon/run
chmod +x /etc/runit/sv/sins-daemon/run
ln -s /etc/runit/sv/sins-daemon /var/service/
```

---

## 🧩 Система модульности
SINS использует **Go Build Tags**, чтобы сохранять бинарные файлы компактными.
- `dbus`: Включает bridge для D-Bus.
- `notify`: Включает поддержку сокета Notify.
- `timers`: Включает логику демона таймеров.
- `sockets`: Включает поддержку активации сокетов.
- `cgroups`: Включает логику ограничений Cgroups v2.

---

## 📜 Лицензия
MIT

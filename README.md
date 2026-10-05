# 🚀 Alirun

[English](#-english) | [Русский](#-русский)

---

## 🇬🇧 English

**Alirun** is a fast, visual, and transparent Linux service, daemon, and autostart manager. Built in **Go (Golang)**, it combines an intuitive CLI, an interactive creation wizard, and a full-screen modern TUI dashboard.

### ✨ Key Features

- **Automated Service & Timer Creation**:
  - Turn any binary, Python/Node/Bash script, or arbitrary command line into a running systemd service in seconds.
  - **Modern Cron Replacement**: First-class support for **Systemd Timers** (`--preset timer --schedule "15m"` / `"*-*-* 03:00:00"`) paired with oneshot services.
- **Transparency First**:
  - Live syntax-highlighted preview of generated `.service` and `.timer` files before touching disk.
  - Clear display of target paths (`~/.config/systemd/user/` or `/etc/systemd/system/`).
  - Option to open and customize units in `$EDITOR` (nano, vim, micro) prior to installation.
  - Explicit notification of executed system commands (`daemon-reload`, `enable --now`).
- **Live Resource Metrics & Visual Gauges**:
  - Real-time CPU% and RAM usage monitoring with Unicode sparkline graphs (` ▂▃▄▅▆▇█`) and progress gauges (`[████░░░░]`).
  - Task/thread count and precise uptime tracking.
- **Safe Non-Root by Default**: Services default to User scope (`~/.config/systemd/user/`), protecting system integrity without requiring `sudo`.
- **Multi-Init Modular Architecture**: Abstracted via `initsys.Manager` interface. Primary provider: **Systemd**, with ready modular slots for **XDG Autostart**, **OpenRC**, and **Runit**.
- **Interactive TUI Dashboard**: Split-screen view featuring a service table with timer indicators (`⏱`), details inspector with live metrics & sparklines, live `journalctl -f` log tailing, and an embedded `[N]ew Daemon` creation wizard.
- **Effortless Updates**:
  - Built-in `alirun update` command for self-updating the binary directly from GitHub Releases.
  - `make update-deps` for updating all Go dependencies in one command.

### 🛠 Installation & Build

```bash
git clone https://github.com/khameleonium/alirun.git
cd alirun

# Build optimized binary bin/alirun
make build

# Install to ~/.local/bin/alirun (for current user)
make install

# System-wide installation (requires sudo)
sudo make install-system
```

### 📖 CLI Usage

```bash
# Interactive wizard from a script
alirun create ./myscript.sh

# Create daemon from an arbitrary command
alirun create "python3 -m http.server 8080" --name my-web --preset web --now

# List active user services
alirun list

# Filter services by name or description
alirun list --search telegram

# Inspect service status and recent journal logs
alirun status my-web

# Real-time log streaming
alirun logs my-web -f

# Service lifecycle control
alirun start my-web
alirun stop my-web
alirun restart my-web
alirun enable my-web
alirun disable my-web
alirun delete my-web
```

### 🖥 TUI Dashboard

Launch without arguments:
```bash
alirun
```

**Keybindings:**
- **`N` / `C`**: Open embedded **New Daemon Creation Wizard** (with real-time unit syntax preview)
- **`↑ / ↓` (or `j / k`)**: Navigate services list
- **`Tab`**: Switch focus between Table and Logs viewport (or exit search)
- **`S` / `X` / `R`**: Start / Stop / Restart selected service
- **`E` / `D`**: Enable / Disable autostart
- **`Del` / `Backspace`**: Delete service with confirmation
- **`U`**: Toggle User mode (`~/.config/systemd/user/`) and System mode (`/etc/systemd/system/`)
- **`/`**: Search/filter services (press `Tab`, `Enter`, or `↓/↑` to navigate results, `Esc` to clear)
- **`Esc`**: Clear search filter / Cancel wizard
- **`Q` / `Ctrl+C`**: Quit

---

## 🇷🇺 Русский

**Alirun** — быстрый, наглядный и прозрачный менеджер системных служб, демонов и автозапуска в Linux. Программа создана на **Go (Golang)** и объединяет удобный CLI, интерактивный пошаговый мастер (Wizard) и полноэкранный TUI-дашборд.

### ✨ Ключевые особенности

- **Автоматическое создание служб и таймеров**:
  - Превращение любого бинарника, Python/Node/Bash скрипта или команды в службу systemd за пару секунд.
  - **Современная замена cron**: нативная поддержка **Systemd Timers** (`--preset timer --schedule "15m"` / `"*-*-* 03:00:00"`).
- **Принцип абсолютной прозрачности (Transparency First)**:
  - Предпросмотр сгенерированных `.service` и `.timer` файлов с подсветкой синтаксиса перед записью на диск.
  - Наглядное указание путей (`~/.config/systemd/user/` или `/etc/systemd/system/`).
  - Возможность открыть конфиг в `$EDITOR` (nano, micro, vim) прямо перед сохранением.
  - Четкий список выполняемых системных команд (`daemon-reload`, `enable --now`).
- **Мониторинг ресурсов и живые графики**:
  - Отображение загрузки CPU% и памяти RAM в реальном времени с прогресс-барами (`[████░░░░]`) и Unicode-спарклайнами истории (` ▂▃▄▅▆▇█`).
  - Подсчет задач/потоков (Tasks) и точное время непрерывной работы (Uptime).
- **Безопасный User-Mode по умолчанию**: службы создаются без прав `root` / `sudo` в каталоге пользователя, защищая систему от случайных поломок.
- **Мульти-инит архитектура**: ядро абстрагировано через интерфейс `initsys.Manager`. Основной модуль — **Systemd**, с готовой модульной структурой для **XDG Autostart**, **OpenRC** и **Runit**.
- **Красивый TUI-дашборд**: раздельный экран с деревом служб и таймеров (`⏱`), инспектором свойств с живыми графиками, окном живых логов `journalctl -f` и встроенным мастером создания `[N]ew Daemon`.
- **Простота обновлений**:
  - Встроенная команда `alirun update` для самообновления бинарника из GitHub Releases.
  - Команда `make update-deps` для мгновенного обновления зависимостей проекта.

### 🛠 Установка и сборка

```bash
git clone https://github.com/khameleonium/alirun.git
cd alirun

# Сборка бинарника bin/alirun
make build

# Установка в ~/.local/bin (для текущего пользователя)
make install

# Общесистемная установка (требует sudo)
sudo make install-system
```

### 📖 Примеры использования (CLI)

```bash
# Запуск интерактивного мастера с предпросмотром
alirun create ./myscript.sh

# Создание демона из произвольной команды
alirun create "python3 -m http.server 8080" --name my-web --preset web --now

# Список активных служб пользователя
alirun list

# Поиск по имени или описанию
alirun list --search telegram

# Детальный статус и последние логи
alirun status my-web

# Живой стриминг логов в реальном времени
alirun logs my-web -f

# Управление жизненным циклом
alirun start my-web
alirun stop my-web
alirun restart my-web
alirun enable my-web
alirun disable my-web
alirun delete my-web
```

### 🖥 Полноэкранный TUI-интерфейс

Запустите команду без аргументов:
```bash
alirun
```

**Горячие клавиши:**
- **`N` / `C`**: Открыть встроенный **Мастер создания нового демона** (с live-preview)
- **`↑ / ↓` (или `j / k`)**: Навигация по списку служб
- **`Tab`**: Переключение фокуса между таблицей и логами
- **`S` / `X` / `R`**: Запустить / Остановить / Перезапустить
- **`E` / `D`**: Включить / Выключить автозапуск
- **`Del` / `Backspace`**: Удалить службу с подтверждением
- **`U`**: Переключить режим (User / System)
- **`/`**: Поиск и фильтрация списка (при этом `Tab`, `Enter` или `↓/↑` переключают фокус на найденные службы, а `Esc` сбрасывает фильтр)
- **`Esc`**: Сброс фильтра / Отмена создания
- **`Q` / `Ctrl+C`**: Выход

---

## 🧩 Архитектура проекта

```
alirun/
├── cmd/                      # CLI команды (Cobra + Huh)
│   ├── alirun/main.go        # Точка входа
│   ├── root.go               # Запуск TUI по умолчанию, глобальные флаги
│   ├── create.go             # Интерактивный визард создания юнита
│   ├── list.go               # Таблица служб (Lip Gloss)
│   ├── status.go             # Детальный статус и инспектор
│   ├── control.go            # start/stop/restart/enable/disable/delete
│   ├── logs.go               # Стриминг journalctl
│   ├── tui.go                # Вызов TUI-дашборда
│   └── version.go            # Команды version и update
├── internal/tui/             # Полноэкранный TUI (Bubble Tea)
│   ├── app.go                # Реактивная модель, стейт и цикл событий
│   ├── metrics.go            # Мониторинг CPU%/RAM, спарклайны и прогресс-бары
│   └── styles.go             # Цвета, границы и бейджи
├── pkg/
│   ├── initsys/              # Модульное ядро мульти-инит систем
│   │   ├── manager.go        # Интерфейс initsys.Manager
│   │   ├── registry.go       # Реестр и автоопределение инит-системы
│   │   ├── systemd/          # Провайдер Systemd (D-Bus API, go-systemd)
│   │   ├── xdg/              # Провайдер XDG (~/.config/autostart/)
│   │   └── openrc/           # Провайдер OpenRC (модульная заготовка)
│   ├── detector/             # Анализ файлов, shebang, chmod, virtualenv
│   ├── highlighter/          # Подсветка синтаксиса и diff (Chroma)
│   └── updater/              # Самообновление бинарника (go-selfupdate)
├── Makefile                  # Сборка, тесты, установка, обновление
└── go.mod                    # Go зависимости
```

---

## 📄 Лицензия / License

Распространяется под лицензией MIT. Подробности в файле [LICENSE](LICENSE).

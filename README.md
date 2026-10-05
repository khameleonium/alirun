# 🚀 Alirun

> **Создатель: Илья Ульянов | khameleonium** ([@khameleonium](https://github.com/khameleonium))  
> **Author / Creator: Ilya Ulyanov | khameleonium**

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

# List active user services (compact view)
alirun list

# Detailed view with CPU, RAM, Uptime, and PID
alirun list -d

# Sort services by CPU, RAM, Uptime, or Status
alirun list --sort cpu -d
alirun list --sort ram -d
alirun list --sort uptime -r

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

# Cron Jobs Management (Classic Crontab)
alirun cron list                        # List user cron jobs with human-readable schedules & next run
alirun cron list --system               # List system crontab jobs (/etc/crontab and /etc/cron.d/*)
alirun cron add -s "*/15 * * * *" -c "/backup.sh" -m "DB backup"  # Add new cron job
alirun cron toggle cron-1               # Enable / Disable job (comments out with # without deleting)
alirun cron run cron-1                  # Test run a cron job command immediately on demand
alirun cron remove cron-1               # Delete cron job from crontab
```

### 🖥 TUI Dashboard

Launch without arguments:
```bash
alirun
```

**Keybindings:**
- **`M`**: Switch active **Manager** (**Systemd** ⟷ **Cron**)
- **`N` / `C`**: Open embedded **New Daemon / Job Creation Wizard** (with real-time syntax preview)
- **`V`**: Toggle **Table View Mode** (Compact vs Detailed with CPU, RAM, Uptime, PID)
- **`O` / `P`**: Cycle **Sort Field** (Name, Status, Start, Uptime, CPU, RAM) / Reverse Sort Direction (`▲` / `▼`)
- **`1`..`6`**: Direct sort by Name (1), Status (2), Start Time (3), Uptime (4), CPU (5), RAM (6)
- **`↑ / ↓` (or `j / k`)**: Navigate services list
- **`Tab`**: Switch focus between Table and Logs viewport (or exit search)
- **`S` / `X` / `R`**: Start (or Run Now for cron) / Stop / Restart selected service
- **`E` / `D`**: Enable / Disable autostart (or comment/uncomment for cron)
- **`Del` / `Backspace`**: Delete service / cron job with confirmation
- **`U`**: Toggle User mode (`~/.config/systemd/user/` / user crontab) and System mode (`/etc/systemd/system/` / `/etc/crontab`)
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

# Список активных служб пользователя (компактный вид)
alirun list

# Подробный вид таблицы (с ЦП, ОЗУ, Аптаймом и PID)
alirun list -d

# Сортировка по нагрузке процессора, памяти, аптайму или статусу
alirun list --sort cpu -d
alirun list --sort ram -d
alirun list --sort uptime -r

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

# Управление классическими задачами Cron (crontab)
alirun cron list                        # Список пользовательских задач cron с читаемым расписанием и временем следующего запуска
alirun cron list --system               # Общесистемные задачи cron (/etc/crontab и /etc/cron.d/*)
alirun cron add -s "*/15 * * * *" -c "/backup.sh" -m "Резервное копирование БД" # Добавить новую задачу
alirun cron toggle cron-1               # Включить / Выключить (комментирует через # без удаления)
alirun cron run cron-1                  # Запустить команду cron немедленно по требованию
alirun cron remove cron-1               # Удалить задачу из crontab
```

### 🖥 Полноэкранный TUI-интерфейс

Запустите команду без аргументов:
```bash
alirun
```

**Горячие клавиши:**
- **`M`**: Переключить активный **Менеджер** (**Systemd** ⟷ **Cron**)
- **`N` / `C`**: Открыть встроенный **Мастер создания нового демона / cron-задачи** (с live-preview)
- **`V`**: Переключить **Вид таблицы** (Упрощенный / Подробный с ЦП, ОЗУ, Аптаймом и PID)
- **`O` / `P`**: Переключить **Поле сортировки** (Имя, Статус, Время запуска, Аптайм, ЦП, ОЗУ) / Направление (`▲` / `▼`)
- **`1`..`6`**: Быстрая сортировка по Имени (1), Статусу (2), Времени запуска (3), Аптайму (4), ЦП (5), ОЗУ (6)
- **`↑ / ↓` (или `j / k`)**: Навигация по списку служб
- **`Tab`**: Переключение фокуса между таблицей и логами
- **`S` / `X` / `R`**: Запустить (или Запустить прямо сейчас для cron) / Остановить / Перезапустить
- **`E` / `D`**: Включить / Выключить (раскомментировать/закомментировать для cron)
- **`Del` / `Backspace`**: Удалить службу / cron-задачу с подтверждением
- **`U`**: Переключить режим (User: `~/.config/systemd/user/` или `crontab` / System: `/etc/systemd/system/` или `/etc/crontab`)
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
│   ├── cron.go               # Управление задачами cron (list, add, toggle, run, rm)
│   ├── list.go               # Таблица служб (Lip Gloss)
│   ├── status.go             # Детальный статус и инспектор
│   ├── control.go            # start/stop/restart/enable/disable/delete
│   ├── logs.go               # Стриминг journalctl
│   ├── tui.go                # Вызов TUI-дашборда
│   └── version.go            # Команды version и update
├── internal/tui/             # Полноэкранный TUI (Bubble Tea)
│   ├── app.go                # Реактивная модель, стейт и цикл событий
│   ├── metrics.go            # Мониторинг CPU%/RAM, спарклайны и прогресс-бары
│   ├── sort.go               # Модели сортировки, переключение режимов таблицы и форматтеры
│   └── styles.go             # Цвета, границы и бейджи
├── pkg/
│   ├── initsys/              # Модульное ядро мульти-инит систем
│   │   ├── manager.go        # Интерфейс initsys.Manager
│   │   ├── registry.go       # Реестр и автоопределение инит-системы
│   │   ├── systemd/          # Провайдер Systemd (D-Bus API, go-systemd)
│   │   ├── cron/             # Провайдер Cron (crontab, /etc/crontab, parser, humanizer)
│   │   ├── xdg/              # Провайдер XDG (~/.config/autostart/)
│   │   └── openrc/           # Провайдер OpenRC (модульная заготовка)
│   ├── detector/             # Анализ файлов, shebang, chmod, virtualenv
│   ├── highlighter/          # Подсветка синтаксиса и diff (Chroma)
│   └── updater/              # Самообновление бинарника (go-selfupdate)
├── Makefile                  # Сборка, тесты, установка, обновление
└── go.mod                    # Go зависимости
```

---

## 👤 Создатель / Author

**Илья Ульянов | khameleonium**
- **GitHub**: [https://github.com/khameleonium](https://github.com/khameleonium)
- **Репозиторий**: [https://github.com/khameleonium/alirun](https://github.com/khameleonium/alirun)

---

## 📄 Лицензия / License

Распространяется под лицензией MIT. Подробности в файле [LICENSE](LICENSE).

# 🚀 Alirun

> **Создатель: Илья Ульянов | khameleonium** ([@khameleonium](https://github.com/khameleonium))  
> **Author / Creator: Ilya Ulyanov | khameleonium**

[English](#-english) | [Русский](#-русский) | [📚 Справочник по Init и Автозапуску](справочник%20--%20Системы%20инициализации%20и%20автозагрузка%20в%20Linux.md)

---

## 🇬🇧 English

**Alirun** is a fast, visual, and transparent Linux service, daemon, and autostart manager. Built in **Go (Golang)**, it combines an intuitive CLI, an interactive creation wizard, and a full-screen modern TUI dashboard.

### ✨ Key Features

- **Automated Service & Timer Creation**:
  - Turn any binary, Python/Node/Bash script, or arbitrary command line into a running systemd service in seconds.
  - **Modern Cron Replacement**: First-class support for **Systemd Timers** (`--preset timer --schedule "15m"` / `"*-*-* 03:00:00"`) paired with oneshot services.
- **XDG Desktop Autostart Management**:
  - Full support for `~/.config/autostart/*.desktop` and `/etc/xdg/autostart/`.
  - Add, toggle, run, and remove desktop autostart applications directly from CLI and TUI.
- **Quick Config Editor ($EDITOR & In-TUI F4)**:
  - Edit systemd units, crontabs, and `.desktop` files in `$EDITOR` (nano, vim, micro) with automatic daemon reload upon save.
  - Press `F4` in TUI to suspend, edit, and immediately reload selected service without restarting Alirun.
- **Listening Network Ports Inspector**:
  - Built-in pure Go `/proc/net` parser automatically detects open listening ports (`tcp:8080`, `udp:53`, etc.) for running services in both CLI and TUI.
- **Automated Diagnostic Assistant (Doctor)**:
  - Detects failed units, parses exit codes (203/EXEC, 127, 200/CHDIR, 217/USER), OOM killer terminations, and port conflicts with instant actionable fixes.
- **Portable YAML Backup & Restore**:
  - `alirun export` packages systemd units, crontabs, and desktop autostart into a single YAML archive.
  - `alirun import` restores configurations with `--dry-run` simulation support.
- **Enhanced Log Viewer**:
  - Tail `journalctl` in real-time, cycle priority filters (`L`: ALL / WARN+ERR / ERR), pause/resume streaming (`Space`), and search within logs (`/`).
- **Transparency First**:
  - Live syntax-highlighted preview of generated `.service` and `.timer` files before touching disk.
  - Clear display of target paths (`~/.config/systemd/user/` or `/etc/systemd/system/`).
  - Explicit notification of executed system commands (`daemon-reload`, `enable --now`).
- **Live Resource Metrics & Visual Gauges**:
  - Real-time CPU% and RAM usage monitoring with Unicode sparkline graphs (` ▂▃▄▅▆▇█`) and progress gauges (`[████░░░░]`).
  - Task/thread count and precise uptime tracking.
- **Safe Non-Root by Default**: Services default to User scope (`~/.config/systemd/user/`), protecting system integrity without requiring `sudo`.
- **Multi-Init Modular Architecture**: Seamless switching between **Systemd**, **Cron**, and **XDG Autostart**.
- **Effortless Updates**:
  - Built-in `alirun update` command for self-updating the binary directly from GitHub Releases.
  - `make update-deps` for updating all Go dependencies in one command.

### 📚 Init Systems & Autostart Handbook

For those who want to understand under the hood how services, daemons, cron, and autostart work in Linux:
👉 **[Справочник: Системы инициализации, демоны и автозагрузка в Linux](справочник%20--%20Системы%20инициализации%20и%20автозагрузка%20в%20Linux.md)** *(Comprehensive manual guide in Russian)*

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

# List active user services (compact or detailed with CPU, RAM, Uptime, PID)
alirun list
alirun list -d
alirun list --sort cpu -d

# Inspect service status, open listening ports, and recent journal logs
alirun status my-web

# Automated Diagnostic Assistant: identify failures & suggest fixes
alirun doctor               # diagnose all failed units
alirun doctor my-web        # deep dive into specific service

# Quick Configuration Editor ($EDITOR / nano / vim)
alirun edit my-web          # systemd unit
alirun edit                 # user crontab (when --init cron)
alirun edit app.desktop     # XDG autostart file

# Real-time log streaming
alirun logs my-web -f

# Service lifecycle control
alirun start my-web
alirun stop my-web
alirun restart my-web
alirun enable my-web
alirun disable my-web
alirun delete my-web

# XDG Desktop Autostart (~/.config/autostart/*.desktop)
alirun xdg list                                         # List autostart apps
alirun xdg add -n "MyTool" -c "/usr/local/bin/mytool"   # Create desktop autostart entry
alirun xdg toggle mytool                                # Enable / Disable autostart
alirun xdg run mytool                                   # Launch desktop app on demand
alirun xdg remove mytool                                # Delete desktop entry

# Cron Jobs Management (Classic Crontab)
alirun cron list                        # List user cron jobs with human-readable schedules & next run
alirun cron list --system               # List system crontab jobs (/etc/crontab and /etc/cron.d/*)
alirun cron add -s "*/15 * * * *" -c "/backup.sh" -m "DB backup"  # Add new cron job
alirun cron toggle cron-1               # Enable / Disable job (comments out with # without deleting)
alirun cron run cron-1                  # Test run a cron job command immediately on demand
alirun cron remove cron-1               # Delete cron job from crontab

# Portable Backup & Restore (YAML)
alirun export                           # Export user services, crontab, & autostart to YAML
alirun export backup.yaml               # Export to specific file
alirun import backup.yaml --dry-run     # Simulate restoration
alirun import backup.yaml               # Restore configurations & reload daemons
```

### 🖥 TUI Dashboard

Launch without arguments:
```bash
alirun
```

**Keybindings:**
- **`M`**: Cycle active **Manager** (**Systemd** ➔ **Cron** ➔ **XDG Autostart**)
- **`F4` / `Ctrl+E`**: Open selected service/crontab/desktop entry in **`$EDITOR`** (nano, vim) with auto daemon reload
- **`N` / `C`**: Open embedded **New Daemon / Job Creation Wizard** (with real-time syntax preview)
- **`V`**: Toggle **Table View Mode** (Compact vs Detailed with CPU, RAM, Uptime, PID)
- **`O` / `P`**: Cycle **Sort Field** (Name, Status, Start, Uptime, CPU, RAM) / Reverse Sort Direction (`▲` / `▼`)
- **`1`..`6`**: Direct sort by Name (1), Status (2), Start Time (3), Uptime (4), CPU (5), RAM (6)
- **`↑ / ↓` (or `j / k`)**: Navigate services list
- **`Tab`**: Switch focus between Table and Logs viewport
- **`S` / `X` / `R`**: Start (or Run Now for cron/xdg) / Stop / Restart selected service
- **`E` / `D`**: Enable / Disable autostart
- **`Del` / `Backspace`**: Delete service / cron job / desktop file with confirmation
- **`U`**: Toggle User mode (`~/.config/` / user crontab) and System mode (`/etc/`)
- **`/`**: Search filter (in Table when table is focused, or in Logs when logs viewport is focused)
- **`L`**: Cycle Log Priority Filter (**ALL** ➔ **WARN+ERR** ➔ **ERR**)
- **`Space`**: Pause / Resume live log tailing (when logs viewport is focused)
- **`Esc`**: Clear search filter / Cancel wizard
- **`Q` / `Ctrl+C`**: Quit

---

## 🇷🇺 Русский

**Alirun** — быстрый, наглядный и прозрачный менеджер системных служб, демонов и автозапуска в Linux. Программа создана на **Go (Golang)** и объединяет удобный CLI, интерактивный пошаговый мастер (Wizard) и полноэкранный TUI-дашборд.

### ✨ Ключевые особенности

- **Автоматическое создание служб и таймеров**:
  - Превращение любого бинарника, Python/Node/Bash скрипта или команды в службу systemd за пару секунд.
  - **Современная замена cron**: нативная поддержка **Systemd Timers** (`--preset timer --schedule "15m"` / `"*-*-* 03:00:00"`).
- **Управление XDG Desktop Autostart**:
  - Полная поддержка `~/.config/autostart/*.desktop` и `/etc/xdg/autostart/`.
  - Добавление, включение/выключение, запуск и удаление приложений автозапуска из CLI и TUI.
- **Быстрый редактор конфигураций ($EDITOR и F4 в TUI)**:
  - Редактирование юнитов systemd, crontab и `.desktop` файлов в `$EDITOR` (nano, vim, micro) с автоматическим `systemctl daemon-reload` или перезаписью crontab при сохранении.
  - Нажатие `F4` в TUI бесшовно приостанавливает интерфейс, открывает редактор и обновляет дашборд.
- **Инспектор открытых сетевых портов**:
  - Встроенный чистый Go парсер `/proc/net` определяет прослушиваемые сокеты (`tcp:8080`, `udp:53` и др.) для работающих служб в CLI и TUI.
- **Автоматический диагностический ассистент (Doctor)**:
  - Команда `alirun doctor [служба]` анализирует упавшие службы, коды выхода (203/EXEC, 127, 200/CHDIR, 217/USER), убийство OOM Killer, конфликты портов и предлагает конкретные решения.
- **Портативный бэкап и перенос (YAML)**:
  - `alirun export`: сборка служб, crontab и автозапуска в единый переносимый YAML-файл.
  - `alirun import`: безопасное восстановление с поддержкой симуляции `--dry-run`.
- **Улучшенный просмотрщик логов**:
  - Фильтрация приоритета (`L`: ALL / WARN+ERR / ERR), живая пауза/возобновление стриминга (`Space`) и поиск по логам (`/`).
- **Принцип абсолютной прозрачности (Transparency First)**:
  - Предпросмотр сгенерированных `.service` и `.timer` файлов с подсветкой синтаксиса перед записью на диск.
  - Наглядное указание путей (`~/.config/systemd/user/` или `/etc/systemd/system/`).
  - Четкий список выполняемых системных команд (`daemon-reload`, `enable --now`).
- **Мониторинг ресурсов и живые графики**:
  - Отображение загрузки CPU% и памяти RAM в реальном времени с прогресс-барами (`[████░░░░]`) и Unicode-спарклайнами истории (` ▂▃▄▅▆▇█`).
  - Подсчет задач/потоков (Tasks) и точное время непрерывной работы (Uptime).
- **Безопасный User-Mode по умолчанию**: службы создаются без прав `root` / `sudo` в каталоге пользователя, защищая систему от случайных поломок.
- **Мульти-инит архитектура**: быстрое переключение между **Systemd**, **Cron** и **XDG Autostart**.
- **Простота обновлений**:
  - Встроенная команда `alirun update` для самообновления бинарника из GitHub Releases.
  - Команда `make update-deps` для мгновенного обновления зависимостей проекта.

### 📚 Справочник для тех, кто желает сам разобраться как всё работает

Если вы хотите глубоко понять, как устроены службы, демоны, планировщики и автозагрузка в Linux, понимать принципы их ручной настройки и синтаксис всех файлов конфигурации — ознакомьтесь с подробным иллюстрированным руководством:

👉 **[Справочник: Системы инициализации, демоны и автозагрузка в Linux](справочник%20--%20Системы%20инициализации%20и%20автозагрузка%20в%20Linux.md)**

Внутри руководства:
- **Базовые понятия:** Процессы, демоны, PID 1, разница между System (root) и User scope.
- **Сетевая активность:** Слушающие серверные порты (`LISTEN`) против исходящих клиентских сессий (`ESTABLISHED`) — почему Nginx слушает порт, а Telegram соединяется наружу.
- **Systemd:** Построчный разбор файлов `.service`, типы запуска (`simple`, `forking`, `oneshot`), механизм переопределений (`override.conf`), иерархия каталогов (`/run/` vs `/usr/lib/` vs `~/.config/`).
- **Systemd Timers:** Синтаксис `OnCalendar`, монотонные таймеры и защита от пропуска запусков (`Persistent=true`).
- **Cron:** Правила 5 и 6 полей, и **3 главные ловушки новичков** (минимальный `$PATH`, рабочий каталог, потеря вывода).
- **XDG Autostart:** Управление `.desktop` файлами и их связь с генераторами systemd.
- **OpenRC:** Службы в контейнерах и дистрибутивах без systemd (Alpine Linux).
- **Диагностика:** Пошаговый поиск причин сбоев, анализ логов, коды выхода и конфликты портов.

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

# Список активных служб пользователя (компактный вид или с ЦП, ОЗУ, Аптаймом)
alirun list
alirun list -d
alirun list --sort cpu -d

# Детальный статус, открытые порты и последние логи
alirun status my-web

# Автоматическая диагностика сбоев и рекомендации по исправлению
alirun doctor               # диагностика всех упавших служб
alirun doctor my-web        # детальный анализ конкретной службы

# Быстрый редактор конфигурации ($EDITOR / nano / vim)
alirun edit my-web          # юнит systemd
alirun edit                 # пользовательский crontab (при --init cron)
alirun edit app.desktop     # файл автозапуска XDG

# Живой стриминг логов в реальном времени
alirun logs my-web -f

# Управление жизненным циклом
alirun start my-web
alirun stop my-web
alirun restart my-web
alirun enable my-web
alirun disable my-web
alirun delete my-web

# Автозапуск XDG Desktop (~/.config/autostart/*.desktop)
alirun xdg list                                         # Список автозапускаемых приложений
alirun xdg add -n "MyTool" -c "/usr/local/bin/mytool"   # Добавить в автозапуск
alirun xdg toggle mytool                                # Включить / Выключить
alirun xdg run mytool                                   # Запустить по требованию
alirun xdg remove mytool                                # Удалить из автозапуска

# Управление классическими задачами Cron (crontab)
alirun cron list                        # Список пользовательских задач cron
alirun cron list --system               # Общесистемные задачи cron (/etc/crontab и /etc/cron.d/*)
alirun cron add -s "*/15 * * * *" -c "/backup.sh" -m "Резервное копирование БД" # Добавить задачу
alirun cron toggle cron-1               # Включить / Выключить (комментирует через #)
alirun cron run cron-1                  # Запустить команду cron немедленно
alirun cron remove cron-1               # Удалить задачу из crontab

# Резервное копирование и перенос (YAML)
alirun export                           # Экспорт служб, crontab и автозапуска в YAML
alirun export backup.yaml               # Экспорт в указанный файл
alirun import backup.yaml --dry-run     # Симуляция восстановления (без записи)
alirun import backup.yaml               # Восстановление конфигураций и перезагрузка демонов
```

### 🖥 Полноэкранный TUI-интерфейс

Запустите команду без аргументов:
```bash
alirun
```

**Горячие клавиши:**
- **`M`**: Переключить активный **Менеджер** (**Systemd** ➔ **Cron** ➔ **XDG Autostart**)
- **`F4` / `Ctrl+E`**: Открыть выбранную службу/crontab/desktop в **`$EDITOR`** (nano, vim) с автоперезагрузкой
- **`N` / `C`**: Открыть встроенный **Мастер создания нового демона / задачи** (с live-preview)
- **`V`**: Переключить **Вид таблицы** (Упрощенный / Подробный с ЦП, ОЗУ, Аптаймом и PID)
- **`O` / `P`**: Переключить **Поле сортировки** (Имя, Статус, Время запуска, Аптайм, ЦП, ОЗУ) / Направление (`▲` / `▼`)
- **`1`..`6`**: Быстрая сортировка по Имени (1), Статусу (2), Времени запуска (3), Аптайму (4), ЦП (5), ОЗУ (6)
- **`↑ / ↓` (или `j / k`)**: Навигация по списку служб
- **`Tab`**: Переключение фокуса между таблицей и логами
- **`S` / `X` / `R`**: Запустить (или Запустить прямо сейчас) / Остановить / Перезапустить
- **`E` / `D`**: Включить / Выключить
- **`Del` / `Backspace`**: Удалить службу / задачу / desktop-файл с подтверждением
- **`U`**: Переключить режим (User: `~/.config/` / crontab или System: `/etc/`)
- **`/`**: Поиск и фильтрация (в таблице при фокусе на таблице, или по тексту логов при фокусе на логах)
- **`L`**: Переключить фильтр важности логов (**ALL** ➔ **WARN+ERR** ➔ **ERR**)
- **`Space`**: Пауза / Возобновление живого потока логов (при фокусе на логах)
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
│   ├── doctor.go             # Автоматическая диагностика сбоев
│   ├── edit.go               # Интеграция с $EDITOR (F4 / alirun edit)
│   ├── export.go             # Экспорт конфигураций в портативный YAML
│   ├── import.go             # Импорт и восстановление из YAML
│   ├── list.go               # Таблица служб (Lip Gloss)
│   ├── status.go             # Детальный статус и инспектор портов
│   ├── control.go            # start/stop/restart/enable/disable/delete
│   ├── logs.go               # Стриминг journalctl
│   ├── tui.go                # Вызов TUI-дашборда
│   ├── version.go            # Команды version и update
│   └── xdg.go                # Управление автозапуском XDG Desktop
├── internal/tui/             # Полноэкранный TUI (Bubble Tea)
│   ├── app.go                # Реактивная модель, стейт и цикл событий
│   ├── metrics.go            # Мониторинг CPU%/RAM, спарклайны и прогресс-бары
│   ├── sort.go               # Модели сортировки, переключение режимов таблицы и форматтеры
│   └── styles.go             # Цвета, границы и бейджи
├── pkg/
│   ├── backup/               # Модуль экспорта и импорта YAML-бэкапов
│   ├── detector/             # Анализ файлов, shebang, chmod, virtualenv
│   ├── diagnose/             # Движок диагностики сбоев и предложений по фиксу
│   ├── editor/               # Враппер $EDITOR для Bubble Tea и CLI
│   ├── highlighter/          # Подсветка синтаксиса и diff (Chroma)
│   ├── initsys/              # Модульное ядро мульти-инит систем
│   │   ├── manager.go        # Интерфейс initsys.Manager
│   │   ├── registry.go       # Реестр и автоопределение инит-системы
│   │   ├── systemd/          # Провайдер Systemd (D-Bus API, go-systemd)
│   │   ├── cron/             # Провайдер Cron (crontab, /etc/crontab, parser, humanizer)
│   │   ├── xdg/              # Провайдер XDG (~/.config/autostart/)
│   │   └── openrc/           # Провайдер OpenRC (модульная заготовка)
│   ├── netinfo/              # Парсер сокетов /proc/net для инспекции портов
│   └── updater/              # Самообновление бинарника (go-selfupdate)
├── Makefile                  # Сборка, тесты, установка, обновление
├── go.mod                    # Go зависимости
└── справочник -- Системы инициализации и автозагрузка в Linux.md # Подробный иллюстрированный справочник
```

---

## 👤 Создатель / Author

**Илья Ульянов | khameleonium**
- **GitHub**: [https://github.com/khameleonium](https://github.com/khameleonium)
- **Репозиторий**: [https://github.com/khameleonium/alirun](https://github.com/khameleonium/alirun)

---

## 📄 Лицензия / License

Распространяется под лицензией MIT. Подробности в файле [LICENSE](LICENSE).

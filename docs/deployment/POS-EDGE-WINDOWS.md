# POS Edge Windows package

Статус: реализовано сейчас для alpha packaging.

## Состав пакета

Реализовано сейчас: есть два способа сборки.

`scripts/package-pos-edge-windows.sh` собирает portable-каталоги:

- `windows-amd64/pos-edge.exe`;
- `windows-386/pos-edge.exe`;
- `config/pos-edge.json`;
- `migrations/sqlite/001_init.sql`;
- `ui/pos-ui/*`;
- `webwallpaper/config.pos-edge.example.json`.

POS UI собирается с `VITE_POS_API_BASE=/api/v1`, поэтому при открытии `http://127.0.0.1:8080/` UI ходит в API того же `pos-edge.exe`.

`scripts/build-pos-edge-installer.ps1` собирает Windows NSIS installer:

- staging-каталог с `pos-edge.exe`, `start-pos-edge.cmd`, `migrations`, `ui/pos-ui`;
- `config/pos-edge.install.json` с preset `POS_HTTP_ADDR`, `LICENSE_SERVER_URL` и `POS_CLOUD_SYNC_URL`;
- optional `webwallpaper/gowebwallpaper.exe` и `webwallpaper/config.pos-edge.example.json`, если задан `-WebWallpaperExe`;
- install-time страницу настроек, где оператор задает порт POS backend, адрес Cloud server и адрес License server;
- `myhoreca-pos-edge-<version>-<arch>-setup.exe`.

NSIS installer ставится в user scope: `%LOCALAPPDATA%\MyHoreca\POS Edge`. Это не требует admin rights и оставляет `data/` доступным runtime-процессу.

## Portable сборка

Из корня репозитория:

```bash
scripts/package-pos-edge-windows.sh
```

Артефакты пишутся в `dist/pos-edge-windows`. Для другого каталога:

```bash
scripts/package-pos-edge-windows.sh /tmp/pos-edge-windows
```

## NSIS сборка на Windows

Требования на Windows-хосте:

- Go версии проекта;
- Node.js + npm;
- NSIS 3.x с `makensis.exe` в `PATH`.

Из PowerShell в корне репозитория:

```powershell
Set-ExecutionPolicy -Scope Process Bypass
.\scripts\build-pos-edge-installer.ps1 `
  -Version "0.1.9" `
  -Arch "amd64" `
  -PosHttpPort 8080 `
  -LicenseServerUrl "https://license.example.com" `
  -CloudSyncUrl "https://cloud.example.com"
```

`-PosHttpPort`, `-LicenseServerUrl` и `-CloudSyncUrl` задают значения по умолчанию, которые будут показаны в installer UI. На этапе установки оператор может заменить их без пересборки installer.

Для 32-bit alpha smoke:

```powershell
.\scripts\build-pos-edge-installer.ps1 `
  -Version "0.1.9" `
  -Arch "386" `
  -PosHttpPort 8080 `
  -LicenseServerUrl "https://license.example.com" `
  -CloudSyncUrl "https://cloud.example.com"
```

Если рядом нужно положить kiosk host:

```powershell
.\scripts\build-pos-edge-installer.ps1 `
  -Version "0.1.9" `
  -Arch "amd64" `
  -PosHttpPort 8080 `
  -LicenseServerUrl "https://license.example.com" `
  -CloudSyncUrl "https://cloud.example.com" `
  -WebWallpaperExe "C:\tools\gowebwallpaper.exe"
```

Результат: `dist\pos-edge-installer\myhoreca-pos-edge-0.1.9-amd64-setup.exe`.

## Запуск на Windows

Portable package запускается из каталога пакета:

```powershell
$env:POS_CONFIG_PATH="config\pos-edge.json"
.\pos-edge.exe
```

NSIS installer создает Start Menu shortcut `MyHoreca\POS Edge`, который запускает `start-pos-edge.cmd` с рабочим каталогом установки и `POS_CONFIG_PATH=config\pos-edge.json`.

Перед portable-запуском нужно заполнить `LICENSE_SERVER_URL` в `config/pos-edge.json`; без license authority POS Edge завершает startup fail-fast.

Для NSIS installer реализовано сейчас:

- installer показывает страницу `POS Edge connection settings`;
- поле `POS backend port` записывается как `POS_HTTP_ADDR = 127.0.0.1:<port>`;
- поле `Cloud server URL` записывается в `POS_CLOUD_SYNC_URL`;
- поле `License server URL` записывается в `LICENSE_SERVER_URL`;
- значения из build script используются только как defaults installer UI;
- при установке обновляется активный `config/pos-edge.json`: сохраняются остальные существующие параметры, но `MH_POS_VERSION`, порт, Cloud URL, License URL и стандартные Windows runtime paths приводятся к выбранным installer values.

`POS_UI_DIST_DIR=ui/pos-ui` включает отдачу React SPA самим `pos-edge.exe`. API остается на `/api/v1`, healthcheck на `/health`.

## Обновление

Реализовано сейчас:

- NSIS installer можно запускать поверх существующей установки;
- installer заменяет `pos-edge.exe`, `migrations`, `ui` и optional `webwallpaper`;
- installer сохраняет `data/`, `data/backups`, `data/archives` и существующий `config/pos-edge.json`;
- preset config нового installer кладется как `config/pos-edge.install.json`;
- installer применяет выбранные install-time настройки к активному `config/pos-edge.json`, включая `MH_POS_VERSION`;
- если в installer включен `gowebwallpaper.exe`, installer создает активный `webwallpaper/config.pos-edge.json` с URL `http://127.0.0.1:<POS backend port>/`;
- если `config/pos-edge.json` отсутствует, installer создает его из preset и выбранных install-time настроек.

Это покрывает ручное обновление: скачать новый setup, запустить, затем перезапустить POS Edge. Данные мигрируются штатным startup policy POS Edge: backup до schema/data upgrade, `db_runtime_versions`, checksum/version gate и schema verification.

Запланировано далее: автоматическое обновление через version storage.

Целевой contract:

1. POS Edge при startup и далее периодически читает version manifest из provider storage.
2. Manifest содержит `product`, `channel`, `version`, `arch`, `min_supported_from`, `installer_url`, `sha256`, подпись, release notes URL и migration flags.
3. POS Edge скачивает installer в `data/updates`, проверяет checksum и подпись.
4. Перед update POS Edge создает SQLite backup тем же runtime backup mechanism, что migrations.
5. Отдельный updater process останавливает POS Edge, запускает NSIS installer в silent mode, стартует POS Edge обратно и пишет локальный update journal.
6. Новый POS Edge стартует, выполняет migrations и schema verification; при ошибке оператор получает fail-fast состояние с backup path.

Вне текущего объема: runtime updater code и version storage API. Их нельзя безопасно завершить до появления хранилища версий, подписи manifest и политики каналов.

## Kiosk host

Для операторского экрана не запускаем обычный браузер вручную. Используем внешний `gowebwallpaper` как kiosk/webview host:

1. Для NSIS installer передать `-WebWallpaperExe "C:\tools\gowebwallpaper.exe"` при сборке.
2. Installer положит `gowebwallpaper.exe` в `%LOCALAPPDATA%\MyHoreca\POS Edge\webwallpaper`.
3. Installer создаст `webwallpaper\config.pos-edge.json` с URL, соответствующим порту POS backend, выбранному в installer UI.
4. Запустить POS Edge через Start Menu shortcut `MyHoreca\POS Edge`.
5. Запустить kiosk host из каталога установки:

```powershell
cd "$env:LOCALAPPDATA\MyHoreca\POS Edge\webwallpaper"
.\gowebwallpaper.exe .\config.pos-edge.json
```

6. На первом запуске выбрать монитор в tray menu, если kiosk host требует привязку монитора.

Для portable package:

1. Положить `gowebwallpaper.exe` рядом с пакетом или в отдельный каталог оператора.
2. Использовать `webwallpaper/config.pos-edge.example.json` как шаблон и выставить URL на фактический порт POS Edge.
3. Выбрать монитор в tray menu.

`webwallpaper/config.pos-edge.example.json` оставлен как пример URL. Активный `webwallpaper/config.pos-edge.json` в NSIS установке генерируется на install-time и использует выбранный порт. Не подменять его обычным браузером: операторский экран должен открываться через нативный fullscreen/kiosk host.

## Матрица Windows

Реализовано сейчас:

- основной supported target: Windows 10/11, `amd64`;
- smoke/alpha fallback: Windows 10/11, `386`;
- `pos-edge.exe` собирается без CGO и не требует внешнего SQLite DLL.

Разъяснение по 32-bit Windows:

- сборка `-Arch "386"` предусмотрена и выпускает 32-bit `pos-edge.exe` и NSIS setup;
- это не основной production target, а alpha/smoke fallback для Windows 10/11 x86;
- backend-часть собирается Go cross-compile без CGO, поэтому отдельная SQLite DLL для x86 не нужна;
- практическая приемка 32-bit должна отдельно проверить запуск процесса, SQLite startup/migrations, `/health`, отдачу UI и pairing/smoke на реальном 32-bit Windows окружении;
- kiosk host/webview зависит от конкретного внешнего `gowebwallpaper.exe`: если нужен kiosk на 32-bit Windows, нужен 32-bit compatible kiosk binary и отдельная проверка его WebView/runtime зависимостей.

Вне текущего объема:

- единый production target для Windows 7/8/8.1;
- security-supported WebView2 на Windows 7;
- MSI, Windows service wrapper и autoupdate POS Edge.

Windows 7/x86 возможен только как отдельный legacy track: pinned Go 1.20.x toolchain, pinned old WebView2/Edge 109 runtime, отдельная приемка железа и явное согласование security risk. Текущий Go 1.26 runtime проекта не является честным production target для Windows 7.

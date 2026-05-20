# Python master-data scripts design

Статус: реализовано сейчас для локального pre-pilot рабочего места.

## Цель

Нужно заменить устаревающую PowerShell-only скриптовую часть переносимым Python 3 контуром, который на Fedora/Linux и Windows предзаполняет Cloud master data через штатные HTTP API, выполняет POS Edge provisioning и проверяет Cloud -> Edge синхронизацию через POS API.

## Архитектура

Реализация делится на переносимое Python-ядро и тонкие оболочки под окружения. Python-ядро использует только стандартную библиотеку, чтобы не требовать `pip install`, `jq` или PowerShell runtime на Linux. Shell/PowerShell wrappers только выбирают интерпретатор и передают аргументы в Python.

Канонический путь:

1. Проверить health endpoints Cloud/POS/License.
2. Создать Cloud-owned demo data через Cloud master-data API.
3. Опубликовать master-data package через Cloud publication API.
4. Выполнить POS Edge pairing/provisioning через license code, с fallback на Cloud assignment.
5. Проверить initial read model через POS API: PIN login, halls, tables, menu items, modifiers.
6. После pairing создать дополнительную Cloud-owned menu позицию, повторно опубликовать master data и дождаться, пока POS sync sender получит данные через authenticated `sync/exchange`.
7. Проверить POS read model повторным GET к POS API.

## Компоненты

- `scripts/lib/mhpos_http.py` — HTTP JSON client, retry/wait helpers и safe error reporting.
- `scripts/lib/mhpos_seed.py` — доменные сценарии seed/provision/verify без привязки к CLI.
- `scripts/seed-cloud-masterdata.py` — полуавтоматическое заполнение Cloud справочников.
- `scripts/provision-pos-edge.py` — POS Edge provisioning через штатные API.
- `scripts/verify-sync.py` — проверка read model POS Edge и ожидание Cloud -> Edge sync.
- `scripts/run-local-masterdata-smoke.py` — полный локальный сценарий для Docker stack.
- `scripts/*.sh` и `scripts/*.ps1` wrappers — тонкий запуск Python-скриптов.

## Ошибки и безопасность

Скрипты не пишут напрямую в PostgreSQL или SQLite. PIN передается только в request body Cloud/POS API и не печатается в error dump как secret payload. При HTTP ошибке выводится method, URL, status и safe response body; request body с PIN не логируется.

## Проверка

Unit tests покрывают URL normalization, порядок Cloud API calls, POS read-model validation и ожидание sync. Интеграционный smoke запускается вручную на поднятом Docker stack.

## Документация

`docs/backend/LOCAL-DOCKER-STACK.md` и `README.md` должны показывать Linux/Fedora-first команды. Документация должна явно сказать, что demo seed dataset нужно расширять вместе с развитием справочников и пользовательских сценариев, чтобы ручной наглядный тест оставался полезным.

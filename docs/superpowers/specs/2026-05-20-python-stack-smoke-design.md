# Python Stack Smoke Design

Статус: реализовано сейчас как отдельный переносимый тестовый модуль для локального Docker stack.

## Цель

Нужна одна Python-утилита, которая проверяет не только Cloud -> Edge master-data seed flow, но и базовую готовность всех основных сервисов: Cloud API, POS Edge API и License Server. Утилита должна оставаться portable для Linux/Windows и использовать только Python standard library.

## Архитектура

Реализация делится на reusable library и CLI:

- `scripts/lib/mhpos_stack.py` содержит model результата, контекст клиентов, suite runner и проверки.
- `scripts/run-stack-smoke.py` является единой точкой входа для локального stack smoke.
- `scripts/lib/mhpos_seed.py` остается доменным helper для Cloud master-data seed, POS provisioning и Cloud -> Edge sync verification.
- `docs/api/mhpos-local-smoke.openapi.json` остается contract source для HTTP calls; новые stack checks добавляют операции в OpenAPI и вызывают их через `operationId`.

Runner выполняет независимые suites и возвращает structured JSON. Каждый suite имеет status `passed`, `failed` или `skipped`, duration, safe details и safe error. PIN, pairing code и token не печатаются в открытом виде.

## Suites

Реализовано сейчас:

- `health`: проверяет `/health` для Cloud, POS Edge и License Server.
- `license_pairing`: напрямую регистрирует одноразовый pairing code в License Server, resolve-ит его и проверяет, что code consumed.
- `cloud_to_edge_masterdata`: создает Cloud-owned demo справочники, выполняет POS Edge provisioning через License Code с Cloud assignment fallback, проверяет POS read model и post-pairing Cloud -> Edge sync.

Запланировано далее:

- `cashier_runtime`: runtime путь `login -> shift/cash session -> order -> precheck -> payment -> check -> reprint -> cancellation/refund -> close shifts`.
- service-specific suites для новых Cloud, Edge и License endpoints, когда они становятся частью smoke acceptance.

## Error Handling

Каждый suite ловит exception и превращает его в failed result с безопасным error string. HTTP timeout и connection errors остаются `HttpError` с `status=0`. CLI завершает процесс с code `1`, если хотя бы один выбранный suite failed.

## CLI Contract

Команда:

```bash
python3 scripts/run-stack-smoke.py --suite all --json-output scripts/.stack-smoke-result.json
```

Аргументы:

- `--cloud-base`, `--pos-base`, `--license-base`;
- `--suite` с повторением или comma-separated list: `all`, `health`, `license_pairing`, `cloud_to_edge_masterdata`;
- `--output` для существующего seed summary;
- `--json-output` для полного stack smoke result;
- `--skip-post-pairing-sync-check`, `--wait-seconds`, `--interval-seconds`.

## Проверка

Unit tests проверяют result model, suite selection, health suite, license pairing suite и CLI-safe JSON shape. Интеграционный smoke запускается только на поднятом локальном stack.

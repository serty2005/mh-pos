# mh-pos

Короткая карта репозитория `serty2005/mh-pos`. Подробные правила, контракты и планы находятся в профильных документах, а не в README.

## Текущее состояние

Реализовано сейчас:

- POS Edge backend поддерживает cashier runtime `Order -> Precheck -> Payment -> Check`.
- `IssuePrecheck` блокирует заказ, создает immutable financial snapshot precheck и фиксирует `currency_code`, subtotal, discounts, surcharges, taxes, grand total, paid/remaining totals и breakdown строк/налогов/скидок/надбавок.
- POS Edge backend содержит MVP `Pricing` boundary: line/order discounts, manual/service/PB1 surcharge foundation, единый ordered discount/surcharge pipeline по `application_index`, percentage/fixed amounts, percentage/fixed tax rules, inclusive/exclusive tax foundation и deterministic integer rounding.
- `CancelPrecheck` требует manager override, проверяет PIN/permission и возвращает unpaid active precheck order в `open`.
- Оплата выполняется через `precheck_id`; partial payments разрешены; final check создается только после полной оплаты.
- `POST /api/v1/payments/{id}/refund` переводит captured payment в `refunded` и уменьшает `paid_total` у precheck/check.
- Reprint precheck/check строится из immutable snapshot.
- Cloud -> Edge master-data ingest в POS Edge runtime поддерживает только потоки `restaurants`, `devices`, `staff`, `floor`, `catalog`, `menu`.
- Cloud schema содержит foundation для modifier/recipe/master-data authority; SQLite schema содержит foundation для recipes/inventory. Это не означает готовый cashier runtime для modifiers или inventory consumption.

Вне текущего runtime:

- POS order modifiers runtime и cashier UI modifiers;
- automatic recipe expansion / stock consumption engine;
- real payment processor module, PSP webhooks и fiscal adapter;
- ClickHouse runtime pipeline;
- подтвержденный `sqlc` persistence rollout.

## Структура

- `pos-backend/` — POS Edge Go backend, SQLite runtime, cashier API.
- `pos-ui/` — Vue/Quasar cashier UI.
- `cloud-backend/` — Cloud API, PostgreSQL sync receiver и master-data authority foundation.
- `license-server/` — license/pairing support service.
- `shared/` — общие platform helpers.
- `scripts/` — локальные bootstrap/smoke scripts.
- `docs/` — профильная документация.

## Главные документы

- `SPECv1.3.md` — frozen cashier pilot contract до первого pilot.
- `ROADMAP.md` — статусы, блокеры и следующий план.
- `docs/backend/POS-BACKEND-SPEC.md` — фактический POS backend contract.
- `docs/backend/POS-DATA-AND-MIGRATIONS.md` — SQLite/PostgreSQL schema и migration policy.
- `docs/ui/POS-UI-SPEC.md` — фактический cashier UI contract.
- `docs/architecture/DDD-CONTEXT-MAP.md` — bounded contexts и ownership boundaries.
- `docs/adr/ADR-015-persistence-and-analytics-strategy.md` — решение по persistence/analytics strategy.
- `AGENTS.md` — только правила работы агентов и процесса разработки.

## Локальный запуск

Docker stack:

```powershell
docker compose -f docker-compose.local.yml up --build
```

POS UI:

```powershell
cd pos-ui
npm install
npm run dev
```

POS backend:

```powershell
cd pos-backend
go mod tidy
go test ./...
```

Cloud backend:

```powershell
cd cloud-backend
go mod tidy
go test ./...
```

UI build:

```powershell
cd pos-ui
npm install
npm run build
```

## Документационное правило

Если код и документ расходятся, фактический runtime проверяется по коду и тестам. Документ после этого обновляется под подтвержденное поведение. Planned decisions должны быть явно помечены как `запланировано до пилота`, `запланировано далее`, `после пилота` или `вне текущего объема`, а не как реализованные функции.

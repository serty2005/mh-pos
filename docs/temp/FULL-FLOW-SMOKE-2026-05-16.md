# Full Flow Smoke 2026-05-16

## Статус

реализовано сейчас: production-way local flow повторен на локальном Docker stack `mh-pos-local`.

## Найдено

- POS Edge Docker image собирается на Alpine в `docker/pos-edge.Dockerfile` и `pos-backend/docker/Dockerfile`; runtime image уже устанавливает `tzdata`, сборочный лог подтвердил `tzdata (2026b-r0)`.
- Старые файлы `docs/temp/FULL-FLOW-SMOKE-2026-05-15.md` и `docs/temp/DOCUMENTATION-AUDIT-2026-05-15.md` отсутствовали в рабочем дереве на момент итерации; новый отчет создан в `docs/temp`.
- Host port `5432` был занят существующим локальным PostgreSQL контейнером `mh-pos-cloud-postgres`; stack поднят с `CLOUD_POSTGRES_HOST_PORT=55432`, внутренний DSN `cloud-postgres:5432` не менялся.

## Изменено

- Provisioning/POS Edge: повторный `PollCloudAssignment` и `pair-via-license` после состояния `paired` возвращают текущий paired status без повторного Cloud/license resolve, snapshot apply, `PairEdgeNode` и без нового `EdgeNodePaired` local event/outbox row.
- POS regression tests: добавлены проверки IANA timezone при открытии личной смены, идемпотентного paired provisioning и сохранения `restaurants.active = 1` при Cloud -> Edge ingest.
- Cloud publication-state: `GET /api/v1/restaurants/{id}/master-data/publication-state` до первой публикации возвращает `200 null`; Cloud UI optional request parsing принимает `null` как empty state.
- Cloud publication regression: snapshot опубликованного active restaurant проверяется на `active: true`.
- Smoke script `scripts/bootstrap-production-way.ps1`: runtime smoke теперь вызывает reprint precheck и reprint final check до refund.
- Local compose: host binding PostgreSQL вынесен в `CLOUD_POSTGRES_HOST_PORT`, чтобы не останавливать чужой локальный PostgreSQL.

## Smoke

Команда:

```powershell
$env:CLOUD_POSTGRES_HOST_PORT='55432'
docker compose -f docker-compose.local.yml up --build -d
powershell -NoProfile -ExecutionPolicy Bypass -File scripts\bootstrap-production-way.ps1 -CloudBaseUrl http://localhost:8090 -EdgeBaseUrl http://localhost:8080 -RunRuntimeSmoke -VerboseOutput
```

Результат:

- Cloud health: ok.
- License health: ok.
- POS Edge health: ok.
- Cloud UI/master data path: restaurant, roles, employees, hall/table, catalog/menu created.
- Publication: `publication_id = 81c5e586-690b-4ec2-9f9a-bfc35df16657`.
- Pairing: `node_device_id = 233b817a-5b04-457b-9189-7046da563c8f`, status `paired`.
- PIN login: cashier PIN `1111`, manager PIN `2222`.
- Runtime: personal employee shift opened, cash shift opened, table order created, two lines added, precheck issued, payment captured, final check created, order closed.
- Reprint: local event log contains `PrecheckReprinted = 1` and `CheckReprinted = 1`.
- Refund: `refund_operation_id = 01c03511-d4f4-41b3-a0e3-65aa32575979`, status `recorded`.
- Idempotency spot check after paired state: repeated provisioning status + `pair-via-license` did not increase `EdgeNodePaired`; local event log kept `EdgeNodePaired = 1`.

Local event summary after smoke:

```text
AuthSessionStarted 7
CashSessionClosed 1
CashSessionOpened 2
CheckCreated 1
CheckReprinted 1
EdgeNodePaired 1
OrderClosed 1
OrderCreated 1
OrderLineAdded 2
PaymentCaptured 1
PrecheckIssued 1
PrecheckReprinted 1
RefundRecorded 1
ShiftClosed 1
ShiftOpened 2
```

## Проверки

- `cd pos-backend; go test ./...` - passed.
- `cd cloud-backend; go test ./...` - passed.
- `cd license-server; go test ./...` - passed.
- `cd pos-ui; npm.cmd install; npm.cmd run build` - passed.
- `cd cloud-ui; npm.cmd install; npm.cmd run build` - passed.
- Targeted packages for POS app/API/provisioning and Cloud masterdata/cloudsync - passed.

## Осталось

запланировано далее:

- Наблюдать sync delivery counters в Cloud после более длинного фонового worker interval.
- Добавить браузерный Playwright smoke поверх UI, если понадобится проверять именно клики Cloud UI/POS UI, а не production API flow.

вне текущего объема:

- Modifiers runtime beyond current selected modifiers, automatic inventory consumption, real PSP, fiscal adapter, ClickHouse runtime и sqlc rollout.
- Детальный Cloud financial operation projection по refund item scopes.

## Runtime Code

затрагивался runtime code: да, POS provisioning service и Cloud masterdata HTTP empty-state handler.

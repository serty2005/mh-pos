# Ручной UI-аудит 2026-06-28

Статус: промежуточное сканирование чистого локального стека через ручные Playwright-проходы. Runtime-код не изменялся.

Проверенный контур:

- Cloud UI: `http://localhost:5174`, `cloud-ui-g`.
- POS UI: `http://localhost:3000`, `pos-ui-g`.
- License Server operator page: `http://localhost:8095`.
- Backend health: Cloud `8090`, POS Edge `8080`, License Server `8095`.

## Выполненные проходы

Реализовано сейчас:

- На чистом Cloud UI создан ресторан `Audit Cafe`.
- Через License Server UI создан entitlement snapshot `local-tenant/cloud-local` с активными модулями.
- В Cloud UI выбран ресторан, созданы catalog item, menu item, зал, стол, роль и сотрудник с PIN.
- Pending POS Edge назначен ресторану через Cloud UI.
- POS UI перешел из provisioning на PIN login, вход по созданному PIN прошел.
- В POS UI открыты личная смена и кассовая смена.

## Найдено

Блокеры:

- Cloud License page до ручного snapshot показывает только `errors.license.unavailable`; пользователь не получает понятного действия восстановления. License Server требует знание admin token и редактирование snapshot через отдельную страницу.
- POS UI показывает `ТЕРМИНАЛ ЗАБЛОКИРОВАН` из-за `503 /api/v1/license/entitlements`, но при этом вход по PIN и кассовые операции доступны. Это противоречивое состояние лицензии.
- POS floor/table сценарий не исполняется: `GET /api/v1/halls?...` возвращает `503 LICENSE_AUTHORITY_UNAVAILABLE`, хотя зал и стол созданы в Cloud.
- POS `НАЧАТЬ ЗАКАЗ` отправляет `POST /api/v1/orders` и получает `201`, но UI остается в состоянии `ЗАКАЗ НЕ ВЫБРАН`; созданный заказ не открывается оператору.
- Cloud delivery после Edge assignment показывает `Ожидает ACK`; понятного действия для оператора, чтобы запустить/проверить Edge exchange и применить master-data, нет.

Проблемы удобства:

- License Server UI использует raw `Entitlements JSON`; нужны переключатели модулей и пресеты без ручного JSON.
- Cloud menu uses raw `Availability JSON`; pricing/taxes uses `Tax profiles JSON array`, `Tax rules JSON array`, `Service charge rules JSON array`. Эти формы требуют JSON вместо кнопок, select, checkbox и repeatable rows.
- Cloud Catalog/Menu/Pricing/Receipt Templates содержат много `input/select/textarea` без `id`, `name`, `aria-label` или связанного label; формы плохо автоматизируются и хуже доступны.
- Cloud Staff onboarding разорван: сотрудника нельзя создать без роли, но форма сотрудника не предлагает создать роль-пресет рядом; роль создается в другой вкладке через плотную permission matrix.
- Mobile Cloud dashboard после выбора ресторана показывает UUID ресторана вместо названия.
- License Server mobile имеет горизонтальный overflow: `scrollWidth 477` при viewport `390`.
- Cloud Inventory и Reports в активном UI остаются заглушками после выбора ресторана.

## Промпты для исправления

## Итерация 2026-06-29

Реализовано сейчас по `POS-91`:

- POS UI выбирает созданный counter-order по `order_id`, поэтому после `POST /api/v1/orders` заказ открывается без reload даже без выбранного стола.
- Для режима без `table-mode` раздел `Заказы` стал основным экраном продажи: слева крупная кнопка `+`, справа последние закрытые заказы с типом оплаты и суммой, клик открывает модалку состава заказа.
- Ручная кнопка пречека в режиме без залов и столов отсутствует; оплата использует `counter-payment`, где backend автоматически выпускает precheck под капотом.
- Блок итогов заказа открывает модалку расшифровки backend totals.

Запланировано далее:

- Edge-настройка поведения после оплаты: возвращаться на экран с `+` или сразу создавать следующий counter-order.
- QR ticket modal/reprint для билетных позиций после явного POS DTO-контракта QR-флага номенклатуры и данных первого прохода.

1. Canonical licensing flow: Plane `POS-95`.

```text
Свести licensing flow к canonical contract docs/backend/LICENSE-ENTITLEMENTS.md: сохранить hyphen module IDs, заменить License Server raw Entitlements JSON на toggles/presets, исправить POS license-state flow, проверить Cloud assignment -> Edge license status -> module routes, сохранить базовую кассу доступной без дополнительных module entitlements. Исторический симптом из аудита POS-90 помечен в Plane как duplicate/delete candidate.
```

2. POS order creation UI: Plane `POS-91`.

```text
Исправь POS UI flow `НАЧАТЬ ЗАКАЗ`: после успешного `POST /api/v1/orders` с HTTP 201 созданный заказ должен стать текущим выбранным заказом без reload. Если backend возвращает safe error, показывать локализованный banner без raw details. Acceptance: Playwright/manual path login -> open shifts -> start order показывает текущий заказ и меню, regression test или минимальный e2e check в `pos-ui-g`.
```

3. Cloud forms without raw JSON: Plane `POS-92`.

```text
Заменить raw JSON поля в Cloud UI на простые controls: Menu availability as status/days/time windows rows, Pricing tax/service-charge arrays as repeatable form rows. License entitlements controls перенесены в POS-95. Не добавлять новый UI kit. Все тексты через i18n, payload остается совместимым с текущими backend DTO. Acceptance: пользователь может заполнить эти данные без JSON, invalid states подсвечиваются рядом с полем, `npm run build` в `cloud-ui-g`.
```

4. Cloud setup onboarding: Plane `POS-93`.

```text
Сделать guided setup для чистого Cloud ресторана: Restaurant -> License snapshot health -> Role preset + employee -> Catalog item -> Menu item -> Floor -> Edge assignment/delivery readiness. Использовать существующие route-backed forms, без нового workflow engine. Acceptance: на чистых данных пользователь видит следующий шаг и primary action, Staff form предлагает создать роль-пресет, mobile dashboard показывает restaurant name, не UUID.
```

5. Accessibility/layout cleanup: Plane `POS-94`.

```text
Проставить устойчивые labels/id/aria-label/name для Cloud Catalog/Menu/Pricing/Receipt Template controls. License Server mobile overflow перенесен в POS-95. Acceptance: Playwright locator by label работает для основных форм, build/test соответствующего UI.
```

## Вне текущего объема аудита

- Исправления runtime-кода.
- Полный seed/smoke `scripts/seed-dev-system.py --run-minimal-flow`.
- Проверка физической печати, Telegram, QR checker и production auth.


## Промпт для итераций

Возьми Plane-задачу POS-XX и реализуй её в репозитории `/home/master/repos/myhoreca-pos`.

Сначала прочитай задачу в Plane, затем профильный аудит:
`docs/ui/UI-MANUAL-AUDIT-2026-06-28.md`.

Работай по правилам AGENTS.md:
- отвечай на русском;
- не откатывай чужие изменения;
- пользовательские UI-строки только через i18n;
- не показывай raw backend/SQL/errors/PIN/token/request dumps;
- не добавляй новые UI kits/dependencies без необходимости;
- исправляй минимально, по существующим паттернам проекта.

Цель:
- устранить проблему, описанную в Plane-задаче;
- не менять unrelated runtime;
- обновить профильную документацию, если меняются UI flow, API usage, license behavior, sync/delivery behavior или error handling;
- добавить/обновить минимальные проверки, которые реально ловят регрессию.

Обязательные шаги:
1. Прочитай код затронутого UI/backend flow end-to-end.
2. Подтверди root cause по фактическому коду/API, не лечи только симптом.
3. Внеси минимальные правки.
4. Запусти релевантные проверки:
   - для `pos-ui-g`: `npm run build`;
   - для `cloud-ui-g`: `npm run build`;
   - для Go backend, если затронут: `go test ./...` в соответствующем модуле;
   - ручной Playwright smoke для исправленного сценария.
5. Обнови Plane-задачу кратким результатом: что исправлено, какие проверки прошли, что осталось вне объема.
6. В финале дай краткий и полный отчёт: найдено, изменено, файлы, проверки, риски, далее, вне объема, затрагивался ли runtime code.

Не реализуй соседние задачи из аудита, если они не нужны для закрытия POS-XX.

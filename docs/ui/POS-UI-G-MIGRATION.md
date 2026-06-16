# POS UI Migration Direction

Статус: обязательное направление разработки POS UI с момента принятия этого документа.

## Решение

- `pos-ui/` удален из runtime tree после периода deprecated.
- Новая разработка POS UI в `pos-ui/` больше не ведется.
- Активной кодовой базой POS UI считается `pos-ui-g/`.
- Изменения в `pos-ui/` больше не выполняются; нужные сценарии переносятся в `pos-ui-g/`.

## Что можно брать из исторического `pos-ui/`

Из исторического `pos-ui/` допускается переносить в `pos-ui-g/` только через сверку с текущим backend runtime и документами:

- подтвержденные runtime-условия;
- backend-authoritative правила;
- RBAC visibility rules;
- safe error, empty, loading и no-permission patterns;
- проверенные cashier, waiter и KDS flow assumptions;
- тестовые сценарии и acceptance assumptions.

Перенос не должен быть механическим копированием legacy component structure. Новый код в `pos-ui-g/` должен строиться через общий модульный component layer.

## Активный UI stack

`pos-ui-g/` является активным React/Vite/TypeScript POS UI. Новые POS UI изменения должны проектироваться под этот стек.

Реализовано сейчас: waiter mobile order/precheck runtime живет в `pos-ui-g/src/components/waiter/POSWaiterSection.tsx` и подключается через terminal mode `waiter` в `pos-ui-g/src/App.tsx`. Default waiter surface не вызывает payment/refund/cash drawer/cash shift/fiscal endpoints; backend остается авторитетным для permissions, locked state, totals и order/precheck transitions.

## Component Modularity Standard

Все новые UI элементы в `pos-ui-g/` должны быть модульными и переиспользуемыми, если они могут повториться в другом экране или flow.

В общий компонентный слой должны выноситься:

- buttons и action controls;
- layout panels и containers;
- navigation и shell элементы;
- notifications, banners и alerts;
- empty, loading, error и no-permission states;
- modal/dialog shells;
- forms, fields и validation messages;
- tables, lists, cards и data rows;
- badges, chips, counters и metric cards;
- quantity/stepper controls;
- drawers, side panels и action rails.

## Правила разработки в `pos-ui-g/`

- Feature screen не должен создавать локальный вариант кнопки, модального окна, notification, row/card или loading/error state, если pattern может быть общим.
- Сначала расширяется существующий reusable component или создается новый primitive/composite component, затем он используется в feature layer.
- Reusable components должны быть presentational/dumb: принимать данные, callbacks и state props, но не владеть POS business logic.
- Backend остается authoritative для totals, permissions, transitions, inventory/payment/fiscal boundaries и validation enforcement.
- UI не должен показывать неподтвержденные backend/API flows как работающий runtime. Такие действия допускаются только как disabled/readiness/backlog state с причиной.
- User-visible text должен идти через localization/text layer `pos-ui-g/`, а не хардкодиться внутри reusable primitives.
- Цвета, spacing, typography и visual states должны идти через общий design-system layer/tokens/classes.

## Документационное правило

Последующие документы должны описывать `pos-ui-g/` как активный POS UI. `pos-ui/` можно упоминать только как удаленный legacy source в историческом контексте migration.

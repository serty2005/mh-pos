# POS UI Agent Rules

Статус: обязательные правила для UI-агента/кодогенератора при проектировании и изменении POS UI.

## Source of truth

- Код, тесты и backend specs являются источником истины.
- Не объявлять UI-функцию реализованной, если backend/API ее не поддерживает.
- Запланированные/будущие функции можно описывать только как reserved/backlog/disabled design.

## Product mode

- POS UI проектируется как waiter-first restaurant workspace.
- Главный UX-контекст: table + active order, либо active order в single-table/quick-service режиме.
- Не строить интерфейс как generic admin panel или retail checkout.

## Navigation

- Основной shell: bottom quick access bar + скрываемое section menu.
- Разделы: залы и столы, заказы, активность, отчеты, касса.
- Sidebar скрыт по умолчанию и не является primary navigation.
- Избегать глубокой route navigation внутри POS runtime.

## Layout

- Каждый раздел использует общий layout: 3/4 основная область, 1/4 action rail.
- На compact/touch screens action rail превращается в drawer/bottom sheet.
- Пользователь выбирает объект слева, работает с действиями справа.

## Touch-first

- Все critical flows должны работать на Android/Windows touch devices.
- Не использовать hover-only или keyboard-only действия.
- Минимальный touch target 48x48 px.
- Long press разрешен для context menu, но всегда нужен альтернативный путь через action rail/overflow.

## Modals

- Модалки используются для процессов: payment, refund, manager override, cash drawer event, precheck cancel, запланированные modifiers/split.
- Modal не меняет текущий section/context.
- Financial/destructive modal должен показывать сумму, действие, последствия и confirmation.

## Drag and drop

- Drag & drop является ускорителем, не единственным способом действия.
- Financial/destructive drag запрещен.
- Transfer через drag требует preview и confirmation.
- Если backend не поддерживает операцию, UI не должен показывать ее как активную.

## Backend boundaries

- Сейчас реализованы: halls/tables read, menu/catalog read, service item секция, selected modifiers при добавлении order line, orders, order lines quantity/void, backend pricing preview, precheck, payments, reprint, whole-check и partial `order_line`/quantity cancellation/refund, compatibility payment refund, shifts, cash sessions, cash drawer events, sync status.
- Сейчас не реализованы как UI runtime: rich partial cancellation/refund scopes by line/modifier/service/tip, cashier discount/surcharge editor, tax policy editor, split bill, transfer/merge tables, KDS lifecycle, delivery/pickup/QR/reservations, real PSP, fiscal adapter.
- Для будущих функций компоненты можно проектировать extensible, но не показывать активную кнопку без ручки.

## RBAC

- Backend permissions authoritative.
- UI visibility - только UX.
- При сомнении действие либо скрыть, либо disabled с понятной причиной, но backend остается security boundary.

## Error and state

- Использовать safe backend error envelope/message_key.
- Не показывать raw SQL/Go errors, PIN, tokens, payment-sensitive payloads.
- Для каждого раздела нужны loading, empty, error, no-permission, locked/offline/sync states.

## i18n

- Пользовательский текст через vue-i18n.
- Не хардкодить русский/английский текст внутри компонентов.

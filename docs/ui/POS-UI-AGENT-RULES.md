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

## Базовые UX-принципы Шнейдермана

POS UI должен руководствоваться восемью золотыми правилами интерфейсного дизайна Бена Шнейдермана. Это обязательная UX-база для всех экранов, компонентов и кодогенерации.

1. Стремиться к согласованности.
   - Одно и то же действие должно выглядеть и работать одинаково в разделах `Залы и столы`, `Заказы`, `Активность`, `Отчеты` и `Касса`.
   - Иконки, цвета, иерархия кнопок, модальные окна, бейджи статусов и empty/error states должны использовать единые правила.

2. Давать опытным пользователям ускорители.
   - Частые POS-действия должны быть доступны за 1-2 касания.
   - Favorites, quick groups, context chips, bottom quick access bar, long press и overflow можно использовать как ускорители.
   - Ускоритель не может быть единственным способом выполнить действие: всегда нужен обычный путь через action rail, modal или меню.

3. Давать информативную обратную связь.
   - Каждая операция заказа, пречека, оплаты, возврата, синхронизации и кассового ящика должна показывать понятный результат: выполнено, выполняется, ожидает, ошибка или требуется действие.
   - Offline/sync состояния должны быть видимы, но не должны ломать нормальный ввод заказа, если операция может быть выполнена локально.

4. Завершать диалоги понятным итогом.
   - Payment, refund, precheck cancel, manager override, cash drawer event и sync retry должны завершаться явным состоянием: выполнено, отменено, ошибка или требуется действие.
   - Пользователь не должен гадать, прошла ли оплата, выпущен ли чек, закрыт ли заказ, отправлен ли возврат или напечатан ли документ.

5. Предотвращать ошибки.
   - Опасные, финансовые и необратимые действия требуют подтверждения, RBAC и понятного описания последствий.
   - Disabled action должен объяснять причину недоступности, если причина не очевидна.
   - UI не должен показывать будущую или backend-неподдержанную функцию как активную кнопку.

6. Разрешать отмену и исправление там, где это допустимо бизнес-правилами.
   - Где возможно, должны быть понятные flow для void, cancel, refund, reprint и retry.
   - Если действие необратимо или ограничено фискальными/юридическими правилами, UI обязан предупредить об этом до подтверждения.

7. Сохранять ощущение контроля у пользователя.
   - POS не должен неожиданно уводить пользователя из текущего стола, заказа или раздела.
   - Modal сохраняет текущий контекст и после закрытия возвращает пользователя туда же.
   - Backend остается авторитетным, но UI обязан объяснять состояние и следующий возможный шаг.

8. Снижать нагрузку на кратковременную память.
   - Текущий стол, заказ, гость/seat, смена, кассовая смена, пречек, оплата и sync state должны быть видимы или доступны без запоминания.
   - Пользователь не должен помнить UUID, технический id, состояние пречека, остаток оплаты или количество несинхронизированных операций.

## Темы оформления

- POS UI должен поддерживать две обязательные темы: светлую и темную.
- Обе темы являются продуктовым и архитектурным требованием, а не косметической опцией.
- Темная тема обязательна для баров, вечерних смен, затемненных залов, kitchen-adjacent зон и экранов, которые долго смотрят в помещении с низким светом.
- Светлая тема обязательна для дневной работы, ярких помещений, backoffice-like сценариев и экранов с высокой информационной плотностью.
- Значение статусов не должно меняться между темами: danger остается danger, success остается success, warning остается warning, info остается info.
- Цвет не должен быть единственным носителем смысла: статус дополнительно поддерживается текстом, иконкой, бейджем или формой компонента.

## Theme tokens

- Не использовать raw/hardcoded colors внутри POS-компонентов.
- Все цвета должны идти через semantic design tokens.
- Минимальный набор токенов:
  - `--pos-bg`;
  - `--pos-surface`;
  - `--pos-surface-raised`;
  - `--pos-border`;
  - `--pos-text-primary`;
  - `--pos-text-secondary`;
  - `--pos-text-disabled`;
  - `--pos-action-primary`;
  - `--pos-action-secondary`;
  - `--pos-status-success`;
  - `--pos-status-warning`;
  - `--pos-status-danger`;
  - `--pos-status-info`;
  - `--pos-sync-pending`;
  - `--pos-sync-error`;
  - `--pos-payment-fiscal`;
  - `--pos-payment-non-fiscal`;
  - `--pos-selected-table`;
  - `--pos-selected-order`.
- Компонент не должен знать, светлая сейчас тема или темная. Компонент использует semantic token, а тема определяет фактическое значение.

## Theme testing

Каждый critical POS flow должен проверяться в светлой и темной теме:

- PIN login;
- выбор зала/стола;
- ввод заказа;
- выбор модификаторов;
- выпуск пречека;
- locked order после пречека;
- оплата;
- возврат/refund;
- перепечатка/reprint;
- кассовые операции;
- offline/sync warning;
- error state;
- no-permission state;
- empty state.

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

- Сейчас реализованы: halls/tables read, menu/catalog read, service item секция, selected modifiers при добавлении order line, orders, order lines quantity/void, backend stop-list sale blocking error handling, backend pricing preview, precheck, payments, reprint, bounded ledger history reads, whole-check и partial `order_line`/quantity cancellation/refund, compatibility payment refund, shifts, cash sessions, cash drawer events, sync status.
- Сейчас не реализованы как UI runtime: rich partial cancellation/refund scopes by line/modifier/service/tip, cashier discount/surcharge editor, tax policy editor, split bill, transfer/merge tables, KDS lifecycle, delivery/pickup/QR/reservations, real PSP, fiscal adapter.
- Для будущих функций компоненты можно проектировать extensible, но не показывать активную кнопку без ручки.

## RBAC

- Backend permissions authoritative.
- UI visibility - только UX.
- При сомнении действие либо скрыть, либо disabled с понятной причиной, но backend остается security boundary.

## Error and state

- Использовать safe backend error envelope/message_key.
- Не показывать raw SQL/Go errors, PIN, tokens, payment-sensitive payloads.
- Для stop-list conflict использовать backend `message_key` `errors.stopListConflict`; UI не должен сам считать stock availability или блокировать продажу по stock balance.
- Для каждого раздела нужны loading, empty, error, no-permission, locked/offline/sync states.

## i18n

- Пользовательский текст через vue-i18n.
- Не хардкодить русский/английский текст внутри компонентов.

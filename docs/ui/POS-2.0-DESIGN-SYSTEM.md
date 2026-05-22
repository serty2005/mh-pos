# POS 2.0 Design System

Статус: обязательный дизайн-контракт для POS UI, BackOffice UI и будущих экранов MyHoreca POS.

Документ фиксирует не философию, а практические правила: как должны выглядеть и вести себя компоненты, состояния, темы, статусы и пользовательские сценарии. Он дополняет `design.md`, `docs/ui/POS-UX-GUIDELINES.md` и `docs/ui/POS-UI-AGENT-RULES.md`.

Если дизайн-правило конфликтует с фактическим backend/API/runtime, источником истины остаются код, тесты и backend specs. UI не должен показывать backend-неподдержанную функцию как активную.

## 1. Назначение

`POS 2.0 Design System` нужен, чтобы несколько разработчиков, дизайнеров и UI-агентов создавали единый продукт, а не набор визуально разных экранов.

Design System отвечает на вопросы:

- какие цвета, размеры, отступы и типографика используются;
- какие компоненты допустимы;
- как выглядят состояния заказа, стола, оплаты, смены и синхронизации;
- какие действия считаются primary/secondary/destructive;
- как работают light/dark темы;
- как проектировать payment, refund, precheck, manager override и sync states;
- что запрещено показывать без backend-контракта.

## 2. Главные принципы

### 2.1 POS — это рабочий инструмент, а не сайт

POS используется в шумной, быстрой и стрессовой ресторанной среде. Поэтому:

- скорость важнее декоративности;
- ясность важнее визуальных эффектов;
- предотвращение ошибок важнее плотности функций;
- каждое действие должно иметь понятный результат;
- пользователь не должен гадать, что произойдет после нажатия.

### 2.2 Четкость и прямота

Пользователь должен легко понимать:

- какие действия сейчас доступны;
- какие действия недоступны и почему;
- что произойдет после нажатия;
- является ли действие безопасным, финансовым или destructive;
- завершилась ли операция успешно.

Правила:

- primary action всегда визуально выделен;
- destructive action не выглядит как обычная кнопка;
- disabled action объясняет причину, если она не очевидна;
- финансовое действие показывает сумму и последствия;
- после payment/refund/precheck/cash operation всегда показывается результат.

### 2.3 Снижение когнитивной нагрузки

UI должен помогать пользователю сосредоточиться на текущей задаче.

Правила:

- показывать только релевантные действия текущего контекста;
- не выводить backend/debug-детали в runtime UI;
- не перегружать POS-экран backoffice-функциями;
- не заставлять пользователя помнить id, суммы, состояние пречека или sync state;
- частые действия держать ближе, редкие — в overflow/modal;
- группировать связанные функции: заказ отдельно, оплата отдельно, касса отдельно, диагностика отдельно.

### 2.4 Доступность

POS должен быть пригоден для пользователей с разным зрением, моторикой и уровнем опыта.

Правила:

- минимальный touch target 48x48 px;
- icon-only actions требуют `aria-label` и tooltip/hint;
- цвет не является единственным носителем смысла;
- критичные статусы имеют label + icon/badge + цвет;
- контраст проверяется в светлой и темной теме;
- focus states обязательны для Windows terminals и keyboard fallback;
- формы должны иметь видимые labels;
- ошибки формы должны быть связаны с конкретными полями.

### 2.5 Визуальная иерархия

Информация распределяется по важности через размер, цвет, контраст, отступы, позицию и плотность.

Правила:

- сумма заказа и остаток оплаты крупнее вторичных деталей;
- primary action крупнее secondary actions;
- опасное действие визуально отделено от обычных;
- статус заказа/пречека/оплаты видим без поиска;
- технические детали спрятаны в diagnostics/debug;
- пользовательский взгляд ведется по рабочему сценарию: объект -> состояние -> действие -> результат.

### 2.6 Шнейдерман обязателен

Весь UI должен следовать восьми правилам Шнейдермана:

1. согласованность;
2. ускорители для опытных пользователей;
3. информативная обратная связь;
4. завершение диалогов понятным итогом;
5. предотвращение ошибок;
6. возможность отмены/исправления там, где это допустимо;
7. ощущение контроля у пользователя;
8. снижение нагрузки на кратковременную память.

## 3. Visual identity

Целевой стиль: чистый B2B SaaS + industrial restaurant POS.

Ориентиры:

- Square — простота, item grid, быстрый checkout;
- Toast — table-service логика, check details, действия рядом с заказом;
- Lightspeed — понятная навигация и операционные разделы;
- iiko/Syrve — ресторанная плотность, столы, гости, пречек, касса;
- Stripe/Linear-like SaaS — аккуратная визуальная дисциплина для BackOffice.

Запрещено:

- декоративный glassmorphism;
- 3D-иконки в runtime POS;
- слабый контраст;
- мелкий серый текст для critical info;
- цветные элементы без семантики;
- разные стили иконок на одном экране;
- лишние анимации, которые не дают обратной связи.

## 4. Theme system

POS UI обязан поддерживать две полноценные темы:

- светлую;
- темную.

Темы являются архитектурным требованием.

### 4.1 Светлая тема

Используется для:

- дневных ресторанов и кафе;
- ярких помещений;
- BackOffice;
- отчетов;
- настройки номенклатуры, меню, цен и прав;
- экранов с высокой информационной плотностью.

### 4.2 Темная тема

Используется для:

- баров;
- вечерних и ночных смен;
- затемненных залов;
- kitchen-adjacent зон;
- терминалов, на которые долго смотрят при низком освещении.

### 4.3 Правила тем

- Семантика цветов не меняется между темами.
- Danger остается danger.
- Success остается success.
- Warning остается warning.
- Info остается info.
- Fiscal/payment status не должен конфликтовать с danger/success.
- Цвет не является единственным носителем смысла.
- Компонент не должен знать, какая тема активна. Он использует semantic token.

## 5. Design tokens

Компоненты не используют hardcoded colors. Все визуальные значения идут через токены.

### 5.1 Color tokens

```css
--pos-bg
--pos-surface
--pos-surface-raised
--pos-border
--pos-border-strong
--pos-text-primary
--pos-text-secondary
--pos-text-muted
--pos-text-disabled
--pos-action-primary
--pos-action-primary-hover
--pos-action-secondary
--pos-status-success
--pos-status-warning
--pos-status-danger
--pos-status-info
--pos-sync-pending
--pos-sync-error
--pos-payment-fiscal
--pos-payment-non-fiscal
--pos-selected-table
--pos-selected-order
--pos-focus-ring
```

### 5.2 Layout tokens

```css
--pos-bottom-nav-height
--pos-action-rail-width
--pos-action-rail-min-width
--pos-card-radius
--pos-modal-radius
--pos-sheet-radius
--pos-gap-xs
--pos-gap-sm
--pos-gap-md
--pos-gap-lg
--pos-gap-xl
```

### 5.3 Touch tokens

```css
--pos-touch-target-min: 48px
--pos-button-height: 56px
--pos-button-height-critical: 64px
--pos-menu-tile-height: 112px
--pos-table-tile-height: 120px
```

## 6. Color semantics

| Роль | Значение | Правило |
|---|---|---|
| Primary | основное безопасное действие | одно главное действие в контексте |
| Secondary | вторичное действие | не конкурирует с primary |
| Success | выполнено, оплачено, готово | не использовать декоративно |
| Warning | pending, внимание, требуется проверка | не использовать для обычного акцента |
| Danger | ошибка, отмена, destructive | всегда требует осторожности |
| Info | справочная информация | не конкурирует с действиями |
| Fiscal | фискальная операция/состояние | отдельно от обычной оплаты |
| Sync pending | ожидает синхронизации | не путать с ошибкой оплаты |
| Sync error | проблема синхронизации | показывать следующий шаг |
| Disabled | недоступно | причина должна быть понятна |

## 7. Typography

POS typography функциональная, не маркетинговая.

Роли:

- `screen-title` — название раздела;
- `panel-title` — заголовок панели/rail;
- `entity-title` — стол, заказ, блюдо, чек;
- `order-total` — итог заказа;
- `payment-remaining` — остаток оплаты;
- `price` — цена позиции;
- `status-label` — статус;
- `helper-text` — подсказка;
- `error-text` — ошибка;
- `debug-text` — только diagnostics/debug.

Правила:

- суммы, цены и short ids используют tabular numbers;
- critical info не выводится мелким шрифтом;
- helper text не должен конкурировать с основным действием;
- BackOffice может быть плотнее POS, но не за счет читаемости.

## 8. Layout system

### 8.1 POS shell

Базовая схема:

```text
Основная область 3/4 + action rail 1/4 + bottom quick access bar
```

Правила:

- слева пользователь выбирает объект или вводит заказ;
- справа видит текущий объект и действия;
- bottom bar отвечает за навигацию и краткий контекст;
- financial/destructive flows открываются modal;
- на compact screens action rail становится drawer/bottom sheet.

### 8.2 BackOffice shell

BackOffice не должен копировать POS layout.

BackOffice использует:

- data tables;
- filter bars;
- saved views;
- breadcrumbs;
- side panels;
- forms;
- import/export;
- audit timelines;
- permission matrices.

## 9. Touch target rules

| Элемент | Минимум |
|---|---:|
| Любое touch-действие | 48x48 px |
| Обычная POS-кнопка | 52-64 px высотой |
| Critical POS-кнопка | 64-80 px высотой |
| Плитка блюда | 96-128 px высотой |
| Плитка стола | 96-140 px высотой |
| Icon-only action | 48x48 px + tooltip/aria-label |

Запрещено:

- hover-only critical actions;
- tiny links для оплаты/void/refund;
- мелкие чекбоксы в POS runtime;
- destructive action без подтверждения;
- drag-only сценарии без альтернативы.

## 10. Icon system

Иконки должны быть частью единой иконографики.

Правила:

- использовать один набор иконок в runtime UI;
- не смешивать outline/fill/3D/emoji;
- одинаковая толщина stroke;
- icon-only action требует tooltip/aria-label;
- для payment methods желательно icon + label;
- неизвестная пользователю иконка без label запрещена.

Доменные иконки:

| Сущность | Иконка |
|---|---|
| Заказ | ReceiptText |
| Стол | Armchair / LayoutGrid |
| Гости | Users |
| Оплата | CreditCard |
| Наличные | Banknote |
| Касса | Wallet |
| Смена | Clock |
| Кухня | ChefHat |
| Склад | Warehouse |
| Блюдо | Utensils |
| Товар | Package |
| Модификатор | SlidersHorizontal |
| Скидка | BadgePercent |
| Лояльность | Gift |
| Доставка | Truck |
| Организация | Building2 |
| Ресторан | Store |
| Отчеты | BarChart3 |
| Настройки | Settings |

## 11. Component system

### 11.1 Base components

- Button;
- IconButton;
- POSButton;
- Badge;
- StatusBadge;
- Card;
- Modal;
- Drawer;
- BottomSheet;
- Tabs;
- SearchInput;
- NumericKeypad;
- Toast;
- ConfirmDialog;
- EmptyState;
- ErrorState;
- Skeleton;
- NoPermissionState.

Для каждого компонента должны быть определены:

- назначение;
- размеры;
- варианты;
- состояния;
- disabled behavior;
- loading behavior;
- error behavior;
- light/dark behavior;
- accessibility requirements.

### 11.2 POS components

- FloorSectionTabs;
- TableTile;
- ActiveOrderChip;
- MenuCategoryChip;
- MenuItemTile;
- ModifierPicker;
- OrderRail;
- OrderLine;
- PrecheckBanner;
- PaymentModal;
- PaymentMethodButton;
- CashInput;
- RefundModal;
- ManagerOverrideModal;
- CashDrawerEventModal;
- SyncStatusIndicator;
- ShiftStatusBadge;
- CashSessionStatusBadge.

### 11.3 BackOffice components

- DataTable;
- FilterBar;
- SavedViewTabs;
- EntityForm;
- EntityDrawer;
- AuditTimeline;
- ImportExportPanel;
- PermissionMatrix;
- NomenclatureTree;
- PriceListEditor;
- TaxRuleEditor;
- StockMovementTable.

## 12. Button hierarchy

| Тип | Назначение | Правило |
|---|---|---|
| Primary | главное безопасное действие | одно на контекст |
| Secondary | дополнительное действие | не конкурирует с primary |
| Financial | оплата/возврат/касса | показывает сумму/последствие |
| Destructive | отмена/void/delete/refund | confirmation/RBAC |
| Ghost | вторичная навигация | не для critical action |
| Disabled | недоступное действие | понятная причина |

Правила:

- часто используемые кнопки крупнее и заметнее;
- редкие действия меньше и уходят в overflow/modal;
- кнопки платежных методов должны иметь понятный label;
- финансовая кнопка не должна быть icon-only.

## 13. Status system

### 13.1 Order status

- open;
- precheck issuing;
- precheck issued;
- locked;
- partially paid;
- paid;
- closed;
- cancelled;
- refund pending;
- refund completed;
- sync pending;
- sync failed.

### 13.2 Order line status

- added;
- quantity changed;
- voided;
- requires modifiers;
- unavailable / stop-list conflict;
- planned: sent to kitchen;
- planned: preparing;
- planned: ready;
- planned: served.

### 13.3 Table status

- free;
- occupied;
- selected;
- reserved;
- soon reservation;
- dirty / needs cleaning;
- merged;
- conflict.

Statuses without backend contract must be documented as planned and not shown as active runtime actions.

### 13.4 Payment status

- not started;
- pending;
- partially paid;
- captured;
- failed;
- refunded;
- partially refunded;
- fiscal pending;
- fiscal failed;
- sync pending;
- sync failed.

## 14. Payment UI rules

Payment UI is a critical flow.

Rules:

- payment opens as modal from current order/precheck context;
- user always sees total, paid amount, remaining amount and change if applicable;
- payment method buttons use icon + label;
- no decorative animation in payment flow;
- animation is allowed only as informative feedback, for example processing;
- forms are simple and vertical: label above field;
- primary payment action is visually dominant;
- payment result is explicit: paid, partial, failed, cancelled, requires action;
- user must never be uncertain whether payment was captured;
- split/tip/PSP/fiscal features are shown only when backend supports them.

## 15. Forms

Rules:

- labels above inputs;
- one logical group per form section;
- avoid side-by-side fields in POS runtime;
- numeric input uses numeric keypad where appropriate;
- validation errors are close to the field;
- form submit shows loading state;
- destructive submit requires confirmation;
- long forms belong to BackOffice, not POS runtime.

## 16. Offline and sync states

Required states:

- online;
- offline;
- unstable connection;
- sync pending;
- sync failed;
- local operation saved;
- retry required;
- fiscal device unavailable;
- payment terminal unavailable;
- printer unavailable;
- kitchen printer unavailable.

Rules:

- sync error is not payment error;
- offline does not automatically mean operation failed;
- if local operation is saved, UI says so explicitly;
- retry action must show result;
- sync status is visible but not panic-inducing.

## 17. Modal and destructive action rules

Modal is used for process, not navigation.

Modal required for:

- payment;
- refund;
- precheck cancel;
- manager override;
- cash drawer event;
- destructive confirmation;
- future split bill;
- future transfer table/order;
- future PSP/fiscal problem resolution.

Rules:

- modal preserves current context;
- modal shows action, object, amount if financial, consequences and result;
- destructive action never runs silently;
- financial/destructive drag is forbidden;
- after modal close user returns to same order/table/section.

## 18. Empty, loading and error states

Every section/component needs:

- loading skeleton matching real layout;
- empty state with next action;
- safe error state;
- no-permission state;
- locked/read-only state where applicable;
- offline/sync warning where applicable.

State copy must answer:

1. что произошло;
2. почему это важно;
3. что пользователь может сделать дальше.

## 19. Accessibility checklist

- minimum touch target 48x48 px;
- no color-only status;
- aria-label for icon-only actions;
- tooltip/hint for unfamiliar icons;
- visible focus state;
- contrast checked in light and dark themes;
- text does not rely on tiny font sizes;
- forms have labels;
- error messages are specific;
- keyboard fallback exists for Windows/desktop terminals.

## 20. Motion and sound

Rules:

- animation is informative only;
- no decorative animation in payment/order entry;
- loading animation must not block reading key state;
- sound can be used for KDS/payment/error only as configurable feedback;
- sound is never the only status signal.

## 21. Security and trust in UI

POS handles financial and sensitive operational data.

Rules:

- automatic lock/logout behavior must be visible and predictable;
- manager approval is required for configured sensitive actions;
- payment-sensitive payloads are never shown;
- customer data is minimized;
- UI should not expose raw tokens, PINs, SQL/Go errors or request dumps;
- audit-sensitive actions should show actor, time and result where backend provides it.

## 22. Role-based UI

UI adapts to role and permissions:

- waiter sees order entry/table workflow;
- cashier sees payment/cash session capabilities;
- manager sees override/refund/reprint/cash controls;
- admin/backoffice user sees configuration and reports;
- disabled or hidden actions must match backend permissions.

UI visibility is only UX. Backend remains authoritative.

## 23. Customization and flexibility

Allowed customization:

- favorites/quick groups;
- menu category ordering;
- device start section;
- light/dark theme;
- role-based visible sections;
- future: restaurant/device profile presets.

Rules:

- customization must not break consistency;
- customization must not expose unsupported backend actions;
- per-restaurant/device settings are Cloud-owned and synced to Edge.

## 24. Do / Don't

### Do

- Keep current order visible.
- Show operation result.
- Use semantic tokens.
- Use large touch targets.
- Pair payment icons with labels.
- Explain disabled states.
- Keep payment forms simple.
- Test in light and dark themes.
- Keep POS and BackOffice density different.

### Don't

- Do not use hardcoded colors.
- Do not mix icon styles.
- Do not hide payment in deep navigation.
- Do not use decorative animation in critical flows.
- Do not use tiny destructive links.
- Do not show future functions as active.
- Do not rely on color only.
- Do not leave payment/refund/precheck result ambiguous.
- Do not put inventory/tax/reporting admin UI into order entry.

## 25. External UX references to track

These sources are useful as supporting UX references, not as direct product specs:

- FasterCapital: POS UI principles — clarity, consistency, speed, error recovery, accessibility, security, integrations, feedback, layout grouping, role adaptation and iterative testing.
- Bright Inventions POS development guide — POS feature map, hardware integrations, offline mode, cloud/on-prem tradeoffs, payments, KDS, split bill, inventory, compliance and user personas.
- Bright Inventions payment UI/UX — payment should be fast, animations informative only, payment methods need understandable icons with labels, button hierarchy matters, forms should be simple, split payment should hide calculations from the user.

## 26. Acceptance criteria

A POS UI change is not ready if it:

- violates Shneiderman principles;
- lacks light/dark support;
- uses hardcoded colors;
- has critical action below 48x48 px;
- uses hover-only interaction for critical flow;
- shows unsupported backend functions as active;
- lacks loading/empty/error/no-permission states;
- lacks explicit result after payment/refund/precheck/cash operation;
- uses color as the only status signal;
- makes the user remember context that should be visible.

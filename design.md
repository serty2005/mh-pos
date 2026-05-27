# POS UI/UX design plan

Статус: рабочий дизайн-план для пересмотра `pos-ui` после frozen POS pilot.

Документ фиксирует правила проектирования интерфейса, основанные на фактически реализованном POS Edge backend и текущих UI-контрактах. Он не объявляет будущие backend-возможности реализованными. Если план конфликтует с кодом, тестами, `SPECv1.3.md` или профильной документацией, источником истины остается фактический runtime.

## Цель

Сделать главный POS-экран быстрым для сотрудников ресторана: минимум переходов, крупные touch-targets, постоянный доступ к текущему заказу, понятное состояние смены/кассы/синхронизации и единая навигация по всем ежедневным операциям.

Основной UX-ориентир:

- Square POS: простая нижняя навигация, быстрый item grid, favorites/shortcuts, понятный checkout.
- iiko/Syrve: ресторанная плотность, залы/столы, гости, работа с заказом, пречек, касса, модификаторы, курсы подачи и операции менеджера.
- Toast/Lightspeed/Poster/Clover/r_keeper: table service workflows, список заказов, quick order, видимость статусов столов, быстрые платежи, кассовые операции, отчеты и операционный профиль сотрудника.

## Зафиксированные продуктовые решения

### Primary workflow

POS UI проектируется как waiter-first restaurant workspace. Основной пользователь - официант, который выбирает зал/стол, ведет активный заказ, работает с гостями и позициями, выпускает пречек и передает заказ к оплате. Кассирские, менеджерские и отчетные операции доступны в тех же разделах по RBAC, но не являются главным навигационным сценарием.

Happy path:

```text
PIN login -> зал/стол -> активный заказ -> добавление позиций -> гости/количества/void -> пречек -> оплата/закрытие
```

### Главный UX-контекст

Главный UX-якорь - рабочий контекст обслуживания. В table-service режиме это `стол + активный заказ`. В quick-service/single-table режиме это `активный заказ`. Пречеки, платежи, гости, скидки, возвраты и manager override являются дочерними процессами выбранного контекста.

UI не должен заставлять пользователя думать в терминах backend-сущностей. Backend-сущности используются для состояния, прав и действий, но интерфейс показывает их как единый рабочий процесс обслуживания гостя.

### Стартовый раздел после PIN

Стартовый раздел определяется умно:

1. Если в Cloud-настройках ресторана/устройства явно задан стартовый POS section, открыть его.
2. Иначе если в ресторане больше одного стола, открыть `залы и столы`.
3. Иначе открыть `заказы`.

До появления Cloud-owned setting UI использует fallback-правило:

```text
tables_count > 1 ? floor : orders
```

Настройка должна быть запланирована как Cloud-owned restaurant/device setting, доставляемая на Edge через master/config sync.

### Kitchen/KDS

Kitchen Display System является обязательным будущим направлением, но проектируется как отдельный shell, а не как раздел основного POS. Основной POS UI должен быть готов к будущим kitchen fields на order lines: статус приготовления, станция, course, send/hold/fire state, elapsed time. До появления backend-контракта KDS-статусы не показываются как активные действия.

### Touch-first devices

Основные устройства: Android touch tablets/terminals и Windows touch POS terminals. UI является web-интерфейсом от локального Go Edge service. Все critical flows должны работать без физической клавиатуры, hover-only interaction и desktop-only shortcuts.

### Long press and context menus

Для touch-экранов long press является стандартным способом открытия контекстного меню. Он используется для редких или объектных действий:

- строка заказа: void сейчас; split, move to guest, note и modifier edit после добавления backend/UI contract;
- гость: split/check by guest, rename и move items только `запланировано далее`;
- стол: transfer, merge, reservation и mark status только `запланировано далее`;
- closed order/payment: refund, reprint, details.

Каждое long-press действие должно иметь альтернативный доступ через action rail или overflow.

### Drag and drop

Drag & drop поддерживается как ускоритель ресторанных операций, но не как единственный способ выполнить действие.

Первичные сценарии:

- перенос заказа со стола на стол после появления backend contract;
- перенос позиций между гостями — `запланировано далее`;
- перенос позиций между заказами — `запланировано далее`;
- редактирование floor plan — `запланировано далее`;
- сортировка favorites/menu shortcuts — `запланировано далее`.

Правила:

- financial/destructive действия через drag запрещены;
- перенос заказа требует preview и подтверждение;
- при отсутствии backend-контракта drag-сценарий остается в design/backlog, но компоненты проектируются с учетом будущей поддержки;
- любой drag-flow обязан иметь touch menu fallback.

### Multi-order mode

По умолчанию table-service режим разрешает один активный заказ на один стол. В single-table/quick-service режиме ресторан может разрешить несколько активных заказов на одном столе. Тогда активные заказы отображаются как переключаемые order tabs/chips в нижней панели или в action rail до момента оплаты/закрытия.

Multi-order mode является Cloud-owned restaurant/device setting. До появления backend-поддержки UI не должен показывать множественные активные заказы как реализованную возможность.

## Фактический backend surface

### Реализовано сейчас

POS Edge backend уже дает устойчивую runtime-основу для waiter-first restaurant workspace:

- Pairing/provisioning: health, pairing status, license/cloud pairing.
- Auth/session: PIN login, logout, session lookup, actor context и permissions.
- Смены сотрудников: открыть смену, закрыть смену, текущая смена, последние смены.
- Кассовые смены: открыть/закрыть кассовую смену, получить текущую кассовую смену.
- Кассовый ящик: cash in, cash out, no sale, cash count через `cash-drawer-events`.
- Залы и столы: чтение halls/tables из Edge read model.
- Каталог и меню: чтение catalog/menu items, services и modifier groups/options из текущего menu item contract.
- Заказы: создать заказ, получить текущий заказ по столу, получить заказ, получить закрытые заказы, добавить позицию с selected modifiers, изменить количество, удалить позицию через void, закрыть заказ.
- Пречек: выпустить, получить, получить список по заказу, отменить через manager override, перепечатать копию.
- Оплата и чек: принять payment по precheck, получить check, перепечатать check, вернуть captured payment.
- Sync/операционная диагностика: status, outbox, local events, retry failed outbox.
- RBAC: backend permissions являются авторитетными, UI visibility остается UX-слоем.

### Ограничения текущего runtime

Эти области нельзя показывать как готовую функциональность без отдельного backend/API изменения:

- расширенное редактирование modifiers после добавления строки и отдельный cashier modifier catalog editor;
- ручной cashier editor для скидок, надбавок, tax rules и price overrides;
- inventory consumption, recipe expansion и списания;
- KDS lifecycle;
- delivery/channel runtime;
- real PSP/payment terminal integration;
- fiscal adapter;
- полноценные отчеты за пределами доступных closed orders, payments/checks и sync/cash-session data;
- перенос/объединение столов, split bill, split by guest, courses, kitchen send/hold/stay.

### UI gap относительно backend

Текущий `pos-ui` уже использует большую часть runtime surface, но организован как один терминал с верхним status bar, левым выбором столов, центральным заказом и правым catalog/checkout/actions panel. Для новой навигации нужно не добавлять много экранов поверх старого terminal view, а разложить уже реализованные операции по постоянным разделам:

- `залы и столы`: halls/tables/current order by table/create order.
- `заказы`: menu grid + active order rail + order actions/precheck/payment modals.
- `активность`: closed orders, final checks, payments, refund, reprint.
- `отчеты`: только реализуемые сейчас operational summaries; расширенные отчеты пометить как `запланировано далее`.
- `касса`: employee shift, cash session, cash drawer, sync health, service diagnostics.

## Принципы из референсов

### Square POS

Сильные стороны:

- item grid превращает кассу в быстрый touch-интерфейс;
- favorites/shortcuts позволяют вынести частые товары, скидки и операции на один экран;
- нижняя навигация не конкурирует с рабочей областью;
- на малом экране checkout превращается в последовательный review/pay flow.

Что берем:

- крупная плиточная сетка меню;
- быстрые группы и избранное;
- постоянный bottom quick access bar;
- checkout/order rail всегда рядом с выбором блюд.

Что адаптируем:

- Square ближе к retail/quick checkout, поэтому для ресторана добавляем залы, столы, гости, пречек, кассовую смену и manager override.

### iiko/Syrve

Сильные стороны:

- ресторанная логика видна на первом экране: гости, столы, заказ, категории, касса;
- плотность выше, чем у retail POS, но важные действия остаются крупными;
- хорошо ложатся flows с модификаторами, курсами подачи, разделением по гостям и оплатой;
- менеджерские операции не спрятаны слишком далеко, но защищены правами и PIN.

Что берем:

- статусную карту зала;
- заказ как постоянную рабочую сущность;
- категории блюд рядом с плитками;
- bottom action bar для основных операций;
- модальные процессы для оплаты, отмены пречека, модификаторов, скидок и возвратов.

Что адаптируем:

- не копируем перегруженность старых desktop POS; оставляем единый layout и предсказуемые панели.

### Toast

Сильные стороны:

- table service screen показывает гостя, официанта, сумму и время открытия заказа на столе;
- table details pane можно скрывать;
- quick order и table order имеют разные наборы быстрых действий;
- менее частые действия уходят в overflow текущего заказа;
- на handheld важные действия вроде print/pay закреплены внизу.

Что берем:

- карточка стола должна показывать статус, сумму, время, гостя/сотрудника, если backend это отдает;
- боковое меню и action rail должны уметь скрываться;
- overflow только для редких действий текущего раздела;
- оплата и печать должны быть постоянно доступны из заказа, когда состояние разрешает.

### Lightspeed Restaurant

Сильные стороны:

- основная навигация сведена к нескольким частым разделам;
- Tables открывают floor plan и статус столов;
- Orders List дает общий список и поиск по заказам;
- Profile объединяет cash drawer, reports, shift/day close.

Что берем:

- ровно пять нижних разделов для daily operations;
- `активность` как список закрытых заказов/платежей с фильтрами;
- `касса` как профиль операционной смены, кассового ящика и технической готовности.

### Poster/Clover/r_keeper

Сильные стороны:

- разные service modes: dine-in, takeout, delivery;
- частичные оплаты, reprint, edit payment method, split receipts;
- handheld/table-side operations;
- интеграция online orders в единый поток.

Что берем как направление:

- будущие service modes проектировать как режимы внутри `заказы`, а не как отдельные приложения;
- split/merge/transfer, delivery и KDS размещать в карте будущих backend contracts;
- не перегружать pilot UI кнопками без работающей ручки.

## Целевая информационная архитектура

### Bottom quick access bar

Нижняя строка быстрого доступа является постоянной на всех основных POS-экранах и отвечает за навигацию и быстрый контекст, а не за полный набор действий раздела.

Структура:

1. Слева: кнопка текущего раздела — иконка + короткое название. Нажатие открывает/закрывает скрываемое боковое меню разделов.
2. Центр: context chips текущего рабочего контекста, максимум 3-5 элементов: выбранный стол, активный заказ, гость/seat, статус пречека, quick-service order tab, если такой режим разрешен settings/backend.
3. Справа: lock/logout, sync status, actor/session status и compact cash/shift status, если они помещаются без визуального перегруза.

Основные разделы POS:

- `Залы и столы`;
- `Заказы`;
- `Активность`;
- `Отчеты`;
- `Касса`.

Правила:

- active section должен быть виден без чтения длинного текста: иконка + короткий label;
- нажатие на левую кнопку bottom bar повторно закрывает открытое боковое меню;
- bottom bar не должен перекрывать важные кнопки оплаты, пречека и кассовых операций;
- context chips не заменяют основной экран и не превращаются в список всех действий;
- запрещено размещать в bottom bar действия, которые требуют длинного ввода, подтверждения, manager override или financial/destructive flow. Такие действия открывают modal;
- служебные UUID, node id и debug-информация не показываются в bottom bar runtime UI.

### Скрываемое боковое меню

Боковое меню служит навигацией между разделами и коротким операционным контекстом. Оно не занимает постоянное место на экране и не конкурирует с рабочей областью.

Правила:

- полностью скрыто по умолчанию;
- открывается по кнопке текущего раздела в bottom bar;
- показывает все пять разделов: `Залы и столы`, `Заказы`, `Активность`, `Отчеты`, `Касса`;
- показывает краткие статусы: сотрудник, личная смена, кассовая смена, синхронизация, роль/права;
- закрывается при выборе раздела, по клику вне меню, по `Escape` и по повторному нажатию на кнопку текущего раздела;
- на tablet/desktop открывается overlay-панелью шириной примерно 280-320 px;
- на handheld/mobile открывается как full-height sheet;
- не содержит длинные формы, кассовые процессы, оплату, refund или диагностику outbox в раскрытом виде. Эти сценарии остаются в разделах или modal-процессах.

### Единый layout раздела

Каждый раздел использует одну схему:

```text
┌──────────────────────────────────────────────────────────────┐
│ Основная рабочая область, 3/4        │ Action rail, 1/4       │
│                                      │                       │
│ Пользователь выбирает объект         │ Пользователь видит     │
│ или работает с основной сеткой       │ выбранный объект и     │
│                                      │ доступные операции     │
└──────────────────────────────────────┴───────────────────────┘
```

Правило:

- слева пользователь выбирает или редактирует основной объект;
- справа пользователь видит текущий объект и доступные действия;
- долгие, сложные и опасные операции не раскрываются прямо в rail, а открывают modal;
- на малых экранах action rail становится bottom sheet или drawer, но сохраняет тот же mental model.

CSS-правило для desktop/tablet:

```css
.pos-section-layout {
  display: grid;
  grid-template-columns: minmax(0, 3fr) minmax(320px, 1fr);
  min-height: calc(100dvh - var(--bottom-nav-height));
}
```

## Первый этап редизайна: экран выбранного заказа

Первым реализуемым экраном нового POS shell становится экран выбранного заказа в разделе `Заказы`. Это не общий список заказов и не кассовая админ-панель, а рабочий Square-like экран table-service/quick-service order entry: 3/4 экрана занимает меню блюд плитками, 1/4 экрана занимает текущий заказ.

Целевой layout:

```text
┌──────────────────────────────────────────────────────────────┐
│ Верхний компактный контекст, если нужен                      │
├──────────────────────────────────────────┬───────────────────┤
│ Меню плитками, 3/4 экрана                 │ Текущий заказ     │
│                                          │ 1/4 экрана        │
│ [Категории / Избранное / Поиск]           │                   │
│                                          │ Стол / заказ      │
│ [Плитка блюда] [Плитка блюда]             │ Состав заказа     │
│ [Плитка блюда] [Плитка блюда]             │ Суммы             │
│ [Плитка блюда] [Плитка блюда]             │                   │
│                                          │ [Действия]        │
│                                          │ [Пречек]          │
└──────────────────────────────────────────┴───────────────────┘
│ Нижняя строка быстрого доступа                                │
└──────────────────────────────────────────────────────────────┘
```

### Левая часть: Square-like menu grid

В 3/4 основной области располагаются:

- категории меню, избранное/быстрые группы и поиск;
- спокойная сетка крупных touch-friendly плиток блюд;
- skeleton, повторяющий реальную сетку плиток, а не абстрактную строку;
- пустое состояние с понятным следующим действием.

Плитка блюда должна поддерживать:

- название;
- цену;
- картинку, если появится подтвержденный media contract;
- fallback с иконкой/initials, если картинки нет;
- статус доступности;
- disabled-состояние с понятным hint;
- добавление позиции в заказ по нажатию;
- modal выбора modifiers для позиций с `modifier_groups`, потому что текущий UI/backend уже поддерживает selected modifiers при добавлении строки.

Визуальные требования: крупные плитки 96-128 px на tablet/desktop, единые отступы, минимум мелких рамок, понятная типографическая иерархия, без технических статусов backend на поверхности меню.

### Правая часть: current order rail

В 1/4 action rail показывается только текущий заказ и разрешенные действия над ним:

- заголовок текущего заказа;
- контекст: стол, short id заказа, статус заказа, статус пречека, если есть;
- список строк заказа;
- quantity, цена строки и итог;
- backend-provided totals;
- empty/error/no-permission states;
- две крупные основные кнопки внизу до выпуска пречека: `Действия` и `Пречек`.

Правая панель не должна содержать меню блюд, операции кассы, служебную диагностику, технические UUID, node id, лишние backend-статусы или длинные формы. Технические детали уходят в диагностику, dev/debug режим или раздел `Касса`.

### Кнопка `Действия`

`Действия` открывает modal поверх текущего UI и не меняет раздел. В modal можно показывать только действия, которые реально поддержаны backend/API и разрешены текущим состоянием заказа/RBAC.

Реализуемые сейчас операции по текущему backend/UI surface:

- изменение quantity строки;
- void строки;
- выбор modifiers при добавлении позиции с modifier groups;
- перепечать пречека/check там, где уже есть snapshot и permission;
- refund captured payment через compatibility route в контексте закрытых заказов, а не как произвольная операция редактируемого заказа.

Запланировано далее/backlog, не показывать как активные кнопки без backend-контракта:

- split bill;
- перенос заказа на другой стол;
- merge/transfer столов;
- добавить гостя или переместить позиции между гостями, если нет активного backend/settings support;
- комментарий к заказу/кухне;
- ручная скидка/надбавка/price override;
- check-level cancellation/refund ledger UI by line/quantity/scope;
- manager override для новых сценариев;
- KDS send/hold/fire lifecycle;
- реальные PSP/fiscal операции.

### Кнопка `Пречек` и locked state

До выпуска пречека справа остаются две основные кнопки: `Действия` и `Пречек`. При нажатии `Пречек` запускается выпуск пречека; если требуется подтверждение, открывается modal. Во время выпуска нужен loading state и защита от повторного нажатия.

После успешного выпуска пречека заказ переходит в locked state:

- строки заказа read-only и не меняются свободно;
- добавление блюд из меню блокируется или меню становится disabled с понятным hint;
- `Действия` скрывается или становится disabled, если текущий locked state запрещает операции;
- `Пречек` заменяется на две крупные кнопки: `Касса` и `Отмена пречека`;
- состояние `Пречек выпущен / заказ заблокирован` визуально очевидно в заголовке rail.

Целевое состояние rail после пречека:

```text
Текущий заказ
Стол 4
Пречек выпущен
Заказ заблокирован

[Состав заказа read-only]

Итого: 1 490 ₽

[Касса]
[Отмена пречека]
```

### Modal оплаты

Кнопка `Касса` открывает modal оплаты поверх текущего заказа. Пользователя не нужно переводить в отдельный раздел только для оплаты выбранного заказа.

Modal оплаты показывает:

- сумму к оплате и remaining total;
- методы оплаты, поддержанные текущим backend: cash и trusted manual card;
- состояние личной/кассовой смены;
- безопасные ошибки через normalized `message_key`;
- результат оплаты;
- переход заказа/чека после полной оплаты согласно текущему backend flow: final check создается после полного покрытия precheck payments, затем доступно закрытие заказа и reprint check.

Real PSP/payment terminal integration и fiscal adapter остаются `запланировано далее` как целевой backend contract.

### Modal отмены пречека

Кнопка `Отмена пречека` открывает modal отмены пречека. Modal должен показывать предупреждение, reason, manager override input, безопасную ошибку и результат. При успешной отмене unpaid precheck заказ возвращается в редактируемое состояние `open`.

Текущий backend/UI уже поддерживает отмену unpaid issued precheck через manager override. Отмена оплаченного пречека, произвольные partial check cancellations и rich refund ledger UI не должны выглядеть как готовые операции.

### Состояния экрана выбранного заказа

- `No selected order`: левая часть может показывать меню в disabled/preview режиме или empty state; правая часть объясняет, что нужно выбрать стол/заказ, и дает переход в `Залы и столы`.
- `Editable order`: меню активно, добавление позиций доступно по permissions, quantity/void доступны по permissions, справа кнопки `Действия` и `Пречек`.
- `Precheck issuing`: loading state, idempotency/retry-safety на уровне UX, защита от повторного нажатия, безопасная ошибка.
- `Precheck issued / locked order`: заказ read-only, меню disabled или add blocked с hint, вместо `Пречек` показываются `Касса` и `Отмена пречека`.
- `Payment modal open`: контекст заказа сохраняется, modal оплаты поверх UI, после успешной оплаты показывается понятное подтверждение и backend-состояние check/order.
- `Closed order`: read-only, действия редуцированы до view/reprint/refund там, где это разрешено и относится к `Активность`.
- `Error / no permission / offline`: безопасные ошибки, no-permission state, sync/offline warning, без raw SQL/Go/backend details.

### Ограничения текущего backend/UI для первого этапа

Первый этап не должен обещать как рабочие функции: split bill, transfer/merge tables, произвольное добавление гостей/seat allocation без settings/API, ручные скидки, KDS lifecycle, полноценные отчеты, real PSP/payment terminal, fiscal adapter и delivery/channel runtime. Эти пункты можно держать в design/backlog только как `запланировано далее` или как целевой backend contract.

## Разделы

### Залы и столы

Цель: быстро понять занятость ресторана, выбрать стол и открыть нужный активный заказ. Этот раздел отделен от экрана выбранного заказа: здесь пользователь выбирает table-service контекст, а не вводит блюда.

Целевой layout:

```text
┌──────────────────────────────────────────┬───────────────────┐
│ Столы большими плитками, 3/4              │ Заказы быстрым    │
│                                          │ списком, 1/4      │
│ [Зал: Основной] [Зал: Терраса]            │                   │
│                                          │ Активные заказы   │
│ [Стол 1] [Стол 2] [Стол 3]                │ #101 Стол 1       │
│ [Стол 4] [Стол 5] [Стол 6]                │ #102 Стол 5       │
│                                          │ #103 Без стола    │
└──────────────────────────────────────────┴───────────────────┘
```

Основная область 3/4:

- tabs/chips залов сверху;
- большие touch-friendly плитки столов или будущий floor-plan;
- каждая плитка стола показывает номер/название, статус, наличие активного заказа;
- если данные доступны из backend/read model, плитка дополнительно показывает сумму заказа, время открытия/возраст заказа, количество гостей и официанта/сотрудника;
- плитка стола является главным входом в table-service заказ.

Action rail 1/4:

- вертикальный список всех активных заказов для быстрого выбора;
- short id заказа, стол, сумма/статус, если доступно;
- фильтр/поиск, если список длинный;
- выбранный заказ можно открыть в разделе `Заказы`;
- создание заказа для выбранного стола;
- обновление данных.

Не нужно перегружать rail кассовыми действиями, оплатой, refund или sync diagnostics.

Реализуемо сейчас:

- halls/tables, выбор стола, current order by table, create order.

Запланировано далее:

- единый endpoint/модель активных заказов для rail, визуальный drag floor plan, расширенные статусы занятости beyond active order, server assignment, reservations, transfer/merge.

### Заказы

Цель: рабочий экран выбранного заказа для быстрого ввода блюд и контроля текущего заказа. Это не общий список заказов. Поиск/выбор активного заказа относится к `Залы и столы`, а закрытые заказы, платежи, reprint и refund относятся к `Активность`.

Если заказ открыт со стола, экран `Заказы` сохраняет table context. Возврат назад ведет к тому же столу/залу. В single-table/quick-service режиме `Заказы` может быть стартовым разделом; переключение между несколькими активными заказами через chips/tabs допустимо только когда это разрешено backend/settings.

Основная область 3/4:

- категории меню, избранное/быстрые группы и поиск;
- Square-like сетка крупных плиток меню;
- плитки показывают название, цену, картинку при наличии media contract, fallback, availability/status и disabled hint;
- добавление позиции в заказ по нажатию;
- modal выбора modifiers для menu items with modifier groups;
- skeleton повторяет реальную сетку плиток.

Action rail 1/4:

- текущий открытый заказ;
- стол, short id заказа, статус заказа, статус пречека;
- строки заказа с quantity/void по permissions и состоянию заказа;
- backend totals;
- две крупные основные кнопки в editable state: `Действия` и `Пречек`;
- после issued precheck/locked state: `Касса` и `Отмена пречека`;
- closed/read-only state с редуцированными действиями view/reprint, если такой заказ открыт из `Активность`.

Реализуемо сейчас:

- menu items/services, selected modifiers при добавлении строки, add line, quantity, void, issue precheck, cancel unpaid precheck через manager override, payment cash/card manual, reprint precheck/check, close order, closed orders/refund compatibility flow.

Запланировано далее/backlog:

- favorites/quick groups на отдельном backend/config contract, menu item images media contract, richer order actions modal, comments, guest/seat allocation, split bill, transfer/merge tables, ручные discounts/surcharges/price overrides, KDS send/hold/fire lifecycle.

### Активность

Цель: быстро найти закрытые заказы, платежи, чеки и выполнить разрешенные постоперации.

Основная область 3/4:

- список/фильтры закрытых заказов и платежей;
- поиск и фильтры: дата, стол, сумма, payment status, refund status, если эти поля надежно доступны;
- compact timeline-card заказа: стол, сумма, check status, payments, business date.

Action rail 1/4:

- детали выбранного закрытого заказа;
- платежи и статусы;
- reprint check;
- refund captured payment через compatibility route, если есть permission и открыта cash session;
- перепечатка и безопасная история действий, если backend предоставляет данные;
- sync/audit hint по операции.

Реализуемо сейчас:

- closed orders, check payments, refund captured payment через compatibility route, reprint check.

Запланировано далее:

- возвраты/отмены по строкам, количествам и scope через rich ledger UI, расширенный поиск по receipt/customer/card last digits, edit payment method, split refunds, audit trail view.

### Отчеты

Цель: дать сотруднику и менеджеру легкие операционные итоги без обещания полноценной аналитики.

Основная область 3/4:

- текущая личная смена;
- текущая кассовая смена;
- закрытые заказы за локальный business date, если данные надежно доступны;
- суммы оплат по методам, если backend уже позволяет корректно посчитать по check/payment snapshots.

Action rail 1/4:

- переход к кассовой смене;
- sync health;
- предупреждение о том, что расширенные отчеты пока `запланировано далее`;
- печать/экспорт отчета только после появления backend contract.

Реализуемо сейчас:

- только легкие operational summaries на основе current/recent shift, current cash session, closed orders и sync status.

Запланировано далее / целевой backend contract:

- Z/X reports, revenue by category, staff sales, tax reports, inventory/cost reports, Cloud analytics.

### Касса

Цель: все сменные, кассовые и локально-операционные сервисные действия в одном месте. Оплата выбранного заказа запускается modal из раздела `Заказы`; раздел `Касса` нужен для смен, cash drawer и диагностики.

Основная область 3/4:

- личная смена;
- кассовая смена;
- cash drawer events;
- sync health;
- local diagnostics;
- lock/logout entry points, если они не перегружают bottom bar.

Action rail 1/4:

- открыть/закрыть личную смену;
- открыть/закрыть кассовую смену;
- cash in/cash out/no sale/cash count modal;
- sync retry для ролей с правом;
- lock terminal/logout.

Реализуемо сейчас:

- open/close employee shift, current/recent shifts, open/close cash session, record cash drawer event, sync status/outbox/local events/retry.

Запланировано далее:

- list cash drawer events, close day, reconciliation report, fiscal close, cash discrepancy workflow.

## Модалки и процессы

Модалки используются для процессов, а не для постоянной навигации. Modal не меняет текущий раздел: после закрытия пользователь возвращается к тому же заказу, столу или выбранному объекту.

Обязательные modal-процессы:

- оплата выбранного пречека;
- действия над заказом;
- выбор модификатора;
- выбор скидки/надбавки, когда появится подтвержденный backend/API contract;
- отмена пречека;
- manager override;
- возврат оплаты;
- cash in/cash out/no sale/cash count;
- подтверждение destructive operations;
- перепечатка, если требуется выбор причины/копии;
- будущий split bill;
- будущий перенос заказа на другой стол.

Правила:

- modal не должен менять текущий раздел;
- закрытие modal возвращает пользователя к тому же заказу/столу;
- financial modal всегда показывает сумму, метод, статус смены/кассы и безопасную ошибку;
- manager override требует отдельного ввода manager id/PIN и reason;
- raw backend errors, SQL errors, PIN, tokens, request dumps и payment-sensitive payloads не показывать;
- если backend-контракта нет, действие в modal не выглядит как рабочая активная кнопка: только `запланировано далее`, disabled design или backlog note.

## Визуальные правила

### Square-like заказ

- Экран заказа должен выглядеть как рабочий POS, а не как админка.
- Плитки меню являются главным визуальным объектом.
- Правая панель заказа спокойная, читаемая и не перегруженная техническими деталями.
- UUID, node id, внутренние идентификаторы и debug-информация не показываются в основном runtime UI.
- Основные действия всегда находятся внизу правого rail.
- В editable order это две крупные кнопки: `Действия` и `Пречек`.
- В locked order после пречека это `Касса` и `Отмена пречека`.
- Редкие действия открываются через modal `Действия`.
- Locked state после пречека визуально очевиден: read-only строки, lock/banner/status и disabled add-line hints.
- Не использовать много мелких border-рамок; вместо этого применять surfaces/cards, отступы, typography hierarchy и status colors.

### Плотность и touch

- Минимальный touch target: 48x48 px.
- Основные POS-кнопки: 52-64 px высотой.
- Основные плитки меню: примерно 96-128 px высотой на tablet/desktop, 80-104 px на compact screens.
- Важные actions визуально отделены от secondary actions.
- Числа, суммы, short IDs: tabular/monospace.
- Карточки использовать только для повторяемых объектов: столы, блюда, заказы, платежи.
- На compact screens layout адаптируется в drawer/bottom sheet, но сохраняет тот же mental model: объект/меню слева по приоритету, действия в rail/sheet.

### Цвет и состояние

Цвет кодирует статус, а не украшение:

- нейтральный фон рабочей области;
- один основной accent для primary action;
- отдельные status colors: success, warning, danger, info;
- disabled должен объяснять причину через tooltip/hint, если причина не очевидна;
- sync problems должны быть заметны, но не мешать order entry;
- financial/destructive actions визуально отличаются от обычных, но не доминируют над экраном ввода заказа.

### Типографика

- UI должен быть рабочим, не маркетинговым.
- Заголовки внутри POS-панелей компактные.
- Hero-scale typography в runtime POS не использовать.
- Русский пользовательский текст только через `vue-i18n`.

### Иконки

- Использовать иконки Quasar/Material или выбранный единый набор проекта.
- Кнопки инструментов должны иметь иконку; длинный label только для primary/financial/destructive actions.
- У незнакомых icon-only actions обязателен tooltip/aria-label.

## Состояния

Для каждого раздела обязательны:

- loading skeleton, соответствующий реальной сетке/списку;
- empty state с конкретным следующим действием;
- safe error state через normalized `message_key`;
- no-permission state: действие скрыто или disabled по UX, backend остается авторитетным;
- offline/sync warning, когда операция зависит от локального outbox/sync health;
- locked/read-only state для заказа после precheck.

Для экрана выбранного заказа обязательны отдельные состояния:

- `No selected order`;
- `Editable order`;
- `Precheck issuing`;
- `Precheck issued / locked order`;
- `Payment modal open`;
- `Closed order`;
- `Error / no permission / offline`.

## RBAC

UI обязан использовать canonical permission ids из `pos-ui/src/shared/rbac.ts`, но не считать себя security boundary. В актуальном коде canonical ids включают `pos.employee_shift.*`, `pos.cash_session.*`, `pos.cash_drawer.record_event`, `pos.order.add_line`, `pos.order.change_quantity`, `pos.order.void_line`, `pos.sync.retry_failed` и другие ids из `permissionCatalog`. Если профильная документация перечисляет старые aliases вроде `pos.shift.open` или `pos.order.line.add`, перед реализацией UI нужно синхронизировать ее с кодом/backend.

Разделы скрываются или деградируют по permissions:

- `Залы и столы`: `pos.floor.view`, `pos.order.view`, `pos.order.create`.
- `Заказы`: `pos.menu.view`, `pos.catalog.view`, `pos.order.add_line`, `pos.order.change_quantity`, `pos.order.void_line`, `pos.precheck.issue`, `pos.precheck.cancel.request`, `pos.precheck.reprint`, `pos.payment.cash`, `pos.payment.card.manual`, `pos.order.close`.
- `Активность`: `pos.check.view`, `pos.payment.refund`, `pos.check.reprint`.
- `Отчеты`: зависит от будущих report permissions; до них только разрешенные operational summaries на основе уже доступных reads.
- `Касса`: `pos.employee_shift.open`, `pos.employee_shift.close`, `pos.employee_shift.view_current`, `pos.employee_shift.recent`, `pos.cash_session.open`, `pos.cash_session.close`, `pos.cash_session.view_current`, `pos.cash_drawer.record_event`, `pos.sync.view`, `pos.sync.retry_failed`.

Видимость и disabled-state кнопок `Действия`, `Пречек`, `Касса`, `Отмена пречека`, `Void`, `Refund`, `Reprint` и cash drawer operations зависит одновременно от permissions, состояния заказа, precheck/payment state, личной смены, кассовой смены и sync/offline условий. Backend остается авторитетным enforcement layer.

## Уникальный подход проекта

Наша цель не скопировать Square или iiko, а убрать трение:

- один bottom bar для всего POS, без конкурирующих боковых и верхних меню;
- каждый раздел использует одинаковый mental model: объект слева, действия справа;
- пользователь всегда видит текущий заказ или выбранный объект;
- частые операции доступны за 1-2 касания;
- рискованные операции уходят в modal с явным подтверждением и RBAC;
- будущие функции появляются как disabled/`запланировано далее` только в документах и backlog, не как активные кнопки без API;
- интерфейс проектируется от смены ресторана: быстро выбрать стол, добавить блюда, выпустить пречек, принять оплату, закрыть/вернуть/перепечатать, сверить кассу.
- primary workflow остается waiter-first: кассовые и менеджерские операции доступны по RBAC, но не диктуют основной маршрут.

## Решения для фиксации

Рекомендуемые решения по умолчанию:

- основной маршрут `/pos` становится новым shell с bottom bar и разделами;
- старый terminal view разбирается на section components, а не переписывается целиком;
- первым runtime-экраном редизайна становится `Заказы` как экран выбранного заказа: menu grid 3/4 + current order rail 1/4;
- default section после login выбирается smart: если столов больше одного - `Залы и столы`, если один стол - `Заказы`, в будущем - Cloud-owned restaurant/device setting;
- `Заказы` больше не являются общим списком заказов и не являются безусловным default section после входа;
- primary workflow проектируется как waiter-first restaurant workspace;
- `Активность`, `Касса`, `sync` больше не являются правым drawer внутри checkout panel, а становятся частью разделов или modal-процессов;
- `Отчеты` создается как limited operational section с честными статусами `реализовано сейчас` и `запланировано далее`;
- sidebar по умолчанию полностью скрыт и открывается только из bottom bar;
- action rail всегда справа на desktop/tablet и как bottom sheet/drawer на compact screens.

Нужно подтвердить перед реализацией:

- точный Cloud contract для restaurant/device UI settings;
- backend contract для transfer/merge/split;
- backend contract для KDS;
- backend/UI contract для advanced modifier editing и ручных discounts/surcharges;
- media contract для menu item images;
- report backend scope.

## Первый этап реализации

1. Создать POS shell:
   - постоянный bottom quick access bar;
   - скрываемое section menu;
   - section layout `3fr / 1fr`;
   - routing/state for active section;
   - context chips для table/order/precheck без превращения bottom bar в панель действий.
2. Реализовать первым экран выбранного заказа:
   - перенести menu grid в 3/4 рабочей области;
   - собрать current order rail в 1/4 справа;
   - оставить в rail две основные кнопки editable state: `Действия` и `Пречек`;
   - после issued precheck показывать locked state и кнопки `Касса` / `Отмена пречека`;
   - вынести payment flow в modal, не переводя пользователя в раздел `Касса`;
   - сохранить текущие backend capabilities: selected modifiers, quantity, void, precheck, cancel precheck manager override, cash/manual card payment, reprint.
3. Разложить текущие компоненты по будущим разделам:
   - `FloorTableSelector` в раздел `Залы и столы`;
   - menu grid из `CatalogCheckoutPanel` и active order из `OrderWorkspace` в раздел `Заказы`;
   - `ClosedOrdersDrawer` в раздел `Активность`;
   - shift/cash/sync panels из `CatalogCheckoutPanel` и `SyncDrawer` в раздел `Касса`;
   - limited reports section как read-only operational dashboard.
4. Обновить визуальную систему:
   - tokens for surfaces, status colors, bottom nav height, action rail width;
   - consistent skeleton/empty/error/no-permission/locked states;
   - крупные table/order/menu cards без технического шума.
5. Перевести drawer-only flows в modals/section surfaces:
   - payment;
   - order actions;
   - refund;
   - cash drawer event;
   - precheck cancel;
   - sync diagnostics.
6. Обновить документацию и e2e:
   - `docs/ui/POS-UI-SPEC.md`;
   - `docs/ui/POS-UI-RBAC.md`;
   - Playwright сценарий: login -> shift/cash -> table -> order -> precheck -> payment modal -> activity -> refund/reprint -> cash.

## Источники

- Square Support: item grid, favorites and shortcuts: https://square.site/help/us/en/article/8334-set-up-item-grid
- Toast Support: ordering screens, check details, quick order/table order: https://support.toasttab.com/en/article/New-POS-Experience-Ordering-Screens
- Toast Support: table service screen and table details pane: https://support.toasttab.com/en/article/New-POS-Managing-Tables
- Lightspeed Support: Restaurant POS navigation tabs: https://resto-support.lightspeedhq.com/hc/en-us/articles/360005777873-About-navigation-in-Restaurant-POS
- Syrve Front of House: floor plan, POS, payments, delivery/takeaway, prompts: https://www.syrve.com/en-gb/front-of-house-software
- iiko table service: залы, обслуживание гостей, разделение чека, курсы подачи: https://iiko.ru/solutions/iikorms-tableservice
- Poster Help Center: service modes, order/payment/split/reprint flows: https://help.joinposter.com/en/collections/2995013-order
- Локальные референсы: `docs/ui/square-point-of-sale.png`, `docs/ui/iiko-pin.jpg`, `docs/ui/iiko-5-0.webp`.

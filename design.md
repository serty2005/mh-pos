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

- строка заказа: future split, move to guest, void, note, modifier edit;
- гость: future split/check by guest, rename, move items;
- стол: future transfer, merge, reservation, mark status;
- closed order/payment: refund, reprint, details.

Каждое long-press действие должно иметь альтернативный доступ через action rail или overflow.

### Drag and drop

Drag & drop поддерживается как ускоритель ресторанных операций, но не как единственный способ выполнить действие.

Первичные сценарии:

- перенос заказа со стола на стол;
- future перенос позиций между гостями;
- future перенос позиций между заказами;
- future редактирование floor plan;
- future сортировка favorites/menu shortcuts.

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
- Каталог и меню: чтение catalog/menu items.
- Заказы: создать заказ, получить текущий заказ по столу, получить заказ, получить закрытые заказы, добавить позицию, изменить количество, удалить позицию через void, закрыть заказ.
- Пречек: выпустить, получить, получить список по заказу, отменить через manager override, перепечатать копию.
- Оплата и чек: принять payment по precheck, получить check, перепечатать check, вернуть captured payment.
- Sync/операционная диагностика: status, outbox, local events, retry failed outbox.
- RBAC: backend permissions являются авторитетными, UI visibility остается UX-слоем.

### Ограничения текущего runtime

Эти области нельзя показывать как готовую функциональность без отдельного backend/API изменения:

- модификаторы в order line runtime;
- скидки, надбавки, tax engine и ручные price overrides;
- inventory consumption, recipe expansion и списания;
- KDS lifecycle;
- delivery/channel runtime;
- real PSP/payment terminal integration;
- fiscal adapter;
- полноценные отчеты beyond available closed orders, payments/checks и sync/cash-session data;
- перенос/объединение столов, split bill, split by guest, courses, kitchen send/hold/stay.

### UI gap относительно backend

Текущий `pos-ui` уже использует большую часть runtime surface, но организован как один терминал с верхним status bar, левым выбором столов, центральным заказом и правым catalog/checkout/actions panel. Для новой навигации нужно не добавлять много экранов поверх старого terminal view, а разложить уже реализованные операции по постоянным разделам:

- `залы и столы`: halls/tables/current order by table/create order.
- `заказы`: menu grid + active order + precheck/payment actions.
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

Нижняя строка является постоянной на всех POS-экранах:

1. Слева: кнопка текущего раздела. Нажатие открывает/закрывает боковое меню разделов.
2. Центр: context chips и быстрые действия текущего раздела, максимум 3-5 штук.
3. Справа: lock, sync badge, actor/session badge или compact status.

Разделы:

- `залы и столы`;
- `заказы`;
- `активность`;
- `отчеты`;
- `касса`.

Правила:

- active section должен быть виден без чтения длинного текста: иконка + короткий label;
- нижняя строка не должна перекрывать важные кнопки оплаты/пречека;
- каждый раздел определяет свои quick actions;
- в центре bottom bar могут отображаться context chips: выбранный стол, активный заказ, гость или quick-service order tabs;
- context chips не заменяют основной экран, а ускоряют переключение рабочего контекста;
- bottom bar не должен превращаться в перегруженную панель действий;
- запрещено размещать в bottom bar действия, которые требуют длинного ввода или подтверждения. Такие действия открывают modal/side dialog.

### Скрываемое боковое меню

Боковое меню служит навигацией между разделами и дополнительным контекстом:

- полностью скрыто по умолчанию после выбора раздела;
- открывается по кнопке текущего раздела в bottom bar;
- показывает все разделы, состояние смены, кассы, синхронизации и роль сотрудника;
- закрывается по выбору раздела, клику вне меню или `Esc`;
- на tablet/desktop занимает 280-320 px overlay;
- на mobile/full handheld открывается как full-height sheet.

### Единый layout раздела

Каждый раздел использует одну схему:

- основная рабочая область слева: примерно 3/4 ширины;
- action rail справа: примерно 1/4 ширины;
- на малых экранах action rail становится bottom sheet или drawer, но сохраняет те же приоритеты.

CSS-правило для desktop/tablet:

```css
.pos-section-layout {
  display: grid;
  grid-template-columns: minmax(0, 3fr) minmax(320px, 1fr);
  min-height: calc(100dvh - var(--app-header-height) - var(--bottom-nav-height));
}
```

UX-правило:

- слева пользователь выбирает объект работы;
- справа пользователь видит текущий выбранный объект и доступные операции;
- модалки используются для процессов, а не для постоянной навигации.

## Разделы

### Залы и столы

Цель: быстро понять занятость ресторана и открыть нужный заказ.

Этот раздел является default section для table-service ресторана. Стол является визуальным входом в заказ: пользователь не создает заказ "из воздуха", если ресторан работает в table-service режиме. Он сначала выбирает стол, затем открывает существующий активный заказ или создает новый заказ в контексте выбранного стола.

Основная область:

- tabs/chips залов сверху;
- сетка или floor-plan столов;
- карточка стола показывает минимум: название, мест, текущий статус, наличие активного заказа;
- когда backend даст данные, добавить: сумма заказа, время открытия, сотрудник, количество гостей.

Action rail:

- выбранный стол;
- текущий заказ на столе;
- создать заказ;
- перейти к заказу;
- обновить данные;
- будущие действия: перенос стола, объединение, бронь, посадка гостей.

Реализуемо сейчас:

- halls/tables, выбор стола, current order, create order.

Запланировано далее:

- визуальный drag floor plan, статусы занятости beyond active order, server assignment, reservations, transfer/merge.

### Заказы

Цель: самый быстрый экран для ввода блюд и контроля текущего заказа.

Если заказ открыт со стола, экран заказов всегда сохраняет table context. Возврат назад ведет не в общий список заказов, а к тому же столу/залу. В single-table режиме экран заказов может быть стартовым экраном. Single-table режим поддерживает переключение между активными заказами через chips/tabs, но только когда backend/settings это разрешают.

Основная область:

- слева внутри 3/4 области: категории/избранное/поиск;
- основная сетка плиток меню;
- плитки должны поддерживать картинку блюда, initials fallback, цену, availability/status;
- sticky category rail допустим сверху или слева внутри основной области;
- search не должен вытеснять сетку на маленьком экране.

Action rail:

- текущий открытый заказ;
- строки заказа с quantity stepper и void;
- итоговые суммы от backend;
- пречек;
- оплата;
- reprint;
- состояние locked/closed.

Реализуемо сейчас:

- menu items, add line, quantity, void, issue precheck, payment cash/card, reprint precheck/check, close order.

Запланировано далее:

- modifiers modal, discount modal, course/send/hold, guest/seat allocation, split bill, comments to kitchen.

### Активность

Цель: быстро найти закрытые заказы, платежи и выполнить разрешенные постоперации.

Основная область:

- список закрытых заказов;
- поиск и фильтры: дата, стол, сумма, payment status, refund status;
- компактная timeline-card заказа: стол, сумма, check status, payments, business date.

Action rail:

- детали выбранного closed order;
- платежи и статусы;
- reprint check;
- refund captured payment, если есть permission и открыта cash session;
- sync/audit hint по операции.

Реализуемо сейчас:

- closed orders, check payments, refund captured payment, reprint check.

Запланировано далее:

- расширенный поиск по receipt/customer/card last digits, edit payment method, split refunds, audit trail view.

### Отчеты

Цель: дать сотруднику и менеджеру быстрые операционные итоги без обещания полноценной аналитики.

Основная область:

- текущая личная смена;
- текущая кассовая смена;
- закрытые заказы за локальный business date, если можно собрать из доступного endpoint;
- суммы оплат по методам только если данные надежно доступны из закрытых заказов/check snapshots.

Action rail:

- печать/экспорт отчета, когда появится backend contract;
- переход к кассовой смене;
- sync health;
- предупреждение о том, что расширенные отчеты пока `запланировано далее`.

Реализуемо сейчас:

- только легкие operational summaries на основе current/recent shift, current cash session, closed orders и sync status.

Запланировано далее:

- Z/X reports, revenue by category, staff sales, tax reports, inventory/cost reports, Cloud analytics.

### Касса

Цель: все сменные и кассовые операции в одном месте.

Основная область:

- личная смена сотрудника;
- кассовая смена устройства;
- кассовый ящик;
- последние cash drawer events, если backend добавит list endpoint;
- sync health и local diagnostics.

Action rail:

- открыть/закрыть личную смену;
- открыть/закрыть кассовую смену;
- cash in/cash out/no sale/cash count modal;
- sync retry для ролей с правом;
- lock terminal/logout.

Реализуемо сейчас:

- open/close shift, current/recent shifts, open/close cash session, record cash drawer event, sync status/outbox/local events/retry.

Запланировано далее:

- list cash drawer events, close day, reconciliation report, fiscal close, cash discrepancy workflow.

## Модалки и процессы

Модалки используются для изолированных операций, где нужно удержать контекст заказа и выполнить подтверждение:

- оплата;
- manager override;
- отмена пречека;
- возврат оплаты;
- кассовый ящик;
- модификаторы блюда;
- скидка/надбавка;
- split bill;
- печать/перепечать;
- подтверждение destructive operations.

Правила:

- modal не должен менять текущий раздел;
- закрытие modal возвращает пользователя к тому же заказу/столу;
- financial modal должен показывать сумму, метод, статус смены и безопасную ошибку;
- manager override всегда требует отдельного ввода manager id/PIN и reason;
- raw backend errors, SQL errors, PIN, tokens и payment-sensitive payloads не показывать.

## Визуальные правила

### Плотность и touch

- Минимальный touch target: 48x48 px.
- Основные плитки меню: 96-128 px высотой на desktop/tablet, 80-104 px на compact screens.
- Важные actions: 52-56 px высотой.
- Числа, суммы, short IDs: tabular/monospace.
- Карточки использовать только для повторяемых объектов: столы, блюда, заказы, платежи.

### Цвет и состояние

Цвет кодирует статус, а не украшение:

- нейтральный фон рабочей области;
- один основной accent для primary action;
- отдельные status colors: success, warning, danger, info;
- disabled должен объяснять причину через tooltip/hint, если причина не очевидна;
- sync problems должны быть заметны, но не мешать order entry.

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
- locked order state после precheck.

## RBAC

UI обязан использовать canonical permission ids из `pos-ui/src/shared/rbac.ts`, но не считать себя security boundary.

Разделы скрываются или деградируют по permissions:

- `залы и столы`: `pos.floor.view`, `pos.order.view`, `pos.order.create`.
- `заказы`: `pos.menu.view`, `pos.order.add_line`, `pos.order.change_quantity`, `pos.order.void_line`, `pos.precheck.issue`, payment permissions.
- `активность`: `pos.check.view`, `pos.payment.refund`, `pos.check.reprint`.
- `отчеты`: зависит от будущих report permissions; до них только разрешенные operational summaries.
- `касса`: employee shift, cash session, cash drawer, sync permissions.

## Уникальный подход проекта

Наша цель не скопировать Square или iiko, а убрать трение:

- один bottom bar для всего POS, без конкурирующих боковых и верхних меню;
- каждый раздел использует одинаковый mental model: объект слева, действия справа;
- пользователь всегда видит текущий заказ или выбранный объект;
- частые операции доступны за 1-2 касания;
- рискованные операции уходят в modal с явным подтверждением и RBAC;
- будущие функции появляются как disabled/planned только в документах и backlog, не как активные кнопки без API;
- интерфейс проектируется от смены ресторана: быстро выбрать стол, добавить блюда, выпустить пречек, принять оплату, закрыть/вернуть/перепечатать, сверить кассу.
- primary workflow остается waiter-first: кассовые и менеджерские операции доступны по RBAC, но не диктуют основной маршрут.

## Решения для фиксации

Рекомендуемые решения по умолчанию:

- основной маршрут `/pos` становится новым shell с bottom bar и разделами;
- старый terminal view разбирается на section components, а не переписывается целиком;
- default section после login выбирается smart: если столов больше одного - `залы и столы`, если один стол - `заказы`, в будущем - Cloud-owned restaurant/device setting;
- `заказы` больше не являются безусловным default section после входа;
- primary workflow проектируется как waiter-first restaurant workspace;
- `активность`, `касса`, `sync` больше не являются правым drawer внутри checkout panel, а становятся частью разделов;
- `отчеты` создается как limited operational section с честными статусами `реализовано сейчас` и `запланировано далее`;
- sidebar по умолчанию скрыт;
- action rail всегда справа на desktop/tablet и как bottom sheet на compact screens.

Нужно подтвердить перед реализацией:

- точный Cloud contract для restaurant/device UI settings;
- backend contract для transfer/merge/split;
- backend contract для KDS;
- backend contract для modifiers/discounts;
- media contract для menu item images;
- report backend scope.

## Первый этап реализации

1. Создать POS shell:
   - bottom quick access bar;
   - collapsible section menu;
   - section layout `3fr / 1fr`;
   - routing/state for active section.
2. Перенести текущие компоненты:
   - `FloorTableSelector` в раздел `залы и столы`;
   - menu grid и order workspace в раздел `заказы`;
   - `ClosedOrdersDrawer` в раздел `активность`;
   - shift/cash/sync panels в раздел `касса`;
   - limited reports section как read-only operational dashboard.
3. Обновить визуальную систему:
   - tokens for surfaces, status colors, bottom nav height, action rail width;
   - consistent skeleton/empty/error states;
   - compact table/order/menu cards.
4. Перевести drawer-only flows в modals/section surfaces:
   - refund;
   - cash drawer event;
   - precheck cancel;
   - sync diagnostics.
5. Обновить документацию и e2e:
   - `docs/ui/POS-UI-SPEC.md`;
   - `docs/ui/POS-UI-RBAC.md`;
   - Playwright сценарий: login -> shift/cash -> table -> order -> precheck -> payment -> activity -> refund/reprint -> cash.

## Источники

- Square Support: item grid, favorites and shortcuts: https://square.site/help/us/en/article/8334-set-up-item-grid
- Toast Support: ordering screens, check details, quick order/table order: https://support.toasttab.com/en/article/New-POS-Experience-Ordering-Screens
- Toast Support: table service screen and table details pane: https://support.toasttab.com/en/article/New-POS-Managing-Tables
- Lightspeed Support: Restaurant POS navigation tabs: https://resto-support.lightspeedhq.com/hc/en-us/articles/360005777873-About-navigation-in-Restaurant-POS
- Syrve Front of House: floor plan, POS, payments, delivery/takeaway, prompts: https://www.syrve.com/en-gb/front-of-house-software
- iiko table service: залы, обслуживание гостей, разделение чека, курсы подачи: https://iiko.ru/solutions/iikorms-tableservice
- Poster Help Center: service modes, order/payment/split/reprint flows: https://help.joinposter.com/en/collections/2995013-order
- Локальные референсы: `docs/ui/square-point-of-sale.png`, `docs/ui/iiko-pin.jpg`, `docs/ui/iiko-5-0.webp`.

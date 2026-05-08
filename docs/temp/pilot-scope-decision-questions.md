# Вопросы для фиксации pilot scope decisions

## Назначение

Этот временный документ нужен для обсуждения с коллегами трех pre-pilot решений:

- `business_date_local`;
- reprint;
- waiter payment.

Цель обсуждения — зафиксировать, что входит в первый пилот, что явно остается `out of scope`, и какие backend-инварианты должны быть реализованы до pilot readiness.

После принятия решений этот документ нужно либо удалить, либо перенести итоговые решения в профильные документы:

- `ROADMAP.md`;
- `docs/backend/POS-BACKEND-SPEC.md`;
- `docs/backend/POS-DATA-AND-MIGRATIONS.md`;
- `docs/ui/POS-UI-SPEC.md`;
- `docs/ui/POS-UI-RBAC.md`.

---

## 1. `business_date_local`

Главный вопрос: пилот живет по календарной дате терминала или по ресторанному бизнес-дню?

Рекомендуемая позиция для обсуждения: для пилота ввести `business_date_local` как обязательный backend-owned инвариант для финансовых и сменных сущностей. Значение вычисляется backend-ом по timezone ресторана и правилу начала бизнес-дня, сохраняется неизменяемо на записи.

### Вопросы

1. Как ресторан закрывает операционный день: по полуночи или по сменной границе, например 04:00/05:00?
2. Может ли заведение работать после полуночи, когда продажи 00:30 должны попасть во вчерашний бизнес-день?
3. Нужна ли настройка `business_day_start_time` на уровне ресторана/филиала, или для пилота фиксируем одно значение?
4. Какие сущности обязаны хранить `business_date_local`: `orders`, `prechecks`, `checks`, `payments`, `shifts`, `cash_sessions`, `cash_drawer_events`, `local_event_log`, `outbox`?
5. Должна ли `business_date_local` быть immutable после создания записи?
6. Что делать, если заказ открыт до полуночи, а оплачен после полуночи: дата заказа, дата оплаты и дата чека совпадают или могут отличаться?
7. Что важнее для отчетов: дата открытия заказа, дата оплаты, дата финального чека или дата закрытия кассовой смены?
8. Может ли manager вручную перекинуть операцию на другой бизнес-день, или это строго запрещено?
9. Как обрабатываем смену timezone ресторана: новые операции идут по новой timezone, старые записи не пересчитываются?
10. Нужен ли отдельный объект `business_day` / `day_close`, или для пилота достаточно поля `business_date_local` на операционных записях?

### Варианты решения

- Вариант A: не вводить `business_date_local`, использовать календарную дату. Быстро, но рискованно для ночных ресторанов.
- Вариант B: ввести `business_date_local` без отдельного day-close flow. Это минимальный безопасный pilot scope.
- Вариант C: реализовать полноценный business day lifecycle с открытием/закрытием дня. Надежнее, но может раздуть первый пилот.

### Предварительная рекомендация

Вариант B.

---

## 2. Reprint

Главный вопрос: разрешаем ли в пилоте повторную печать пречека/финального чека, и если да, из какого источника истины?

Рекомендуемая позиция для обсуждения: либо явно оставить reprint `out of scope`, либо реализовать только controlled reprint из immutable snapshot с permission и audit. Не делать reprint из текущего состояния заказа.

### Вопросы

1. Reprint нужен для пречека, финального чека или обоих документов?
2. Это операционная необходимость пилота или nice-to-have?
3. Какие реальные сценарии нужно покрыть: принтер не напечатал, гость потерял чек, кассир ошибся, кухня/официант просит копию?
4. Reprint должен печатать точную копию оригинала или документ с пометкой "копия"?
5. Нужен ли manager permission для reprint финального чека?
6. Нужно ли указывать reason при reprint?
7. Нужно ли audit-событие: кто, когда, какой документ, причина, terminal/client_device_id?
8. Что является источником печати: immutable print snapshot, final check payload, precheck snapshot или текущий order state?
9. Если оригинальная печать failed, это reprint или retry print?
10. Нужна ли отдельная политика для fiscal/legal чеков, если позже появится фискальный регистратор?
11. Должен ли reprint попадать в sync/outbox как отдельное событие?
12. Что делать, если заказ уже закрыт, смена закрыта, а reprint нужен?

### Варианты решения

- Вариант A: reprint полностью `out of scope` для пилота.
- Вариант B: только precheck reprint, без final check reprint.
- Вариант C: precheck/final check reprint из immutable snapshot, с RBAC и audit.
- Вариант D: реализовать print retry failure flow, но не reprint по запросу оператора.

### Предварительная рекомендация

Если пилот без реального принтера/фискальника — вариант A. Если печать реально проверяется в пилоте — вариант C минимальным scope.

---

## 3. Waiter payment

Главный вопрос: может ли официант принимать оплату, или все оплаты в пилоте проходят через кассира/manager на POS terminal?

Рекомендуемая позиция для обсуждения: для cashier-first pilot оставить waiter payment `out of scope`, если нет явного требования пилотного ресторана. Backend должен не давать официанту оплату без permission.

### Вопросы

1. В пилотном заведении кто фактически принимает деньги: кассир, официант, manager?
2. Есть ли сценарий оплаты за столом официантом, или гость идет к кассе?
3. Если официант принимает оплату, это cash, card manual, terminal card, QR, room charge или other?
4. У официанта есть собственная кассовая смена/cash session или он работает через общую кассу?
5. Кто несет ответственность за cash drawer discrepancy: официант или кассир?
6. Может ли официант принимать cash без доступа к cash drawer?
7. Нужен ли manager override для waiter payment?
8. Можно ли официанту принимать частичную оплату?
9. Можно ли официанту закрывать заказ после полной оплаты?
10. Какие роли получают permissions: `waiter`, `senior_cashier`, `manager`?
11. Нужно ли разделять permission `pos.payment.cash` для кассира и отдельный `pos.payment.waiter_cash`?
12. Нужно ли отображать waiter payment в UI уже сейчас, или достаточно backend запрета/разрешения?
13. Как waiter payment влияет на отчеты по сменам, accountability и sync events?

### Варианты решения

- Вариант A: waiter payment `out of scope`, оплаты только cashier/senior_cashier/manager.
- Вариант B: waiter может инициировать payment request, но backend payment capture делает кассир.
- Вариант C: waiter может принимать только card/manual non-cash.
- Вариант D: waiter full payment flow с отдельной ответственностью, permissions, UI и audit.

### Предварительная рекомендация

Вариант A для первого пилота, если бизнес явно не требует оплату официантом.

---

## Decision template

```md
## Pilot Scope Decision

### business_date_local

Decision: A/B/C
Pilot rule:
Backend invariant:
Out of scope:
Reason:
Owner:
Deadline:

### reprint

Decision: A/B/C/D
Allowed documents:
Required permissions:
Audit requirements:
Snapshot/source of truth:
Out of scope:
Owner:
Deadline:

### waiter payment

Decision: A/B/C/D
Allowed roles:
Allowed methods:
Cash session responsibility:
UI scope:
Out of scope:
Owner:
Deadline:
```

---

## Базовый предлагаемый пакет решений

- `business_date_local`: вариант B.
- reprint: вариант A, если печать не входит в пилот; вариант C, если печать реально проверяется.
- waiter payment: вариант A.


### business_date_local

Decision: Hybrid logic (Standard & 24/7)
Pilot rule: 
- Введены два режима: "Стандартный" (граница смены в заданное время) и "24/7" (смена может длиться несколько дней).
- Время закрытия дня (например, `05:00` по умолчанию) настраивается строго на уровне ресторана, а не глобально на сервере.
- Для отчетов используются поля `business_date_local` (Учетный день) и `closed_at` (Время закрытия).
- Финансовая принадлежность определяется только моментом закрытия заказа (созданием чека) и платежом. Время создания заказа не влияет на учетный день.
- Перенос открытого заказа в новую смену разрешен.
- Списание продуктов со склада происходит в момент "отбития" с кухни / подачи, а не в момент оплаты.
Backend invariant: 
- Бэкенд автоматически вычисляет `business_date_local` для чека и платежа на основе конфигурации ресторана (Стандарт или 24/7) в момент их создания.
- После закрытия заказа поле `business_date_local` становится строго неизменяемым (immutable).
Out of scope: 
- Ручной перенос *закрытых* заказов или платежей в другую смену/бизнес-день. 
- Настройка времени закрытия дня глобально на весь инстанс сервера.
Reason: Заведениям 24/7 требуется привязка чеков к реальному дню закрытия без искусственных границ, тогда как стандартным заведениям нужна защита от попадания ночных чеков в следующий календарный день. 
Owner: Backend Team / Code Agent
Deadline: Pilot Readiness

### reprint

Decision: Controlled Reprint from Immutable Snapshot
Allowed documents: Precheck, Final Check.
Required permissions: 
- `pos.precheck.reprint` (доступно: Waiter, Cashier, Manager).
- `pos.check.reprint` (доступно строго: Manager).
Audit requirements: 
- Обязательная запись в `local_event_log` (события `CheckReprinted` / `PrecheckReprinted`) с фиксацией `actor_employee_id`.
- Напечатанный документ обязан содержать явный маркер "КОПИЯ" (COPY).
Snapshot/source of truth: Строго из сохраненных неизменяемых JSON-пейлоадов `check.snapshot` / `precheck.snapshot`. Использование текущего состояния заказа для репринта запрещено.
Out of scope: 
- Указание причины (reason) при репринте.
- Отдельные политики печати для фискальных регистраторов (будет реализовано с появлением модуля ФР).
Owner: Fullstack / Code Agent
Deadline: Pilot Readiness

### waiter payment

Decision: Cashier-first flow (Waiter payment is Out of Scope / Post-MVP)
Allowed roles: Кассирские операции доступны только ролям `cashier`, `senior_cashier`, `manager`.
Allowed methods: Оплата проводится только на кассе (Terminal/Cashier).
Cash session responsibility: Вся ответственность за наличные деньги и расхождения лежит на владельце общей кассовой смены (Кассире). Личные кассы официантов не создаются.
UI scope: Полное скрытие или блокировка кнопок/экранов оплаты во Vue-клиенте, если у авторизованного пользователя (`actor`) нет пермишенов `pos.payment.*`.
Out of scope: 
- Оплата официантом за столом (Server Banking).
- Создание личных кассовых смен для официантов.
- Разделение прав на `pos.payment.cash` и `pos.payment.waiter_cash`.
Owner: Fullstack / Code Agent
Deadline: Post-MVP
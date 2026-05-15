# Cloud UI Spec

## Назначение

реализовано сейчас: `cloud-ui` является отдельным пилотным интерфейсом для `cloud-backend` и Cloud-owned операционных сценариев.

`cloud-ui` не является частью `pos-ui`, не использует POS session, POS Edge runtime endpoints, cashier routes или локальные POS stores.

## Целевой сценарный план

реализовано сейчас:

1. Подключение POS Edge к Cloud через `cloud-backend` provisioning routes.
2. Проверка минимальной готовности ресторана, зала, ролей и сотрудников.
3. Подготовка продаваемого меню поверх существующих Cloud-owned master data.
4. Явная публикация master data package для Edge.
5. Передача опубликованного snapshot на Edge, где далее формируются заказ и продажа.

запланировано далее:

- заменить текущие справочные таблицы сценарными мастерами запуска ресторана;
- добавить readiness-панель по ресторану: устройство назначено, публикация создана, меню имеет продаваемые позиции, сотрудники и роли готовы;
- вывести связи `catalog item -> menu item -> modifier bindings -> pricing policies` как единый сценарий подготовки продажи;
- показывать версии опубликованного пакета и состояние доставки на Edge, когда backend подтвердит такой контракт.

вне текущего объема:

- создание справочников как отдельная бизнес-цель;
- cashier runtime, order/payment/check/precheck flows в Cloud UI;
- Cloud auth/RBAC UI до появления подтвержденного backend-контракта.

## Границы

реализовано сейчас:

- план запуска Cloud UI от подключения Edge-device до продажи на Edge-стороне;
- панель готовности онбординга по выбранному ресторану: ресторан, роли/сотрудники, зал/столы, меню, Edge-device и публикация;
- список незакрепленных Edge-device из `/api/v1/devices/unassigned`;
- назначение Edge-device ресторану через `/api/v1/restaurants/{restaurant_id}/devices/{node_device_id}/assign`;
- проверка assignment status через `/api/v1/devices/{node_device_id}/assignment-status`;
- генерация pairing code через `/api/v1/restaurants/{restaurant_id}/devices/generate-pairing-code`;
- управление ресторанами;
- роли и сотрудники; роли создаются через предустановленные профили и матрицу POS-прав, а не через ручной ввод `permissions_json`;
- catalog items;
- catalog folders;
- folder parameters;
- catalog tags;
- item tags как command-only привязка;
- modifier groups, options и bindings;
- pricing policies;
- halls и tables;
- menu items;
- menu category create как command-only операция, потому что list/update routes не подтверждены;
- publication summary и явная публикация master data.

вне текущего объема:

- KDS;
- PSP;
- fiscalization;
- inventory runtime;
- recipe consumption;
- delivery;
- cashier runtime;
- POS order/payment/check/precheck flows.

## UX

реализовано сейчас: интерфейс перестроен из чистой admin surface в операционный центр с двумя слоями:

- сценарный слой запуска: план внедрения и подключение Edge-device;
- технический слой master data: существующие таблицы и формы для подтвержденных backend routes.

Правила UI:

- первое действие оператора — открыть план запуска или подключить Edge-device;
- выбор ресторана остается обязательным для restaurant-scoped операций;
- технические связи между сущностями выбираются из загруженных справочников; пользователь не вводит ID вручную в подтвержденных связях;
- pairing code flow не требует ввода `node_device_id`: Cloud генерирует device id на backend-стороне;
- publication flow позволяет выбрать известное Edge-устройство из UI-состояния или опубликовать общий пакет без ручного ввода ID;
- роли выбираются из профилей `cashier`, `senior_cashier`, `waiter`, `manager`, `kitchen`, `support_admin`, после чего оператор может изменить права в матрице;
- Edge-device flow не показывает секреты кроме одноразового pairing code, который возвращает backend;
- command-only разделы не показывают неподтвержденную таблицу;
- Cloud UI показывает безопасные локализованные ошибки: message key, support code, correlation id и безопасные details, если backend их вернул;
- пользовательские тексты идут через `vue-i18n`.

## API

реализовано сейчас: API client `cloud-ui/src/shared/api.ts` использует подтвержденные routes из:

- `cloud-backend/internal/provisioning/api/router.go` для Edge-device provisioning;
- `cloud-backend/internal/masterdata/api/router.go` для master data и публикации.

Для entities без подтвержденного `GET list` route UI показывает форму команды и поясняет, что list route не подтвержден.

## Runtime Code

реализовано сейчас: runtime backend code не изменялся.

реализовано сейчас: для запуска Cloud UI из браузера `cloud-backend` разрешает local CORS origin `http://localhost:5174`, `http://127.0.0.1:5174` и `http://host.docker.internal:5174`.

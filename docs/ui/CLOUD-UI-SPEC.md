# Cloud UI Spec

## Назначение

реализовано сейчас: `cloud-ui` является отдельным пилотным интерфейсом для `cloud-backend` и Cloud-owned master data.

`cloud-ui` не является частью `pos-ui`, не использует POS session, POS Edge runtime endpoints, cashier routes или локальные POS stores.

## Границы

реализовано сейчас:

- управление ресторанами;
- роли и сотрудники;
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
- POS order/payment/check/precheck flows;
- Cloud auth/RBAC UI до появления подтвержденного backend-контракта.

## UX

реализовано сейчас: интерфейс построен как плотный admin surface:

- левое меню разделов;
- выбор ресторана для restaurant-scoped справочников;
- таблица текущего раздела;
- форма создания и редактирования;
- отдельные actions только для подтвержденных routes;
- command-only разделы не показывают неподтвержденную таблицу.

## API

реализовано сейчас: API client `cloud-ui/src/shared/api.ts` использует только routes из `cloud-backend/internal/masterdata/api/router.go`.

Для entities без подтвержденного `GET list` route UI показывает форму команды и поясняет, что list route не подтвержден.

## Runtime Code

реализовано сейчас: для запуска Cloud UI из браузера `cloud-backend` разрешает local CORS origin `http://localhost:5174`, `http://127.0.0.1:5174` и `http://host.docker.internal:5174`.

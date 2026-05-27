# Deprecated POS UI

`pos-ui/` объявлен deprecated.

Новая разработка POS UI в этой папке больше не ведется. Активная POS UI кодовая база находится в `pos-ui-g/`.

Допустимо использовать `pos-ui/` как источник для переноса подтвержденных runtime-условий, backend-authoritative правил, RBAC visibility, safe error/empty/loading/no-permission patterns, acceptance assumptions и проверенных flow идей.

Новые реализации должны выполняться в `pos-ui-g/` через модульные переиспользуемые компоненты: buttons, screens, notifications, modal/dialog shells, forms, cards, rows, layout panels, drawers и shared state views.

Изменения в `pos-ui/` допускаются только для compatibility fixes без добавления новой функциональности или новых UI patterns.

См. `docs/ui/POS-UI-G-MIGRATION.md`.

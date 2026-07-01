# Промпт реализации: единый раздел `Каталог и меню`

Использовать после согласования задачи на кодовые правки.

```text
Мы продолжаем разработку RMS-POS системы для ресторанов и выставочного alpha launch.

Нужно реализовать в `cloud-ui-g` единый Cloud backoffice раздел `Каталог и меню` вместо раздельного пользовательского опыта Catalog/Menu.

Перед правками прочитай:
- AGENTS.md;
- docs/ui/CLOUD-UI-SPEC.md;
- docs/ui/CLOUD-CATALOG-MENU-UX.md;
- docs/ui/CLOUD-UI-RMS-MANAGER-BACKEND-GAPS.md;
- docs/ui/CLOUD-UI-RMS-MANAGER-CLOUD-API-SPEC.md;
- docs/project-management/EXHIBITION-ALPHA-PILOT-REQUIREMENTS.md;
- docs/ui/myhoreca-rms-manager/src/components/MenuPanel.tsx только как UX reference;
- текущие `cloud-ui-g/src/features/catalog/**`;
- текущие `cloud-ui-g/src/features/menu/**`;
- `cloud-ui-g/src/shared/api/endpoints.ts`;
- `cloud-ui-g/src/shared/api/schemas.ts`;
- `cloud-ui-g/src/shared/i18n/ru.ts` и `cloud-ui-g/src/i18n/ru.ts`.

Сначала проверь `git status --short`. Не откатывай чужие изменения.

Целевое поведение:
- Navigation показывает один раздел `Каталог и меню`.
- Раздел доступен без выбранного ресторана и позволяет заполнять tenant catalog.
- Без ресторана доступны catalog folders/items/tags; menu-specific controls hidden/disabled.
- При выбранном ресторане загружаются catalog folders/items/tags и menu items выбранного ресторана.
- Режим `Только каталог` по умолчанию строит дерево по `catalog_folders`, показывает весь catalog и подсвечивает catalog items, выставленные на продажу в выбранном ресторане.
- Режим `Только меню` доступен только при выбранном ресторане, строит дерево по menu categories и показывает только menu items. Если list/edit/archive menu categories backend routes отсутствуют, реализуй минимальный blocked/readiness state и не подменяй его моками.
- Если текущие catalog routes требуют `restaurant_id`, не создавай fake restaurant для tenant catalog authoring. Зафиксируй backend gap или реализуй только тот подрежим, который подтвержден текущим контрактом.
- Все create/edit формы для folders/items/tags/menu items/menu categories открываются как modal dialogs.
- Добавь right-click context menu shell на folders/items/categories/menu items; первый slice содержит только существующие безопасные действия или disabled future actions.
- Для catalog item с menu item выбранного ресторана detail panel показывает menu overrides: name, price, category, tag, tax profile, availability, runtime status.
- Для catalog item без menu item выбранного ресторана detail panel показывает действие `Выставить на продажу`.

Ограничения:
- Не добавлять новый UI kit, tree library, state library или drag-and-drop dependency.
- Не копировать mockData/reference runtime logic.
- Не делать manual publish.
- Не делать destructive/support actions без backend RBAC/idempotency/audit.
- Все пользовательские строки через i18n.
- Не добавлять русские hardcoded UI strings в components.
- Не менять backend без отдельного явного требования. Если нужен backend route, обнови docs/gap и покажи blocked UI.

Проверки:
- `cd cloud-ui-g && npm run build`;
- если меняются forms/helpers, запусти существующие relevant tests или добавь минимальный тест рядом;
- по возможности Playwright/screenshot smoke desktop/mobile для нового раздела.

Финальный отчет на русском:
- что найдено;
- что изменено;
- измененные файлы;
- какие проверки запущены;
- какие проверки не удалось запустить;
- оставшиеся риски;
- что запланировано далее;
- что вне текущего объема;
- затрагивался ли runtime code;
- краткий и полный комментарии о выполненных работах.
```

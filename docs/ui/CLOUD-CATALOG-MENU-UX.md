# Cloud UI: раздел `Каталог и меню`

Статус: UX/design contract активного `cloud-ui-g` slice.

## Назначение

`Каталог и меню` является единой manager-facing точкой входа для верхнеуровневой номенклатуры tenant и restaurant menu overrides.

Цель экрана — убрать разрыв между отдельными CRUD-блоками `Catalog` и `Menu`: менеджер видит общий catalog tree, выбирает ресторан и сразу понимает, какие позиции выставлены на продажу в этом ресторане.

## Ownership

реализовано сейчас:

- tenant владеет catalog item identity;
- restaurant владеет menu item overrides;
- `menu item` ссылается на `catalog item`;
- изменение одного restaurant menu item не меняет catalog item и меню других ресторанов;
- Cloud -> Edge delivery автоматическая, manual publish action отсутствует.

## Доступность Без Ресторана

реализовано сейчас:

- Раздел `Каталог и меню` доступен до создания и выбора ресторана.
- Без выбранного ресторана экран работает как tenant catalog editor.
- Менеджер может заранее создать catalog folders, catalog items, services, tags и item tag assignments без fake restaurant.
- Restaurant-specific controls скрыты или disabled до выбора ресторана.
- Hint о выборе ресторана не блокирует catalog authoring.

## Режимы Вида

реализовано сейчас:

- `Только каталог` — режим по умолчанию. Иерархия строится по `catalog_folders`; отображается весь tenant catalog. Если ресторан выбран, catalog items с menu item выбранного ресторана подсвечиваются как `выставлено на продажу`.
- `Только меню` — доступен только при выбранном ресторане. Иерархия строится по restaurant `menu categories`; отображаются только menu items выбранного ресторана.

Не добавлять третий смешанный режим до отдельного требования.

## Дерево И Detail Panel

реализовано сейчас:

- Основной layout: compact toolbar сверху, дерево слева, detail panel справа.
- Дерево должно поддерживать folders/categories и item rows с устойчивой высотой, loading skeleton, empty state и error state.
- Выбранный узел показывает detail panel:
  - catalog folder: name, parent, sort order, status и действия;
  - catalog item: tenant fields, folder, kind, tags, status и restaurant sale state;
  - menu category: name, sort order, status и count items;
  - menu item: catalog identity плюс restaurant overrides.
- Для catalog item при выбранном ресторане detail panel показывает:
  - `не выставлено` и действие `Выставить на продажу`;
  - или `выставлено на продажу` и restaurant overrides: menu name, price, category, tag, tax profile, availability, runtime status.
- Хотелка: подсветка `выставлено на продажу` цветом выбранного ресторана после появления стабильного restaurant color/accent DTO. До этого использовать нейтральный semantic accent.

## Toolbar

реализовано сейчас:

- Toolbar содержит `+ Новая`, поиск, переключатель режима и refresh.
- `+ Новая` открывает menu с действиями, доступными в текущем контексте:
  - без ресторана: catalog folder, catalog item, tag;
  - с рестораном: catalog folder, catalog item, tag, menu category, menu item from catalog item.
- Secondary actions группируются в menus/icon buttons, чтобы экран не превращался в панель равновесных кнопок.

## Модалки

реализовано сейчас:

- Все create/edit формы открываются в modal dialog.
- Inline editing внутри дерева/списка не используется.
- Формы используют i18n, labels, ids/name/aria-label и безопасные error states.
- Raw JSON поля для availability не являются целевым UX. Первый slice может сохранить payload compatibility, но должен заменить пользовательский ввод на controls или явно оставить blocked state до отдельной формы.

## Правый Клик

реализовано сейчас:

- Right click перехватывается на catalog folders/items и menu categories/items.
- Первый slice может показывать context menu shell с существующими безопасными действиями и disabled future actions.
- Destructive/support actions запрещены без backend RBAC, idempotency key и audit contract.

## Menu Categories

`catalog folder` — папка общего tenant catalog.

`menu category` — папка ресторанного меню. Она принадлежит ресторану и группирует menu items, которые уже выставлены на продажу в этом ресторане.

Пример:

- catalog folder: `Билеты`;
- catalog item: `Входной билет`;
- restaurant A menu category: `Основные билеты`;
- restaurant B menu category: `VIP и пакеты`.

## Backend Gaps

реализовано сейчас:

- Catalog authoring без выбранного ресторана использует optional `restaurant_id` для catalog folders/items/tags; UI не создает fake restaurant.
- Режим `Только меню` использует `GET /master-data/menu/categories?restaurant_id=...`.
- Редактирование menu category использует `PATCH /master-data/menu/categories/{id}`.
- Lifecycle menu category использует `POST /master-data/menu/categories/{id}/archive`.

запланировано далее:

- Toolbar filters, tags filter, настройки вида и delivery status.
- Restaurant color/accent DTO для подсветки `выставлено на продажу`.

## Вне Текущего Объема

- Drag-and-drop сортировка.
- Массовое редактирование.
- Manual publish.
- Mock recipe/cost simulations из `docs/ui/myhoreca-rms-manager`.
- Frontend-authoritative financial, stock или delivery decisions.

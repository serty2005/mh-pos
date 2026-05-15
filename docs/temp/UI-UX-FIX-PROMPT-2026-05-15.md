# Отдельный промпт на исправление UI/UX после аудита 2026-05-15

Используй этот промпт для отдельной задачи/PR по UI/UX hardening. Не меняй backend contracts без явного подтверждения.

```text
Ты работаешь в репозитории /workspace/mh-pos. Ответы, документация и комментарии — на русском языке; identifiers — на английском. Соблюдай AGENTS.md. Пользовательские UI-строки добавляй только через vue-i18n. Runtime backend code не менять, если не обнаружен blocker, который невозможно решить на UI/mock уровне.

Задача: исправить UI/UX проблемы, найденные аудитом 2026-05-15, для обоих интерфейсов: POS Edge UI (`pos-ui`) и Cloud UI (`cloud-ui`).

Контекст аудита:
- `docs/temp/DOCUMENTATION-AUDIT-2026-05-15.md`;
- `docs/ui/POS-UI-SPEC.md`;
- `docs/ui/CLOUD-UI-SPEC.md`;
- `ROADMAP.md`.

Обязательные ограничения:
1. UI не является security boundary; backend RBAC и application checks остаются авторитетными.
2. Не добавлять hardcoded Russian UI strings вне locale files.
3. Не показывать raw Go/SQL/stack/request/PIN/token/secrets errors в UI.
4. Не реализовывать неподтвержденные business features: KDS, PSP, fiscal adapter, inventory consumption, Cloud auth/RBAC UI.
5. Не менять HTTP payloads/routes без профильного обновления docs/backend и docs/ui.

Цели POS Edge UI:
1. Снизить когнитивную плотность cashier terminal:
   - основной путь должен читаться как `готовность смены -> стол -> заказ -> пречек -> оплата`;
   - secondary operations (`sync`, `closed orders`, `cash drawer`, service diagnostics) оставить доступными, но визуально отделить от primary sale flow.
2. На tablet width checkout/precheck/payment не должны визуально отрываться от active order. Пересмотри breakpoint около 1100px и поведение `action-pane`.
3. Унифицировать blocking states (`noShift`, `noCashSession`, locked order, permission-disabled action): показать причину, следующее действие и роль/permission, если применимо.
4. Проверить wording для refund/cancel/reprint dialogs: безопасные формулировки, без raw technical details.
5. Сохранить текущие backend API calls and authoritative totals from backend.

Цели Cloud UI:
1. Разбить `cloud-ui/src/App.vue` на поддерживаемые компоненты без изменения runtime behavior:
   - shell/navigation;
   - launch readiness/checklist;
   - edge device assignment/pairing;
   - publication panel;
   - resource table/list;
   - resource form;
   - role permission matrix.
2. Сделать launch/onboarding flow primary journey:
   - clear readiness panel: restaurant selected, roles/employees ready, halls/tables ready, menu sellable, Edge assigned, publication created, snapshot available;
   - для каждого blocked item показывать next best action button.
3. Master-data CRUD оставить secondary/admin layer, не primary first-screen journey.
4. Для narrow screens добавить card/list fallback для ключевых launch/edge/publication states; не полагаться только на table `min-width: 720px`.
5. Ошибки API показывать контекстно возле failed step с safe recovery action (`retry`, `select restaurant`, `open section`) и без sensitive payload.

Проверки:
- `cd pos-ui && npm install && npm run build && npm run test`;
- `cd cloud-ui && npm install && npm run build`;
- если возможно установить Playwright browsers: добавить/запустить smoke сценарии для POS и Cloud;
- если Playwright browsers недоступны, явно указать limitation и приложить source-level/a11y checklist.

Definition of done:
- UI changes не ломают текущие flows and i18n.
- `docs/ui/POS-UI-SPEC.md`, `docs/ui/CLOUD-UI-SPEC.md` и `ROADMAP.md` обновлены, если изменился UX contract/status.
- Есть краткий отчет с before/after проблемами, проверками и оставшимися рисками.
```

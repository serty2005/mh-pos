# UI Icon System

## Назначение документа

Этот документ фиксирует единую систему иконок для MyHoreca POS UI и BackOffice UI.

Цель: исключить выбор иконок “по вкусу” отдельного разработчика или дизайнера и обеспечить единый визуальный язык во всех интерфейсах проекта.

Документ обязателен для:

- POS UI;
- BackOffice UI;
- дизайн-макетов;
- frontend-разработки;
- документации по UI;
- будущих Codex-итераций.

## 1. Базовая библиотека

Для MVP используется единая базовая библиотека иконок:

```text
Lucide Icons
```

Правила:

- использовать только Lucide Icons, если нет отдельного архитектурного решения;
- не смешивать Lucide с FontAwesome, Material Icons, Heroicons, Bootstrap Icons и другими наборами;
- не использовать цветные pictogram/icon packs в рабочих интерфейсах POS;
- не использовать разные стили иконок в одном модуле;
- иконка должна усиливать смысл действия или раздела, а не быть декоративным шумом.

## 2. Общий стиль иконок

Базовый стиль:

| Параметр | Значение |
|----------|----------|
| Style | outline |
| Stroke width | 1.75–2px |
| Fill | запрещен, кроме специально утвержденных случаев |
| Цвет | через design token / currentColor |
| Скругления | стандартные Lucide |
| Анимации | только для состояния loading/progress, если необходимо |

Иконки не должны быть самостоятельным источником цвета. Цвет задается состоянием компонента:

- normal;
- hover/active;
- selected;
- disabled;
- warning;
- danger;
- success.

Запрещено:

- делать иконки разноцветными без системной причины;
- использовать разные stroke width в соседних элементах;
- смешивать outline и filled icons;
- подбирать похожие, но не одинаковые иконки для одного и того же доменного смысла;
- использовать иконки как замену тексту там, где смысл может быть непонятен пользователю.

## 3. Размеры

### 3.1 Базовые размеры

| Размер | Назначение |
|--------|------------|
| 16 px | compact/backoffice secondary actions, inline labels |
| 20 px | стандартные backoffice controls |
| 24 px | крупные backoffice actions, toolbar buttons |
| 28 px | POS secondary actions |
| 32 px | POS primary actions, navigation, touch-first controls |

### 3.2 POS UI

POS UI проектируется в первую очередь под touch/touchpad.

Для POS:

- основная иконка кнопки: 28/32 px;
- иконка в нижней навигации: 28/32 px;
- иконка в плитке блюда/раздела: 28/32 px;
- иконка danger/confirmation action: 28/32 px;
- иконка внутри мелкого поясняющего текста допускается 20/24 px, но не как основной action target.

Минимальная зона нажатия остается больше самой иконки. Иконка 32 px не означает кнопку 32 px. Кнопка должна оставаться touch-friendly.

### 3.3 BackOffice UI

Для BackOffice:

- inline/table icons: 16 px;
- toolbar/filter buttons: 20 px;
- primary page actions: 24 px;
- крупные пустые состояния/empty states: 32 px по необходимости.

BackOffice допускает более компактные элементы, но не должен ломать общую визуальную систему проекта.

## 4. Цвета и состояния

Иконки должны наследовать цвет от компонента или design token.

Рекомендуемые роли:

| Состояние | Цветовая роль |
|-----------|---------------|
| Default | text-secondary / icon-default |
| Primary action | accent / primary |
| Selected | accent / selected |
| Disabled | text-muted / disabled |
| Success | success |
| Warning | warning |
| Danger | danger |
| Info | info |

Важно:

- не задавать цвет каждой иконке вручную;
- не создавать локальные one-off цвета;
- danger-иконка должна использоваться только для действительно разрушительных или рискованных действий;
- warning не должен заменять danger;
- success не должен использоваться просто как “красиво зеленое”.

## 5. Доменные иконки

### 5.1 Основные POS/BackOffice модули

| Модуль / раздел | Основная иконка | Альтернатива | Комментарий |
|-----------------|-----------------|--------------|-------------|
| Заказы | `ReceiptText` | `ShoppingCart` | `ReceiptText` предпочтительнее для ресторанного заказа/чека |
| Столы | `Armchair` | `LayoutGrid` | `Armchair` для ресторанного смысла, `LayoutGrid` для схемы/плитки |
| Гости | `Users` | - | Гости, количество гостей, клиентская группа |
| Оплата | `CreditCard` | - | Без привязки к конкретному провайдеру |
| Наличные | `Banknote` | - | Cash payment, cash drawer events |
| Смена | `Clock` | `CalendarClock` | `CalendarClock` для смен с business date |
| Касса | `Landmark` | `Wallet` | `Wallet` допустим для cash drawer, `Landmark` для кассового раздела |
| Кухня | `ChefHat` | - | Kitchen/KDS |
| Склад | `Warehouse` | - | Inventory/warehouse |
| Номенклатура | `Package` | - | Catalog/SKU/product master |
| Блюда | `Utensils` | - | Menu items/dishes |
| Модификаторы | `ListPlus` | `SlidersHorizontal` | `ListPlus` для добавления, `SlidersHorizontal` для настройки |
| Скидки | `BadgePercent` | - | Discounts/promotions |
| Лояльность | `Gift` | `HeartHandshake` | `Gift` для бонусов, `HeartHandshake` для loyalty relationship |
| Фискализация | `FileCheck` | - | Fiscal document/check validation |
| Настройки | `Settings` | - | System/settings |
| Организация | `Building2` | - | Legal/org structure |
| Ресторан / точка | `Store` | - | Location/outlet |
| Отчёты | `ChartNoAxesColumn` | `BarChart3` | Metrics/reports |
| Персонал | `UserCog` | - | Staff, roles, permissions |
| Доставка | `Truck` | - | Delivery/channel delivery |
| Резервы | `CalendarCheck` | - | Reservations/bookings |

### 5.2 Операционные действия

| Действие | Иконка | Комментарий |
|----------|--------|-------------|
| Добавить | `Plus` | Универсальное добавление |
| Создать заказ | `ReceiptText` + `Plus` | Можно использовать рядом текстом, не обязательно комбинировать иконки |
| Удалить / void | `Trash2` | Только для удаления/void, не для cancel business operation |
| Отмена | `X` | Закрыть окно или отменить текущую операцию |
| Отмена операции | `Ban` | Бизнес-отмена/cancellation |
| Возврат | `Undo2` | Refund/return |
| Подтвердить | `Check` | Confirmation |
| Сохранить | `Save` | BackOffice forms |
| Печать | `Printer` | Print/reprint |
| Повторить | `RefreshCw` | Retry/reload/sync retry |
| Поиск | `Search` | Search input |
| Фильтр | `Filter` | Filters |
| Больше действий | `MoreHorizontal` | Secondary menu/actions |
| Назад | `ArrowLeft` | Navigation back |
| Вперед | `ArrowRight` | Navigation forward |
| Вверх | `ChevronUp` | Vertical scroll/navigation |
| Вниз | `ChevronDown` | Vertical scroll/navigation |
| Влево | `ChevronLeft` | Horizontal scroll/navigation |
| Вправо | `ChevronRight` | Horizontal scroll/navigation |
| Заблокировано | `Lock` | Locked order/precheck state |
| Разблокировано | `Unlock` | Unlock/override where allowed |
| Предупреждение | `TriangleAlert` | Warning |
| Ошибка | `CircleAlert` | Error |
| Информация | `Info` | Info/help |
| Успешно | `CircleCheck` | Success |

## 6. Примеры использования для POS

### 6.1 Bottom navigation

POS bottom navigation использует крупные иконки 28/32 px и короткие подписи.

Пример:

| Раздел | Иконка | Размер |
|--------|--------|--------|
| Залы/столы | `Armchair` | 32 px |
| Заказы | `ReceiptText` | 32 px |
| Активность | `Clock` / `ReceiptText` | 28/32 px |
| Отчёты | `ChartNoAxesColumn` | 28/32 px |
| Касса | `Wallet` / `Landmark` | 32 px |

Правила:

- иконка + короткий текст лучше, чем одна иконка без подписи;
- активный раздел должен иметь selected-состояние;
- disabled-разделы не должны выглядеть как доступные;
- не использовать длинные подписи, которые ломают navigation.

### 6.2 Action buttons

Для основных POS-действий:

| Action | Иконка | Комментарий |
|--------|--------|-------------|
| Пречек | `Printer` / `FileCheck` | Выпуск/печать precheck |
| Касса/оплата | `CreditCard` | Переход к оплате или payment modal |
| Наличные | `Banknote` | Cash payment method |
| Карта | `CreditCard` | Manual card/payment method |
| Отмена пречека | `Ban` / `X` | Prefer `Ban` для бизнес-смысла |
| Возврат | `Undo2` | Refund |
| Manager override | `ShieldCheck` | Подтверждение менеджером |

### 6.3 Scroll controls

Для кастомных touch-first scroll controls:

| Направление | Иконка |
|-------------|--------|
| Вверх | `ChevronUp` |
| Вниз | `ChevronDown` |
| Влево | `ChevronLeft` |
| Вправо | `ChevronRight` |

Scroll buttons должны быть крупными, полупрозрачными и размещаться у краев scrollable-области.

## 7. Примеры использования для BackOffice

BackOffice допускает более плотную компоновку.

Пример:

| UI элемент | Размер иконки | Пример иконки |
|------------|---------------|---------------|
| Table row action | 16 px | `Edit`, `Trash2`, `MoreHorizontal` |
| Toolbar action | 20 px | `Plus`, `Filter`, `Search` |
| Primary page action | 24 px | `Plus`, `Save` |
| Empty state | 32 px | доменная иконка раздела |

Правила:

- в таблицах не перегружать каждую строку большим набором иконок;
- destructive actions лучше прятать в menu/confirmation flow;
- иконки рядом с текстом должны быть выровнены по baseline/center;
- если действие неочевидно, нужна подпись или tooltip.

## 8. Правила Шнейдермана для UI

В UI проекта следует руководствоваться восемью золотыми правилами Бена Шнейдермана, адаптированными под POS/BackOffice.

### 8.1 Стремиться к согласованности

Одинаковые действия, иконки, цвета, состояния и layout-паттерны должны вести себя одинаково во всех разделах.

Пример: `Undo2` всегда означает возврат/refund, а не произвольное “назад”.

### 8.2 Обеспечивать быстрые пути для опытных пользователей

POS-операторы работают быстро и повторяют одни и те же действия много раз.

Нужно поддерживать:

- крупные быстрые кнопки;
- predictable layout;
- минимум лишних шагов;
- keyboard shortcuts там, где уместно для backoffice;
- ускоренные повторные операции без нарушения безопасности.

### 8.3 Давать информативную обратную связь

Каждое действие должно иметь видимый результат:

- loading;
- success;
- warning;
- error;
- disabled reason;
- locked state;
- sync/payment/precheck status.

### 8.4 Проектировать завершенные диалоги

Каждый бизнес-процесс должен иметь понятное начало, середину и конец.

Пример:

```text
Пречек -> locked state -> оплата/отмена -> результат -> возврат к заказу/активности
```

Модалки должны явно показывать результат операции и не оставлять пользователя в неопределенном состоянии.

### 8.5 Предотвращать ошибки

UI должен не просто показывать ошибку, а предотвращать ее:

- блокировать недоступные действия;
- показывать причины disabled state;
- требовать confirmation для danger actions;
- использовать manager override для рискованных операций;
- не позволять случайно изменить locked/precheck order.

### 8.6 Позволять отмену действий

Где бизнес-логика допускает отмену, пользователь должен иметь безопасный путь назад.

Важно: отмена UI-действия и финансовая/business cancellation — разные вещи. Их нельзя смешивать одной и той же иконкой или текстом.

### 8.7 Поддерживать внутренний локус контроля пользователя

Пользователь должен чувствовать, что он управляет системой, а не система неожиданно переключает его между экранами.

Пример: оплата открывается modal-окном поверх текущего заказа, а не уводит пользователя в другой раздел без необходимости.

### 8.8 Снижать нагрузку на кратковременную память

Интерфейс должен показывать нужный контекст на экране:

- текущий стол;
- текущий заказ;
- состояние пречека;
- сумма;
- текущая смена;
- выбранный payment method;
- причины блокировки/ошибки.

Пользователь не должен помнить UUID, backend statuses или технические детали.

## 9. Требования к реализации

Frontend должен иметь единый слой маппинга доменных иконок.

Рекомендуемый подход:

```typescript
export const domainIcons = {
  orders: ReceiptText,
  tables: Armchair,
  guests: Users,
  payment: CreditCard,
  cash: Banknote,
  shift: CalendarClock,
  cashDesk: Wallet,
  kitchen: ChefHat,
  inventory: Warehouse,
  catalog: Package,
  dishes: Utensils,
  modifiers: ListPlus,
  discounts: BadgePercent,
  loyalty: Gift,
  fiscal: FileCheck,
  settings: Settings,
  organization: Building2,
  restaurant: Store,
  reports: ChartNoAxesColumn,
  staff: UserCog,
  delivery: Truck,
  reservations: CalendarCheck,
} as const;
```

Правила реализации:

- не импортировать иконки хаотично в каждом компоненте без маппинга;
- для доменных разделов использовать `domainIcons` или аналогичный централизованный registry;
- для action icons использовать отдельный `actionIcons` registry;
- размеры и stroke width задавать через компонент-обертку или design tokens;
- компонент кнопки должен принимать semantic icon key, а не случайный icon component, если это доменная кнопка;
- тесты/линтинг по возможности должны выявлять запрещенные icon packs.

## 10. Что нельзя делать

Запрещено:

- смешивать несколько icon packs в одном UI без архитектурного решения;
- использовать filled/color icons рядом с outline Lucide;
- выбирать иконки по личному вкусу без сверки с этим документом;
- менять доменную иконку модуля в одном месте, не меняя общий mapping;
- использовать слишком мелкие иконки на POS touch-screen;
- делать icon-only кнопки для неочевидных бизнес-действий без подписи/tooltip;
- использовать одну и ту же иконку для разных опасных бизнес-смыслов;
- использовать разные иконки для одного и того же действия в разных разделах.

## 11. Связанные документы

- `REDESIGN_PLAN.md`
- `docs/ui/POS-UI-SPEC.md`
- `docs/ui/POS-UI-RBAC.md`

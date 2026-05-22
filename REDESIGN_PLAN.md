# POS UI Redesign Plan - Промышленный UI для RMS-POS

## Executive Summary

Текущий `pos-ui` уже имеет хорошую архитектуру, соответствующую design.md, но требует визуального и UX улучшения для достижения промышленного уровня. Этот документ описывает конкретные изменения.

## 1. Информационная архитектура

### 1.1 Структура разделов (согласно design.md)

```
Разделы POS (5 основных):
├── Залы и столы (floor) - выбор стола, floor plan
├── Заказы (orders) - экран ввода блюд, menu grid 3/4 + order rail 1/4
├── Активность (activity) - закрытые заказы, платежи, reprint, refund
├── Отчеты (reports) - операционные summaries
└── Касса (cash) - смены, cash drawer, sync diagnostics
```

**Текущее состояние:**
- ✅ Разделы существуют
- ⚠️ Названия: `shift` → `cash`, `analytics` → `activity` + `reports`

### 1.2 Bottom Quick Access Bar

**Требуемая структура (design.md строки 234-256):**
```
┌─────────────────────────────────────────────────────────────┐
│ [Раздел] │ Context Chips (стол, заказ, гость, пречек) │ [Status] │
└─────────────────────────────────────────────────────────────┘
```

**Текущее состояние:**
- ✅ Есть базовый bottom bar
- ❌ Нет context chips
- ❌ Статусы (sync, actor, shift) не показаны компактно

## 2. Визуальная система

### 2.1 CSS Design Tokens

```css
:root {
  /* Surface colors */
  --pos-bg: #f5f7f9;
  --pos-surface: #ffffff;
  --pos-surface-muted: #f0f2f5;
  --pos-rail: #fafbfc;
  
  /* Border colors */
  --pos-border: #dfe3e8;
  --pos-border-strong: #c9cfd6;
  
  /* Text colors */
  --pos-text-primary: #1a1f24;
  --pos-text-secondary: #5c6773;
  --pos-text-muted: #8a95a1;
  
  /* Accent & Status */
  --pos-accent: #2563eb;
  --pos-accent-hover: #1d4ed8;
  --pos-success: #16a34a;
  --pos-warning: #ea580c;
  --pos-danger: #dc2626;
  --pos-info: #0891b2;
  
  /* Dimensions */
  --pos-bottom-nav-height: 64px;
  --pos-action-rail-width: 380px;
  --pos-touch-target-min: 48px;
  --pos-touch-target-main: 56px;
  
  /* Typography */
  --pos-font-family: 'Inter', system-ui, sans-serif;
  --pos-font-mono: 'JetBrains Mono', 'Consolas', monospace;
  
  /* Radius */
  --pos-radius-sm: 6px;
  --pos-radius-md: 10px;
  --pos-radius-lg: 14px;
}
```

### 2.2 Touch Targets (design.md строки 656-665)

| Элемент | Размер | Примечание |
|---------|--------|------------|
| Минимальный touch target | 48×48 px | Обязательно |
| Основные POS кнопки | 56px высота | Primary actions |
| Плитки меню (tablet/desktop) | 96-128 px | Menu grid |
| Плитки меню (compact) | 80-104 px | Mobile adaptation |

### 2.3 Главный viewport и правила скролла

Главное POS-окно всегда должно занимать ровно размер экрана/viewport и не должно иметь собственных scrollbar-ов.

Правило обязательное:

- `html`, `body`, корневой app container и основной POS shell должны работать в пределах `100vw` × `100vh`;
- главное окно POS не должно прокручиваться целиком;
- прокрутка допускается только внутри внутренних областей: menu grid, order lines, activity list, reports table, cash events, modal body;
- bottom bar, action rail, modal stack и основные navigation controls не должны уезжать за пределы экрана;
- при нехватке места должен скроллиться внутренний список/панель, а не весь POS shell;
- любые новые страницы/разделы должны проходить проверку: у главного окна нет document-level scrollbars.

Рекомендуемый CSS-инвариант:

```css
html,
body,
#app {
  width: 100vw;
  height: 100vh;
  overflow: hidden;
}

.pos-shell {
  width: 100vw;
  height: 100vh;
  overflow: hidden;
  display: flex;
  flex-direction: column;
}

.pos-main {
  min-height: 0;
  flex: 1;
  overflow: hidden;
}

.pos-scroll-area {
  min-height: 0;
  overflow: auto;
}
```

### 2.4 Touch-first scrollbar и управление без мыши

POS-интерфейс проектируется в первую очередь под touch/touchpad-работу. Работа мышью является вторичной.

Обязательные требования:

- все кнопки, action controls, list rows и scroll controls должны быть увеличены под палец;
- мелкие desktop-style controls запрещены на рабочих POS-экранах;
- scrollbar должен быть заметным, крупным и удобным для touch;
- стандартный узкий desktop scrollbar не должен быть единственным способом навигации;
- для длинных списков желательно иметь дополнительные touch-friendly кнопки прокрутки.

Специальное правило для кастомных scrollbars:

- убрать классический маленький ползунок между кнопками, если он неудобен для touch;
- по краям scrollable-области отрисовывать две полупрозрачные кнопки прокрутки;
- для горизонтального скролла использовать кнопки с иконками `left` / `right`;
- для вертикального скролла использовать кнопки с иконками `up` / `down`;
- кнопки должны быть достаточно крупными и не перекрывать критичный контент;
- кнопки могут появляться только когда соответствующая прокрутка возможна;
- при удержании кнопки допускается continuous scroll;
- wheel/mouse scrolling и touch/trackpad scrolling должны оставаться рабочими.

### 2.5 UI Icon System

Иконки UI не выбираются локально “по вкусу” разработчика или дизайнера.

Для всех POS и BackOffice экранов обязателен отдельный документ:

```text
docs/ui/UI-ICON-SYSTEM.md
```

Он является источником правил для:

- базовой icon library;
- размеров;
- stroke width;
- цветов и состояний;
- доменного маппинга иконок;
- запрета на смешивание icon packs и визуальных стилей;
- примеров использования в POS и BackOffice.

Краткое правило MVP:

- базовая библиотека: Lucide Icons;
- стиль: outline;
- stroke width: 1.75–2px;
- POS размеры: 28/32 px;
- BackOffice размеры: 16/20/24 px;
- доменные иконки должны браться из централизованного mapping/registry.

### 2.6 Тянущийся дизайн и обрезка текста

Весь POS UI должен быть тянущимся и адаптивным. Интерфейс не должен ломаться из-за длинных названий блюд, столов, категорий, сотрудников, ресторанов или статусов.

Правило обязательное:

- если текст не помещается в кнопку, плитку, строку таблицы, chip, tab или action control, текст не нужно пытаться впихнуть уменьшением шрифта до нечитаемого размера;
- вместо этого текст должен обрезаться и завершаться `...`;
- layout должен сохранять размеры и не раздвигать соседние элементы;
- компоненты должны поддерживать `min-width: 0`, `overflow: hidden`, `text-overflow: ellipsis`, `white-space: nowrap` там, где это уместно;
- для важных длинных значений можно использовать tooltip/details modal, но рабочий экран не должен переполняться.

Рекомендуемый CSS-паттерн:

```css
.pos-ellipsis {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
```

### 2.7 Общие UI-принципы Шнейдермана

При проектировании POS и BackOffice UI следует руководствоваться восемью золотыми правилами Бена Шнейдермана.

В практическом смысле для проекта это означает:

- согласованность действий, иконок, цветов, layout-паттернов и терминологии;
- быстрые пути для опытных пользователей, особенно в POS;
- информативную обратную связь на каждое действие;
- завершенные диалоги с понятным началом, результатом и выходом;
- предотвращение ошибок до выполнения операции;
- безопасную отмену там, где она допустима бизнес-логикой;
- ощущение контроля у пользователя, без неожиданных переходов и скрытых side effects;
- снижение нагрузки на память пользователя через видимый контекст: стол, заказ, пречек, сумма, смена, статус.

Подробная адаптация этих правил зафиксирована в `docs/ui/UI-ICON-SYSTEM.md`.

## 3. Экран выбранного заказа (Приоритет #1)

### 3.1 Layout

```
┌──────────────────────────────────────────────────────────────┐
│ Bottom bar (постоянный)                                       │
├──────────────────────────────────┬───────────────────────────┤
│ Menu Grid (3/4)                  │ Current Order Rail (1/4)  │
│                                  │                           │
│ [Категории/Поиск]                │ Стол 4 · Заказ #A1B2      │
│                                  │                           │
│ [Плитка] [Плитка] [Плитка]       │ ┌─────────────────────┐   │
│ [Плитка] [Плитка] [Плитка]       │ │ Позиции заказа      │   │
│ [Плитка] [Плитка] [Плитка]       │ │                     │   │
│                                  │ │ • Бургер ×2         │   │
│                                  │ │ • Кола ×1           │   │
│                                  │ │                     │   │
│                                  │ ├─────────────────────┤   │
│                                  │ │ Итого: 1 490 ₽      │   │
│                                  │ │                     │   │
│                                  │ │ [Действия] [Пречек] │   │
│                                  │ └─────────────────────┘   │
└──────────────────────────────────┴───────────────────────────┘
```

### 3.2 States (design.md строки 437-446)

| State | Описание | Визуальные признаки |
|-------|----------|---------------------|
| No selected order | Нет выбранного заказа | Empty state с CTA |
| Editable order | Заказ редактируется | Меню активно, кнопки Действия/Пречек |
| Precheck issuing | Выпуск пречека | Loading state, защита от повтора |
| Precheck issued / locked | Пречек выпущен | Read-only, lock banner, Касса/Отмена |
| Payment modal open | Оплата | Modal поверх UI |
| Closed order | Заказ закрыт | Read-only, действия reduced |
| Error / offline | Ошибка | Safe error message |

### 3.3 Locked State Visual Design

```vue
<template>
  <div class="locked-order-banner" role="alert">
    <q-icon name="lock" size="24px" color="warning" />
    <div>
      <strong>Пречек выпущен</strong>
      <small>Заказ заблокирован до оплаты или отмены пречека</small>
    </div>
  </div>
  
  <div class="order-rail locked">
    <!-- read-only lines -->
    <button class="action-btn primary" :disabled="true">Касса</button>
    <button class="action-btn danger">Отмена пречека</button>
  </div>
</template>
```

## 4. Разделы

### 4.1 Залы и столы

**Layout:**
```
┌─────────────────────────────────┬──────────────────────────┐
│ Столы плитками (3/4)            │ Активные заказы (1/4)    │
│                                 │                          │
│ [Зал: Основной ▼]               │ Активные заказы:         │
│                                 │                          │
│ [Стол 1] [Стол 2] [Стол 3]     │ #A1B2 Стол 1 · 1 200 ₽  │
│ [Стол 4] [Стол 5] [Стол 6]     │ #C3D4 Стол 5 · 890 ₽    │
│                                 │ #E5F6 Без стола · 450 ₽ │
└─────────────────────────────────┴──────────────────────────┘
```

**Table card data:**
- Номер/название стола
- Статус (free/open/precheck/paid/unavailable)
- Наличие активного заказа
- Сумма заказа (если есть)
- Время открытия заказа
- Количество гостей (если backend предоставляет)

### 4.2 Заказы

**Menu Grid требования:**
- Категории меню (tabs/chips)
- Поиск по меню
- Плитки 96-128px с названием, ценой, картинкой (если есть)
- Fallback с initials если нет картинки
- Disabled state с hint
- Modifier modal при клике на item с modifier_groups

### 4.3 Активность

**Filters:**
- Поиск по ID заказа, столу, сумме
- Фильтр по дате (business date)
- Фильтр по статусу оплаты (all/refundable)
- Pagination

**Order card:**
- Стол
- Short ID заказа
- Сумма
- Статус чека/оплаты
- Business date

### 4.4 Касса

**Operations:**
- Открыть/закрыть личную смену
- Открыть/закрыть кассовую смену
- Cash in/cash out/no sale/cash count
- Sync health view
- Retry failed sync (если есть permission)

### 4.5 Отчеты

**Metrics:**
- Закрытые заказы за период
- Суммы по методам оплаты
- Sync status
- Shift readiness

## 5. Модалки

### 5.0 Общие правила архитектуры модальных окон и UI-модулей

POS UI должен использовать централизованный блокирующий стек модальных окон и переиспользуемые компонентные модули для всех рабочих окон, кнопок, таблиц, панелей заказа, плиток меню и action controls.

#### 5.0.1 Модальные окна являются блокирующими overlay-окнами

Все модальные окна POS-интерфейса должны блокировать взаимодействие со всем экраном под ними.

Когда открыто модальное окно:

- основной экран должен быть визуально затемнен или заблокирован;
- клики/тапы по фону не должны выполнять действия на основном экране;
- фокус клавиатуры/тача должен оставаться внутри активного модального окна;
- опасные операции должны требовать явного подтверждения внутри модального окна;
- состояние заказа, стола, оплаты или кассы не должно случайно изменяться через фоновые элементы.

Это правило применяется ко всем операционным модальным окнам:

- окно оплаты;
- окно отмены пречека;
- окно manager override;
- окно возврата/отмены;
- окно удаления/void строки заказа;
- окно подтверждения повторной печати;
- будущие окна split bill / merge / transfer;
- любые другие POS action dialogs.

#### 5.0.2 Стек модальных окон и вложенные модальные окна

POS UI должен поддерживать стек модальных окон.

Одно модальное окно может открыть второе модальное окно, второе может открыть третье и так далее, если этого требует бизнес-процесс.

Пример:

```text
Экран заказа
  -> Модальное окно оплаты
      -> Модальное окно manager override
          -> Модальное окно подтверждения
```

Поведение должно быть стековым:

- активным является только верхнее модальное окно;
- нижние модальные окна остаются заблокированными, пока открыт дочерний modal;
- закрытие дочернего окна возвращает управление предыдущему модальному окну;
- закрытие/отмена должны происходить в обратном порядке;
- основной POS-экран остается заблокированным до закрытия всего стека модальных окон;
- каждое модальное окно должно сохранять свое локальное состояние, пока поверх него открыто дочернее окно.

Модальные окна не должны реализовываться как разрозненные ad-hoc overlay. Для всех модальных окон должен использоваться общий modal stack manager / composable / provider.

#### 5.0.3 Переиспользуемая компонентная архитектура

Все элементы POS UI должны разрабатываться как переиспользуемые модули/компоненты, а не как одноразовые блоки внутри конкретной страницы.

В отдельные компоненты должны быть вынесены:

- кнопки;
- action-кнопки;
- danger-кнопки;
- таблицы и grid-компоненты;
- строки заказа;
- панели суммы заказа;
- модальные окна;
- заголовки и футеры модальных окон;
- окна подтверждения;
- окна manager override;
- кнопки блюд / menu item tiles;
- кнопки категорий;
- кнопки способов оплаты;
- status badges;
- empty-state блоки;
- loading/error блоки;
- numeric/keypad inputs;
- quantity controls.

Каждый компонент должен иметь четкую ответственность и не должен содержать несвязанную бизнес-логику.

Бизнес-логика должна находиться в composables / services / API layers, а компоненты должны в основном получать данные и отправлять пользовательские события.

#### 5.0.4 Правила консистентности POS-компонентов

Все переиспользуемые POS-компоненты должны следовать единым правилам дизайна:

- touch-friendly размеры;
- единые отступы и spacing;
- единая типографика;
- единые состояния disabled / loading / error;
- понятное active / selected состояние;
- отдельное визуальное оформление destructive actions;
- отсутствие raw UUID / debug / backend details в рабочем интерфейсе;
- отсутствие дублирующихся реализаций одного и того же визуального паттерна.

Перед созданием нового UI-блока разработчик должен проверить, можно ли переиспользовать или расширить уже существующий POS-компонент.

#### 5.0.5 Требование к разработке

Во время реализации Codex/разработчики не должны хардкодить модальные окна, кнопки, плитки меню, таблицы или action panels напрямую внутри больших page-компонентов, если для этого нет сильной причины.

Предпочтительная структура:

```text
page = только composition/layout
feature component = POS-specific workflow
shared component = reusable UI primitive
composable/service = state и бизнес-логика
API client/schema = backend contract
```

Это правило обязательно, чтобы POS UI оставался поддерживаемым при добавлении следующих процессов:

- возвраты;
- отмены;
- split bill;
- перенос столов;
- объединение заказов;
- скидки;
- фискальные операции;
- KDS;
- складские операции;
- отчеты.

### 5.1 Оплата

```
┌─────────────────────────────────────┐
│ Оплата заказа                       │
├─────────────────────────────────────┤
│ Сумма к оплате: 1 490 ₽             │
│                                     │
│ [Наличные] [Карта] [Другое]        │
│                                     │
│ Личная смена: открыта ✓             │
│ Кассовая смена: открыта ✓           │
│                                     │
│ [Отмена] [Оплатить]                 │
└─────────────────────────────────────┘
```

### 5.2 Действия над заказом

- Изменение quantity
- Void строки
- Выбор modifiers
- Перепечатка пречека/check

### 5.3 Отмена пречека

- Предупреждение
- Reason input
- Manager override PIN
- Результат

## 6. RBAC Integration

Использовать canonical permission ids из `rbac.ts`:

```typescript
const permissions = {
  floor: ['pos.floor.view', 'pos.order.view', 'pos.order.create'],
  orders: ['pos.menu.view', 'pos.catalog.view', 'pos.order.add_line', 
           'pos.order.change_quantity', 'pos.order.void_line',
           'pos.precheck.issue', 'pos.precheck.cancel.request'],
  activity: ['pos.check.view', 'pos.payment.refund', 'pos.check.reprint'],
  cash: ['pos.employee_shift.open', 'pos.cash_session.open', 
         'pos.cash_drawer.record_event', 'pos.sync.retry_failed'],
};
```

## 7. Implementation Priority

### Phase 1 (Неделя 1-2)
1. Обновить CSS tokens и визуальную систему
2. Добавить context chips в bottom bar
3. Улучшить visual hierarchy order rail
4. Реализовать четкий locked state
5. Зафиксировать viewport без document-level scrollbars, touch-first scroll controls, Lucide icons и ellipsis overflow rules
6. Подключить `docs/ui/UI-ICON-SYSTEM.md` как обязательный источник правил по иконкам и UI-принципам Шнейдермана

### Phase 2 (Неделя 3-4)
1. Переименовать разделы согласно design.md
2. Улучшить menu grid с категориями и поиском
3. Добавить skeleton states matching real grid
4. Улучшить empty/error states

### Phase 3 (Неделя 5-6)
1. Оптимизировать responsive breakpoints
2. Добавить long-press context menus
3. Улучшить modal flows
4. E2E тесты для critical paths

## 8. Responsive Breakpoints

```css
/* Desktop */
@media (min-width: 1200px) {
  --pos-action-rail-width: 380px;
  .dish-grid { grid-template-columns: repeat(auto-fill, minmax(140px, 1fr)); }
}

/* Tablet */
@media (max-width: 1199px) and (min-width: 900px) {
  --pos-action-rail-width: 320px;
  .dish-grid { grid-template-columns: repeat(auto-fill, minmax(120px, 1fr)); }
}

/* Compact tablet */
@media (max-width: 899px) {
  .pos-layout { grid-template-columns: 1fr; }
  .action-rail { border-top: 1px solid var(--pos-border); }
}

/* Mobile */
@media (max-width: 520px) {
  .dish-grid { grid-template-columns: repeat(2, 1fr); }
  .floor-table-grid { grid-template-columns: repeat(2, 1fr); }
}
```

## 9. Accessibility

- Все интерактивные элементы имеют aria-label
- Keyboard navigation support
- Focus indicators visible
- Color contrast WCAG AA compliant
- Screen reader friendly structure

## 10. Performance

- Lazy load section components
- Virtual scrolling для длинных списков
- Image lazy loading для menu items
- Optimistic updates для частых операций
- Debounced search inputs

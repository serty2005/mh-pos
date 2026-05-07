# AGENTS.md

## Назначение

Этот файл задает правила работы с документацией и архитектурными границами репозитория `mh-pos`.

Цель проста:

- не смешивать в одном документе текущий runtime, целевую архитектуру и план работ;
- не тащить compatibility-хвосты до первого пилота;
- не допускать расхождения между кодом, тестами и документацией.

Проект еще не имеет production-эксплуатации. До первого пилота действуют правила first-launch development, а не legacy-support development.

## Документы и их владельцы

### Этот файл отвечает за

- границы между документами;
- порядок приоритетов при конфликте источников;
- правила поддержки документации;
- правила clean-before-pilot;
- требования к compatibility tails.

### Этот файл не отвечает за

- полный перечень HTTP endpoints;
- полный перечень экранов UI;
- полное описание схемы БД;
- детальный roadmap по стадиям.

Для этого существуют отдельные документы.

## Карта источников истины

### Код и тесты

Код и тесты — источник истины для того, что **реально реализовано сейчас**.

Если документация утверждает одно, а runtime и тесты делают другое, то для описания текущего состояния приоритет у кода и тестов.

### SPECv1.3

`SPECv1.3.md` — источник истины для:

- архитектурных инвариантов;
- финансовой модели;
- identity model;
- sync model;
- security baseline;
- pilot topology.

### UI-спецификация

`docs/ui/POS-UI-SPEC.md` — источник истины для:

- текущих и целевых экранов;
- пользовательских сценариев;
- UI flow;
- границ frontend responsibility;
- supported и unsupported UX surface.

### UI RBAC

`docs/ui/POS-UI-RBAC.md` — источник истины для:

- ролей сотрудников;
- permission catalog;
- матрицы прав по UI-операциям;
- правил manager override.

### Backend-спецификация

`docs/backend/POS-BACKEND-SPEC.md` — источник истины для:

- публичного API;
- state transitions;
- event catalog;
- текущих compatibility tails.

### Backend data/migrations

`docs/backend/POS-DATA-AND-MIGRATIONS.md` — источник истины для:

- ключевых сущностей;
- связей между сущностями;
- first-launch schema policy;
- reset/migration policy;
- требований к БД.

### Roadmap

`ROADMAP.md` — источник истины для:

- what is done / next / blocked;
- рисков;
- pilot gates;
- sequencing.

## Порядок приоритета при конфликте

Если есть конфликт между документами, использовать такой порядок:

1. Безопасность и архитектурные инварианты: `SPECv1.3.md`.
2. Текущее фактическое поведение: код и тесты.
3. UI / backend surface contracts: профильные документы в `docs/ui/` и `docs/backend/`.
4. План выполнения и статусы: `ROADMAP.md`.
5. `README.md` — только обзорный документ, не финальный арбитр архитектуры.

## Clean-before-pilot policy

До первого пилота запрещено:

- поддерживать старое поведение "на всякий случай", если его не существует в production;
- добавлять dual-write;
- добавлять исторические DB migrations ради обратной совместимости dev-схем;
- тащить legacy aliases без владельца и срока удаления;
- документировать unsupported future behavior как будто оно уже доступно.

Разрешено:

- переписывать canonical first-launch schema;
- удалять deprecated transport/API tails;
- пересоздавать dev/test БД;
- переименовывать сущности и endpoints так, чтобы модель становилась чище.

## Политика схемы БД до первого пилота

До первого пилота SQLite развивается по правилу:

- один канонический `001_init.sql`;
- никакой обязательной исторической цепочки `002`, `003`, `004` для локального runtime;
- любое изменение схемы до пилота делается через обновление canonical init;
- dev/test databases регенерируются с нуля.

Если потребуется редкий временный migration script для локальной разработки, он не становится частью canonical pilot path без отдельного архитектурного решения.

## Политика compatibility tails

Любой compatibility tail допустим только если у него есть:

- владелец;
- причина существования;
- срок удаления или milestone удаления;
- тест, который подтверждает текущее поведение;
- запись в профильной спецификации backend/UI.

Если у compatibility tail нет срока удаления, он не должен быть merged.

## Что считается compatibility tail

Примеры:

- deprecated alias endpoint;
- legacy transport field;
- старое enum name;
- временный adapter слой между old/new payload;
- документация со старым названием сущности, оставленная "ради привычки".

## Правило синхронного обновления документации

Любой PR, который меняет одно из следующих, обязан менять профильную документацию в том же PR:

- HTTP routes or payloads;
- UI screens or user flows;
- permission model;
- DB schema / invariants;
- sync event catalog;
- migration/reset policy.

## Минимальный чек перед merge

Перед merge проверить:

- тесты не спорят с документацией;
- `README.md` не обещает лишнего;
- профильный документ обновлен;
- deprecated tails либо удалены, либо помечены с kill date;
- документация отделяет `implemented now` от `target later`.

## Правило формулировок

В документации использовать только три формулировки статуса:

- `implemented now`
- `planned next`
- `out of scope`

Формулировки вида:

- `temporary for now`
- `legacy but maybe keep`
- `later we will see`

не использовать.

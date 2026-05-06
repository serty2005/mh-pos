---
description: Code quality + refactoring discipline + review mode (front_new)
globs:
  - *.{ts,tsx,js,jsx}"
alwaysApply: true
---

# CODE QUALITY (FRONTEND)

## ❌ Запрещено

- дублирование
- магия и скрытая логика
- «TODO в проде» без тикета/плана
- неявные side effects

## ✅ Принцип

- explicit > implicit
- readable > clever

## REFACTORING DISCIPLINE

- Не делать «рефакторинг ради рефакторинга» без запроса.
- Рефакторинг допустим только если он **необходим** для задачи (и минимален).

## REVIEW MODE (MANDATORY)

Перед финальным ответом/PR ассистент обязан проверить:

1. масштабируемо ли (perf, поддержка роста функционала)
2. читаемо ли (простая навигация, адекватные имена)
3. нет ли скрытой логики
4. учтены ли edge cases
5. обработаны ли ошибки
6. соблюдены ли security правила
7. прошёл бы строгий code review

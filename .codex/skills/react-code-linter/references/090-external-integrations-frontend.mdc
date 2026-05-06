---
description: Паттерны внешних интеграций (frontend): contracts, resilience, idempotency
globs:
  - *.{ts,tsx,js,jsx}"
alwaysApply: true
---

# EXTERNAL INTEGRATIONS (FRONTEND)

## Контракты и совместимость

- API контракты должны быть типизированы (DTO) и иметь явный маппинг в UI модели.
- Backward compatibility: UI должен быть устойчив к отсутствию новых полей и появлению дополнительных.

## Resilience

- Таймауты/отмена запросов (если поддерживается клиентом).
- Retry только для **идемпотентных** запросов и transient ошибок (network/5xx), с backoff.
- Circuit breaker — если есть единый слой/библиотека и это оправдано (иначе не выдумывать).

## Idempotency / double submit

- Защита от двойных сабмитов: disable кнопок на loading, dedupe запросов по ключу, явные «in-flight» флаги.

## Запрещено

- хранить «сырой» response в UI состоянии без нормализации/маппинга
- скрывать ошибки интеграции от пользователя без UX решения

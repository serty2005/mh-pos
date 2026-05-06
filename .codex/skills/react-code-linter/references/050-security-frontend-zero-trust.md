---
description: Security (frontend zero-trust): input, auth, tokens, XSS/CSRF, storage для React
globs:
  - *.{ts,tsx,js,jsx}"
alwaysApply: true
---

# SECURITY (FRONTEND, ZERO TRUST)

## Принцип

Любой вход (UI input, query params, localStorage, backend response) — потенциально вредоносный или неконсистентный.

## Обязательно

- Валидация пользовательского ввода до отправки (и нормализация).
- Экранирование/санитизация при отображении user-generated content.
- Никаких `dangerouslySetInnerHTML` без жесткой санитизации и обоснования.
- Токены/секреты: не логировать, не хранить в открытом виде.
- Не использовать localStorage для высокочувствительных токенов, если можно избежать (предпочтение httpOnly cookies — если backend так устроен).

## Запрещено

- Хардкод секретов, ключей, токенов, приватных URL в репозитории.
- Вывод в UI/логи ошибок с чувствительными деталями (stack traces, токены, сырые ответы).

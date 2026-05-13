# Pilot Scope Decision Questions

Статус: архивная рабочая заметка.

Эта заметка не является актуальным контрактом. Текущий frozen cashier pilot contract находится в `SPECv1.3.md`; статусы и блокеры находятся в `ROADMAP.md`.

Ранее обсуждавшиеся вопросы по `business_date_local`, reprint, waiter payment и refund закрыты или уточнены фактическим кодом:

- `business_date_local` реализован для shifts, cash sessions, payments и checks.
- Controlled reprint precheck/check реализован из immutable snapshots.
- Waiter payment остается вне текущего объема.
- Refund backend route и cashier UI flow реализованы; pilot hardening по audit/sync/reporting policy остается задачей roadmap.

Новые решения по pricing/discounts, modifiers, recipes и inventory consumption должны фиксироваться в `SPECv1.3.md`, `ROADMAP.md` и профильных docs, а не в этой архивной заметке.

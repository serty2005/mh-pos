{a:center}{{.restaurant_name}}
{a:center}{{.restaurant_address}}
---
{a:center}{f:double}TICKET
{a:center}No. {{.ticket_display_number}}
---
{a:center}{f:double}{{.service_name}}
{a:center}{{.category_name}}
---
{qr:size=6:{{.qr_payload}}}
---
{a:center}{{.sale_date_local}} {{.sale_time_local}}
{a:center}Amount: {{.price_minor | money}}
{if:validity_date_local}
{a:center}Valid until: {{.validity_date_local}}
{/if}
{if:is_copy}{a:center}*** COPY ***{/if}
{a:center}{{.cashier_name}} | Shift No.{{.shift_number}}
{s:4}
{cut}

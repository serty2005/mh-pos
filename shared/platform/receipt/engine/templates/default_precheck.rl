---
{a:center}{{.restaurant_name}}
{a:center}{{.restaurant_address}}
---
{a:center}RECEIPT
{a:center}No. {{.precheck_number}}
---
{w:auto,6,12}{a:left,right,right}Item	Qty	Total
---
{each:lines}
{w:auto,6,12}{a:left,right,right}{{.name}}	{{.quantity}}	{{.total_minor | money}}
{if:modifiers}{each:modifiers}
{a:left}+ {{.name}}: {{.price_minor | money}}
{/each}{/if}
{/each}
---
{w:auto,16}{a:left,right}Subtotal:	{{.subtotal_minor | money}}
{if:discount_total_minor}
{w:auto,16}{a:left,right}Discount:	-{{.discount_total_minor | money}}
{/if}
{if:taxes}{each:taxes}
{w:auto,16}{a:left,right}{{.name}}:	{{.amount_minor | money}}
{/each}{/if}
---
{w:auto,16}{a:left,right}TOTAL:	{{.total_minor | money}}
{a:center}{{.cashier_name}}
{a:center}{{.business_date}} | Shift No.{{.shift_number}}
{if:is_copy}{a:center}*** COPY ***{/if}
{s:4}
{cut}

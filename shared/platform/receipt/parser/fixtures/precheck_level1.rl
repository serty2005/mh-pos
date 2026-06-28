---
{a:center}{{.restaurant_name}}
{a:center}{{.restaurant_address}}
---
{a:center}ПРЕДЧЕК
{w:auto,6,8}{a:left,left,right}Наименование	Кол	Сумма
{each:lines}
{w:auto,6,8}{a:left,left,right}{{.name}}	{{.quantity}}	{{.total_minor | money}}
{if:modifiers}{each:modifiers}
{a:left}  + {{.name}}: {{.price_minor | money}}
{/each}{/if}
{/each}
---
{if:is_copy}{a:center}*** КОПИЯ ***{/if}
{cut}

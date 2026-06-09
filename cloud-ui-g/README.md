# MyHoReCa Cloud UI G

`cloud-ui-g` — активный React/Vite Cloud-бэкофис для `cloud-backend`.

## Статус

реализовано сейчас:

- отдельное React 19 / Vite / TypeScript приложение;
- API base по умолчанию берется из `VITE_CLOUD_API_BASE=http://localhost:8090/api/v1`;
- route-backed разделы dashboard, restaurants, Edge sync, catalog, menu, modifiers, pricing/taxes, staff/permissions, floor и publications;
- безопасный API client с Zod-схемами и safe error banner;
- локальный i18n слой для пользовательских строк.

запланировано далее:

- перенос нужных manager-facing Cloud сценариев из устаревшего `cloud-ui` только поверх подтвержденных backend routes;
- развитие inventory/reporting экранов в React-каталоге без обращения к POS Edge runtime endpoints.

вне текущего объема:

- новые Cloud UI правки в `cloud-ui`;
- cashier runtime, KDS runtime, PSP, fiscalization и POS order/payment/check/precheck flows в Cloud UI.

## Запуск

```powershell
cd cloud-ui-g
npm install
npm run dev
```

Открой `http://localhost:5174`.

## Проверки

```powershell
cd cloud-ui-g
npm run lint
npm run test
npm run build
```

import type { Hall, RestaurantTable } from '../../shared/api/schemas';
import { useI18n } from '../../shared/i18n/I18nProvider';

type Props = {
  halls: Hall[];
  tables: RestaurantTable[];
};

export default function FloorPreview({ halls, tables }: Props) {
  const { t } = useI18n();
  const tablesByHall = new Map<string, RestaurantTable[]>();
  for (const table of tables) {
    const list = tablesByHall.get(table.hall_id) ?? [];
    list.push(table);
    tablesByHall.set(table.hall_id, list);
  }

  return (
    <section className="space-y-4 rounded-2xl border border-slate-200 bg-white p-6">
      <div>
        <h3 className="text-base font-semibold text-slate-900">{t('floor.preview.title')}</h3>
        <p className="mt-1 text-sm text-slate-600">{t('floor.preview.description')}</p>
      </div>
      {halls.length === 0 ? <p className="text-sm text-slate-600">{t('floor.preview.empty')}</p> : null}
      <div className="grid gap-3 md:grid-cols-2">
        {halls.map((hall) => {
          const hallTables = tablesByHall.get(hall.id) ?? [];
          const publishedCount = hallTables.filter((table) => table.status === 'published').length;
          return (
            <article key={hall.id} className="rounded-xl border border-slate-200 bg-slate-50 p-4">
              <div className="flex flex-wrap items-start justify-between gap-2">
                <div>
                  <p className="text-sm font-semibold text-slate-900">{hall.name}</p>
                  <p className="text-xs text-slate-600">{t('floor.shared.status')}: {t(`catalog.statuses.${hall.status}`)}</p>
                </div>
                <p className="rounded-lg border border-slate-200 bg-white px-2 py-1 text-xs text-slate-700">
                  {publishedCount}/{hallTables.length}
                </p>
              </div>
              {hallTables.length === 0 ? (
                <p className="mt-3 text-xs text-slate-500">{t('floor.preview.noTables')}</p>
              ) : (
                <div className="mt-3 grid grid-cols-2 gap-2 sm:grid-cols-3">
                  {hallTables.map((table) => (
                    <div key={table.id} className="min-h-16 rounded-lg border border-slate-200 bg-white p-2">
                      <p className="truncate text-sm font-medium text-slate-900">{table.name}</p>
                      <p className="text-xs text-slate-600">{table.seats} · {t(`catalog.statuses.${table.status}`)}</p>
                    </div>
                  ))}
                </div>
              )}
            </article>
          );
        })}
      </div>
    </section>
  );
}

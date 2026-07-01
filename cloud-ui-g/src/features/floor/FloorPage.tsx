import { useEffect, useState } from 'react';
import { Table2 } from 'lucide-react';
import {
  archiveHall,
  archiveTable,
  createHall,
  createTable,
  listHalls,
  listRestaurantSections,
  listTables,
  updateHall,
  updateTable,
} from '../../shared/api/endpoints';
import type { Hall, RestaurantSection, RestaurantTable } from '../../shared/api/schemas';
import { useI18n } from '../../shared/i18n/I18nProvider';
import EmptyState from '../../shared/ui/EmptyState';
import FloorPreview from './FloorPreview';
import HallsPanel from './HallsPanel';
import TablesPanel from './TablesPanel';
import {
  buildCreateHallPayload,
  buildCreateTablePayload,
  buildUpdateHallPayload,
  buildUpdateTablePayload,
  type HallFormValues,
  type HallUpdateFormValues,
  type TableFormValues,
} from './floorForms';

type Props = {
  restaurantId: string;
};

type RouteStatus = 'loading' | 'ready' | 'blocked';

export default function FloorPage({ restaurantId }: Props) {
  const { t } = useI18n();
  const [halls, setHalls] = useState<Hall[]>([]);
  const [sections, setSections] = useState<RestaurantSection[]>([]);
  const [tables, setTables] = useState<RestaurantTable[]>([]);
  const [status, setStatus] = useState<RouteStatus>('loading');
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<unknown>(null);

  const reload = async () => {
    setStatus('loading');
    setError(null);
    try {
      const [nextHalls, nextSections, nextTables] = await Promise.all([
        listHalls(restaurantId),
        listRestaurantSections(restaurantId),
        listTables(restaurantId),
      ]);
      setHalls(nextHalls);
      setSections(nextSections);
      setTables(nextTables);
      setStatus('ready');
    } catch (nextError) {
      setStatus('blocked');
      setError(nextError);
    }
  };

  useEffect(() => {
    void reload();
  }, [restaurantId]);

  const mutate = async (action: () => Promise<void>) => {
    setLoading(true);
    setError(null);
    try {
      await action();
      await reload();
    } catch (nextError) {
      setError(nextError);
    } finally {
      setLoading(false);
    }
  };

  return (
    <section className="space-y-4">
      <div className="rounded-2xl border border-slate-200 bg-white p-5 sm:p-6">
        <div className="flex flex-col gap-4 lg:flex-row lg:items-start lg:justify-between">
          <div className="flex items-start gap-3">
            <div className="flex h-10 w-10 shrink-0 items-center justify-center rounded-xl border border-blue-100 bg-blue-50 text-blue-700">
              <Table2 className="h-4 w-4" />
            </div>
            <div>
              <h3 className="text-lg font-semibold tracking-tight text-slate-950">{t('floor.pageTitle')}</h3>
              <p className="mt-1 max-w-3xl text-sm leading-6 text-slate-600">{t('floor.pageDescription')}</p>
            </div>
          </div>
          <p className={status === 'ready' ? 'rounded-full border border-emerald-100 bg-emerald-50 px-3 py-1.5 text-xs font-semibold text-emerald-700' : status === 'loading' ? 'rounded-full border border-blue-100 bg-blue-50 px-3 py-1.5 text-xs font-semibold text-blue-700' : 'rounded-full border border-amber-100 bg-amber-50 px-3 py-1.5 text-xs font-semibold text-amber-700'}>
            {t('catalog.readiness')}: {status === 'ready' ? t('status.ready') : status === 'loading' ? t('status.loading') : t('status.blocked')}
          </p>
        </div>
      </div>
      {status === 'blocked' ? <EmptyState title={t('floor.blockedTitle')} description={t('floor.blockedDescription')} /> : null}
      {status !== 'blocked' ? (
        <>
          <HallsPanel
            halls={halls}
            loading={loading}
            error={error}
            onCreate={(values: HallFormValues) => mutate(async () => { await createHall({ restaurant_id: restaurantId, ...buildCreateHallPayload(values) }); })}
            onUpdate={(id: string, values: HallUpdateFormValues) => mutate(async () => { await updateHall(id, buildUpdateHallPayload(values)); })}
            onArchive={(id: string) => mutate(async () => { await archiveHall(id); })}
          />
          <TablesPanel
            halls={halls}
            sections={sections}
            tables={tables}
            loading={loading}
            error={error}
            onCreate={(values: TableFormValues) => mutate(async () => { await createTable({ restaurant_id: restaurantId, ...buildCreateTablePayload(values) }); })}
            onUpdate={(id: string, values: TableFormValues) => mutate(async () => { await updateTable(id, buildUpdateTablePayload(values)); })}
            onArchive={(id: string) => mutate(async () => { await archiveTable(id); })}
          />
          <FloorPreview halls={halls} tables={tables} />
        </>
      ) : null}
    </section>
  );
}

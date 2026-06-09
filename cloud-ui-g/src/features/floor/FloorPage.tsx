import { useEffect, useState } from 'react';
import {
  archiveHall,
  archiveTable,
  createHall,
  createTable,
  listHalls,
  listTables,
  updateHall,
  updateTable,
} from '../../shared/api/endpoints';
import type { Hall, RestaurantTable } from '../../shared/api/schemas';
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
  const [tables, setTables] = useState<RestaurantTable[]>([]);
  const [status, setStatus] = useState<RouteStatus>('loading');
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<unknown>(null);

  const reload = async () => {
    setStatus('loading');
    setError(null);
    try {
      const [nextHalls, nextTables] = await Promise.all([
        listHalls(restaurantId),
        listTables(restaurantId),
      ]);
      setHalls(nextHalls);
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
      <div className="rounded-2xl border border-slate-200 bg-white p-6">
        <h3 className="text-base font-semibold text-slate-900">{t('floor.pageTitle')}</h3>
        <p className="mt-1 text-sm text-slate-600">{t('floor.pageDescription')}</p>
        <p className="mt-2 text-xs text-slate-500">{t('catalog.readiness')}: {status === 'ready' ? t('status.ready') : status === 'loading' ? t('status.loading') : t('status.blocked')}</p>
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

import { useState } from 'react';
import { Archive, Building2, Pencil, Store } from 'lucide-react';
import { archiveRestaurant, createRestaurant, updateRestaurant } from '../../shared/api/endpoints';
import type { Restaurant } from '../../shared/api/schemas';
import { useI18n } from '../../shared/i18n/I18nProvider';
import EmptyState from '../../shared/ui/EmptyState';
import PanelHeader from '../../shared/ui/PanelHeader';
import SafeErrorBanner from '../../shared/ui/SafeErrorBanner';
import RestaurantForm, { buildCreateRestaurantPayload, type RestaurantFormValues } from './RestaurantForm';

type RestaurantsPageProps = {
  restaurants: Restaurant[];
  onReload: () => Promise<void>;
};

export default function RestaurantsPage({ restaurants, onReload }: RestaurantsPageProps) {
  const { t } = useI18n();
  const [editing, setEditing] = useState<Restaurant | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<unknown>(null);

  const submitCreate = async (values: RestaurantFormValues) => {
    setLoading(true);
    setError(null);
    try {
      await createRestaurant(buildCreateRestaurantPayload(values));
      await onReload();
    } catch (nextError) {
      setError(nextError);
    } finally {
      setLoading(false);
    }
  };

  const submitEdit = async (values: RestaurantFormValues) => {
    if (!editing) return;
    setLoading(true);
    setError(null);
    try {
      await updateRestaurant(editing.id, values);
      setEditing(null);
      await onReload();
    } catch (nextError) {
      setError(nextError);
    } finally {
      setLoading(false);
    }
  };

  const handleArchive = async (id: string) => {
    setLoading(true);
    setError(null);
    try {
      await archiveRestaurant(id);
      if (editing?.id === id) setEditing(null);
      await onReload();
    } catch (nextError) {
      setError(nextError);
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="space-y-5">
      <section className="rounded-2xl border border-slate-200 bg-white p-5 sm:p-6">
        <PanelHeader
          icon={Building2}
          title={t('restaurants.pageTitle')}
          description={t('restaurants.pageDescription')}
          action={(
            <div className="rounded-2xl border border-slate-200 bg-slate-50 px-4 py-3 text-right">
              <p className="text-xs font-semibold text-slate-500">{t('restaurants.count')}</p>
              <p className="mt-1 font-mono text-2xl font-semibold text-slate-950">{restaurants.length}</p>
            </div>
          )}
        />
      </section>

      {error ? <SafeErrorBanner error={error} /> : null}

      <div className="grid gap-5 xl:grid-cols-[minmax(18rem,0.9fr)_minmax(0,1.4fr)]">
        <section className="space-y-3">
          <div>
            <h4 className="text-sm font-semibold text-slate-900">{t('restaurants.formTitle')}</h4>
            <p className="mt-1 text-sm leading-6 text-slate-600">{t('restaurants.pageDescription')}</p>
          </div>
          <RestaurantForm mode="create" onSubmit={submitCreate} disabled={loading} />
        </section>

        <section className="space-y-3 rounded-2xl border border-slate-200 bg-white p-4 sm:p-5">
          <div className="flex flex-wrap items-start justify-between gap-3">
            <div>
              <h4 className="text-sm font-semibold text-slate-900">{t('restaurants.listTitle')}</h4>
              <p className="mt-1 text-sm leading-6 text-slate-600">{t('restaurants.listDescription')}</p>
            </div>
          </div>
          {restaurants.length === 0 ? <EmptyState title={t('restaurants.empty')} description={t('ui.noDataBody')} /> : null}

          {restaurants.map((restaurant) => (
            <article key={restaurant.id} className="rounded-xl border border-slate-200 p-4">
              <div className="flex flex-wrap items-start justify-between gap-3">
                <div className="flex min-w-0 items-start gap-3">
                  <div className="flex h-10 w-10 shrink-0 items-center justify-center rounded-xl border border-slate-200 bg-slate-50 text-slate-600">
                    <Store className="h-4 w-4" />
                  </div>
                  <div className="min-w-0">
                    <p className="truncate text-sm font-semibold text-slate-900">{restaurant.name}</p>
                    <p className="mt-1 text-xs text-slate-600">{restaurant.timezone} · {restaurant.currency}</p>
                    <span className={restaurant.status === 'active' ? 'mt-2 inline-flex rounded-lg border border-emerald-100 bg-emerald-50 px-2 py-1 text-[11px] font-semibold text-emerald-700' : 'mt-2 inline-flex rounded-lg border border-slate-200 bg-slate-100 px-2 py-1 text-[11px] font-semibold text-slate-600'}>
                      {t(`restaurants.status.${restaurant.status}`)}
                    </span>
                  </div>
                </div>
                <div className="flex flex-wrap gap-2">
                  <button type="button" onClick={() => setEditing(restaurant)} className="inline-flex items-center gap-1.5 rounded-lg border border-slate-300 px-2 py-1 text-xs text-slate-700" disabled={loading}>
                    <Pencil className="h-3.5 w-3.5" />
                    {t('restaurants.actions.edit')}
                  </button>
                  <button type="button" onClick={() => { void handleArchive(restaurant.id); }} className="inline-flex items-center gap-1.5 rounded-lg border border-rose-300 px-2 py-1 text-xs text-rose-700" disabled={loading || restaurant.status === 'archived'}>
                    <Archive className="h-3.5 w-3.5" />
                    {t('restaurants.actions.archive')}
                  </button>
                </div>
              </div>

              {editing?.id === restaurant.id ? (
                <div className="mt-3">
                  <RestaurantForm mode="edit" initial={restaurant} onSubmit={submitEdit} onCancel={() => setEditing(null)} disabled={loading} />
                </div>
              ) : null}
            </article>
          ))}
        </section>
      </div>
    </div>
  );
}

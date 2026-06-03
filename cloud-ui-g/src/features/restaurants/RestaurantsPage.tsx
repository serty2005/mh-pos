import { useState } from 'react';
import { archiveRestaurant, createRestaurant, updateRestaurant } from '../../shared/api/endpoints';
import type { Restaurant } from '../../shared/api/schemas';
import { useI18n } from '../../shared/i18n/I18nProvider';
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
    <section className="space-y-4 rounded-2xl border border-slate-200 bg-white p-6">
      <div>
        <h3 className="text-base font-semibold text-slate-900">{t('restaurants.pageTitle')}</h3>
        <p className="mt-1 text-sm text-slate-600">{t('restaurants.pageDescription')}</p>
      </div>

      <RestaurantForm mode="create" onSubmit={submitCreate} disabled={loading} />
      {error ? <SafeErrorBanner error={error} /> : null}

      <div className="space-y-3">
        <h4 className="text-sm font-semibold text-slate-900">{t('restaurants.listTitle')}</h4>
        {restaurants.length === 0 ? <p className="text-sm text-slate-600">{t('restaurants.empty')}</p> : null}

        {restaurants.map((restaurant) => (
          <article key={restaurant.id} className="rounded-xl border border-slate-200 p-4">
            <div className="flex flex-wrap items-start justify-between gap-2">
              <div>
                <p className="text-sm font-medium text-slate-900">{restaurant.name}</p>
                <p className="text-xs text-slate-600">{restaurant.timezone} · {restaurant.currency}</p>
              </div>
              <div className="flex gap-2">
                <button type="button" onClick={() => setEditing(restaurant)} className="rounded-lg border border-slate-300 px-2 py-1 text-xs text-slate-700" disabled={loading}>{t('restaurants.actions.edit')}</button>
                <button type="button" onClick={() => { void handleArchive(restaurant.id); }} className="rounded-lg border border-rose-300 px-2 py-1 text-xs text-rose-700" disabled={loading || restaurant.status === 'archived'}>{t('restaurants.actions.archive')}</button>
              </div>
            </div>

            {editing?.id === restaurant.id ? (
              <div className="mt-3">
                <RestaurantForm mode="edit" initial={restaurant} onSubmit={submitEdit} onCancel={() => setEditing(null)} disabled={loading} />
              </div>
            ) : null}
          </article>
        ))}
      </div>
    </section>
  );
}

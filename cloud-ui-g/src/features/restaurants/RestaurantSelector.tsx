import { useMemo } from 'react';
import { RefreshCw, Store } from 'lucide-react';
import type { Restaurant } from '../../shared/api/schemas';
import { useI18n } from '../../shared/i18n/I18nProvider';
import LoadingSkeleton from '../../shared/ui/LoadingSkeleton';
import SafeErrorBanner from '../../shared/ui/SafeErrorBanner';

type RestaurantsStatus = 'idle' | 'loading' | 'ready' | 'blocked';

type RestaurantSelectorProps = {
  restaurants: Restaurant[];
  status: RestaurantsStatus;
  error: unknown;
  selectedRestaurantId: string;
  onSelectRestaurant: (restaurantId: string) => void;
  onRetry: () => void;
};

export default function RestaurantSelector({
  restaurants,
  status,
  error,
  selectedRestaurantId,
  onSelectRestaurant,
  onRetry,
}: RestaurantSelectorProps) {
  const { t } = useI18n();

  const selectedRestaurant = useMemo(
    () => restaurants.find((restaurant) => restaurant.id === selectedRestaurantId) ?? null,
    [restaurants, selectedRestaurantId],
  );

  return (
    <section className="rounded-2xl border border-slate-200 bg-white p-4 sm:p-5">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <div className="flex items-start gap-3">
          <div className="flex h-10 w-10 shrink-0 items-center justify-center rounded-xl border border-blue-100 bg-blue-50 text-blue-700">
            <Store className="h-4 w-4" />
          </div>
          <div>
            <h2 className="text-sm font-semibold tracking-tight text-slate-900">{t('restaurants.selectorTitle')}</h2>
            <p className="mt-1 text-xs leading-5 text-slate-500">{t('restaurants.selectorHint')}</p>
          </div>
        </div>
        <button
          type="button"
          onClick={onRetry}
          className="inline-flex items-center gap-2 rounded-xl border border-slate-300 px-3 py-2 text-xs font-semibold text-slate-700 hover:bg-slate-100"
        >
          <RefreshCw className="h-3.5 w-3.5" />
          {t('ui.retry')}
        </button>
      </div>

      {status === 'loading' ? <div className="mt-4"><LoadingSkeleton /></div> : null}
      {status === 'blocked' ? <div className="mt-4"><SafeErrorBanner error={error} /></div> : null}

      {status === 'ready' ? (
        <div className="mt-4 grid gap-3 md:grid-cols-[minmax(0,1fr)_auto] md:items-end">
          <label className="grid gap-1 text-sm font-medium text-slate-700" htmlFor="restaurant-selector">
            {t('restaurants.selectorLabel')}
            <select
              id="restaurant-selector"
              className="w-full rounded-lg border border-slate-300 bg-white px-3 py-2 text-sm text-slate-900"
              value={selectedRestaurantId}
              onChange={(event) => onSelectRestaurant(event.target.value)}
            >
              <option value="">{t('restaurants.selectorPlaceholder')}</option>
              {restaurants.map((restaurant) => (
                <option key={restaurant.id} value={restaurant.id}>
                  {restaurant.name}
                </option>
              ))}
            </select>
          </label>

          {selectedRestaurant ? (
            <p className="rounded-xl border border-emerald-100 bg-emerald-50 px-3 py-2 text-xs text-emerald-800">
              {t('restaurants.selectedLabel')}: <span className="font-medium text-slate-900">{selectedRestaurant.name}</span>
            </p>
          ) : (
            <p className="rounded-xl border border-amber-100 bg-amber-50 px-3 py-2 text-xs text-amber-800">{t('restaurants.notSelected')}</p>
          )}
        </div>
      ) : null}
    </section>
  );
}

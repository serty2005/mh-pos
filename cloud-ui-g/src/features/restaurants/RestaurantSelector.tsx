import { useMemo } from 'react';
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
    <section className="rounded-2xl border border-slate-200 bg-white p-4">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <div>
          <h2 className="text-sm font-semibold text-slate-900">{t('restaurants.selectorTitle')}</h2>
          <p className="text-xs text-slate-500">{t('restaurants.selectorHint')}</p>
        </div>
        <button
          type="button"
          onClick={onRetry}
          className="rounded-lg border border-slate-300 px-3 py-1.5 text-xs font-medium text-slate-700 hover:bg-slate-100"
        >
          {t('ui.retry')}
        </button>
      </div>

      {status === 'loading' ? <div className="mt-4"><LoadingSkeleton /></div> : null}
      {status === 'blocked' ? <div className="mt-4"><SafeErrorBanner error={error} /></div> : null}

      {status === 'ready' ? (
        <div className="mt-4 grid gap-3">
          <label className="text-sm text-slate-700" htmlFor="restaurant-selector">
            {t('restaurants.selectorLabel')}
          </label>
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

          {selectedRestaurant ? (
            <p className="text-xs text-slate-600">
              {t('restaurants.selectedLabel')}: <span className="font-medium text-slate-900">{selectedRestaurant.name}</span>
            </p>
          ) : (
            <p className="text-xs text-slate-600">{t('restaurants.notSelected')}</p>
          )}
        </div>
      ) : null}
    </section>
  );
}

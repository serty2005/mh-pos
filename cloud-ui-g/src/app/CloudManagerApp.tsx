import { useMemo, useState } from 'react';
import Sidebar from './Sidebar';
import { navigationById, navigationItems } from './navigation';
import type { CloudRouteId } from './routes';
import { defaultRoute } from './routes';
import RestaurantSelector from '../features/restaurants/RestaurantSelector';
import { useRestaurants } from '../features/restaurants/useRestaurants';
import { apiBase } from '../shared/api/client';
import { useI18n } from '../shared/i18n/I18nProvider';
import EmptyState from '../shared/ui/EmptyState';
import RestaurantsPage from '../features/restaurants/RestaurantsPage';
import DashboardPage from '../features/dashboard/DashboardPage';
import PublicationPanel from '../features/publications/PublicationPanel';
import EdgeSyncPage from '../features/edge/EdgeSyncPage';
import CatalogPage from '../features/catalog/CatalogPage';

export default function CloudManagerApp() {
  const { t } = useI18n();
  const [activeRouteId, setActiveRouteId] = useState<CloudRouteId>(defaultRoute.id);
  const [selectedRestaurantId, setSelectedRestaurantId] = useState('');
  const [isSidebarOpen, setSidebarOpen] = useState(false);
  const { restaurants, status, error, reload } = useRestaurants();

  const activeItem = useMemo(() => navigationById.get(activeRouteId) ?? navigationItems[0], [activeRouteId]);
  const selectedRestaurant = useMemo(
    () => restaurants.find((restaurant) => restaurant.id === selectedRestaurantId) ?? null,
    [restaurants, selectedRestaurantId],
  );
  const isRestaurantSelected = Boolean(selectedRestaurant);
  const routeRequiresRestaurant = activeItem.route.scope === 'restaurant';

  const openRestaurantsRoute = () => {
    setActiveRouteId('restaurants');
    setSidebarOpen(false);
  };

  const handleNavigate = (routeId: CloudRouteId) => {
    const nextItem = navigationById.get(routeId);
    if (!nextItem) return;

    if (nextItem.route.scope === 'restaurant' && !isRestaurantSelected) {
      return;
    }

    setActiveRouteId(routeId);
    setSidebarOpen(false);
  };

  return (
    <main className="min-h-screen bg-slate-50 p-3 sm:p-4 lg:p-6">
      <div className="mx-auto flex w-full max-w-7xl flex-col gap-4 lg:flex-row">
        <div className="flex items-center justify-between rounded-xl border border-slate-200 bg-white p-3 lg:hidden">
          <div>
            <p className="text-sm font-semibold text-slate-900">{t('app.title')}</p>
            <p className="text-xs text-slate-500">{t('nav.mobileHint')}</p>
          </div>
          <button
            type="button"
            onClick={() => setSidebarOpen((prev) => !prev)}
            className="rounded-lg border border-slate-300 px-3 py-2 text-sm font-medium text-slate-700"
          >
            {isSidebarOpen ? t('nav.closeMenu') : t('nav.openMenu')}
          </button>
        </div>

        <Sidebar
          items={navigationItems}
          activeRouteId={activeRouteId}
          isRestaurantSelected={isRestaurantSelected}
          isOpen={isSidebarOpen}
          onNavigate={handleNavigate}
        />

        <section className="min-w-0 flex-1 space-y-4">
          <header className="rounded-2xl border border-slate-200 bg-white p-4">
            <div className="flex flex-wrap items-center justify-between gap-3">
              <div>
                <h2 className="text-lg font-semibold text-slate-900">{t(activeItem.labelKey)}</h2>
                <p className="text-sm text-slate-500">{t('dashboard.readinessDescription')}</p>
              </div>
              <div className="text-xs text-slate-600">
                <div>{t('app.environment')}: {import.meta.env.MODE}</div>
                <div>{t('app.apiBase')}: {apiBase}</div>
              </div>
            </div>
          </header>

          <RestaurantSelector
            restaurants={restaurants}
            status={status}
            error={error}
            selectedRestaurantId={selectedRestaurantId}
            onSelectRestaurant={setSelectedRestaurantId}
            onRetry={() => {
              void reload();
            }}
          />

          {routeRequiresRestaurant && !isRestaurantSelected ? (
            <section className="rounded-2xl border border-slate-200 bg-white p-6">
              <EmptyState
                title={t('restaurants.actionRequiredTitle')}
                description={t('restaurants.actionRequiredBody')}
              />
              <div className="mt-4 flex flex-wrap gap-2">
                <button
                  type="button"
                  onClick={openRestaurantsRoute}
                  className="rounded-lg border border-slate-300 px-3 py-2 text-sm font-medium text-slate-700 hover:bg-slate-100"
                >
                  {t('restaurants.openDirectory')}
                </button>
                <button
                  type="button"
                  onClick={openRestaurantsRoute}
                  className="rounded-lg bg-slate-900 px-3 py-2 text-sm font-medium text-white hover:bg-slate-700"
                >
                  {t('restaurants.createRestaurant')}
                </button>
              </div>
            </section>
          ) : null}

          {activeRouteId === 'dashboard' ? (
            <DashboardPage restaurantId={selectedRestaurantId} />
          ) : null}

          {activeRouteId === 'restaurants' ? (
            <RestaurantsPage
              restaurants={restaurants}
              onReload={reload}
            />
          ) : null}

          {activeRouteId === 'publications' && isRestaurantSelected ? (
            <PublicationPanel restaurantId={selectedRestaurantId} canPublish={isRestaurantSelected} />
          ) : null}

          {activeRouteId === 'edge-sync' && isRestaurantSelected ? (
            <EdgeSyncPage restaurantId={selectedRestaurantId} />
          ) : null}

          {activeRouteId === 'catalog' && isRestaurantSelected ? (
            <CatalogPage restaurantId={selectedRestaurantId} />
          ) : null}

          {activeRouteId !== 'dashboard' && activeRouteId !== 'restaurants' && activeRouteId !== 'publications' && activeRouteId !== 'edge-sync' && activeRouteId !== 'catalog' && isRestaurantSelected ? (
            <section className="rounded-2xl border border-slate-200 bg-white p-6">
              <h3 className="text-base font-semibold text-slate-900">{t(activeItem.labelKey)}</h3>
              <p className="mt-1 text-sm text-slate-600">{t('sections.blocked')}</p>
              <p className="mt-4 text-sm text-slate-700">
                {t('dashboard.selectedRestaurant')}: {selectedRestaurant?.name}
              </p>
            </section>
          ) : null}
        </section>
      </div>
    </main>
  );
}

import { useEffect, useMemo, useState } from 'react';
import { Menu, X } from 'lucide-react';
import Sidebar from './Sidebar';
import { navigationById, navigationForEntitlements, navigationItems } from './navigation';
import type { CloudRouteId } from './routes';
import { defaultRoute } from './routes';
import { useRestaurants } from '../features/restaurants/useRestaurants';
import { useI18n } from '../shared/i18n/I18nProvider';
import EmptyState from '../shared/ui/EmptyState';
import RestaurantsPage from '../features/restaurants/RestaurantsPage';
import DashboardPage from '../features/dashboard/DashboardPage';
import PublicationPanel from '../features/publications/PublicationPanel';
import EdgeSyncPage from '../features/edge/EdgeSyncPage';
import CatalogPage from '../features/catalog/CatalogPage';
import MenuPage from '../features/menu/MenuPage';
import ModifiersPage from '../features/modifiers/ModifiersPage';
import PricingPage from '../features/pricing/PricingPage';
import StaffPage from '../features/staff/StaffPage';
import FloorPage from '../features/floor/FloorPage';
import { getEntitlements } from '../shared/api/endpoints';

export default function CloudManagerApp() {
  const { t } = useI18n();
  const [activeRouteId, setActiveRouteId] = useState<CloudRouteId>(defaultRoute.id);
  const [selectedRestaurantId, setSelectedRestaurantId] = useState('');
  const [isSidebarOpen, setSidebarOpen] = useState(false);
  const { restaurants, status, error, reload } = useRestaurants();
  const [entitlements, setEntitlements] = useState<Record<string, boolean>>({});

  useEffect(() => {
    void getEntitlements().then((snapshot) => {
      if (snapshot.status === 'active' && new Date(snapshot.expires_at).getTime() > Date.now()) {
        setEntitlements(snapshot.entitlements);
      }
    }).catch(() => setEntitlements({}));
  }, []);

  const licensedNavigation = useMemo(() => navigationForEntitlements(entitlements), [entitlements]);

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
    <main className="min-h-[100dvh] bg-slate-100 font-sans text-slate-800">
      <div className="flex min-h-screen w-full flex-col lg:flex-row">
        <div className="flex items-center justify-between border-b border-slate-200 bg-white p-3 lg:hidden">
          <div>
            <p className="text-sm font-semibold text-slate-900">{t('app.title')}</p>
            <p className="text-xs text-slate-500">{t('nav.mobileHint')}</p>
          </div>
          <button
            type="button"
            onClick={() => setSidebarOpen((prev) => !prev)}
            className="inline-flex h-10 w-10 items-center justify-center rounded-xl border border-slate-300 text-slate-700 transition-colors hover:bg-slate-100"
            aria-label={isSidebarOpen ? t('nav.closeMenu') : t('nav.openMenu')}
          >
            {isSidebarOpen ? <X className="h-4 w-4" /> : <Menu className="h-4 w-4" />}
          </button>
        </div>

        <Sidebar
          items={licensedNavigation}
          activeRouteId={activeRouteId}
          isRestaurantSelected={isRestaurantSelected}
          isOpen={isSidebarOpen}
          restaurants={restaurants}
          restaurantsStatus={status}
          restaurantsError={error}
          selectedRestaurantId={selectedRestaurantId}
          onSelectRestaurant={setSelectedRestaurantId}
          onRetryRestaurants={() => {
            void reload();
          }}
          onNavigate={handleNavigate}
        />

        <section className="min-w-0 flex-1 space-y-5 overflow-y-auto border-l border-slate-200 bg-slate-50/70 p-4 sm:p-6 lg:h-screen lg:p-8">
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

          {activeRouteId === 'menu' && isRestaurantSelected ? (
            <MenuPage restaurantId={selectedRestaurantId} restaurantCurrency={selectedRestaurant?.currency ?? 'RUB'} />
          ) : null}

          {activeRouteId === 'modifiers' && isRestaurantSelected ? (
            <ModifiersPage restaurantId={selectedRestaurantId} />
          ) : null}

          {activeRouteId === 'pricing-taxes' && isRestaurantSelected ? (
            <PricingPage restaurantId={selectedRestaurantId} />
          ) : null}

          {activeRouteId === 'staff-permissions' && isRestaurantSelected ? (
            <StaffPage restaurantId={selectedRestaurantId} />
          ) : null}

          {activeRouteId === 'floor' && isRestaurantSelected ? (
            <FloorPage restaurantId={selectedRestaurantId} />
          ) : null}

          {activeRouteId !== 'dashboard' && activeRouteId !== 'restaurants' && activeRouteId !== 'publications' && activeRouteId !== 'edge-sync' && activeRouteId !== 'catalog' && activeRouteId !== 'menu' && activeRouteId !== 'modifiers' && activeRouteId !== 'pricing-taxes' && activeRouteId !== 'staff-permissions' && activeRouteId !== 'floor' && isRestaurantSelected ? (
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

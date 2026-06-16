import {
  BarChart3,
  Building2,
  ChevronDown,
  ChefHat,
  ClipboardList,
  Layers3,
  Percent,
  RefreshCw,
  Settings2,
  ShieldCheck,
  Store,
  Table2,
  Utensils,
  Warehouse,
} from 'lucide-react';
import type { NavigationItem } from './navigation';
import type { CloudRouteId } from './routes';
import { apiBase } from '../shared/api/client';
import type { Restaurant } from '../shared/api/schemas';
import { useI18n } from '../shared/i18n/I18nProvider';

type RestaurantsStatus = 'idle' | 'loading' | 'ready' | 'blocked';

type SidebarProps = {
  items: NavigationItem[];
  activeRouteId: CloudRouteId;
  isRestaurantSelected: boolean;
  isOpen: boolean;
  restaurants: Restaurant[];
  restaurantsStatus: RestaurantsStatus;
  restaurantsError: unknown;
  selectedRestaurantId: string;
  onSelectRestaurant: (restaurantId: string) => void;
  onRetryRestaurants: () => void;
  onNavigate: (routeId: CloudRouteId) => void;
};

export default function Sidebar({
  items,
  activeRouteId,
  isRestaurantSelected,
  isOpen,
  restaurants,
  restaurantsStatus,
  restaurantsError,
  selectedRestaurantId,
  onSelectRestaurant,
  onRetryRestaurants,
  onNavigate,
}: SidebarProps) {
  const { t } = useI18n();
  const iconByRoute: Partial<Record<CloudRouteId, typeof BarChart3>> = {
    dashboard: BarChart3,
    restaurants: Building2,
    'edge-sync': RefreshCw,
    catalog: ClipboardList,
    menu: Utensils,
    modifiers: Layers3,
    'pricing-taxes': Percent,
    'staff-permissions': ShieldCheck,
    floor: Table2,
    publications: Store,
    inventory: Warehouse,
    reports: ChefHat,
  };
  const selectedRestaurant = restaurants.find((restaurant) => restaurant.id === selectedRestaurantId) ?? null;
  const selectorDisabled = restaurantsStatus !== 'ready';
  const restaurantCode = selectedRestaurant
    ? `${selectedRestaurant.currency} / ${selectedRestaurant.timezone}`
    : t('restaurants.selectorGlobalCode');

  return (
    <aside
      className={[
        'flex h-[calc(100dvh-4rem)] w-full shrink-0 select-none flex-col border-r border-slate-800 bg-[#0f172a] text-white shadow-2xl shadow-slate-950/20 lg:h-screen lg:w-72',
        isOpen ? 'flex' : 'hidden lg:flex',
      ].join(' ')}
    >
      <div className="flex items-center gap-3 border-b border-slate-800 p-5">
        <div className="flex h-9 w-9 items-center justify-center rounded-xl bg-blue-600 text-lg font-black text-white shadow-md">
          M
        </div>
        <div className="min-w-0">
          <h1 className="truncate text-base font-bold leading-none tracking-tight text-white">{t('app.title')}</h1>
          <p className="mt-0.5 font-mono text-[8px] font-semibold uppercase tracking-wider text-slate-500">{t('app.subtitle')}</p>
        </div>
      </div>

      <div className="border-b border-slate-800 px-4 pb-3 pt-4">
        <label className="mb-1.5 block font-mono text-[9px] font-bold uppercase tracking-widest text-slate-500" htmlFor="sidebar-restaurant-selector">
          {t('restaurants.sidebarLabel')}
        </label>
        <div className="relative">
          <Store className="pointer-events-none absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-slate-400" />
          <select
            id="sidebar-restaurant-selector"
            value={selectedRestaurantId}
            disabled={selectorDisabled}
            onChange={(event) => onSelectRestaurant(event.target.value)}
            className="w-full cursor-pointer appearance-none rounded-xl border border-slate-800 bg-slate-900 py-2.5 pl-9 pr-9 text-xs font-bold text-white transition-colors focus:border-blue-500 focus:outline-none disabled:cursor-not-allowed disabled:opacity-55"
          >
            <option value="" className="bg-slate-900 text-white">{t('restaurants.selectorAll')}</option>
            {restaurants.map((restaurant) => (
              <option key={restaurant.id} value={restaurant.id} className="bg-slate-900 text-white">
                {restaurant.name}
              </option>
            ))}
          </select>
          <div className="pointer-events-none absolute right-3 top-1/2 flex -translate-y-1/2 items-center gap-1.5">
            <span className={selectedRestaurant ? 'h-2 w-2 rounded-full bg-emerald-500' : 'h-2 w-2 rounded-full bg-blue-500'} />
            <ChevronDown className="h-3.5 w-3.5 text-slate-400" />
          </div>
        </div>

        <div className="mt-2 flex justify-between gap-3 px-1 font-mono text-[9px] text-slate-500">
          <span>{t('restaurants.sidebarRegion')}</span>
          <span className="truncate font-bold uppercase text-slate-400">{restaurantCode}</span>
        </div>
        <div className="mt-2 flex items-center justify-between gap-2 px-1 text-[10px]">
          <span className="font-mono font-semibold uppercase tracking-wider text-slate-500">
            {restaurantsStatus === 'ready' ? t('restaurants.sidebarLoaded') : restaurantsStatus === 'loading' ? t('status.loading') : t('status.blocked')}
          </span>
          {restaurantsStatus === 'blocked' ? (
            <button
              type="button"
              onClick={onRetryRestaurants}
              className="rounded border border-slate-700 px-2 py-1 font-mono text-[9px] font-bold uppercase tracking-wider text-blue-400 hover:border-blue-500 hover:text-blue-300"
              aria-label={`${t('ui.retry')}: ${String(restaurantsError ? t('status.blocked') : t('restaurants.sidebarLoaded'))}`}
            >
              {t('ui.retry')}
            </button>
          ) : restaurantsStatus === 'loading' ? (
            <RefreshCw className="h-3.5 w-3.5 animate-spin text-blue-400" />
          ) : (
            <span className="font-mono text-[9px] font-bold text-slate-400">{restaurants.length}</span>
          )}
        </div>
      </div>

      <nav className="min-h-0 flex-1 overflow-y-auto py-4">
        <div className="mb-2 px-6 py-2 text-[10px] font-bold uppercase tracking-wider text-slate-500">
          {t('nav.mobileHint')}
        </div>
        <ul className="space-y-1">
          {items.map((item) => {
            const isActive = item.route.id === activeRouteId;
            const isDisabled = item.route.scope === 'restaurant' && !isRestaurantSelected;
            const Icon = iconByRoute[item.route.id] ?? Settings2;

            return (
              <li key={item.route.id}>
                <button
                  type="button"
                  disabled={isDisabled}
                  onClick={() => onNavigate(item.route.id)}
                  className={[
                    'flex w-full items-center justify-between border-l-4 px-6 py-3.5 text-left text-sm transition-all duration-200',
                    isActive
                      ? 'border-blue-500 bg-blue-600/10 font-semibold text-blue-400'
                      : 'border-transparent text-slate-400 hover:bg-slate-800/40 hover:text-white',
                    isDisabled ? 'cursor-not-allowed opacity-45' : '',
                  ].join(' ')}
                >
                  <span className="flex min-w-0 items-center gap-3">
                    <Icon className={['h-4 w-4 shrink-0 transition-transform', isActive ? 'scale-110' : ''].join(' ')} />
                    <span className="truncate">{t(item.labelKey)}</span>
                  </span>
                  {isDisabled ? <span className="ml-2 text-[10px] uppercase tracking-wide text-slate-500">{t('nav.locked')}</span> : null}
                </button>
              </li>
            );
          })}
        </ul>
      </nav>

      <div className="mt-auto border-t border-slate-800 bg-slate-950/60 p-4 font-sans">
        <div className="flex items-center gap-3 rounded-xl border border-slate-800 bg-slate-900/50 p-3">
          <div className="flex h-7 w-7 shrink-0 items-center justify-center rounded-lg bg-blue-600 text-xs font-bold leading-none text-white shadow-inner">
            {t('app.footerInitials')}
          </div>
          <div className="min-w-0 flex-1">
            <p className="truncate text-xs font-bold text-slate-100">{t('app.operatorName')}</p>
            <p className="font-mono text-[8px] font-semibold uppercase tracking-wider text-slate-500">{import.meta.env.MODE}</p>
          </div>
        </div>
        <div className="mt-3 flex items-center justify-between gap-3 px-1 font-mono text-[10px] text-slate-500">
          <div className="flex min-w-0 items-center gap-1.5">
            <span className="h-1.5 w-1.5 shrink-0 animate-pulse rounded-full bg-green-500" />
            <span className="truncate text-[9px] font-bold uppercase tracking-wider text-green-500">{t('app.syncActive')}</span>
          </div>
          <span className="truncate text-[9px]">{apiBase}</span>
        </div>
      </div>
    </aside>
  );
}

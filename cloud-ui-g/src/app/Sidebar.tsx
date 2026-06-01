import type { NavigationItem } from './navigation';
import type { CloudRouteId } from './routes';
import { useI18n } from '../shared/i18n/I18nProvider';

type SidebarProps = {
  items: NavigationItem[];
  activeRouteId: CloudRouteId;
  isRestaurantSelected: boolean;
  isOpen: boolean;
  onNavigate: (routeId: CloudRouteId) => void;
};

export default function Sidebar({
  items,
  activeRouteId,
  isRestaurantSelected,
  isOpen,
  onNavigate,
}: SidebarProps) {
  const { t } = useI18n();

  return (
    <aside
      className={[
        'w-full border border-slate-200 bg-white lg:sticky lg:top-4 lg:h-[calc(100vh-2rem)] lg:w-80 lg:rounded-2xl',
        isOpen ? 'block' : 'hidden lg:block',
      ].join(' ')}
    >
      <div className="border-b border-slate-200 p-4">
        <h1 className="text-base font-semibold text-slate-900">{t('app.title')}</h1>
        <p className="mt-1 text-xs text-slate-500">{t('app.subtitle')}</p>
      </div>

      <nav className="max-h-[50vh] overflow-y-auto p-2 lg:max-h-[calc(100vh-8rem)]">
        <ul className="space-y-1">
          {items.map((item) => {
            const isActive = item.route.id === activeRouteId;
            const isDisabled = item.route.scope === 'restaurant' && !isRestaurantSelected;

            return (
              <li key={item.route.id}>
                <button
                  type="button"
                  disabled={isDisabled}
                  onClick={() => onNavigate(item.route.id)}
                  className={[
                    'flex w-full items-center justify-between rounded-lg px-3 py-2 text-left text-sm transition-colors',
                    isActive ? 'bg-slate-900 text-white' : 'text-slate-700 hover:bg-slate-100',
                    isDisabled ? 'cursor-not-allowed opacity-45' : '',
                  ].join(' ')}
                >
                  <span>{t(item.labelKey)}</span>
                  {isDisabled ? <span className="text-[10px] uppercase tracking-wide">{t('nav.locked')}</span> : null}
                </button>
              </li>
            );
          })}
        </ul>
      </nav>
    </aside>
  );
}

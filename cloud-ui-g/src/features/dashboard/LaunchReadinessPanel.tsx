import { useEffect, useMemo, useState } from 'react';
import { CheckCircle2, CircleDashed, Gauge } from 'lucide-react';
import {
  listCatalogItems,
  listEmployees,
  listHalls,
  listMenuItems,
  listModifierBindings,
  listModifierGroups,
  listModifierOptions,
  listPricingPolicies,
  listRoles,
  listRestaurantDevices,
  listTables,
} from '../../shared/api/endpoints';
import { useI18n } from '../../shared/i18n/I18nProvider';
import LoadingSkeleton from '../../shared/ui/LoadingSkeleton';
import PanelHeader from '../../shared/ui/PanelHeader';
import SafeErrorBanner from '../../shared/ui/SafeErrorBanner';

type LaunchReadinessPanelProps = {
  restaurantId: string;
  hasPublication: boolean;
};

export default function LaunchReadinessPanel({ restaurantId, hasPublication }: LaunchReadinessPanelProps) {
  const { t } = useI18n();
  const [error, setError] = useState<unknown>(null);
  const [loading, setLoading] = useState(false);
  const [checks, setChecks] = useState<Record<string, boolean>>({});

  useEffect(() => {
    if (!restaurantId) {
      setChecks({});
      return;
    }

    setLoading(true);
    setError(null);
    Promise.all([
      listRoles(restaurantId),
      listEmployees(restaurantId),
      listHalls(restaurantId),
      listTables(restaurantId),
      listCatalogItems(restaurantId),
      listMenuItems(restaurantId),
      listModifierGroups(restaurantId),
      listModifierOptions(restaurantId),
      listModifierBindings(restaurantId),
      listPricingPolicies(restaurantId),
      listRestaurantDevices(restaurantId),
    ])
      .then(([roles, employees, halls, tables, catalogItems, menuItems, modifierGroups, modifierOptions, modifierBindings, pricing, devices]) => {
        const edgeAssigned = devices.some((device) => device.restaurant_id === restaurantId && device.status === 'assigned');
        setChecks({
          rolesEmployees: roles.length > 0 && employees.length > 0,
          hallsTables: halls.length > 0 && tables.length > 0,
          catalogItems: catalogItems.length > 0,
          menuItems: menuItems.length > 0,
          modifiersPricing: (modifierGroups.length > 0 || modifierOptions.length > 0 || modifierBindings.length > 0) && pricing.length > 0,
          edgeAssigned,
          publicationExists: hasPublication,
        });
      })
      .catch((nextError) => {
        setError(nextError);
      })
      .finally(() => {
        setLoading(false);
      });
  }, [hasPublication, restaurantId]);

  const items = useMemo(() => [
    { key: 'restaurantSelected', label: t('dashboard.readiness.restaurantSelected'), ready: Boolean(restaurantId) },
    { key: 'rolesEmployees', label: t('dashboard.readiness.rolesEmployees'), ready: Boolean(checks.rolesEmployees) },
    { key: 'hallsTables', label: t('dashboard.readiness.hallsTables'), ready: Boolean(checks.hallsTables) },
    { key: 'catalogItems', label: t('dashboard.readiness.catalogItems'), ready: Boolean(checks.catalogItems) },
    { key: 'menuItems', label: t('dashboard.readiness.menuItems'), ready: Boolean(checks.menuItems) },
    { key: 'modifiersPricing', label: t('dashboard.readiness.modifiersPricing'), ready: Boolean(checks.modifiersPricing) },
    { key: 'edgeAssigned', label: t('dashboard.readiness.edgeAssigned'), ready: Boolean(checks.edgeAssigned) },
    { key: 'publicationExists', label: t('dashboard.readiness.publicationExists'), ready: Boolean(checks.publicationExists) },
  ], [checks, restaurantId, t]);
  const readyCount = items.filter((item) => item.ready).length;
  const readinessPercent = Math.round((readyCount / items.length) * 100);

  return (
    <section className="rounded-2xl border border-slate-200 bg-white p-5 sm:p-6">
      <PanelHeader
        icon={Gauge}
        title={t('dashboard.readinessTitle')}
        description={t('dashboard.readinessDescription')}
        action={(
          <div className="min-w-[9rem] rounded-2xl border border-slate-200 bg-slate-50 px-4 py-3 text-right">
            <p className="text-xs font-semibold text-slate-500">{t('catalog.readiness')}</p>
            <p className="mt-1 font-mono text-2xl font-semibold text-slate-950">{readyCount}/{items.length}</p>
            <div className="mt-3 h-1.5 overflow-hidden rounded-full bg-slate-200">
              <div className="h-full rounded-full bg-blue-600" style={{ width: `${readinessPercent}%` }} />
            </div>
          </div>
        )}
      />
      {error ? <div className="mt-3"><SafeErrorBanner error={error} /></div> : null}
      {loading ? (
        <div className="mt-5">
          <LoadingSkeleton cards={4} className="grid gap-3 sm:grid-cols-2 xl:grid-cols-4" />
        </div>
      ) : (
        <div className="mt-5 grid gap-3 sm:grid-cols-2 xl:grid-cols-4">
          {items.map((item) => (
            <div key={item.key} className="rounded-2xl border border-slate-200 bg-slate-50/70 px-3 py-3 text-sm">
              <div className="flex items-start justify-between gap-3">
                <p className="leading-5 text-slate-800">{item.label}</p>
                {item.ready ? <CheckCircle2 className="h-4 w-4 shrink-0 text-emerald-600" /> : <CircleDashed className="h-4 w-4 shrink-0 text-amber-600" />}
              </div>
              <p className={item.ready ? 'mt-2 text-xs font-semibold text-emerald-700' : 'mt-2 text-xs font-semibold text-amber-700'}>{item.ready ? t('dashboard.readiness.ready') : t('dashboard.readiness.pending')}</p>
            </div>
          ))}
        </div>
      )}
    </section>
  );
}

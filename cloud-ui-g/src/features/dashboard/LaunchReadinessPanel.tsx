import { useEffect, useMemo, useState } from 'react';
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
  listTables,
  listUnassignedDevices,
} from '../../shared/api/endpoints';
import { useI18n } from '../../shared/i18n/I18nProvider';
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
      listUnassignedDevices(),
    ])
      .then(([roles, employees, halls, tables, catalogItems, menuItems, modifierGroups, modifierOptions, modifierBindings, pricing, devices]) => {
        const edgeAssigned = devices.some((device) => device.assigned_restaurant_id === restaurantId || device.status === 'assigned');
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

  return (
    <section className="rounded-2xl border border-slate-200 bg-white p-6">
      <h3 className="text-base font-semibold text-slate-900">{t('dashboard.readinessTitle')}</h3>
      <p className="mt-1 text-sm text-slate-600">{t('dashboard.readinessDescription')}</p>
      {loading ? <p className="mt-3 text-sm text-slate-600">{t('ui.loading')}</p> : null}
      {error ? <div className="mt-3"><SafeErrorBanner error={error} /></div> : null}
      <div className="mt-4 grid gap-2 sm:grid-cols-2">
        {items.map((item) => (
          <div key={item.key} className="rounded-lg border border-slate-200 px-3 py-2 text-sm">
            <p className="text-slate-800">{item.label}</p>
            <p className={item.ready ? 'text-emerald-700' : 'text-amber-700'}>{item.ready ? t('dashboard.readiness.ready') : t('dashboard.readiness.pending')}</p>
          </div>
        ))}
      </div>
    </section>
  );
}

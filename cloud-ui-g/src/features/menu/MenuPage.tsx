import { useEffect, useState } from 'react';
import { Utensils } from 'lucide-react';
import { archiveMenuItem, createMenuCategory, createMenuItem, listCatalogItems, listMenuItems, updateMenuItem } from '../../shared/api/endpoints';
import type { CatalogItem, MenuItem } from '../../shared/api/schemas';
import { useI18n } from '../../shared/i18n/I18nProvider';
import EmptyState from '../../shared/ui/EmptyState';
import MenuCategoryCommandPanel from './MenuCategoryCommandPanel';
import MenuItemsPanel from './MenuItemsPanel';
import { buildCreateMenuItemPayload, type MenuCategoryFormValues, type MenuItemFormValues } from './menuForms';

type Props = {
  restaurantId: string;
  restaurantCurrency: string;
};

type RouteStatus = 'loading' | 'ready' | 'blocked';

export default function MenuPage({ restaurantId, restaurantCurrency }: Props) {
  const { t } = useI18n();
  const [menuItems, setMenuItems] = useState<MenuItem[]>([]);
  const [catalogItems, setCatalogItems] = useState<CatalogItem[]>([]);
  const [status, setStatus] = useState<RouteStatus>('loading');
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<unknown>(null);
  const [categorySuccess, setCategorySuccess] = useState(false);

  const reload = async () => {
    setStatus('loading');
    setError(null);
    try {
      const [nextMenuItems, nextCatalogItems] = await Promise.all([
        listMenuItems(restaurantId),
        listCatalogItems(restaurantId),
      ]);
      setMenuItems(nextMenuItems);
      setCatalogItems(nextCatalogItems);
      setStatus('ready');
    } catch (nextError) {
      setStatus('blocked');
      setError(nextError);
    }
  };

  useEffect(() => {
    void reload();
  }, [restaurantId]);

  const mutate = async (action: () => Promise<void>) => {
    setLoading(true);
    setCategorySuccess(false);
    setError(null);
    try {
      await action();
      await reload();
    } catch (nextError) {
      setError(nextError);
    } finally {
      setLoading(false);
    }
  };

  return (
    <section className="space-y-4">
      <div className="rounded-2xl border border-slate-200 bg-white p-5 sm:p-6">
        <div className="flex flex-col gap-4 lg:flex-row lg:items-start lg:justify-between">
          <div className="flex items-start gap-3">
            <div className="flex h-10 w-10 shrink-0 items-center justify-center rounded-xl border border-blue-100 bg-blue-50 text-blue-700">
              <Utensils className="h-4 w-4" />
            </div>
            <div>
              <h3 className="text-lg font-semibold tracking-tight text-slate-950">{t('menu.pageTitle')}</h3>
              <p className="mt-1 max-w-3xl text-sm leading-6 text-slate-600">{t('menu.pageDescription')}</p>
            </div>
          </div>
          <p className={status === 'ready' ? 'rounded-full border border-emerald-100 bg-emerald-50 px-3 py-1.5 text-xs font-semibold text-emerald-700' : status === 'loading' ? 'rounded-full border border-blue-100 bg-blue-50 px-3 py-1.5 text-xs font-semibold text-blue-700' : 'rounded-full border border-amber-100 bg-amber-50 px-3 py-1.5 text-xs font-semibold text-amber-700'}>
            {t('catalog.readiness')}: {status === 'ready' ? t('status.ready') : status === 'loading' ? t('status.loading') : t('status.blocked')}
          </p>
        </div>
      </div>
      {status === 'blocked' ? <EmptyState title={t('menu.blockedTitle')} description={t('menu.blockedDescription')} /> : null}
      {status !== 'blocked' ? (
        <>
          <MenuItemsPanel
            items={menuItems}
            catalogItems={catalogItems}
            restaurantCurrency={restaurantCurrency}
            loading={loading}
            error={error}
            onCreate={(values: MenuItemFormValues) => mutate(async () => { await createMenuItem({ restaurant_id: restaurantId, ...buildCreateMenuItemPayload(values) }); })}
            onUpdate={(id: string, values: MenuItemFormValues) => mutate(async () => { await updateMenuItem(id, values); })}
            onArchive={(id: string) => mutate(async () => { await archiveMenuItem(id); })}
          />
          <MenuCategoryCommandPanel
            restaurantId={restaurantId}
            loading={loading}
            error={error}
            success={categorySuccess}
            onCreate={(values: MenuCategoryFormValues) => mutate(async () => {
              await createMenuCategory(values);
              setCategorySuccess(true);
            })}
          />
        </>
      ) : null}
    </section>
  );
}

import { useEffect, useState } from 'react';
import { archiveMenuItem, createMenuCategory, createMenuItem, listCatalogItems, listMenuItems, updateMenuItem } from '../../shared/api/endpoints';
import type { CatalogItem, MenuItem } from '../../shared/api/schemas';
import { useI18n } from '../../shared/i18n/I18nProvider';
import EmptyState from '../../shared/ui/EmptyState';
import MenuCategoryCommandPanel from './MenuCategoryCommandPanel';
import MenuItemsPanel from './MenuItemsPanel';
import type { MenuCategoryFormValues, MenuItemFormValues } from './menuForms';

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
      <div className="rounded-2xl border border-slate-200 bg-white p-6">
        <h3 className="text-base font-semibold text-slate-900">{t('menu.pageTitle')}</h3>
        <p className="mt-1 text-sm text-slate-600">{t('menu.pageDescription')}</p>
        <p className="mt-2 text-xs text-slate-500">{t('catalog.readiness')}: {status === 'ready' ? t('status.ready') : status === 'loading' ? t('status.loading') : t('status.blocked')}</p>
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
            onCreate={(values: MenuItemFormValues) => mutate(async () => { await createMenuItem({ restaurant_id: restaurantId, ...values }); })}
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

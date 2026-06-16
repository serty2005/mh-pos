import { useEffect, useState } from 'react';
import { Layers3 } from 'lucide-react';
import {
  createModifierBinding,
  createModifierGroup,
  createModifierOption,
  listCatalogFolders,
  listCatalogItems,
  listCatalogTags,
  listMenuItems,
  listModifierBindings,
  listModifierGroups,
  listModifierOptions,
  updateModifierBinding,
  updateModifierGroup,
  updateModifierOption,
} from '../../shared/api/endpoints';
import type { CatalogFolder, CatalogItem, CatalogTag, MenuItem, ModifierBinding, ModifierGroup, ModifierOption } from '../../shared/api/schemas';
import { useI18n } from '../../shared/i18n/I18nProvider';
import EmptyState from '../../shared/ui/EmptyState';
import ModifierBindingsPanel from './ModifierBindingsPanel';
import ModifierGroupsPanel from './ModifierGroupsPanel';
import ModifierOptionsPanel from './ModifierOptionsPanel';
import {
  buildCreateModifierBindingPayload,
  buildCreateModifierGroupPayload,
  buildCreateModifierOptionPayload,
  type ModifierBindingFormValues,
  type ModifierGroupFormValues,
  type ModifierOptionFormValues,
} from './modifierForms';

type Props = {
  restaurantId: string;
};

type RouteStatus = 'loading' | 'ready' | 'blocked';

export default function ModifiersPage({ restaurantId }: Props) {
  const { t } = useI18n();
  const [groups, setGroups] = useState<ModifierGroup[]>([]);
  const [options, setOptions] = useState<ModifierOption[]>([]);
  const [bindings, setBindings] = useState<ModifierBinding[]>([]);
  const [menuItems, setMenuItems] = useState<MenuItem[]>([]);
  const [catalogItems, setCatalogItems] = useState<CatalogItem[]>([]);
  const [folders, setFolders] = useState<CatalogFolder[]>([]);
  const [tags, setTags] = useState<CatalogTag[]>([]);
  const [status, setStatus] = useState<RouteStatus>('loading');
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<unknown>(null);

  const reload = async () => {
    setStatus('loading');
    setError(null);
    try {
      const [nextGroups, nextOptions, nextBindings, nextMenuItems, nextCatalogItems, nextFolders, nextTags] = await Promise.all([
        listModifierGroups(restaurantId),
        listModifierOptions(restaurantId),
        listModifierBindings(restaurantId),
        listMenuItems(restaurantId),
        listCatalogItems(restaurantId),
        listCatalogFolders(restaurantId),
        listCatalogTags(restaurantId),
      ]);
      setGroups(nextGroups);
      setOptions(nextOptions);
      setBindings(nextBindings);
      setMenuItems(nextMenuItems);
      setCatalogItems(nextCatalogItems);
      setFolders(nextFolders);
      setTags(nextTags);
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
              <Layers3 className="h-4 w-4" />
            </div>
            <div>
              <h3 className="text-lg font-semibold tracking-tight text-slate-950">{t('modifiers.pageTitle')}</h3>
              <p className="mt-1 max-w-3xl text-sm leading-6 text-slate-600">{t('modifiers.pageDescription')}</p>
            </div>
          </div>
          <p className={status === 'ready' ? 'rounded-full border border-emerald-100 bg-emerald-50 px-3 py-1.5 text-xs font-semibold text-emerald-700' : status === 'loading' ? 'rounded-full border border-blue-100 bg-blue-50 px-3 py-1.5 text-xs font-semibold text-blue-700' : 'rounded-full border border-amber-100 bg-amber-50 px-3 py-1.5 text-xs font-semibold text-amber-700'}>
            {t('catalog.readiness')}: {status === 'ready' ? t('status.ready') : status === 'loading' ? t('status.loading') : t('status.blocked')}
          </p>
        </div>
      </div>
      {status === 'blocked' ? <EmptyState title={t('modifiers.blockedTitle')} description={t('modifiers.blockedDescription')} /> : null}
      {status !== 'blocked' ? (
        <>
          <ModifierGroupsPanel
            groups={groups}
            loading={loading}
            error={error}
            onCreate={(values: ModifierGroupFormValues) => mutate(async () => { await createModifierGroup({ restaurant_id: restaurantId, ...buildCreateModifierGroupPayload(values) }); })}
            onUpdate={(id: string, values: ModifierGroupFormValues) => mutate(async () => { await updateModifierGroup(id, values); })}
          />
          <ModifierOptionsPanel
            options={options}
            groups={groups}
            loading={loading}
            error={error}
            onCreate={(values: ModifierOptionFormValues) => mutate(async () => { await createModifierOption({ restaurant_id: restaurantId, ...buildCreateModifierOptionPayload(values) }); })}
            onUpdate={(id: string, values: ModifierOptionFormValues) => mutate(async () => { await updateModifierOption(id, { name: values.name, price_minor: values.price_minor, status: values.status }); })}
          />
          <ModifierBindingsPanel
            bindings={bindings}
            groups={groups}
            menuItems={menuItems}
            catalogItems={catalogItems}
            folders={folders}
            tags={tags}
            loading={loading}
            error={error}
            onCreate={(values: ModifierBindingFormValues) => mutate(async () => { await createModifierBinding({ restaurant_id: restaurantId, ...buildCreateModifierBindingPayload(values) }); })}
            onUpdate={(id: string, values: ModifierBindingFormValues) => mutate(async () => { await updateModifierBinding(id, { sort_order: values.sort_order, status: values.status }); })}
          />
        </>
      ) : null}
    </section>
  );
}

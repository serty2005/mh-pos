import { useEffect, useState } from 'react';
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
      <div className="rounded-2xl border border-slate-200 bg-white p-6">
        <h3 className="text-base font-semibold text-slate-900">{t('modifiers.pageTitle')}</h3>
        <p className="mt-1 text-sm text-slate-600">{t('modifiers.pageDescription')}</p>
        <p className="mt-2 text-xs text-slate-500">{t('catalog.readiness')}: {status === 'ready' ? t('status.ready') : status === 'loading' ? t('status.loading') : t('status.blocked')}</p>
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

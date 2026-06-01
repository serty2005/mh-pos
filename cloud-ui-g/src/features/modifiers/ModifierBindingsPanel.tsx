import { useMemo, useState } from 'react';
import type { CatalogFolder, CatalogItem, CatalogTag, MenuItem, ModifierBinding, ModifierGroup } from '../../shared/api/schemas';
import { useI18n } from '../../shared/i18n/I18nProvider';
import SafeErrorBanner from '../../shared/ui/SafeErrorBanner';
import { defaultModifierBindingValues, type ModifierBindingFormValues, type ModifierTargetType } from './modifierForms';

type Props = {
  bindings: ModifierBinding[];
  groups: ModifierGroup[];
  menuItems: MenuItem[];
  catalogItems: CatalogItem[];
  folders: CatalogFolder[];
  tags: CatalogTag[];
  loading: boolean;
  error: unknown;
  onCreate: (values: ModifierBindingFormValues) => Promise<void>;
  onUpdate: (id: string, values: ModifierBindingFormValues) => Promise<void>;
};

const targetTypes: ModifierTargetType[] = ['menu_item', 'catalog_item', 'folder', 'tag'];

export default function ModifierBindingsPanel({ bindings, groups, menuItems, catalogItems, folders, tags, loading, error, onCreate, onUpdate }: Props) {
  const { t } = useI18n();
  const [values, setValues] = useState<ModifierBindingFormValues>(defaultModifierBindingValues);
  const targets = useMemo(() => {
    if (values.target_type === 'menu_item') return menuItems.map((item) => ({ id: item.id, name: item.name }));
    if (values.target_type === 'catalog_item') return catalogItems.map((item) => ({ id: item.id, name: item.name }));
    if (values.target_type === 'folder') return folders.map((item) => ({ id: item.id, name: item.name }));
    return tags.map((item) => ({ id: item.id, name: item.name }));
  }, [catalogItems, folders, menuItems, tags, values.target_type]);

  const groupName = (id: string) => groups.find((group) => group.id === id)?.name ?? id;

  return (
    <section className="space-y-4 rounded-2xl border border-slate-200 bg-white p-6">
      <h3 className="text-base font-semibold text-slate-900">{t('modifiers.bindings.title')}</h3>
      <form className="grid gap-3 md:grid-cols-5" onSubmit={(event) => {
        event.preventDefault();
        void onCreate(values).then(() => setValues(defaultModifierBindingValues));
      }}>
        <select value={values.modifier_group_id} onChange={(event) => setValues({ ...values, modifier_group_id: event.target.value })} className="rounded-lg border border-slate-300 px-3 py-2 text-sm" disabled={loading}>
          <option value="">{t('modifiers.options.fields.selectGroup')}</option>
          {groups.filter((group) => group.status !== 'archived').map((group) => <option key={group.id} value={group.id}>{group.name}</option>)}
        </select>
        <select value={values.target_type} onChange={(event) => setValues({ ...values, target_type: event.target.value as ModifierTargetType, target_id: '' })} className="rounded-lg border border-slate-300 px-3 py-2 text-sm" disabled={loading}>
          {targetTypes.map((type) => <option key={type} value={type}>{t(`modifiers.bindings.targetTypes.${type}`)}</option>)}
        </select>
        <select value={values.target_id} onChange={(event) => setValues({ ...values, target_id: event.target.value })} className="rounded-lg border border-slate-300 px-3 py-2 text-sm" disabled={loading}>
          <option value="">{t('modifiers.bindings.fields.selectTarget')}</option>
          {targets.map((target) => <option key={target.id} value={target.id}>{target.name}</option>)}
        </select>
        <input type="number" value={values.sort_order} onChange={(event) => setValues({ ...values, sort_order: Number(event.target.value) })} className="rounded-lg border border-slate-300 px-3 py-2 text-sm" disabled={loading} />
        <button type="submit" disabled={loading || !values.modifier_group_id || !values.target_id} className="rounded-lg bg-slate-900 px-3 py-2 text-sm font-medium text-white disabled:opacity-50">{t('modifiers.bindings.actions.create')}</button>
      </form>
      {error ? <SafeErrorBanner error={error} /> : null}
      {bindings.length === 0 ? <p className="text-sm text-slate-600">{t('modifiers.bindings.empty')}</p> : null}
      {bindings.map((binding) => (
        <article key={binding.id} className="flex flex-wrap items-center justify-between gap-2 rounded-xl border border-slate-200 p-4">
          <p className="text-sm text-slate-900">{groupName(binding.modifier_group_id)} · {t(`modifiers.bindings.targetTypes.${binding.target_type}`)} · {binding.target_id}</p>
          <button type="button" onClick={() => void onUpdate(binding.id, { modifier_group_id: binding.modifier_group_id, target_type: binding.target_type, target_id: binding.target_id, sort_order: binding.sort_order, status: binding.status === 'archived' ? 'published' : 'archived' })} className="rounded-lg border border-slate-300 px-2 py-1 text-xs text-slate-700" disabled={loading}>
            {binding.status === 'archived' ? t('modifiers.options.actions.activate') : t('catalog.shared.archive')}
          </button>
        </article>
      ))}
    </section>
  );
}

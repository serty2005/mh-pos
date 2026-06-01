import type { ModifierGroup, ModifierOption } from '../../shared/api/schemas';
import { useI18n } from '../../shared/i18n/I18nProvider';
import SafeErrorBanner from '../../shared/ui/SafeErrorBanner';
import { useState } from 'react';
import { defaultModifierOptionValues, type ModifierOptionFormValues } from './modifierForms';

type Props = {
  options: ModifierOption[];
  groups: ModifierGroup[];
  loading: boolean;
  error: unknown;
  onCreate: (values: ModifierOptionFormValues) => Promise<void>;
  onUpdate: (id: string, values: ModifierOptionFormValues) => Promise<void>;
};

export default function ModifierOptionsPanel({ options, groups, loading, error, onCreate, onUpdate }: Props) {
  const { t } = useI18n();
  const [values, setValues] = useState<ModifierOptionFormValues>(defaultModifierOptionValues);
  const groupName = (id: string) => groups.find((group) => group.id === id)?.name ?? id;

  return (
    <section className="space-y-4 rounded-2xl border border-slate-200 bg-white p-6">
      <h3 className="text-base font-semibold text-slate-900">{t('modifiers.options.title')}</h3>
      <form className="grid gap-3 md:grid-cols-[1fr_1fr_140px_auto]" onSubmit={(event) => {
        event.preventDefault();
        void onCreate({ ...values, name: values.name.trim() }).then(() => setValues(defaultModifierOptionValues));
      }}>
        <select value={values.modifier_group_id} onChange={(event) => setValues({ ...values, modifier_group_id: event.target.value })} className="rounded-lg border border-slate-300 px-3 py-2 text-sm" disabled={loading}>
          <option value="">{t('modifiers.options.fields.selectGroup')}</option>
          {groups.filter((group) => group.status !== 'archived').map((group) => <option key={group.id} value={group.id}>{group.name}</option>)}
        </select>
        <input value={values.name} onChange={(event) => setValues({ ...values, name: event.target.value })} className="rounded-lg border border-slate-300 px-3 py-2 text-sm" disabled={loading} placeholder={t('modifiers.options.fields.name')} />
        <input type="number" min="0" value={values.price_minor} onChange={(event) => setValues({ ...values, price_minor: Number(event.target.value) })} className="rounded-lg border border-slate-300 px-3 py-2 text-sm" disabled={loading} />
        <button type="submit" disabled={loading || !values.modifier_group_id || !values.name.trim()} className="rounded-lg bg-slate-900 px-3 py-2 text-sm font-medium text-white disabled:opacity-50">{t('modifiers.options.actions.create')}</button>
      </form>
      {error ? <SafeErrorBanner error={error} /> : null}
      {options.length === 0 ? <p className="text-sm text-slate-600">{t('modifiers.options.empty')}</p> : null}
      {options.map((option) => (
        <article key={option.id} className="flex flex-wrap items-center justify-between gap-2 rounded-xl border border-slate-200 p-4">
          <p className="text-sm text-slate-900">{option.name} · {groupName(option.modifier_group_id)} · {option.price_minor}</p>
          <button type="button" onClick={() => void onUpdate(option.id, { modifier_group_id: option.modifier_group_id, name: option.name, price_minor: option.price_minor, status: option.status === 'archived' ? 'published' : 'archived' })} className="rounded-lg border border-slate-300 px-2 py-1 text-xs text-slate-700" disabled={loading}>
            {option.status === 'archived' ? t('modifiers.options.actions.activate') : t('catalog.shared.archive')}
          </button>
        </article>
      ))}
    </section>
  );
}

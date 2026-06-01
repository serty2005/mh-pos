import { useEffect, useState } from 'react';
import type { ModifierGroup } from '../../shared/api/schemas';
import { useI18n } from '../../shared/i18n/I18nProvider';
import SafeErrorBanner from '../../shared/ui/SafeErrorBanner';
import { defaultModifierGroupValues, toModifierGroupValues, type ModifierGroupFormValues } from './modifierForms';

type Props = {
  groups: ModifierGroup[];
  loading: boolean;
  error: unknown;
  onCreate: (values: ModifierGroupFormValues) => Promise<void>;
  onUpdate: (id: string, values: ModifierGroupFormValues) => Promise<void>;
};

const statuses: ModifierGroupFormValues['status'][] = ['draft', 'published', 'archived'];

export default function ModifierGroupsPanel({ groups, loading, error, onCreate, onUpdate }: Props) {
  const { t } = useI18n();
  const [values, setValues] = useState<ModifierGroupFormValues>(defaultModifierGroupValues);
  const [editing, setEditing] = useState<ModifierGroup | null>(null);
  const [editValues, setEditValues] = useState<ModifierGroupFormValues>(defaultModifierGroupValues);

  useEffect(() => {
    if (editing) setEditValues(toModifierGroupValues(editing));
  }, [editing]);

  const renderForm = (formValues: ModifierGroupFormValues, setFormValues: (next: ModifierGroupFormValues) => void, onSubmit: () => Promise<void>, label: string) => (
    <form className="grid gap-3 md:grid-cols-5" onSubmit={(event) => { event.preventDefault(); void onSubmit(); }}>
      <input aria-label={t('modifiers.groups.fields.name')} value={formValues.name} onChange={(event) => setFormValues({ ...formValues, name: event.target.value })} className="rounded-lg border border-slate-300 px-3 py-2 text-sm" disabled={loading} placeholder={t('modifiers.groups.fields.name')} />
      <input type="number" min="0" aria-label={t('modifiers.groups.fields.minCount')} value={formValues.min_count} onChange={(event) => setFormValues({ ...formValues, min_count: Number(event.target.value) })} className="rounded-lg border border-slate-300 px-3 py-2 text-sm" disabled={loading} />
      <input type="number" min="0" aria-label={t('modifiers.groups.fields.maxCount')} value={formValues.max_count} onChange={(event) => setFormValues({ ...formValues, max_count: Number(event.target.value) })} className="rounded-lg border border-slate-300 px-3 py-2 text-sm" disabled={loading} />
      <select value={formValues.status} onChange={(event) => setFormValues({ ...formValues, status: event.target.value as ModifierGroupFormValues['status'] })} className="rounded-lg border border-slate-300 px-3 py-2 text-sm" disabled={loading}>
        {statuses.map((status) => <option key={status} value={status}>{t(`catalog.statuses.${status}`)}</option>)}
      </select>
      <button type="submit" disabled={loading || !formValues.name.trim()} className="rounded-lg bg-slate-900 px-3 py-2 text-sm font-medium text-white disabled:opacity-50">{label}</button>
      <label className="flex items-center gap-2 text-sm text-slate-700 md:col-span-5">
        <input type="checkbox" checked={formValues.required} onChange={(event) => setFormValues({ ...formValues, required: event.target.checked })} disabled={loading} />
        {t('modifiers.groups.fields.required')}
      </label>
    </form>
  );

  return (
    <section className="space-y-4 rounded-2xl border border-slate-200 bg-white p-6">
      <h3 className="text-base font-semibold text-slate-900">{t('modifiers.groups.title')}</h3>
      {renderForm(values, setValues, async () => {
        await onCreate({ ...values, name: values.name.trim() });
        setValues(defaultModifierGroupValues);
      }, t('modifiers.groups.actions.create'))}
      {error ? <SafeErrorBanner error={error} /> : null}
      {groups.length === 0 ? <p className="text-sm text-slate-600">{t('modifiers.groups.empty')}</p> : null}
      {groups.map((group) => (
        <article key={group.id} className="rounded-xl border border-slate-200 p-4">
          <div className="flex flex-wrap justify-between gap-2">
            <p className="text-sm font-medium text-slate-900">{group.name} · {t(`catalog.statuses.${group.status}`)}</p>
            <button type="button" onClick={() => setEditing(group)} className="rounded-lg border border-slate-300 px-2 py-1 text-xs text-slate-700">{t('catalog.shared.edit')}</button>
          </div>
          <p className="mt-1 text-xs text-slate-600">{t('modifiers.groups.fields.minCount')}: {group.min_count} · {t('modifiers.groups.fields.maxCount')}: {group.max_count}</p>
          {editing?.id === group.id ? (
            <div className="mt-3 space-y-2">
              {renderForm(editValues, setEditValues, async () => {
                await onUpdate(group.id, { ...editValues, name: editValues.name.trim() });
                setEditing(null);
              }, t('catalog.shared.save'))}
              <button type="button" onClick={() => setEditing(null)} className="rounded-lg border border-slate-300 px-3 py-2 text-sm text-slate-700">{t('catalog.shared.cancel')}</button>
            </div>
          ) : null}
        </article>
      ))}
    </section>
  );
}

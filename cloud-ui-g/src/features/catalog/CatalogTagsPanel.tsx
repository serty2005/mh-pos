import { useEffect, useState } from 'react';
import type { CatalogTag } from '../../shared/api/schemas';
import { useI18n } from '../../shared/i18n/I18nProvider';
import SafeErrorBanner from '../../shared/ui/SafeErrorBanner';
import {
  defaultCatalogTagValues,
  toCatalogTagValues,
  type CatalogTagFormValues,
  type LifecycleStatus,
} from './catalogForms';

type CatalogTagsPanelProps = {
  tags: CatalogTag[];
  loading: boolean;
  error: unknown;
  onCreate: (values: CatalogTagFormValues) => Promise<void>;
  onUpdate: (id: string, values: CatalogTagFormValues) => Promise<void>;
};

const statuses: LifecycleStatus[] = ['draft', 'published', 'archived'];

export default function CatalogTagsPanel({ tags, loading, error, onCreate, onUpdate }: CatalogTagsPanelProps) {
  const { t } = useI18n();
  const [createValues, setCreateValues] = useState<CatalogTagFormValues>(defaultCatalogTagValues);
  const [editing, setEditing] = useState<CatalogTag | null>(null);
  const [editValues, setEditValues] = useState<CatalogTagFormValues>(defaultCatalogTagValues);

  useEffect(() => {
    if (editing) setEditValues(toCatalogTagValues(editing));
  }, [editing]);

  const form = (
    mode: 'create' | 'edit',
    values: CatalogTagFormValues,
    setValues: (next: CatalogTagFormValues) => void,
    onSubmit: () => Promise<void>,
    onCancel?: () => void,
  ) => (
    <form className="space-y-3 rounded-xl border border-slate-200 bg-slate-50 p-4" onSubmit={(event) => {
      event.preventDefault();
      void onSubmit();
    }}>
      <div className="grid gap-3 md:grid-cols-2">
        <div>
          <label className="mb-1 block text-sm text-slate-700">{t('catalog.tags.fields.name')}</label>
          <input value={values.name} onChange={(event) => setValues({ ...values, name: event.target.value })} className="w-full rounded-lg border border-slate-300 bg-white px-3 py-2 text-sm" disabled={loading} />
        </div>
        <div>
          <label className="mb-1 block text-sm text-slate-700">{t('catalog.tags.fields.code')}</label>
          <input value={values.code} onChange={(event) => setValues({ ...values, code: event.target.value })} className="w-full rounded-lg border border-slate-300 bg-white px-3 py-2 text-sm" disabled={loading} />
        </div>
      </div>

      <div>
        <label className="mb-1 block text-sm text-slate-700">{t('catalog.tags.fields.status')}</label>
        <select value={values.status} onChange={(event) => setValues({ ...values, status: event.target.value as LifecycleStatus })} className="w-full rounded-lg border border-slate-300 bg-white px-3 py-2 text-sm" disabled={loading}>
          {statuses.map((status) => <option key={status} value={status}>{t(`catalog.statuses.${status}`)}</option>)}
        </select>
      </div>

      <div className="flex flex-wrap gap-2">
        <button type="submit" disabled={loading || !values.name.trim() || !values.code.trim()} className="rounded-lg bg-slate-900 px-3 py-2 text-sm font-medium text-white disabled:opacity-50">{mode === 'create' ? t('catalog.tags.actions.create') : t('catalog.shared.save')}</button>
        {onCancel ? <button type="button" onClick={onCancel} className="rounded-lg border border-slate-300 px-3 py-2 text-sm font-medium text-slate-700" disabled={loading}>{t('catalog.shared.cancel')}</button> : null}
      </div>
    </form>
  );

  return (
    <section className="space-y-4 rounded-2xl border border-slate-200 bg-white p-6">
      <h3 className="text-base font-semibold text-slate-900">{t('catalog.tags.title')}</h3>
      {form('create', createValues, setCreateValues, async () => {
        await onCreate({ ...createValues, name: createValues.name.trim(), code: createValues.code.trim() });
        setCreateValues(defaultCatalogTagValues);
      })}
      {error ? <SafeErrorBanner error={error} /> : null}

      <div className="space-y-3">
        <h4 className="text-sm font-semibold text-slate-900">{t('catalog.tags.listTitle')}</h4>
        {tags.length === 0 ? <p className="text-sm text-slate-600">{t('catalog.tags.empty')}</p> : null}
        {tags.map((tag) => (
          <article key={tag.id} className="rounded-xl border border-slate-200 p-4">
            <div className="flex items-start justify-between gap-2">
              <div>
                <p className="text-sm font-medium text-slate-900">{tag.name}</p>
                <p className="text-xs text-slate-600">{tag.code} · {t(`catalog.statuses.${tag.status}`)}</p>
              </div>
              <button type="button" onClick={() => setEditing(tag)} className="rounded-lg border border-slate-300 px-2 py-1 text-xs text-slate-700" disabled={loading}>{t('catalog.shared.edit')}</button>
            </div>
            {editing?.id === tag.id ? (
              <div className="mt-3">
                {form('edit', editValues, setEditValues, async () => {
                  await onUpdate(tag.id, { ...editValues, name: editValues.name.trim(), code: editValues.code.trim() });
                  setEditing(null);
                }, () => setEditing(null))}
              </div>
            ) : null}
          </article>
        ))}
      </div>
    </section>
  );
}

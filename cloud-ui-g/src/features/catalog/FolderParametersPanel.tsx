import { useEffect, useState } from 'react';
import type { CatalogFolder, FolderParameter } from '../../shared/api/schemas';
import { useI18n } from '../../shared/i18n/I18nProvider';
import SafeErrorBanner from '../../shared/ui/SafeErrorBanner';
import {
  defaultFolderParameterValues,
  toFolderParameterValues,
  type FolderParameterFormValues,
  type LifecycleStatus,
} from './catalogForms';

type FolderParametersPanelProps = {
  parameters: FolderParameter[];
  folders: CatalogFolder[];
  loading: boolean;
  error: unknown;
  onCreate: (values: FolderParameterFormValues) => Promise<void>;
  onUpdate: (id: string, values: FolderParameterFormValues) => Promise<void>;
};

const statuses: LifecycleStatus[] = ['draft', 'published', 'archived'];

export default function FolderParametersPanel({ parameters, folders, loading, error, onCreate, onUpdate }: FolderParametersPanelProps) {
  const { t } = useI18n();
  const [createValues, setCreateValues] = useState<FolderParameterFormValues>(defaultFolderParameterValues);
  const [editing, setEditing] = useState<FolderParameter | null>(null);
  const [editValues, setEditValues] = useState<FolderParameterFormValues>(defaultFolderParameterValues);

  useEffect(() => {
    if (editing) setEditValues(toFolderParameterValues(editing));
  }, [editing]);

  const folderOptions = folders.map((folder) => ({ value: folder.id, label: folder.name }));

  const form = (
    mode: 'create' | 'edit',
    values: FolderParameterFormValues,
    setValues: (next: FolderParameterFormValues) => void,
    onSubmit: () => Promise<void>,
    onCancel?: () => void,
  ) => (
    <form className="space-y-3 rounded-xl border border-slate-200 bg-slate-50 p-4" onSubmit={(event) => {
      event.preventDefault();
      void onSubmit();
    }}>
      <div className="grid gap-3 md:grid-cols-2">
        <div>
          <label className="mb-1 block text-sm text-slate-700">{t('catalog.parameters.fields.folder')}</label>
          <select value={values.folder_id} onChange={(event) => setValues({ ...values, folder_id: event.target.value })} className="w-full rounded-lg border border-slate-300 bg-white px-3 py-2 text-sm" disabled={loading || mode === 'edit'}>
            <option value="">{t('catalog.parameters.fields.selectFolder')}</option>
            {folderOptions.map((option) => <option key={option.value} value={option.value}>{option.label}</option>)}
          </select>
        </div>
        <div>
          <label className="mb-1 block text-sm text-slate-700">{t('catalog.parameters.fields.key')}</label>
          <input value={values.parameter_key} onChange={(event) => setValues({ ...values, parameter_key: event.target.value })} className="w-full rounded-lg border border-slate-300 bg-white px-3 py-2 text-sm" disabled={loading || mode === 'edit'} />
        </div>
      </div>

      <div className="grid gap-3 md:grid-cols-2">
        <div>
          <label className="mb-1 block text-sm text-slate-700">{t('catalog.parameters.fields.valueType')}</label>
          <input value={values.value_type} onChange={(event) => setValues({ ...values, value_type: event.target.value })} className="w-full rounded-lg border border-slate-300 bg-white px-3 py-2 text-sm" disabled={loading} />
        </div>
        <div>
          <label className="mb-1 block text-sm text-slate-700">{t('catalog.parameters.fields.status')}</label>
          <select value={values.status} onChange={(event) => setValues({ ...values, status: event.target.value as LifecycleStatus })} className="w-full rounded-lg border border-slate-300 bg-white px-3 py-2 text-sm" disabled={loading}>
            {statuses.map((status) => <option key={status} value={status}>{t(`catalog.statuses.${status}`)}</option>)}
          </select>
        </div>
      </div>

      <div>
        <label className="mb-1 block text-sm text-slate-700">{t('catalog.parameters.fields.valueJson')}</label>
        <textarea value={values.value_json} onChange={(event) => setValues({ ...values, value_json: event.target.value })} rows={3} className="w-full rounded-lg border border-slate-300 bg-white px-3 py-2 text-sm" disabled={loading} />
      </div>

      <div className="flex flex-wrap gap-2">
        <button type="submit" disabled={loading || !values.folder_id || !values.parameter_key.trim() || !values.value_type.trim() || !values.value_json.trim()} className="rounded-lg bg-slate-900 px-3 py-2 text-sm font-medium text-white disabled:opacity-50">{mode === 'create' ? t('catalog.parameters.actions.create') : t('catalog.shared.save')}</button>
        {onCancel ? <button type="button" onClick={onCancel} className="rounded-lg border border-slate-300 px-3 py-2 text-sm font-medium text-slate-700" disabled={loading}>{t('catalog.shared.cancel')}</button> : null}
      </div>
    </form>
  );

  return (
    <section className="space-y-4 rounded-2xl border border-slate-200 bg-white p-6">
      <h3 className="text-base font-semibold text-slate-900">{t('catalog.parameters.title')}</h3>
      {form('create', createValues, setCreateValues, async () => {
        await onCreate({ ...createValues, parameter_key: createValues.parameter_key.trim(), value_type: createValues.value_type.trim(), value_json: createValues.value_json.trim() });
        setCreateValues(defaultFolderParameterValues);
      })}
      {error ? <SafeErrorBanner error={error} /> : null}

      <div className="space-y-3">
        <h4 className="text-sm font-semibold text-slate-900">{t('catalog.parameters.listTitle')}</h4>
        {parameters.length === 0 ? <p className="text-sm text-slate-600">{t('catalog.parameters.empty')}</p> : null}
        {parameters.map((parameter) => (
          <article key={parameter.id} className="rounded-xl border border-slate-200 p-4">
            <div className="flex items-start justify-between gap-2">
              <div>
                <p className="text-sm font-medium text-slate-900">{parameter.parameter_key}</p>
                <p className="text-xs text-slate-600">{parameter.value_type} · {t(`catalog.statuses.${parameter.status}`)}</p>
              </div>
              <button type="button" onClick={() => setEditing(parameter)} className="rounded-lg border border-slate-300 px-2 py-1 text-xs text-slate-700" disabled={loading}>{t('catalog.shared.edit')}</button>
            </div>
            <pre className="mt-2 overflow-x-auto rounded-lg bg-slate-100 p-2 text-xs text-slate-700">{parameter.value_json}</pre>
            {editing?.id === parameter.id ? (
              <div className="mt-3">
                {form('edit', editValues, setEditValues, async () => {
                  await onUpdate(parameter.id, { ...editValues, parameter_key: editValues.parameter_key.trim(), value_type: editValues.value_type.trim(), value_json: editValues.value_json.trim() });
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

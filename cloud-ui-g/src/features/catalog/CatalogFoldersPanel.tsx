import { useEffect, useState } from 'react';
import type { CatalogFolder } from '../../shared/api/schemas';
import { useI18n } from '../../shared/i18n/I18nProvider';
import SafeErrorBanner from '../../shared/ui/SafeErrorBanner';
import {
  defaultCatalogFolderValues,
  toCatalogFolderValues,
  type CatalogFolderFormValues,
  type LifecycleStatus,
} from './catalogForms';

type CatalogFoldersPanelProps = {
  folders: CatalogFolder[];
  loading: boolean;
  error: unknown;
  onCreate: (values: CatalogFolderFormValues) => Promise<void>;
  onUpdate: (id: string, values: CatalogFolderFormValues) => Promise<void>;
  onArchive: (id: string) => Promise<void>;
};

const statuses: LifecycleStatus[] = ['draft', 'published', 'archived'];

export default function CatalogFoldersPanel({ folders, loading, error, onCreate, onUpdate, onArchive }: CatalogFoldersPanelProps) {
  const { t } = useI18n();
  const [createValues, setCreateValues] = useState<CatalogFolderFormValues>(defaultCatalogFolderValues);
  const [editing, setEditing] = useState<CatalogFolder | null>(null);
  const [editValues, setEditValues] = useState<CatalogFolderFormValues>(defaultCatalogFolderValues);

  useEffect(() => {
    if (editing) setEditValues(toCatalogFolderValues(editing));
  }, [editing]);

  const folderOptions = folders.map((folder) => ({ value: folder.id, label: folder.name }));

  const renderForm = (
    mode: 'create' | 'edit',
    values: CatalogFolderFormValues,
    setValues: (next: CatalogFolderFormValues) => void,
    onSubmit: () => Promise<void>,
    onCancel?: () => void,
  ) => (
    <form
      className="space-y-3 rounded-xl border border-slate-200 bg-slate-50 p-4"
      onSubmit={(event) => {
        event.preventDefault();
        void onSubmit();
      }}
    >
      <div className="grid gap-3 md:grid-cols-2">
        <div>
          <label className="mb-1 block text-sm text-slate-700">{t('catalog.folders.fields.name')}</label>
          <input value={values.name} onChange={(event) => setValues({ ...values, name: event.target.value })} className="w-full rounded-lg border border-slate-300 bg-white px-3 py-2 text-sm" disabled={loading} />
        </div>
        <div>
          <label className="mb-1 block text-sm text-slate-700">{t('catalog.folders.fields.parent')}</label>
          <select value={values.parent_id} onChange={(event) => setValues({ ...values, parent_id: event.target.value })} className="w-full rounded-lg border border-slate-300 bg-white px-3 py-2 text-sm" disabled={loading}>
            <option value="">{t('catalog.folders.fields.root')}</option>
            {folderOptions.map((option) => <option key={option.value} value={option.value}>{option.label}</option>)}
          </select>
        </div>
      </div>

      <div className="grid gap-3 md:grid-cols-2">
        <div>
          <label className="mb-1 block text-sm text-slate-700">{t('catalog.folders.fields.sortOrder')}</label>
          <input type="number" value={values.sort_order} onChange={(event) => setValues({ ...values, sort_order: Number(event.target.value) || 0 })} className="w-full rounded-lg border border-slate-300 bg-white px-3 py-2 text-sm" disabled={loading} />
        </div>
        <div>
          <label className="mb-1 block text-sm text-slate-700">{t('catalog.folders.fields.status')}</label>
          <select value={values.status} onChange={(event) => setValues({ ...values, status: event.target.value as LifecycleStatus })} className="w-full rounded-lg border border-slate-300 bg-white px-3 py-2 text-sm" disabled={loading}>
            {statuses.map((status) => <option key={status} value={status}>{t(`catalog.statuses.${status}`)}</option>)}
          </select>
        </div>
      </div>

      <div className="flex flex-wrap gap-2">
        <button type="submit" disabled={loading || !values.name.trim()} className="rounded-lg bg-slate-900 px-3 py-2 text-sm font-medium text-white disabled:opacity-50">{mode === 'create' ? t('catalog.folders.actions.create') : t('catalog.shared.save')}</button>
        {onCancel ? <button type="button" onClick={onCancel} className="rounded-lg border border-slate-300 px-3 py-2 text-sm font-medium text-slate-700" disabled={loading}>{t('catalog.shared.cancel')}</button> : null}
      </div>
    </form>
  );

  return (
    <section className="space-y-4 rounded-2xl border border-slate-200 bg-white p-6">
      <h3 className="text-base font-semibold text-slate-900">{t('catalog.folders.title')}</h3>
      {renderForm('create', createValues, setCreateValues, async () => {
        await onCreate({ ...createValues, name: createValues.name.trim() });
        setCreateValues(defaultCatalogFolderValues);
      })}
      {error ? <SafeErrorBanner error={error} /> : null}

      <div className="space-y-3">
        <h4 className="text-sm font-semibold text-slate-900">{t('catalog.folders.listTitle')}</h4>
        {folders.length === 0 ? <p className="text-sm text-slate-600">{t('catalog.folders.empty')}</p> : null}
        {folders.map((folder) => (
          <article key={folder.id} className="rounded-xl border border-slate-200 p-4">
            <div className="flex flex-wrap items-start justify-between gap-2">
              <div>
                <p className="text-sm font-medium text-slate-900">{folder.name}</p>
                <p className="text-xs text-slate-600">{t(`catalog.statuses.${folder.status}`)} · sort {folder.sort_order}</p>
              </div>
              <div className="flex gap-2">
                <button type="button" onClick={() => setEditing(folder)} className="rounded-lg border border-slate-300 px-2 py-1 text-xs text-slate-700" disabled={loading}>{t('catalog.shared.edit')}</button>
                <button
                  type="button"
                  onClick={() => {
                    if (!window.confirm(t('catalog.shared.archiveConfirm'))) return;
                    void onArchive(folder.id);
                  }}
                  className="rounded-lg border border-rose-300 px-2 py-1 text-xs text-rose-700"
                  disabled={loading || folder.status === 'archived'}
                >
                  {t('catalog.shared.archive')}
                </button>
              </div>
            </div>
            {editing?.id === folder.id ? (
              <div className="mt-3">
                {renderForm('edit', editValues, setEditValues, async () => {
                  await onUpdate(folder.id, { ...editValues, name: editValues.name.trim() });
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

import { useEffect, useState } from 'react';
import type { CatalogFolder, CatalogItem } from '../../shared/api/schemas';
import { useI18n } from '../../shared/i18n/I18nProvider';
import SafeErrorBanner from '../../shared/ui/SafeErrorBanner';
import {
  defaultCatalogItemValues,
  toCatalogItemValues,
  type CatalogItemFormValues,
  type CatalogKind,
  type LifecycleStatus,
} from './catalogForms';

type CatalogItemsPanelProps = {
  items: CatalogItem[];
  folders: CatalogFolder[];
  loading: boolean;
  error: unknown;
  onCreate: (values: CatalogItemFormValues) => Promise<void>;
  onUpdate: (id: string, values: CatalogItemFormValues) => Promise<void>;
  onArchive: (id: string) => Promise<void>;
};

const kinds: CatalogKind[] = ['dish', 'good', 'semi_finished', 'service'];
const statuses: LifecycleStatus[] = ['draft', 'published', 'archived'];

export default function CatalogItemsPanel({ items, folders, loading, error, onCreate, onUpdate, onArchive }: CatalogItemsPanelProps) {
  const { t } = useI18n();
  const [createValues, setCreateValues] = useState<CatalogItemFormValues>(defaultCatalogItemValues);
  const [editing, setEditing] = useState<CatalogItem | null>(null);
  const [editValues, setEditValues] = useState<CatalogItemFormValues>(defaultCatalogItemValues);

  useEffect(() => {
    if (editing) setEditValues(toCatalogItemValues(editing));
  }, [editing]);

  const folderLabel = (folderId: string) => folders.find((folder) => folder.id === folderId)?.name ?? folderId;

  const renderForm = (
    mode: 'create' | 'edit',
    values: CatalogItemFormValues,
    setValues: (next: CatalogItemFormValues) => void,
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
          <label className="mb-1 block text-sm text-slate-700">{t('catalog.items.fields.name')}</label>
          <input value={values.name} onChange={(event) => setValues({ ...values, name: event.target.value })} className="w-full rounded-lg border border-slate-300 bg-white px-3 py-2 text-sm" disabled={loading} />
        </div>
        <div>
          <label className="mb-1 block text-sm text-slate-700">{t('catalog.items.fields.sku')}</label>
          <input value={values.sku} onChange={(event) => setValues({ ...values, sku: event.target.value })} className="w-full rounded-lg border border-slate-300 bg-white px-3 py-2 text-sm" disabled={loading} />
        </div>
      </div>

      <div className="grid gap-3 md:grid-cols-2 lg:grid-cols-4">
        <div>
          <label className="mb-1 block text-sm text-slate-700">{t('catalog.items.fields.kind')}</label>
          <select value={values.kind} onChange={(event) => setValues({ ...values, kind: event.target.value as CatalogKind })} className="w-full rounded-lg border border-slate-300 bg-white px-3 py-2 text-sm" disabled={loading}>
            {kinds.map((kind) => <option key={kind} value={kind}>{t(`catalog.kinds.${kind}`)}</option>)}
          </select>
        </div>
        <div>
          <label className="mb-1 block text-sm text-slate-700">{t('catalog.items.fields.status')}</label>
          <select value={values.status} onChange={(event) => setValues({ ...values, status: event.target.value as LifecycleStatus })} className="w-full rounded-lg border border-slate-300 bg-white px-3 py-2 text-sm" disabled={loading}>
            {statuses.map((status) => <option key={status} value={status}>{t(`catalog.statuses.${status}`)}</option>)}
          </select>
        </div>
        <div>
          <label className="mb-1 block text-sm text-slate-700">{t('catalog.items.fields.baseUnit')}</label>
          <input value={values.base_unit} onChange={(event) => setValues({ ...values, base_unit: event.target.value })} className="w-full rounded-lg border border-slate-300 bg-white px-3 py-2 text-sm" disabled={loading} />
        </div>
        <div>
          <label className="mb-1 block text-sm text-slate-700">{t('catalog.items.fields.folder')}</label>
          <select value={values.folder_id} onChange={(event) => setValues({ ...values, folder_id: event.target.value })} className="w-full rounded-lg border border-slate-300 bg-white px-3 py-2 text-sm" disabled={loading}>
            <option value="">{t('catalog.shared.noFolder')}</option>
            {folders.map((folder) => <option key={folder.id} value={folder.id}>{folder.name}</option>)}
          </select>
        </div>
      </div>

      <div className="grid gap-3 md:grid-cols-2">
        <div>
          <label className="mb-1 block text-sm text-slate-700">{t('catalog.items.fields.kitchenType')}</label>
          <input value={values.kitchen_type} onChange={(event) => setValues({ ...values, kitchen_type: event.target.value })} className="w-full rounded-lg border border-slate-300 bg-white px-3 py-2 text-sm" disabled={loading} />
        </div>
        <div>
          <label className="mb-1 block text-sm text-slate-700">{t('catalog.items.fields.accountingCategory')}</label>
          <input value={values.accounting_category} onChange={(event) => setValues({ ...values, accounting_category: event.target.value })} className="w-full rounded-lg border border-slate-300 bg-white px-3 py-2 text-sm" disabled={loading} />
        </div>
      </div>

      <div className="flex flex-wrap gap-2">
        <button type="submit" disabled={loading || !values.name.trim() || !values.sku.trim() || !values.base_unit.trim()} className="rounded-lg bg-slate-900 px-3 py-2 text-sm font-medium text-white disabled:opacity-50">{mode === 'create' ? t('catalog.items.actions.create') : t('catalog.shared.save')}</button>
        {onCancel ? <button type="button" onClick={onCancel} className="rounded-lg border border-slate-300 px-3 py-2 text-sm font-medium text-slate-700" disabled={loading}>{t('catalog.shared.cancel')}</button> : null}
      </div>
    </form>
  );

  return (
    <section className="space-y-4 rounded-2xl border border-slate-200 bg-white p-6">
      <h3 className="text-base font-semibold text-slate-900">{t('catalog.items.title')}</h3>
      {renderForm('create', createValues, setCreateValues, async () => {
        await onCreate({ ...createValues, name: createValues.name.trim(), sku: createValues.sku.trim(), base_unit: createValues.base_unit.trim() });
        setCreateValues(defaultCatalogItemValues);
      })}
      {error ? <SafeErrorBanner error={error} /> : null}

      <div className="space-y-3">
        <h4 className="text-sm font-semibold text-slate-900">{t('catalog.items.listTitle')}</h4>
        {items.length === 0 ? <p className="text-sm text-slate-600">{t('catalog.items.empty')}</p> : null}
        {items.map((item) => (
          <article key={item.id} className="rounded-xl border border-slate-200 p-4">
            <div className="flex flex-wrap items-start justify-between gap-2">
              <div>
                <p className="text-sm font-medium text-slate-900">{item.name}</p>
                <p className="text-xs text-slate-600">{item.sku} · {t(`catalog.kinds.${item.kind}`)} · {t(`catalog.statuses.${item.status}`)}</p>
                <p className="text-xs text-slate-500">{t('catalog.items.fields.folder')}: {item.folder_id ? folderLabel(item.folder_id) : t('catalog.shared.noFolder')}</p>
              </div>
              <div className="flex gap-2">
                <button type="button" onClick={() => setEditing(item)} className="rounded-lg border border-slate-300 px-2 py-1 text-xs text-slate-700" disabled={loading}>{t('catalog.shared.edit')}</button>
                <button
                  type="button"
                  onClick={() => {
                    if (!window.confirm(t('catalog.shared.archiveConfirm'))) return;
                    void onArchive(item.id);
                  }}
                  className="rounded-lg border border-rose-300 px-2 py-1 text-xs text-rose-700"
                  disabled={loading || item.status === 'archived'}
                >
                  {t('catalog.shared.archive')}
                </button>
              </div>
            </div>
            {editing?.id === item.id ? (
              <div className="mt-3">
                {renderForm('edit', editValues, setEditValues, async () => {
                  await onUpdate(item.id, { ...editValues, name: editValues.name.trim(), sku: editValues.sku.trim(), base_unit: editValues.base_unit.trim() });
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

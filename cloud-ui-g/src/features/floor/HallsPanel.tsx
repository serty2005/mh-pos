import { useEffect, useState } from 'react';
import type { Hall } from '../../shared/api/schemas';
import { useI18n } from '../../shared/i18n/I18nProvider';
import SafeErrorBanner from '../../shared/ui/SafeErrorBanner';
import {
  defaultHallUpdateValues,
  defaultHallValues,
  toHallValues,
  type HallFormValues,
  type HallUpdateFormValues,
} from './floorForms';

type Props = {
  halls: Hall[];
  loading: boolean;
  error: unknown;
  onCreate: (values: HallFormValues) => Promise<void>;
  onUpdate: (id: string, values: HallUpdateFormValues) => Promise<void>;
  onArchive: (id: string) => Promise<void>;
};

const statuses: HallUpdateFormValues['status'][] = ['draft', 'published', 'archived'];

export default function HallsPanel({ halls, loading, error, onCreate, onUpdate, onArchive }: Props) {
  const { t } = useI18n();
  const [createValues, setCreateValues] = useState<HallFormValues>(defaultHallValues);
  const [editing, setEditing] = useState<Hall | null>(null);
  const [editValues, setEditValues] = useState<HallUpdateFormValues>(defaultHallUpdateValues);

  useEffect(() => {
    if (editing) setEditValues(toHallValues(editing));
  }, [editing]);

  const renderCreateForm = () => (
    <form className="space-y-3 rounded-xl border border-slate-200 bg-slate-50 p-4" onSubmit={(event) => { event.preventDefault(); void onCreate(createValues).then(() => setCreateValues(defaultHallValues)); }}>
      <div>
        <label className="mb-1 block text-sm text-slate-700">{t('floor.halls.fields.name')}</label>
        <input value={createValues.name} onChange={(event) => setCreateValues({ name: event.target.value })} className="w-full rounded-lg border border-slate-300 bg-white px-3 py-2 text-sm" disabled={loading} />
      </div>
      <button type="submit" disabled={loading || !createValues.name.trim()} className="rounded-lg bg-slate-900 px-3 py-2 text-sm font-medium text-white disabled:opacity-50">{t('floor.halls.actions.create')}</button>
    </form>
  );

  const renderEditForm = (hall: Hall) => (
    <form className="mt-3 space-y-3 rounded-xl border border-slate-200 bg-slate-50 p-4" onSubmit={(event) => { event.preventDefault(); void onUpdate(hall.id, editValues).then(() => setEditing(null)); }}>
      <div className="grid gap-3 md:grid-cols-2">
        <div>
          <label className="mb-1 block text-sm text-slate-700">{t('floor.halls.fields.name')}</label>
          <input value={editValues.name} onChange={(event) => setEditValues({ ...editValues, name: event.target.value })} className="w-full rounded-lg border border-slate-300 bg-white px-3 py-2 text-sm" disabled={loading} />
        </div>
        <div>
          <label className="mb-1 block text-sm text-slate-700">{t('floor.shared.status')}</label>
          <select value={editValues.status} onChange={(event) => setEditValues({ ...editValues, status: event.target.value as HallUpdateFormValues['status'] })} className="w-full rounded-lg border border-slate-300 bg-white px-3 py-2 text-sm" disabled={loading}>
            {statuses.map((status) => <option key={status} value={status}>{t(`catalog.statuses.${status}`)}</option>)}
          </select>
        </div>
      </div>
      <div className="flex flex-wrap gap-2">
        <button type="submit" disabled={loading || !editValues.name.trim()} className="rounded-lg bg-slate-900 px-3 py-2 text-sm font-medium text-white disabled:opacity-50">{t('catalog.shared.save')}</button>
        <button type="button" onClick={() => setEditing(null)} className="rounded-lg border border-slate-300 px-3 py-2 text-sm text-slate-700" disabled={loading}>{t('catalog.shared.cancel')}</button>
      </div>
    </form>
  );

  return (
    <section className="space-y-4 rounded-2xl border border-slate-200 bg-white p-6">
      <div>
        <h3 className="text-base font-semibold text-slate-900">{t('floor.halls.title')}</h3>
        <p className="mt-1 text-sm text-slate-600">{t('floor.halls.description')}</p>
      </div>
      {renderCreateForm()}
      {error ? <SafeErrorBanner error={error} /> : null}
      {halls.length === 0 ? <p className="text-sm text-slate-600">{t('floor.halls.empty')}</p> : null}
      {halls.map((hall) => (
        <article key={hall.id} className="rounded-xl border border-slate-200 p-4">
          <div className="flex flex-wrap items-start justify-between gap-2">
            <div>
              <p className="text-sm font-medium text-slate-900">{hall.name}</p>
              <p className="text-xs text-slate-600">{t('floor.shared.status')}: {t(`catalog.statuses.${hall.status}`)}</p>
            </div>
            <div className="flex gap-2">
              <button type="button" onClick={() => setEditing(hall)} className="rounded-lg border border-slate-300 px-2 py-1 text-xs text-slate-700" disabled={loading}>{t('catalog.shared.edit')}</button>
              <button type="button" onClick={() => { if (window.confirm(t('floor.shared.archiveConfirm'))) void onArchive(hall.id); }} className="rounded-lg border border-rose-300 px-2 py-1 text-xs text-rose-700" disabled={loading || hall.status === 'archived'}>{t('catalog.shared.archive')}</button>
            </div>
          </div>
          {editing?.id === hall.id ? renderEditForm(hall) : null}
        </article>
      ))}
    </section>
  );
}

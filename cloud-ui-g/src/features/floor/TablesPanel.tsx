import { useEffect, useState } from 'react';
import type { Hall, RestaurantSection, RestaurantTable } from '../../shared/api/schemas';
import { useI18n } from '../../shared/i18n/I18nProvider';
import SafeErrorBanner from '../../shared/ui/SafeErrorBanner';
import { defaultTableValues, toTableValues, type TableFormValues } from './floorForms';

type Props = {
  halls: Hall[];
  sections: RestaurantSection[];
  tables: RestaurantTable[];
  loading: boolean;
  error: unknown;
  onCreate: (values: TableFormValues) => Promise<void>;
  onUpdate: (id: string, values: TableFormValues) => Promise<void>;
  onArchive: (id: string) => Promise<void>;
};

const statuses: TableFormValues['status'][] = ['draft', 'published', 'archived'];

export default function TablesPanel({ halls, sections, tables, loading, error, onCreate, onUpdate, onArchive }: Props) {
  const { t } = useI18n();
  const activeHalls = halls.filter((hall) => hall.status !== 'archived');
  const firstHallId = activeHalls[0]?.id ?? '';
  const activeSections = sections.filter((section) => section.is_active && section.mode === 'hall_section');
  const sectionOptionsForHall = (hallId: string) => activeSections.filter((section) => !section.hall_id || !hallId || section.hall_id === hallId);
  const firstSectionIdForHall = (hallId: string) => sectionOptionsForHall(hallId)[0]?.id ?? '';
  const [createValues, setCreateValues] = useState<TableFormValues>({
    ...defaultTableValues,
    hall_id: firstHallId,
    section_id: firstSectionIdForHall(firstHallId),
  });
  const [editing, setEditing] = useState<RestaurantTable | null>(null);
  const [editValues, setEditValues] = useState<TableFormValues>(defaultTableValues);

  useEffect(() => {
    setCreateValues((prev) => {
      const hallId = prev.hall_id || firstHallId;
      const options = sectionOptionsForHall(hallId);
      const sectionStillValid = options.some((section) => section.id === prev.section_id);
      return {
        ...prev,
        hall_id: hallId,
        section_id: sectionStillValid ? prev.section_id : options[0]?.id ?? '',
      };
    });
  }, [firstHallId, sections]);

  useEffect(() => {
    if (editing) setEditValues(toTableValues(editing));
  }, [editing]);

  const hallLabel = (hallId: string) => halls.find((hall) => hall.id === hallId)?.name ?? t('floor.tables.noHall');
  const sectionLabel = (sectionId: string) => sections.find((section) => section.id === sectionId)?.name ?? sectionId;

  const renderForm = (
    values: TableFormValues,
    setValues: (next: TableFormValues) => void,
    onSubmit: () => Promise<void>,
    actionLabel: string,
    includeStatus: boolean,
  ) => {
    const sectionOptions = sectionOptionsForHall(values.hall_id);
    return (
    <form className="space-y-3 rounded-xl border border-slate-200 bg-slate-50 p-4" onSubmit={(event) => { event.preventDefault(); void onSubmit(); }}>
      <div className="grid gap-3 md:grid-cols-2 lg:grid-cols-4">
        <div>
          <label className="mb-1 block text-sm text-slate-700">{t('floor.tables.fields.hall')}</label>
          <select
            value={values.hall_id}
            onChange={(event) => {
              const hallId = event.target.value;
              setValues({ ...values, hall_id: hallId, section_id: firstSectionIdForHall(hallId) });
            }}
            className="w-full rounded-lg border border-slate-300 bg-white px-3 py-2 text-sm"
            disabled={loading}
          >
            <option value="">{t('floor.tables.fields.selectHall')}</option>
            {activeHalls.map((hall) => <option key={hall.id} value={hall.id}>{hall.name}</option>)}
          </select>
        </div>
        <div>
          <label className="mb-1 block text-sm text-slate-700">{t('floor.tables.fields.section')}</label>
          <select value={values.section_id} onChange={(event) => setValues({ ...values, section_id: event.target.value })} className="w-full rounded-lg border border-slate-300 bg-white px-3 py-2 text-sm" disabled={loading || sectionOptions.length === 0}>
            <option value="">{t('floor.tables.fields.selectSection')}</option>
            {sectionOptions.map((section) => <option key={section.id} value={section.id}>{section.name}</option>)}
          </select>
        </div>
        <div>
          <label className="mb-1 block text-sm text-slate-700">{t('floor.tables.fields.name')}</label>
          <input value={values.name} onChange={(event) => setValues({ ...values, name: event.target.value })} className="w-full rounded-lg border border-slate-300 bg-white px-3 py-2 text-sm" disabled={loading} />
        </div>
        <div>
          <label className="mb-1 block text-sm text-slate-700">{t('floor.tables.fields.seats')}</label>
          <input type="number" min="1" value={values.seats} onChange={(event) => setValues({ ...values, seats: Number(event.target.value) })} className="w-full rounded-lg border border-slate-300 bg-white px-3 py-2 text-sm" disabled={loading} />
        </div>
        {includeStatus ? (
          <div>
            <label className="mb-1 block text-sm text-slate-700">{t('floor.shared.status')}</label>
            <select value={values.status} onChange={(event) => setValues({ ...values, status: event.target.value as TableFormValues['status'] })} className="w-full rounded-lg border border-slate-300 bg-white px-3 py-2 text-sm" disabled={loading}>
              {statuses.map((status) => <option key={status} value={status}>{t(`catalog.statuses.${status}`)}</option>)}
            </select>
          </div>
        ) : null}
      </div>
      <button type="submit" disabled={loading || !values.hall_id || !values.section_id || !values.name.trim()} className="rounded-lg bg-slate-900 px-3 py-2 text-sm font-medium text-white disabled:opacity-50">{actionLabel}</button>
    </form>
    );
  };

  return (
    <section className="space-y-4 rounded-2xl border border-slate-200 bg-white p-6">
      <div>
        <h3 className="text-base font-semibold text-slate-900">{t('floor.tables.title')}</h3>
        <p className="mt-1 text-sm text-slate-600">{t('floor.tables.description')}</p>
      </div>
      {activeHalls.length === 0 ? <p className="text-sm text-slate-600">{t('floor.tables.noHalls')}</p> : null}
      {activeSections.length === 0 ? <p className="text-sm text-slate-600">{t('floor.tables.noSections')}</p> : null}
      {renderForm(createValues, setCreateValues, async () => {
        await onCreate(createValues);
        setCreateValues({ ...defaultTableValues, hall_id: firstHallId, section_id: firstSectionIdForHall(firstHallId) });
      }, t('floor.tables.actions.create'), false)}
      {error ? <SafeErrorBanner error={error} /> : null}
      {tables.length === 0 ? <p className="text-sm text-slate-600">{t('floor.tables.empty')}</p> : null}
      {tables.map((table) => (
        <article key={table.id} className="rounded-xl border border-slate-200 p-4">
          <div className="flex flex-wrap items-start justify-between gap-2">
            <div>
              <p className="text-sm font-medium text-slate-900">{table.name}</p>
              <p className="text-xs text-slate-600">{hallLabel(table.hall_id)} · {sectionLabel(table.section_id)} · {table.seats} · {t(`catalog.statuses.${table.status}`)}</p>
            </div>
            <div className="flex gap-2">
              <button type="button" onClick={() => setEditing(table)} className="rounded-lg border border-slate-300 px-2 py-1 text-xs text-slate-700" disabled={loading}>{t('catalog.shared.edit')}</button>
              <button type="button" onClick={() => { if (window.confirm(t('floor.shared.archiveConfirm'))) void onArchive(table.id); }} className="rounded-lg border border-rose-300 px-2 py-1 text-xs text-rose-700" disabled={loading || table.status === 'archived'}>{t('catalog.shared.archive')}</button>
            </div>
          </div>
          {editing?.id === table.id ? (
            <div className="mt-3">
              {renderForm(editValues, setEditValues, async () => {
                await onUpdate(table.id, editValues);
                setEditing(null);
              }, t('catalog.shared.save'), true)}
              <button type="button" onClick={() => setEditing(null)} className="mt-2 rounded-lg border border-slate-300 px-3 py-2 text-sm text-slate-700" disabled={loading}>{t('catalog.shared.cancel')}</button>
            </div>
          ) : null}
        </article>
      ))}
    </section>
  );
}

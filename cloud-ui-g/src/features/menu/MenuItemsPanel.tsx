import { useEffect, useState } from 'react';
import type { CatalogItem, MenuItem } from '../../shared/api/schemas';
import { useI18n } from '../../shared/i18n/I18nProvider';
import SafeErrorBanner from '../../shared/ui/SafeErrorBanner';
import { defaultMenuItemValues, normalizeMenuItemValues, toMenuItemValues, type MenuItemFormValues } from './menuForms';

type Props = {
  items: MenuItem[];
  catalogItems: CatalogItem[];
  restaurantCurrency: string;
  loading: boolean;
  error: unknown;
  onCreate: (values: MenuItemFormValues) => Promise<void>;
  onUpdate: (id: string, values: MenuItemFormValues) => Promise<void>;
  onArchive: (id: string) => Promise<void>;
};

const statuses: MenuItemFormValues['status'][] = ['draft', 'published', 'archived'];

export default function MenuItemsPanel({ items, catalogItems, restaurantCurrency, loading, error, onCreate, onUpdate, onArchive }: Props) {
  const { t } = useI18n();
  const [createValues, setCreateValues] = useState<MenuItemFormValues>({ ...defaultMenuItemValues, currency: restaurantCurrency });
  const [editing, setEditing] = useState<MenuItem | null>(null);
  const [editValues, setEditValues] = useState<MenuItemFormValues>(defaultMenuItemValues);

  useEffect(() => {
    setCreateValues((prev) => ({ ...prev, currency: restaurantCurrency }));
  }, [restaurantCurrency]);

  useEffect(() => {
    if (editing) setEditValues(toMenuItemValues(editing));
  }, [editing]);

  const catalogLabel = (id: string) => catalogItems.find((item) => item.id === id)?.name ?? id;

  const renderForm = (
    values: MenuItemFormValues,
    setValues: (next: MenuItemFormValues) => void,
    onSubmit: () => Promise<void>,
    actionLabel: string,
  ) => (
    <form className="space-y-3 rounded-xl border border-slate-200 bg-slate-50 p-4" onSubmit={(event) => { event.preventDefault(); void onSubmit(); }}>
      <div className="grid gap-3 md:grid-cols-2">
        <div>
          <label className="mb-1 block text-sm text-slate-700">{t('menu.items.fields.catalogItem')}</label>
          <select value={values.catalog_item_id} onChange={(event) => setValues({ ...values, catalog_item_id: event.target.value })} className="w-full rounded-lg border border-slate-300 px-3 py-2 text-sm" disabled={loading}>
            <option value="">{t('menu.items.fields.selectCatalogItem')}</option>
            {catalogItems.filter((item) => item.status !== 'archived').map((item) => <option key={item.id} value={item.id}>{item.name}</option>)}
          </select>
        </div>
        <div>
          <label className="mb-1 block text-sm text-slate-700">{t('menu.items.fields.name')}</label>
          <input value={values.name} onChange={(event) => setValues({ ...values, name: event.target.value })} className="w-full rounded-lg border border-slate-300 px-3 py-2 text-sm" disabled={loading} />
        </div>
      </div>
      <div className="grid gap-3 md:grid-cols-4">
        <div>
          <label className="mb-1 block text-sm text-slate-700">{t('menu.items.fields.price')}</label>
          <input type="number" min="0" value={values.price} onChange={(event) => setValues({ ...values, price: Number(event.target.value) })} className="w-full rounded-lg border border-slate-300 px-3 py-2 text-sm" disabled={loading} />
        </div>
        <div>
          <label className="mb-1 block text-sm text-slate-700">{t('menu.items.fields.currency')}</label>
          <input value={values.currency} onChange={(event) => setValues({ ...values, currency: event.target.value })} className="w-full rounded-lg border border-slate-300 px-3 py-2 text-sm" disabled={loading} />
        </div>
        <div>
          <label className="mb-1 block text-sm text-slate-700">{t('menu.items.fields.status')}</label>
          <select value={values.status} onChange={(event) => setValues({ ...values, status: event.target.value as MenuItemFormValues['status'] })} className="w-full rounded-lg border border-slate-300 px-3 py-2 text-sm" disabled={loading}>
            {statuses.map((status) => <option key={status} value={status}>{t(`catalog.statuses.${status}`)}</option>)}
          </select>
        </div>
        <div>
          <label className="mb-1 block text-sm text-slate-700">{t('menu.items.fields.station')}</label>
          <input value={values.station_routing_key} onChange={(event) => setValues({ ...values, station_routing_key: event.target.value })} className="w-full rounded-lg border border-slate-300 px-3 py-2 text-sm" disabled={loading} />
        </div>
      </div>
      <div>
        <label className="mb-1 block text-sm text-slate-700">{t('menu.items.fields.availability')}</label>
        <textarea value={values.availability_json} onChange={(event) => setValues({ ...values, availability_json: event.target.value })} className="min-h-24 w-full rounded-lg border border-slate-300 px-3 py-2 font-mono text-sm" disabled={loading} />
      </div>
      <button type="submit" disabled={loading || !values.catalog_item_id || !values.name.trim()} className="rounded-lg bg-slate-900 px-3 py-2 text-sm font-medium text-white disabled:opacity-50">{actionLabel}</button>
    </form>
  );

  return (
    <section className="space-y-4 rounded-2xl border border-slate-200 bg-white p-6">
      <h3 className="text-base font-semibold text-slate-900">{t('menu.items.title')}</h3>
      {renderForm(createValues, setCreateValues, async () => {
        await onCreate(normalizeMenuItemValues(createValues));
        setCreateValues({ ...defaultMenuItemValues, currency: restaurantCurrency });
      }, t('menu.items.actions.create'))}
      {error ? <SafeErrorBanner error={error} /> : null}
      {items.length === 0 ? <p className="text-sm text-slate-600">{t('menu.items.empty')}</p> : null}
      {items.map((item) => (
        <article key={item.id} className="rounded-xl border border-slate-200 p-4">
          <div className="flex flex-wrap justify-between gap-2">
            <div>
              <p className="text-sm font-medium text-slate-900">{item.name}</p>
              <p className="text-xs text-slate-600">{catalogLabel(item.catalog_item_id)} · {item.price} {item.currency} · {t(`catalog.statuses.${item.status}`)}</p>
            </div>
            <div className="flex gap-2">
              <button type="button" onClick={() => setEditing(item)} className="rounded-lg border border-slate-300 px-2 py-1 text-xs text-slate-700" disabled={loading}>{t('catalog.shared.edit')}</button>
              <button type="button" onClick={() => { if (window.confirm(t('catalog.shared.archiveConfirm'))) void onArchive(item.id); }} className="rounded-lg border border-rose-300 px-2 py-1 text-xs text-rose-700" disabled={loading || item.status === 'archived'}>{t('catalog.shared.archive')}</button>
            </div>
          </div>
          {editing?.id === item.id ? (
            <div className="mt-3">
              {renderForm(editValues, setEditValues, async () => {
                await onUpdate(item.id, normalizeMenuItemValues(editValues));
                setEditing(null);
              }, t('catalog.shared.save'))}
              <button type="button" onClick={() => setEditing(null)} className="mt-2 rounded-lg border border-slate-300 px-3 py-2 text-sm text-slate-700">{t('catalog.shared.cancel')}</button>
            </div>
          ) : null}
        </article>
      ))}
    </section>
  );
}

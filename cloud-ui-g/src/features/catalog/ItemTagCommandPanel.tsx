import { useState } from 'react';
import type { CatalogItem, CatalogTag } from '../../shared/api/schemas';
import { useI18n } from '../../shared/i18n/I18nProvider';
import SafeErrorBanner from '../../shared/ui/SafeErrorBanner';
import { defaultItemTagCommandValues, type ItemTagCommandFormValues } from './catalogForms';

type ItemTagCommandPanelProps = {
  items: CatalogItem[];
  tags: CatalogTag[];
  loading: boolean;
  error: unknown;
  success: boolean;
  onAssign: (values: ItemTagCommandFormValues) => Promise<void>;
};

export default function ItemTagCommandPanel({ items, tags, loading, error, success, onAssign }: ItemTagCommandPanelProps) {
  const { t } = useI18n();
  const [values, setValues] = useState<ItemTagCommandFormValues>(defaultItemTagCommandValues);

  return (
    <section className="space-y-4 rounded-2xl border border-slate-200 bg-white p-6">
      <div>
        <h3 className="text-base font-semibold text-slate-900">{t('catalog.itemTags.title')}</h3>
        <p className="mt-1 text-sm text-slate-600">{t('catalog.itemTags.commandOnly')}</p>
      </div>

      <form className="space-y-3 rounded-xl border border-slate-200 bg-slate-50 p-4" onSubmit={(event) => {
        event.preventDefault();
        void onAssign(values).then(() => setValues(defaultItemTagCommandValues));
      }}>
        <div>
          <label className="mb-1 block text-sm text-slate-700">{t('catalog.itemTags.fields.item')}</label>
          <select value={values.catalog_item_id} onChange={(event) => setValues({ ...values, catalog_item_id: event.target.value })} className="w-full rounded-lg border border-slate-300 bg-white px-3 py-2 text-sm" disabled={loading}>
            <option value="">{t('catalog.itemTags.fields.selectItem')}</option>
            {items.map((item) => <option key={item.id} value={item.id}>{item.name}</option>)}
          </select>
        </div>
        <div>
          <label className="mb-1 block text-sm text-slate-700">{t('catalog.itemTags.fields.tag')}</label>
          <select value={values.tag_id} onChange={(event) => setValues({ ...values, tag_id: event.target.value })} className="w-full rounded-lg border border-slate-300 bg-white px-3 py-2 text-sm" disabled={loading}>
            <option value="">{t('catalog.itemTags.fields.selectTag')}</option>
            {tags.map((tag) => <option key={tag.id} value={tag.id}>{tag.name}</option>)}
          </select>
        </div>
        <button type="submit" disabled={loading || !values.catalog_item_id || !values.tag_id} className="rounded-lg bg-slate-900 px-3 py-2 text-sm font-medium text-white disabled:opacity-50">{t('catalog.itemTags.actions.assign')}</button>
      </form>

      {success ? <div className="rounded-lg border border-emerald-200 bg-emerald-50 p-3 text-sm text-emerald-800">{t('catalog.itemTags.success')}</div> : null}
      {error ? <SafeErrorBanner error={error} /> : null}
    </section>
  );
}

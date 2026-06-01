import { useState } from 'react';
import { useI18n } from '../../shared/i18n/I18nProvider';
import SafeErrorBanner from '../../shared/ui/SafeErrorBanner';
import { defaultMenuCategoryValues, type MenuCategoryFormValues } from './menuForms';

type Props = {
  restaurantId: string;
  loading: boolean;
  error: unknown;
  success: boolean;
  onCreate: (values: MenuCategoryFormValues) => Promise<void>;
};

export default function MenuCategoryCommandPanel({ restaurantId, loading, error, success, onCreate }: Props) {
  const { t } = useI18n();
  const [values, setValues] = useState<MenuCategoryFormValues>({ ...defaultMenuCategoryValues, restaurant_id: restaurantId });

  return (
    <section className="space-y-4 rounded-2xl border border-slate-200 bg-white p-6">
      <div>
        <h3 className="text-base font-semibold text-slate-900">{t('menu.categories.title')}</h3>
        <p className="mt-1 text-sm text-slate-600">{t('menu.categories.commandOnly')}</p>
      </div>
      <form
        className="grid gap-3 md:grid-cols-[1fr_160px_auto]"
        onSubmit={(event) => {
          event.preventDefault();
          void onCreate({ ...values, restaurant_id: restaurantId, name: values.name.trim() }).then(() => {
            setValues({ ...defaultMenuCategoryValues, restaurant_id: restaurantId });
          });
        }}
      >
        <div>
          <label className="mb-1 block text-sm text-slate-700">{t('menu.categories.fields.name')}</label>
          <input value={values.name} onChange={(event) => setValues({ ...values, name: event.target.value })} className="w-full rounded-lg border border-slate-300 px-3 py-2 text-sm" disabled={loading} />
        </div>
        <div>
          <label className="mb-1 block text-sm text-slate-700">{t('menu.categories.fields.sortOrder')}</label>
          <input type="number" value={values.sort_order} onChange={(event) => setValues({ ...values, sort_order: Number(event.target.value) })} className="w-full rounded-lg border border-slate-300 px-3 py-2 text-sm" disabled={loading} />
        </div>
        <div className="flex items-end">
          <button type="submit" disabled={loading || !values.name.trim()} className="rounded-lg bg-slate-900 px-3 py-2 text-sm font-medium text-white disabled:opacity-50">{t('menu.categories.actions.create')}</button>
        </div>
      </form>
      {success ? <p className="text-sm text-emerald-700">{t('menu.categories.success')}</p> : null}
      {error ? <SafeErrorBanner error={error} /> : null}
    </section>
  );
}

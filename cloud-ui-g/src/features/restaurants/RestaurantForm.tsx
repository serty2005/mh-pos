import { useEffect, useState } from 'react';
import type { Restaurant } from '../../shared/api/schemas';
import { useI18n } from '../../shared/i18n/I18nProvider';

export type RestaurantFormValues = {
  name: string;
  timezone: string;
  currency: string;
  business_day_mode: string;
  business_day_boundary_local_time: string;
  status: 'active' | 'archived';
};

type RestaurantFormProps = {
  mode: 'create' | 'edit';
  initial?: Restaurant | null;
  disabled?: boolean;
  onSubmit: (values: RestaurantFormValues) => Promise<void>;
  onCancel?: () => void;
};

export const defaultRestaurantValues: RestaurantFormValues = {
  name: '',
  timezone: 'Europe/Moscow',
  currency: 'RUB',
  business_day_mode: 'standard',
  business_day_boundary_local_time: '04:00',
  status: 'active',
};

export default function RestaurantForm({ mode, initial, disabled = false, onSubmit, onCancel }: RestaurantFormProps) {
  const { t } = useI18n();
  const [values, setValues] = useState<RestaurantFormValues>(defaultRestaurantValues);

  useEffect(() => {
    if (!initial) {
      setValues(defaultRestaurantValues);
      return;
    }

    setValues({
      name: initial.name,
      timezone: initial.timezone,
      currency: initial.currency,
      business_day_mode: initial.business_day_mode,
      business_day_boundary_local_time: initial.business_day_boundary_local_time,
      status: initial.status,
    });
  }, [initial]);

  const handleChange = (key: keyof RestaurantFormValues, value: string) => {
    setValues((prev) => ({ ...prev, [key]: value }));
  };

  const submitDisabled = disabled || !values.name.trim();

  return (
    <form
      className="space-y-3 rounded-xl border border-slate-200 bg-slate-50 p-4"
      onSubmit={(event) => {
        event.preventDefault();
        void onSubmit({ ...values, name: values.name.trim() });
      }}
    >
      <div>
        <label className="mb-1 block text-sm text-slate-700" htmlFor="restaurant-name">{t('restaurants.form.name')}</label>
        <input
          id="restaurant-name"
          value={values.name}
          onChange={(event) => handleChange('name', event.target.value)}
          className="w-full rounded-lg border border-slate-300 bg-white px-3 py-2 text-sm"
          disabled={disabled}
        />
      </div>

      <div className="grid gap-3 md:grid-cols-2">
        <div>
          <label className="mb-1 block text-sm text-slate-700" htmlFor="restaurant-timezone">{t('restaurants.form.timezone')}</label>
          <input id="restaurant-timezone" value={values.timezone} onChange={(event) => handleChange('timezone', event.target.value)} className="w-full rounded-lg border border-slate-300 bg-white px-3 py-2 text-sm" disabled={disabled} />
        </div>
        <div>
          <label className="mb-1 block text-sm text-slate-700" htmlFor="restaurant-currency">{t('restaurants.form.currency')}</label>
          <input id="restaurant-currency" value={values.currency} onChange={(event) => handleChange('currency', event.target.value)} className="w-full rounded-lg border border-slate-300 bg-white px-3 py-2 text-sm" disabled={disabled} />
        </div>
      </div>

      <div className="grid gap-3 md:grid-cols-2">
        <div>
          <label className="mb-1 block text-sm text-slate-700" htmlFor="restaurant-business-day-mode">{t('restaurants.form.businessDayMode')}</label>
          <input id="restaurant-business-day-mode" value={values.business_day_mode} onChange={(event) => handleChange('business_day_mode', event.target.value)} className="w-full rounded-lg border border-slate-300 bg-white px-3 py-2 text-sm" disabled={disabled} />
        </div>
        <div>
          <label className="mb-1 block text-sm text-slate-700" htmlFor="restaurant-business-day-boundary">{t('restaurants.form.businessDayBoundary')}</label>
          <input id="restaurant-business-day-boundary" value={values.business_day_boundary_local_time} onChange={(event) => handleChange('business_day_boundary_local_time', event.target.value)} className="w-full rounded-lg border border-slate-300 bg-white px-3 py-2 text-sm" disabled={disabled} />
        </div>
      </div>

      <div>
        <label className="mb-1 block text-sm text-slate-700" htmlFor="restaurant-status">{t('restaurants.form.status')}</label>
        <select id="restaurant-status" value={values.status} onChange={(event) => handleChange('status', event.target.value)} className="w-full rounded-lg border border-slate-300 bg-white px-3 py-2 text-sm" disabled={disabled}>
          <option value="active">{t('restaurants.status.active')}</option>
          <option value="archived">{t('restaurants.status.archived')}</option>
        </select>
      </div>

      <div className="flex flex-wrap gap-2">
        <button type="submit" className="rounded-lg bg-slate-900 px-3 py-2 text-sm font-medium text-white disabled:opacity-50" disabled={submitDisabled}>
          {mode === 'create' ? t('restaurants.form.create') : t('restaurants.form.save')}
        </button>
        {onCancel ? (
          <button type="button" onClick={onCancel} className="rounded-lg border border-slate-300 px-3 py-2 text-sm font-medium text-slate-700" disabled={disabled}>
            {t('restaurants.form.cancel')}
          </button>
        ) : null}
      </div>
    </form>
  );
}

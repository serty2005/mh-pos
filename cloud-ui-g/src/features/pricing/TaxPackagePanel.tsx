import { useState } from 'react';
import { useI18n } from '../../shared/i18n/I18nProvider';
import SafeErrorBanner from '../../shared/ui/SafeErrorBanner';
import { buildPricingPolicyPackagePayload, defaultTaxPackageDraft, parseRows, type TaxPackageDraft } from './pricingForms';

type Props = {
  restaurantId: string;
  loading: boolean;
  error: unknown;
  success: boolean;
  onLoad: (nodeDeviceId: string) => Promise<TaxPackageDraft | null>;
  onSave: (payload: ReturnType<typeof buildPricingPolicyPackagePayload>) => Promise<void>;
};

export default function TaxPackagePanel({ restaurantId, loading, error, success, onLoad, onSave }: Props) {
  const { t } = useI18n();
  const [draft, setDraft] = useState<TaxPackageDraft>({ ...defaultTaxPackageDraft, restaurant_id: restaurantId });
  const [taxProfilesJson, setTaxProfilesJson] = useState('[]');
  const [taxRulesJson, setTaxRulesJson] = useState('[]');
  const [serviceChargeJson, setServiceChargeJson] = useState('[]');
  const [validationKey, setValidationKey] = useState('');

  const load = async () => {
    if (!draft.node_device_id.trim()) return;
    const loaded = await onLoad(draft.node_device_id);
    if (!loaded) {
      setValidationKey('pricing.taxPackage.emptyLoaded');
      return;
    }
    setDraft(loaded);
    setTaxProfilesJson(JSON.stringify(loaded.tax_profiles, null, 2));
    setTaxRulesJson(JSON.stringify(loaded.tax_rules, null, 2));
    setServiceChargeJson(JSON.stringify(loaded.service_charge_rules, null, 2));
    setValidationKey('');
  };

  const save = async () => {
    try {
      const nextDraft = {
        ...draft,
        restaurant_id: restaurantId,
        tax_profiles: parseRows(taxProfilesJson),
        tax_rules: parseRows(taxRulesJson),
        service_charge_rules: parseRows(serviceChargeJson),
      };
      setValidationKey('');
      await onSave(buildPricingPolicyPackagePayload(nextDraft));
      setDraft(nextDraft);
    } catch {
      setValidationKey('pricing.taxPackage.invalidJson');
    }
  };

  return (
    <section className="space-y-4 rounded-2xl border border-slate-200 bg-white p-6">
      <div>
        <h3 className="text-base font-semibold text-slate-900">{t('pricing.taxPackage.title')}</h3>
        <p className="mt-1 text-sm text-slate-600">{t('pricing.taxPackage.description')}</p>
      </div>
      <div className="grid gap-3 md:grid-cols-[1fr_140px_1fr_auto_auto]">
        <input value={draft.node_device_id} onChange={(event) => setDraft({ ...draft, node_device_id: event.target.value })} className="rounded-lg border border-slate-300 px-3 py-2 text-sm" disabled={loading} placeholder={t('pricing.taxPackage.fields.nodeDeviceId')} />
        <input type="number" min="1" value={draft.cloud_version} onChange={(event) => setDraft({ ...draft, cloud_version: Number(event.target.value) })} className="rounded-lg border border-slate-300 px-3 py-2 text-sm" disabled={loading} />
        <input value={draft.full_snapshot_reason} onChange={(event) => setDraft({ ...draft, full_snapshot_reason: event.target.value })} className="rounded-lg border border-slate-300 px-3 py-2 text-sm" disabled={loading} placeholder={t('pricing.taxPackage.fields.reason')} />
        <button type="button" onClick={() => void load()} disabled={loading || !draft.node_device_id.trim()} className="rounded-lg border border-slate-300 px-3 py-2 text-sm font-medium text-slate-700 disabled:opacity-50">{t('pricing.taxPackage.actions.load')}</button>
        <button type="button" onClick={() => void save()} disabled={loading || !draft.node_device_id.trim()} className="rounded-lg bg-slate-900 px-3 py-2 text-sm font-medium text-white disabled:opacity-50">{t('pricing.taxPackage.actions.save')}</button>
      </div>
      <div className="grid gap-3 lg:grid-cols-3">
        <label className="text-sm text-slate-700">
          {t('pricing.taxPackage.fields.taxProfiles')}
          <textarea value={taxProfilesJson} onChange={(event) => setTaxProfilesJson(event.target.value)} className="mt-1 min-h-40 w-full rounded-lg border border-slate-300 px-3 py-2 font-mono text-xs" disabled={loading} />
        </label>
        <label className="text-sm text-slate-700">
          {t('pricing.taxPackage.fields.taxRules')}
          <textarea value={taxRulesJson} onChange={(event) => setTaxRulesJson(event.target.value)} className="mt-1 min-h-40 w-full rounded-lg border border-slate-300 px-3 py-2 font-mono text-xs" disabled={loading} />
        </label>
        <label className="text-sm text-slate-700">
          {t('pricing.taxPackage.fields.serviceChargeRules')}
          <textarea value={serviceChargeJson} onChange={(event) => setServiceChargeJson(event.target.value)} className="mt-1 min-h-40 w-full rounded-lg border border-slate-300 px-3 py-2 font-mono text-xs" disabled={loading} />
        </label>
      </div>
      {validationKey ? <p className="text-sm text-amber-700">{t(validationKey)}</p> : null}
      {success ? <p className="text-sm text-emerald-700">{t('pricing.taxPackage.success')}</p> : null}
      {error ? <SafeErrorBanner error={error} /> : null}
    </section>
  );
}

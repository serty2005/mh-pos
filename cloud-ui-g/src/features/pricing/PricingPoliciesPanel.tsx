import { useEffect, useState } from 'react';
import type { PricingPolicy } from '../../shared/api/schemas';
import { useI18n } from '../../shared/i18n/I18nProvider';
import SafeErrorBanner from '../../shared/ui/SafeErrorBanner';
import {
  defaultPricingPolicyValues,
  findDuplicateApplicationIndex,
  normalizePricingPolicyValues,
  toPricingPolicyValues,
  type PricingAmountKind,
  type PricingPolicyFormValues,
  type PricingPolicyKind,
} from './pricingForms';

type Props = {
  policies: PricingPolicy[];
  loading: boolean;
  error: unknown;
  onCreate: (values: PricingPolicyFormValues) => Promise<void>;
  onUpdate: (id: string, values: PricingPolicyFormValues) => Promise<void>;
};

const kinds: PricingPolicyKind[] = ['discount', 'surcharge'];
const amountKinds: PricingAmountKind[] = ['fixed', 'percentage'];
const statuses: PricingPolicyFormValues['status'][] = ['draft', 'published', 'archived'];

export default function PricingPoliciesPanel({ policies, loading, error, onCreate, onUpdate }: Props) {
  const { t } = useI18n();
  const [values, setValues] = useState<PricingPolicyFormValues>(defaultPricingPolicyValues);
  const [editing, setEditing] = useState<PricingPolicy | null>(null);
  const [editValues, setEditValues] = useState<PricingPolicyFormValues>(defaultPricingPolicyValues);
  const [validationKey, setValidationKey] = useState('');

  useEffect(() => {
    if (editing) setEditValues(toPricingPolicyValues(editing));
  }, [editing]);

  const submit = async (next: PricingPolicyFormValues, action: (normalized: PricingPolicyFormValues) => Promise<void>, currentId = '') => {
    const normalized = normalizePricingPolicyValues(next);
    const duplicate = findDuplicateApplicationIndex([
      ...policies.filter((policy) => policy.id !== currentId).map(toPricingPolicyValues),
      normalized,
    ]);
    if (duplicate !== null) {
      setValidationKey('pricing.policies.validation.duplicateIndex');
      return;
    }
    setValidationKey('');
    await action(normalized);
  };

  const renderForm = (formValues: PricingPolicyFormValues, setFormValues: (next: PricingPolicyFormValues) => void, onSubmit: () => Promise<void>, label: string) => (
    <form className="space-y-3 rounded-xl border border-slate-200 bg-slate-50 p-4" onSubmit={(event) => { event.preventDefault(); void onSubmit(); }}>
      <div className="grid gap-3 md:grid-cols-3">
        <input value={formValues.name} onChange={(event) => setFormValues({ ...formValues, name: event.target.value })} className="rounded-lg border border-slate-300 px-3 py-2 text-sm" disabled={loading} placeholder={t('pricing.policies.fields.name')} />
        <select value={formValues.kind} onChange={(event) => setFormValues({ ...formValues, kind: event.target.value as PricingPolicyKind })} className="rounded-lg border border-slate-300 px-3 py-2 text-sm" disabled={loading}>
          {kinds.map((kind) => <option key={kind} value={kind}>{t(`pricing.policies.kinds.${kind}`)}</option>)}
        </select>
        <input value={formValues.scope} onChange={(event) => setFormValues({ ...formValues, scope: event.target.value })} className="rounded-lg border border-slate-300 px-3 py-2 text-sm" disabled={loading || formValues.kind === 'surcharge'} placeholder={t('pricing.policies.fields.scope')} />
      </div>
      <div className="grid gap-3 md:grid-cols-4">
        <select value={formValues.amount_kind} onChange={(event) => setFormValues({ ...formValues, amount_kind: event.target.value as PricingAmountKind })} className="rounded-lg border border-slate-300 px-3 py-2 text-sm" disabled={loading}>
          {amountKinds.map((kind) => <option key={kind} value={kind}>{t(`pricing.policies.amountKinds.${kind}`)}</option>)}
        </select>
        <input type="number" min="0" value={formValues.amount_minor} onChange={(event) => setFormValues({ ...formValues, amount_minor: Number(event.target.value) })} className="rounded-lg border border-slate-300 px-3 py-2 text-sm" disabled={loading || formValues.amount_kind !== 'fixed'} />
        <input type="number" min="0" value={formValues.value_basis_points} onChange={(event) => setFormValues({ ...formValues, value_basis_points: Number(event.target.value) })} className="rounded-lg border border-slate-300 px-3 py-2 text-sm" disabled={loading || formValues.amount_kind !== 'percentage'} />
        <input type="number" min="1" value={formValues.application_index} onChange={(event) => setFormValues({ ...formValues, application_index: Number(event.target.value) })} className="rounded-lg border border-slate-300 px-3 py-2 text-sm" disabled={loading} />
      </div>
      <div className="grid gap-3 md:grid-cols-[1fr_180px_120px]">
        <input value={formValues.requires_permission} onChange={(event) => setFormValues({ ...formValues, requires_permission: event.target.value })} className="rounded-lg border border-slate-300 px-3 py-2 text-sm" disabled={loading} placeholder={t('pricing.policies.fields.permission')} />
        <select value={formValues.status} onChange={(event) => setFormValues({ ...formValues, status: event.target.value as PricingPolicyFormValues['status'] })} className="rounded-lg border border-slate-300 px-3 py-2 text-sm" disabled={loading}>
          {statuses.map((status) => <option key={status} value={status}>{t(`catalog.statuses.${status}`)}</option>)}
        </select>
        <button type="submit" disabled={loading || !formValues.name.trim()} className="rounded-lg bg-slate-900 px-3 py-2 text-sm font-medium text-white disabled:opacity-50">{label}</button>
      </div>
      <label className="flex items-center gap-2 text-sm text-slate-700">
        <input type="checkbox" checked={formValues.manual} onChange={(event) => setFormValues({ ...formValues, manual: event.target.checked })} disabled={loading} />
        {t('pricing.policies.fields.manual')}
      </label>
    </form>
  );

  return (
    <section className="space-y-4 rounded-2xl border border-slate-200 bg-white p-6">
      <h3 className="text-base font-semibold text-slate-900">{t('pricing.policies.title')}</h3>
      {renderForm(values, setValues, async () => {
        await submit(values, onCreate);
        setValues(defaultPricingPolicyValues);
      }, t('pricing.policies.actions.create'))}
      {validationKey ? <p className="text-sm text-rose-700">{t(validationKey)}</p> : null}
      {error ? <SafeErrorBanner error={error} /> : null}
      {policies.length === 0 ? <p className="text-sm text-slate-600">{t('pricing.policies.empty')}</p> : null}
      {policies.map((policy) => (
        <article key={policy.id} className="rounded-xl border border-slate-200 p-4">
          <div className="flex flex-wrap justify-between gap-2">
            <p className="text-sm font-medium text-slate-900">{policy.name} · {t(`pricing.policies.kinds.${policy.kind}`)} · {t(`catalog.statuses.${policy.status}`)}</p>
            <button type="button" onClick={() => setEditing(policy)} className="rounded-lg border border-slate-300 px-2 py-1 text-xs text-slate-700">{t('catalog.shared.edit')}</button>
          </div>
          <p className="mt-1 text-xs text-slate-600">{t('pricing.policies.fields.applicationIndex')}: {policy.application_index} · {policy.amount_kind}</p>
          {editing?.id === policy.id ? (
            <div className="mt-3 space-y-2">
              {renderForm(editValues, setEditValues, async () => {
                await submit(editValues, (normalized) => onUpdate(policy.id, normalized), policy.id);
                setEditing(null);
              }, t('catalog.shared.save'))}
              <button type="button" onClick={() => setEditing(null)} className="rounded-lg border border-slate-300 px-3 py-2 text-sm text-slate-700">{t('catalog.shared.cancel')}</button>
            </div>
          ) : null}
        </article>
      ))}
    </section>
  );
}

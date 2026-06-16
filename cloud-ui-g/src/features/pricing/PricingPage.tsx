import { useEffect, useState } from 'react';
import { Percent } from 'lucide-react';
import { createPricingPolicy, getPricingPolicyPackage, listPricingPolicies, putPricingPolicyPackage, updatePricingPolicy } from '../../shared/api/endpoints';
import type { PricingPolicy } from '../../shared/api/schemas';
import { useI18n } from '../../shared/i18n/I18nProvider';
import EmptyState from '../../shared/ui/EmptyState';
import PricingPoliciesPanel from './PricingPoliciesPanel';
import TaxPackagePanel from './TaxPackagePanel';
import { buildCreatePricingPolicyPayload, type PricingPolicyFormValues, type TaxPackageDraft } from './pricingForms';

type Props = {
  restaurantId: string;
};

type RouteStatus = 'loading' | 'ready' | 'blocked';

export default function PricingPage({ restaurantId }: Props) {
  const { t } = useI18n();
  const [policies, setPolicies] = useState<PricingPolicy[]>([]);
  const [status, setStatus] = useState<RouteStatus>('loading');
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<unknown>(null);
  const [packageSuccess, setPackageSuccess] = useState(false);

  const reload = async () => {
    setStatus('loading');
    setError(null);
    try {
      setPolicies(await listPricingPolicies(restaurantId));
      setStatus('ready');
    } catch (nextError) {
      setStatus('blocked');
      setError(nextError);
    }
  };

  useEffect(() => {
    void reload();
  }, [restaurantId]);

  const mutate = async (action: () => Promise<void>) => {
    setLoading(true);
    setPackageSuccess(false);
    setError(null);
    try {
      await action();
      await reload();
    } catch (nextError) {
      setError(nextError);
    } finally {
      setLoading(false);
    }
  };

  return (
    <section className="space-y-4">
      <div className="rounded-2xl border border-slate-200 bg-white p-5 sm:p-6">
        <div className="flex flex-col gap-4 lg:flex-row lg:items-start lg:justify-between">
          <div className="flex items-start gap-3">
            <div className="flex h-10 w-10 shrink-0 items-center justify-center rounded-xl border border-blue-100 bg-blue-50 text-blue-700">
              <Percent className="h-4 w-4" />
            </div>
            <div>
              <h3 className="text-lg font-semibold tracking-tight text-slate-950">{t('pricing.pageTitle')}</h3>
              <p className="mt-1 max-w-3xl text-sm leading-6 text-slate-600">{t('pricing.pageDescription')}</p>
            </div>
          </div>
          <p className={status === 'ready' ? 'rounded-full border border-emerald-100 bg-emerald-50 px-3 py-1.5 text-xs font-semibold text-emerald-700' : status === 'loading' ? 'rounded-full border border-blue-100 bg-blue-50 px-3 py-1.5 text-xs font-semibold text-blue-700' : 'rounded-full border border-amber-100 bg-amber-50 px-3 py-1.5 text-xs font-semibold text-amber-700'}>
            {t('catalog.readiness')}: {status === 'ready' ? t('status.ready') : status === 'loading' ? t('status.loading') : t('status.blocked')}
          </p>
        </div>
      </div>
      {status === 'blocked' ? <EmptyState title={t('pricing.blockedTitle')} description={t('pricing.blockedDescription')} /> : null}
      {status !== 'blocked' ? (
        <>
          <PricingPoliciesPanel
            policies={policies}
            loading={loading}
            error={error}
            onCreate={(values: PricingPolicyFormValues) => mutate(async () => { await createPricingPolicy({ restaurant_id: restaurantId, ...buildCreatePricingPolicyPayload(values) }); })}
            onUpdate={(id: string, values: PricingPolicyFormValues) => mutate(async () => { await updatePricingPolicy(id, values); })}
          />
          <TaxPackagePanel
            restaurantId={restaurantId}
            loading={loading}
            error={error}
            success={packageSuccess}
            onLoad={async (nodeDeviceId: string): Promise<TaxPackageDraft | null> => {
              const loaded = await getPricingPolicyPackage(nodeDeviceId);
              if (!loaded) return null;
              return {
                node_device_id: loaded.node_device_id,
                restaurant_id: loaded.restaurant_id,
                sync_mode: loaded.sync_mode,
                full_snapshot_reason: loaded.full_snapshot_reason,
                cloud_version: loaded.cloud_version,
                tax_profiles: loaded.payload_json.tax_profiles,
                tax_rules: loaded.payload_json.tax_rules,
                service_charge_rules: loaded.payload_json.service_charge_rules,
              };
            }}
            onSave={(payload) => mutate(async () => {
              await putPricingPolicyPackage(payload);
              setPackageSuccess(true);
            })}
          />
        </>
      ) : null}
    </section>
  );
}

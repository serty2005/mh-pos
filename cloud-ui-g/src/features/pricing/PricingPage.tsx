import { useEffect, useState } from 'react';
import { createPricingPolicy, getPricingPolicyPackage, listPricingPolicies, putPricingPolicyPackage, updatePricingPolicy } from '../../shared/api/endpoints';
import type { PricingPolicy } from '../../shared/api/schemas';
import { useI18n } from '../../shared/i18n/I18nProvider';
import EmptyState from '../../shared/ui/EmptyState';
import PricingPoliciesPanel from './PricingPoliciesPanel';
import TaxPackagePanel from './TaxPackagePanel';
import type { PricingPolicyFormValues, TaxPackageDraft } from './pricingForms';

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
      <div className="rounded-2xl border border-slate-200 bg-white p-6">
        <h3 className="text-base font-semibold text-slate-900">{t('pricing.pageTitle')}</h3>
        <p className="mt-1 text-sm text-slate-600">{t('pricing.pageDescription')}</p>
        <p className="mt-2 text-xs text-slate-500">{t('catalog.readiness')}: {status === 'ready' ? t('status.ready') : status === 'loading' ? t('status.loading') : t('status.blocked')}</p>
      </div>
      {status === 'blocked' ? <EmptyState title={t('pricing.blockedTitle')} description={t('pricing.blockedDescription')} /> : null}
      {status !== 'blocked' ? (
        <>
          <PricingPoliciesPanel
            policies={policies}
            loading={loading}
            error={error}
            onCreate={(values: PricingPolicyFormValues) => mutate(async () => { await createPricingPolicy({ restaurant_id: restaurantId, ...values }); })}
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

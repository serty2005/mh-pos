import { RefreshCw, RadioTower, UploadCloud } from 'lucide-react';
import { useI18n } from '../../shared/i18n/I18nProvider';
import { formatIsoDateTime } from '../../shared/utils/format';
import EmptyState from '../../shared/ui/EmptyState';
import PanelHeader from '../../shared/ui/PanelHeader';
import SafeErrorBanner from '../../shared/ui/SafeErrorBanner';
import { usePublication } from './usePublication';

type PublicationPanelProps = {
  restaurantId: string;
};

export default function PublicationPanel({ restaurantId }: PublicationPanelProps) {
  const { t } = useI18n();
  const { publication, deliveries, status, error, reload } = usePublication(restaurantId);
  const isLoading = status === 'loading';
  const deliveryStatus = (value: 'pending' | 'synced' | 'error') => {
    if (value === 'synced') return t('publications.statusSynced');
    if (value === 'error') return t('publications.statusError');
    return t('publications.statusPending');
  };

  return (
    <section className="space-y-5 rounded-2xl border border-slate-200 bg-white p-5 sm:p-6">
      <PanelHeader
        icon={UploadCloud}
        title={t('publications.title')}
        description={t('publications.automaticDescription')}
        action={(
          <button type="button" className="inline-flex items-center gap-2 rounded-xl border border-slate-300 px-3 py-2 text-xs font-semibold text-slate-700 disabled:opacity-50" onClick={() => { void reload(); }} disabled={isLoading || !restaurantId}>
            <RefreshCw className={isLoading ? 'h-3.5 w-3.5 animate-spin' : 'h-3.5 w-3.5'} />
            {isLoading ? t('edge.refreshing') : t('ui.retry')}
          </button>
        )}
      />

      {status === 'blocked' ? <SafeErrorBanner error={error} /> : null}
      {status === 'ready' && !publication ? (
        <EmptyState title={t('publications.emptyTitle')} description={t('publications.emptyDescription')} />
      ) : null}

      {publication ? (
        <>
          <div className="grid gap-3 text-sm text-slate-700 md:grid-cols-2 xl:grid-cols-4">
            <p className="rounded-2xl border border-slate-200 bg-slate-50 p-3 text-xs font-semibold text-slate-500">{t('publications.fields.cloudVersion')}<span className="mt-1 block font-mono text-sm font-semibold text-slate-900">{publication.cloud_version}</span></p>
            <p className="rounded-2xl border border-slate-200 bg-slate-50 p-3 text-xs font-semibold text-slate-500">{t('publications.fields.status')}<span className="mt-1 block text-sm font-semibold text-slate-900">{publication.status}</span></p>
            <p className="rounded-2xl border border-slate-200 bg-slate-50 p-3 text-xs font-semibold text-slate-500">{t('publications.fields.updatedAt')}<span className="mt-1 block font-mono text-xs font-semibold text-slate-900">{formatIsoDateTime(publication.published_at)}</span></p>
            <p className="rounded-2xl border border-slate-200 bg-slate-50 p-3 text-xs font-semibold text-slate-500">{t('publications.fields.packageHash')}<span className="mt-1 block truncate font-mono text-xs font-semibold text-slate-900">{publication.package_sha256}</span></p>
          </div>

        </>
      ) : null}

      {status === 'ready' ? (
          <div className="rounded-2xl border border-slate-200 bg-slate-50 p-4">
            <div className="flex items-start gap-3">
              <div className="flex h-9 w-9 shrink-0 items-center justify-center rounded-xl border border-slate-200 bg-white text-slate-700">
                <RadioTower className="h-4 w-4" />
              </div>
              <div className="min-w-0">
                <h4 className="text-sm font-semibold text-slate-900">{t('publications.deliveryTitle')}</h4>
                <p className="mt-1 text-sm leading-6 text-slate-600">{t('publications.deliveryDescription')}</p>
              </div>
            </div>
            {deliveries.length === 0 ? <p className="mt-3 text-sm text-slate-600">{t('publications.deliveryEmpty')}</p> : (
              <div className="mt-3 divide-y divide-slate-200 overflow-hidden rounded-lg border border-slate-200 bg-white">
                {deliveries.map((delivery) => (
                  <div key={delivery.node_device_id} className="grid gap-3 p-3 text-sm md:grid-cols-6">
                    <p className="md:col-span-2"><span className="block text-xs font-semibold text-slate-500">{t('publications.fields.nodeDeviceId')}</span><span className="mt-1 block truncate font-mono text-xs text-slate-900">{delivery.node_device_id}</span></p>
                    <p><span className="block text-xs font-semibold text-slate-500">{t('publications.fields.status')}</span><span className="mt-1 block font-semibold text-slate-900">{deliveryStatus(delivery.status)}</span></p>
                    <p><span className="block text-xs font-semibold text-slate-500">{t('publications.fields.edgeAck')}</span><span className="mt-1 block font-mono tabular-nums text-slate-900">{delivery.edge_ack_version} / {delivery.cloud_version}</span></p>
                    <p><span className="block text-xs font-semibold text-slate-500">{t('publications.fields.lag')}</span><span className="mt-1 block tabular-nums text-slate-900">{delivery.lag}</span></p>
                    <p><span className="block text-xs font-semibold text-slate-500">{t('publications.fields.lastSync')}</span><span className="mt-1 block text-xs text-slate-900">{delivery.last_sync_at ? formatIsoDateTime(delivery.last_sync_at) : t('publications.noSync')}</span></p>
                    <p className="md:col-span-5"><span className="block text-xs font-semibold text-slate-500">{t('publications.fields.error')}</span><span className="mt-1 block font-mono text-xs text-slate-900">{delivery.last_error_code || t('publications.noDeliveryError')}</span></p>
                    <p><span className="block text-xs font-semibold text-slate-500">{t('publications.fields.failures')}</span><span className="mt-1 block tabular-nums text-slate-900">{delivery.consecutive_failures}</span></p>
                  </div>
                ))}
              </div>
            )}
          </div>
      ) : null}
    </section>
  );
}

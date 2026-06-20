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
  const { publication, status, error, reload } = usePublication(restaurantId);
  const isLoading = status === 'loading';

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
            <div className="mt-3 grid gap-3 md:grid-cols-3">
              <p className="rounded-xl border border-slate-200 bg-white p-3 text-xs font-semibold text-slate-500">{t('publications.fields.edgeAck')}<span className="mt-1 block text-sm font-semibold text-slate-900">{t('publications.ackPending')}</span></p>
              <p className="rounded-xl border border-slate-200 bg-white p-3 text-xs font-semibold text-slate-500">{t('publications.fields.lag')}<span className="mt-1 block text-sm font-semibold text-slate-900">{t('publications.lagUnknown')}</span></p>
              <p className="rounded-xl border border-slate-200 bg-white p-3 text-xs font-semibold text-slate-500">{t('publications.fields.error')}<span className="mt-1 block text-sm font-semibold text-slate-900">{t('publications.noDeliveryError')}</span></p>
            </div>
          </div>
        </>
      ) : null}
    </section>
  );
}

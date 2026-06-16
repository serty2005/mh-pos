import { useState } from 'react';
import { RefreshCw, Send, UploadCloud } from 'lucide-react';
import { useI18n } from '../../shared/i18n/I18nProvider';
import EmptyState from '../../shared/ui/EmptyState';
import PanelHeader from '../../shared/ui/PanelHeader';
import SafeErrorBanner from '../../shared/ui/SafeErrorBanner';
import { usePublication } from './usePublication';

type PublicationPanelProps = {
  restaurantId: string;
  canPublish: boolean;
};

export default function PublicationPanel({ restaurantId, canPublish }: PublicationPanelProps) {
  const { t } = useI18n();
  const { publication, status, error, reload, publish } = usePublication(restaurantId);
  const [publishedBy, setPublishedBy] = useState('cloud-ui-g');
  const [nodeDeviceId, setNodeDeviceId] = useState('');
  const [submitError, setSubmitError] = useState<unknown>(null);
  const [submitting, setSubmitting] = useState(false);

  const handlePublish = async () => {
    setSubmitError(null);
    setSubmitting(true);
    try {
      await publish({ published_by: publishedBy.trim(), node_device_id: nodeDeviceId.trim() });
    } catch (nextError) {
      setSubmitError(nextError);
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <section className="space-y-5 rounded-2xl border border-slate-200 bg-white p-5 sm:p-6">
      <PanelHeader
        icon={UploadCloud}
        title={t('publications.title')}
        description={canPublish ? t('publications.manualReady') : t('publications.manualBlocked')}
        action={(
          <button type="button" className="inline-flex items-center gap-2 rounded-xl border border-slate-300 px-3 py-2 text-xs font-semibold text-slate-700" onClick={() => { void reload(); }}>
            <RefreshCw className="h-3.5 w-3.5" />
            {t('ui.retry')}
          </button>
        )}
      />

      {status === 'blocked' ? <SafeErrorBanner error={error} /> : null}
      {status === 'ready' && !publication ? (
        <EmptyState title={t('publications.emptyTitle')} description={t('publications.emptyDescription')} />
      ) : null}

      {publication ? (
        <div className="grid gap-3 text-sm text-slate-700 md:grid-cols-2 xl:grid-cols-4">
          <p className="rounded-2xl border border-slate-200 bg-slate-50 p-3 text-xs font-semibold text-slate-500">{t('publications.fields.version')}<span className="mt-1 block font-mono text-sm font-semibold text-slate-900">{publication.version}</span></p>
          <p className="rounded-2xl border border-slate-200 bg-slate-50 p-3 text-xs font-semibold text-slate-500">{t('publications.fields.status')}<span className="mt-1 block text-sm font-semibold text-slate-900">{publication.status}</span></p>
          <p className="rounded-2xl border border-slate-200 bg-slate-50 p-3 text-xs font-semibold text-slate-500">{t('publications.fields.publishedAt')}<span className="mt-1 block font-mono text-xs font-semibold text-slate-900">{publication.published_at}</span></p>
          <p className="rounded-2xl border border-slate-200 bg-slate-50 p-3 text-xs font-semibold text-slate-500">{t('publications.fields.publishedBy')}<span className="mt-1 block text-sm font-semibold text-slate-900">{publication.published_by}</span></p>
        </div>
      ) : null}

      <div className="rounded-2xl border border-slate-200 bg-slate-50 p-4">
        <div className="flex items-start gap-3">
          <div className="flex h-9 w-9 shrink-0 items-center justify-center rounded-xl border border-slate-200 bg-white text-blue-700">
            <Send className="h-4 w-4" />
          </div>
          <div>
            <h4 className="text-sm font-semibold text-slate-900">{t('publications.checkpointTitle')}</h4>
            <p className="mt-1 text-sm leading-6 text-slate-600">{t('publications.checkpointDescription')}</p>
          </div>
        </div>
        <div className="mt-3 grid gap-3 md:grid-cols-2">
          <input value={publishedBy} onChange={(event) => setPublishedBy(event.target.value)} className="rounded-lg border border-slate-300 bg-white px-3 py-2 text-sm" placeholder={t('publications.fields.publishedBy')} />
          <input value={nodeDeviceId} onChange={(event) => setNodeDeviceId(event.target.value)} className="rounded-lg border border-slate-300 bg-white px-3 py-2 text-sm" placeholder={t('publications.fields.nodeDeviceId')} />
        </div>
        <button type="button" className="mt-3 inline-flex items-center gap-2 rounded-lg bg-slate-900 px-3 py-2 text-sm font-medium text-white disabled:opacity-50" disabled={!canPublish || submitting || !restaurantId || !publishedBy.trim()} onClick={() => { void handlePublish(); }}>
          <Send className="h-4 w-4" />
          {t('publications.publishAction')}
        </button>
        {submitError ? <div className="mt-3"><SafeErrorBanner error={submitError} /></div> : null}
      </div>
    </section>
  );
}

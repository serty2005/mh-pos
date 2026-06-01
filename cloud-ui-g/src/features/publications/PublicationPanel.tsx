import { useState } from 'react';
import { useI18n } from '../../shared/i18n/I18nProvider';
import EmptyState from '../../shared/ui/EmptyState';
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
    <section className="space-y-4 rounded-2xl border border-slate-200 bg-white p-6">
      <div className="flex items-center justify-between gap-2">
        <h3 className="text-base font-semibold text-slate-900">{t('publications.title')}</h3>
        <button type="button" className="rounded-lg border border-slate-300 px-3 py-1 text-xs text-slate-700" onClick={() => { void reload(); }}>{t('ui.retry')}</button>
      </div>

      {status === 'blocked' ? <SafeErrorBanner error={error} /> : null}
      {status === 'ready' && !publication ? (
        <EmptyState title={t('publications.emptyTitle')} description={t('publications.emptyDescription')} />
      ) : null}

      {publication ? (
        <div className="grid gap-2 text-sm text-slate-700 md:grid-cols-2">
          <p>{t('publications.fields.version')}: <span className="font-medium text-slate-900">{publication.version}</span></p>
          <p>{t('publications.fields.status')}: <span className="font-medium text-slate-900">{publication.status}</span></p>
          <p>{t('publications.fields.publishedAt')}: <span className="font-medium text-slate-900">{publication.published_at}</span></p>
          <p>{t('publications.fields.publishedBy')}: <span className="font-medium text-slate-900">{publication.published_by}</span></p>
        </div>
      ) : null}

      <div className="rounded-xl border border-slate-200 bg-slate-50 p-4">
        <p className="text-sm text-slate-700">{canPublish ? t('publications.manualReady') : t('publications.manualBlocked')}</p>
        <div className="mt-3 grid gap-3 md:grid-cols-2">
          <input value={publishedBy} onChange={(event) => setPublishedBy(event.target.value)} className="rounded-lg border border-slate-300 bg-white px-3 py-2 text-sm" placeholder={t('publications.fields.publishedBy')} />
          <input value={nodeDeviceId} onChange={(event) => setNodeDeviceId(event.target.value)} className="rounded-lg border border-slate-300 bg-white px-3 py-2 text-sm" placeholder={t('publications.fields.nodeDeviceId')} />
        </div>
        <button type="button" className="mt-3 rounded-lg bg-slate-900 px-3 py-2 text-sm font-medium text-white disabled:opacity-50" disabled={!canPublish || submitting || !restaurantId || !publishedBy.trim()} onClick={() => { void handlePublish(); }}>
          {t('publications.publishAction')}
        </button>
        {submitError ? <div className="mt-3"><SafeErrorBanner error={submitError} /></div> : null}
      </div>
    </section>
  );
}

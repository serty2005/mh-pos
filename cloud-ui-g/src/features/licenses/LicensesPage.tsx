import { useEffect, useMemo, useState } from 'react';
import { RefreshCw, ShieldCheck } from 'lucide-react';
import { getEntitlements } from '../../shared/api/endpoints';
import type { EntitlementSnapshot } from '../../shared/api/schemas';
import { useI18n } from '../../shared/i18n/I18nProvider';
import EmptyState from '../../shared/ui/EmptyState';
import PanelHeader from '../../shared/ui/PanelHeader';
import SafeErrorBanner from '../../shared/ui/SafeErrorBanner';
import { formatIsoDateTime } from '../../shared/utils/format';

type Status = 'loading' | 'ready' | 'blocked';

const moduleIds = [
  'table-mode',
  'telegram-worker',
  'kitchen-space',
  'waiter-space',
  'checker-flow',
  'warehouse-mode',
] as const;

export default function LicensesPage() {
  const { t } = useI18n();
  const [snapshot, setSnapshot] = useState<EntitlementSnapshot | null>(null);
  const [status, setStatus] = useState<Status>('loading');
  const [error, setError] = useState<unknown>(null);

  const reload = async () => {
    setStatus('loading');
    setError(null);
    try {
      const next = await getEntitlements();
      setSnapshot(next);
      setStatus('ready');
    } catch (nextError) {
      setSnapshot(null);
      setStatus('blocked');
      setError(nextError);
    }
  };

  useEffect(() => {
    void reload();
  }, []);

  const expiresAt = snapshot ? new Date(snapshot.expires_at).getTime() : 0;
  const isCurrent = Boolean(snapshot && snapshot.status === 'active' && expiresAt > Date.now());
  const enabledCount = useMemo(
    () => moduleIds.filter((moduleId) => snapshot?.entitlements[moduleId] === true).length,
    [snapshot],
  );

  return (
    <section className="space-y-5">
      <PanelHeader
        icon={ShieldCheck}
        title={t('licenses.pageTitle')}
        description={t('licenses.pageDescription')}
        action={(
          <button
            type="button"
            onClick={() => { void reload(); }}
            disabled={status === 'loading'}
            className="inline-flex items-center gap-2 rounded-xl border border-slate-300 px-3 py-2 text-xs font-semibold text-slate-700 disabled:opacity-50"
          >
            <RefreshCw className={status === 'loading' ? 'h-3.5 w-3.5 animate-spin' : 'h-3.5 w-3.5'} />
            {status === 'loading' ? t('edge.refreshing') : t('ui.retry')}
          </button>
        )}
      />

      {status === 'blocked' ? <SafeErrorBanner error={error} /> : null}
      {status === 'ready' && !snapshot ? (
        <EmptyState title={t('licenses.emptyTitle')} description={t('licenses.emptyDescription')} />
      ) : null}

      {snapshot ? (
        <>
          <div className="grid gap-3 md:grid-cols-2 xl:grid-cols-4">
            <p className="rounded-2xl border border-slate-200 bg-white p-4 text-xs font-semibold text-slate-500">
              {t('licenses.fields.status')}
              <span className={isCurrent ? 'mt-2 inline-flex rounded-full border border-emerald-200 bg-emerald-50 px-2.5 py-1 text-sm font-semibold text-emerald-700' : 'mt-2 inline-flex rounded-full border border-amber-200 bg-amber-50 px-2.5 py-1 text-sm font-semibold text-amber-700'}>
                {isCurrent ? t('licenses.active') : t('licenses.notActive')}
              </span>
            </p>
            <p className="rounded-2xl border border-slate-200 bg-white p-4 text-xs font-semibold text-slate-500">
              {t('licenses.fields.version')}
              <span className="mt-2 block font-mono text-lg font-semibold text-slate-950">{snapshot.version}</span>
            </p>
            <p className="rounded-2xl border border-slate-200 bg-white p-4 text-xs font-semibold text-slate-500">
              {t('licenses.fields.modules')}
              <span className="mt-2 block font-mono text-lg font-semibold text-slate-950">{enabledCount}/{moduleIds.length}</span>
            </p>
            <p className="rounded-2xl border border-slate-200 bg-white p-4 text-xs font-semibold text-slate-500">
              {t('licenses.fields.expiresAt')}
              <span className="mt-2 block font-mono text-xs font-semibold text-slate-950">{formatIsoDateTime(snapshot.expires_at)}</span>
            </p>
          </div>

          <section className="rounded-2xl border border-slate-200 bg-white p-5 sm:p-6">
            <div className="grid gap-4 lg:grid-cols-[0.8fr_1.2fr]">
              <div className="space-y-3 text-sm">
                <p className="text-xs font-semibold text-slate-500">{t('licenses.fields.tenant')}</p>
                <p className="break-all font-mono text-sm font-semibold text-slate-900">{snapshot.tenant_id}</p>
                <p className="pt-2 text-xs font-semibold text-slate-500">{t('licenses.fields.server')}</p>
                <p className="break-all font-mono text-sm font-semibold text-slate-900">{snapshot.server_id}</p>
                <p className="pt-2 text-xs font-semibold text-slate-500">{t('licenses.fields.issuedAt')}</p>
                <p className="font-mono text-sm text-slate-700">{formatIsoDateTime(snapshot.issued_at)}</p>
              </div>

              <div className="grid gap-2 sm:grid-cols-2">
                {moduleIds.map((moduleId) => {
                  const enabled = snapshot.entitlements[moduleId] === true;
                  return (
                    <div
                      key={moduleId}
                      className={enabled ? 'rounded-xl border border-emerald-200 bg-emerald-50 p-3' : 'rounded-xl border border-slate-200 bg-slate-50 p-3'}
                    >
                      <p className="font-mono text-xs font-semibold text-slate-950">{moduleId}</p>
                      <p className={enabled ? 'mt-1 text-xs font-semibold text-emerald-700' : 'mt-1 text-xs font-semibold text-slate-500'}>
                        {enabled ? t('licenses.enabled') : t('licenses.disabled')}
                      </p>
                    </div>
                  );
                })}
              </div>
            </div>
          </section>
        </>
      ) : null}
    </section>
  );
}

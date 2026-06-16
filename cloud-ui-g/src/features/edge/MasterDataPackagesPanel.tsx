import { useCallback, useEffect, useMemo, useState } from 'react';
import { DatabaseZap, RefreshCw } from 'lucide-react';
import { getMasterDataPackage } from '../../shared/api/endpoints';
import type { MasterDataPackage } from '../../shared/api/schemas';
import { useI18n } from '../../shared/i18n/I18nProvider';
import { formatIsoDateTime } from '../../shared/utils/format';
import EmptyState from '../../shared/ui/EmptyState';
import SafeErrorBanner from '../../shared/ui/SafeErrorBanner';

type MasterDataPackagesPanelProps = {
  nodeDeviceId: string;
};

const streams = ['restaurants', 'devices', 'staff', 'floor', 'catalog', 'menu', 'pricing_policy'] as const;

function payloadSummary(payload: unknown) {
  if (!payload || typeof payload !== 'object' || Array.isArray(payload)) return '';
  return Object.entries(payload)
    .map(([key, value]) => `${key}: ${Array.isArray(value) ? value.length : 1}`)
    .join(' · ');
}

export default function MasterDataPackagesPanel({ nodeDeviceId }: MasterDataPackagesPanelProps) {
  const { t } = useI18n();
  const [packages, setPackages] = useState<MasterDataPackage[]>([]);
  const [status, setStatus] = useState<'idle' | 'loading' | 'ready' | 'blocked'>('idle');
  const [error, setError] = useState<unknown>(null);

  const reload = useCallback(async () => {
    if (!nodeDeviceId) {
      setPackages([]);
      setStatus('idle');
      return;
    }
    setStatus('loading');
    setError(null);
    try {
      const results = await Promise.all(streams.map((stream) => getMasterDataPackage(stream, nodeDeviceId)));
      setPackages(results.filter((item): item is MasterDataPackage => Boolean(item)));
      setStatus('ready');
    } catch (nextError) {
      setStatus('blocked');
      setError(nextError);
    }
  }, [nodeDeviceId]);

  useEffect(() => {
    void reload();
  }, [reload]);

  const selectedTitle = useMemo(() => nodeDeviceId || t('edge.noDeviceSelected'), [nodeDeviceId, t]);

  return (
    <section className="space-y-4 rounded-2xl border border-slate-200 bg-white p-5 sm:p-6">
      <div className="flex flex-wrap items-start justify-between gap-3">
        <div className="flex items-start gap-3">
          <div className="flex h-10 w-10 shrink-0 items-center justify-center rounded-xl border border-slate-200 bg-slate-50 text-slate-600">
            <DatabaseZap className="h-4 w-4" />
          </div>
          <div className="min-w-0">
            <h3 className="text-lg font-semibold tracking-tight text-slate-950">{t('edge.packagesTitle')}</h3>
            <p className="mt-1 truncate font-mono text-xs text-slate-500">{selectedTitle}</p>
          </div>
        </div>
        <button type="button" onClick={() => { void reload(); }} className="inline-flex items-center gap-2 rounded-xl border border-slate-300 px-3 py-2 text-xs font-semibold" disabled={status === 'loading' || !nodeDeviceId}>
          <RefreshCw className={status === 'loading' ? 'h-3.5 w-3.5 animate-spin' : 'h-3.5 w-3.5'} />
          {status === 'loading' ? t('edge.refreshing') : t('ui.retry')}
        </button>
      </div>

      {status === 'blocked' ? <SafeErrorBanner error={error} /> : null}
      {status !== 'blocked' && !nodeDeviceId ? <EmptyState title={t('edge.noDeviceSelected')} description={t('edge.noDeviceSelectedDescription')} /> : null}
      {status === 'ready' && nodeDeviceId && packages.length === 0 ? (
        <EmptyState title={t('edge.emptyPackagesTitle')} description={t('edge.emptyPackagesDescription')} />
      ) : null}

      {packages.length > 0 ? (
        <div className="overflow-x-auto rounded-2xl border border-slate-200">
          <table className="w-full min-w-[720px] text-left text-xs">
            <thead>
              <tr className="bg-slate-50 text-slate-500">
                <th className="px-3 py-3">{t('edge.packages.stream')}</th>
                <th className="px-3 py-3">{t('edge.packages.version')}</th>
                <th className="px-3 py-3">{t('edge.packages.mode')}</th>
                <th className="px-3 py-3">{t('edge.packages.updated')}</th>
                <th className="px-3 py-3">{t('edge.packages.payload')}</th>
              </tr>
            </thead>
            <tbody>
              {packages.map((item) => (
                <tr key={item.stream_name} className="border-t border-slate-100 text-slate-800">
                  <td className="px-3 py-3 font-semibold">{item.stream_name}</td>
                  <td className="px-3 py-3 font-mono">{item.cloud_version}</td>
                  <td className="px-3 py-3">{item.sync_mode}</td>
                  <td className="px-3 py-3">{formatIsoDateTime(item.cloud_updated_at || item.updated_at)}</td>
                  <td className="max-w-80 truncate px-3 py-3 text-slate-600">{payloadSummary(item.payload_json) || t('edge.packages.payloadHidden')}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      ) : null}
    </section>
  );
}

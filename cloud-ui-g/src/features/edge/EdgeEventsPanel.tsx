import { formatIsoDateTime } from '../../shared/utils/format';
import { useI18n } from '../../shared/i18n/I18nProvider';
import { useEdgeEvents } from './useEdgeEvents';
import SafeErrorBanner from '../../shared/ui/SafeErrorBanner';
import EmptyState from '../../shared/ui/EmptyState';
import { ListChecks, RefreshCw } from 'lucide-react';

type EdgeEventsPanelProps = {
  restaurantId: string;
  deviceId: string;
};

export default function EdgeEventsPanel({ restaurantId, deviceId }: EdgeEventsPanelProps) {
  const { t } = useI18n();
  const { events, status, error, reload } = useEdgeEvents(restaurantId, deviceId);

  return (
    <section className="space-y-4 rounded-2xl border border-slate-200 bg-white p-5 sm:p-6">
      <div className="flex flex-wrap items-start justify-between gap-3">
        <div className="flex items-start gap-3">
          <div className="flex h-10 w-10 shrink-0 items-center justify-center rounded-xl border border-slate-200 bg-slate-50 text-slate-600">
            <ListChecks className="h-4 w-4" />
          </div>
          <div>
            <h3 className="text-lg font-semibold tracking-tight text-slate-950">{t('edge.eventsTitle')}</h3>
            <p className="mt-1 truncate font-mono text-xs text-slate-500">{deviceId || t('edge.noDeviceSelected')}</p>
          </div>
        </div>
        <button type="button" onClick={() => { void reload(); }} className="inline-flex items-center gap-2 rounded-xl border border-slate-300 px-3 py-2 text-xs font-semibold" disabled={status === 'loading'}>
          <RefreshCw className={status === 'loading' ? 'h-3.5 w-3.5 animate-spin' : 'h-3.5 w-3.5'} />
          {status === 'loading' ? t('edge.refreshing') : t('ui.retry')}
        </button>
      </div>

      {status === 'blocked' ? <SafeErrorBanner error={error} /> : null}
      {status !== 'blocked' && !deviceId ? <EmptyState title={t('edge.noDeviceSelected')} description={t('edge.noDeviceSelectedDescription')} /> : null}
      {status === 'ready' && events.length === 0 ? <EmptyState title={t('edge.emptyEventsTitle')} description={t('edge.emptyEventsDescription')} /> : null}

      {events.length > 0 ? (
        <div className="overflow-x-auto rounded-2xl border border-slate-200">
          <table className="w-full min-w-[760px] text-left text-xs">
            <thead>
              <tr className="text-slate-500">
                <th className="px-3 py-3">{t('edge.events.receiptId')}</th>
                <th className="px-3 py-3">{t('edge.events.eventType')}</th>
                <th className="px-3 py-3">{t('edge.events.deviceId')}</th>
                <th className="px-3 py-3">{t('edge.events.aggregate')}</th>
                <th className="px-3 py-3">{t('edge.events.occurredAt')}</th>
                <th className="px-3 py-3">{t('edge.events.receivedAt')}</th>
                <th className="px-3 py-3">{t('edge.events.payloadHash')}</th>
              </tr>
            </thead>
            <tbody>
              {events.map((event) => (
                <tr key={`${event.cloud_receipt_id}-${event.event_id}`} className="border-t border-slate-100 text-slate-800">
                  <td className="px-3 py-3 font-mono">{event.cloud_receipt_id}</td>
                  <td className="px-3 py-3">{event.event_type}</td>
                  <td className="px-3 py-3 font-mono">{event.device_id}</td>
                  <td className="px-3 py-3">{event.aggregate_type}:{event.aggregate_id}</td>
                  <td className="px-3 py-3">{formatIsoDateTime(event.occurred_at)}</td>
                  <td className="px-3 py-3">{formatIsoDateTime(event.cloud_received_at)}</td>
                  <td className="max-w-72 truncate px-3 py-3 font-mono">{event.raw_payload_sha256_hex}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      ) : null}
    </section>
  );
}

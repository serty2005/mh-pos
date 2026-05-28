import { formatIsoDateTime } from '../../shared/utils/format';
import { useI18n } from '../../shared/i18n/I18nProvider';
import { useEdgeEvents } from './useEdgeEvents';
import SafeErrorBanner from '../../shared/ui/SafeErrorBanner';
import EmptyState from '../../shared/ui/EmptyState';

type EdgeEventsPanelProps = {
  restaurantId: string;
};

export default function EdgeEventsPanel({ restaurantId }: EdgeEventsPanelProps) {
  const { t } = useI18n();
  const { events, status, error, reload } = useEdgeEvents(restaurantId);

  return (
    <section className="space-y-3 rounded-2xl border border-slate-200 bg-white p-5">
      <div className="flex items-center justify-between">
        <h3 className="text-base font-semibold text-slate-900">{t('edge.eventsTitle')}</h3>
        <button type="button" onClick={() => { void reload(); }} className="rounded-lg border border-slate-300 px-3 py-1 text-xs" disabled={status === 'loading'}>
          {status === 'loading' ? t('edge.refreshing') : t('ui.retry')}
        </button>
      </div>

      {status === 'blocked' ? <SafeErrorBanner error={error} /> : null}
      {status === 'ready' && events.length === 0 ? <EmptyState title={t('edge.emptyEventsTitle')} description={t('edge.emptyEventsDescription')} /> : null}

      {events.length > 0 ? (
        <div className="overflow-x-auto">
          <table className="w-full min-w-[760px] text-left text-xs">
            <thead>
              <tr className="text-slate-500">
                <th className="px-2 py-2">{t('edge.events.receiptId')}</th>
                <th className="px-2 py-2">{t('edge.events.eventType')}</th>
                <th className="px-2 py-2">{t('edge.events.deviceId')}</th>
                <th className="px-2 py-2">{t('edge.events.aggregate')}</th>
                <th className="px-2 py-2">{t('edge.events.occurredAt')}</th>
                <th className="px-2 py-2">{t('edge.events.receivedAt')}</th>
                <th className="px-2 py-2">{t('edge.events.payloadHash')}</th>
              </tr>
            </thead>
            <tbody>
              {events.map((event) => (
                <tr key={`${event.cloud_receipt_id}-${event.event_id}`} className="border-t border-slate-100 text-slate-800">
                  <td className="px-2 py-2 font-mono">{event.cloud_receipt_id}</td>
                  <td className="px-2 py-2">{event.event_type}</td>
                  <td className="px-2 py-2 font-mono">{event.device_id}</td>
                  <td className="px-2 py-2">{event.aggregate_type}:{event.aggregate_id}</td>
                  <td className="px-2 py-2">{formatIsoDateTime(event.occurred_at)}</td>
                  <td className="px-2 py-2">{formatIsoDateTime(event.cloud_received_at)}</td>
                  <td className="px-2 py-2 font-mono">{event.raw_payload_sha256_hex}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      ) : null}
    </section>
  );
}

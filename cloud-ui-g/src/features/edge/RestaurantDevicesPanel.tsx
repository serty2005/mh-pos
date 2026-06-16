import { Cpu, RefreshCw, Wifi } from 'lucide-react';
import type { RestaurantEdgeNode } from '../../shared/api/schemas';
import { useI18n } from '../../shared/i18n/I18nProvider';
import { formatIsoDateTime } from '../../shared/utils/format';
import EmptyState from '../../shared/ui/EmptyState';

type RestaurantDevicesPanelProps = {
  devices: RestaurantEdgeNode[];
  selectedDeviceId: string;
  onSelectDevice: (nodeDeviceId: string) => void;
  onRefresh: () => Promise<void>;
  refreshLoading: boolean;
};

export default function RestaurantDevicesPanel({
  devices,
  selectedDeviceId,
  onSelectDevice,
  onRefresh,
  refreshLoading,
}: RestaurantDevicesPanelProps) {
  const { t } = useI18n();

  return (
    <section className="space-y-4 rounded-2xl border border-slate-200 bg-white p-5 sm:p-6">
      <div className="flex flex-wrap items-start justify-between gap-3">
        <div className="flex items-start gap-3">
          <div className="flex h-10 w-10 shrink-0 items-center justify-center rounded-xl border border-blue-100 bg-blue-50 text-blue-700">
            <Wifi className="h-4 w-4" />
          </div>
          <div>
            <h3 className="text-lg font-semibold tracking-tight text-slate-950">{t('edge.restaurantDevicesTitle')}</h3>
            <p className="mt-1 text-sm leading-6 text-slate-600">{t('edge.restaurantDevicesHint')}</p>
          </div>
        </div>
        <button type="button" onClick={() => { void onRefresh(); }} className="inline-flex items-center gap-2 rounded-xl border border-slate-300 px-3 py-2 text-xs font-semibold" disabled={refreshLoading}>
          <RefreshCw className={refreshLoading ? 'h-3.5 w-3.5 animate-spin' : 'h-3.5 w-3.5'} />
          {refreshLoading ? t('edge.refreshing') : t('ui.retry')}
        </button>
      </div>

      {devices.length === 0 ? (
        <EmptyState title={t('edge.emptyRestaurantDevicesTitle')} description={t('edge.emptyRestaurantDevicesDescription')} />
      ) : (
        <div className="space-y-3">
          {devices.map((device) => {
            const selected = selectedDeviceId === device.node_device_id;
            return (
              <button
                key={device.node_device_id}
                type="button"
                onClick={() => onSelectDevice(device.node_device_id)}
                className={[
                  'flex w-full items-start justify-between gap-3 rounded-2xl border p-4 text-left text-sm transition-colors',
                  selected ? 'border-blue-500 bg-blue-50/60' : 'border-slate-200 bg-slate-50/40 hover:border-slate-300',
                ].join(' ')}
              >
                <span className="flex min-w-0 gap-3">
                  <span className="flex h-9 w-9 shrink-0 items-center justify-center rounded-xl border border-slate-200 bg-white text-slate-700">
                    <Cpu className="h-4 w-4" />
                  </span>
                  <span className="min-w-0">
                    <span className="block truncate font-semibold text-slate-900">{device.display_name || device.node_device_id}</span>
                    <span className="mt-1 block truncate font-mono text-xs text-slate-500">{device.node_device_id}</span>
                    <span className="mt-2 block text-xs text-slate-500">
                      {t('edge.lastSeen')}: {device.last_seen_at ? formatIsoDateTime(device.last_seen_at) : t('edge.notReported')}
                    </span>
                  </span>
                </span>
                <span className="shrink-0 rounded-full border border-emerald-200 bg-emerald-50 px-2 py-1 text-xs font-semibold text-emerald-700">
                  {t(`edge.status.${device.status}`)}
                </span>
              </button>
            );
          })}
        </div>
      )}
    </section>
  );
}

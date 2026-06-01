import { useI18n } from '../../shared/i18n/I18nProvider';
import type { UnassignedEdgeNode } from '../../shared/api/schemas';
import SafeErrorBanner from '../../shared/ui/SafeErrorBanner';
import EmptyState from '../../shared/ui/EmptyState';

type UnassignedDevicesPanelProps = {
  devices: UnassignedEdgeNode[];
  status: 'idle' | 'loading' | 'ready' | 'blocked';
  error: unknown;
  selectedDeviceId: string;
  onSelectDevice: (nodeDeviceId: string) => void;
  onAssign: () => Promise<void>;
  onRefresh: () => Promise<void>;
  refreshLoading: boolean;
  assignLoading: boolean;
  restaurantSelected: boolean;
  assignmentStatus: string;
  actionError: unknown;
};

export default function UnassignedDevicesPanel({
  devices,
  status,
  error,
  selectedDeviceId,
  onSelectDevice,
  onAssign,
  onRefresh,
  refreshLoading,
  assignLoading,
  restaurantSelected,
  assignmentStatus,
  actionError,
}: UnassignedDevicesPanelProps) {
  const { t } = useI18n();

  return (
    <section className="space-y-3 rounded-2xl border border-slate-200 bg-white p-5">
      <div className="flex items-center justify-between">
        <h3 className="text-base font-semibold text-slate-900">{t('edge.devicesTitle')}</h3>
        <button type="button" onClick={() => { void onRefresh(); }} className="rounded-lg border border-slate-300 px-3 py-1 text-xs" disabled={refreshLoading}>
          {refreshLoading ? t('edge.refreshing') : t('ui.retry')}
        </button>
      </div>

      {status === 'blocked' ? <SafeErrorBanner error={error} /> : null}
      {actionError ? <SafeErrorBanner error={actionError} /> : null}
      {status === 'ready' && devices.length === 0 ? (
        <EmptyState title={t('edge.emptyDevicesTitle')} description={t('edge.emptyDevicesDescription')} />
      ) : null}

      {devices.length > 0 ? (
        <div className="grid gap-2">
          {devices.map((device) => (
            <label key={device.node_device_id} className="flex cursor-pointer items-center justify-between rounded-lg border border-slate-200 p-2 text-sm">
              <span>
                <span className="font-medium text-slate-900">{device.display_name || device.node_device_id}</span>
                <span className="ml-2 text-xs text-slate-500">{device.node_device_id}</span>
              </span>
              <span className="flex items-center gap-2">
                <span className="rounded bg-slate-100 px-2 py-1 text-xs text-slate-700">{t(`edge.status.${device.status}`)}</span>
                <input type="radio" name="edge-device" checked={selectedDeviceId === device.node_device_id} onChange={() => onSelectDevice(device.node_device_id)} />
              </span>
            </label>
          ))}
        </div>
      ) : null}

      <button
        type="button"
        onClick={() => { void onAssign(); }}
        className="rounded-lg bg-slate-900 px-3 py-2 text-sm font-medium text-white disabled:opacity-50"
        disabled={!restaurantSelected || !selectedDeviceId || assignLoading}
      >
        {assignLoading ? t('edge.refreshing') : t('edge.assignAction')}
      </button>
      {assignmentStatus ? <p className="text-xs text-slate-600">{t('edge.assignmentStatus')}: {t(`edge.status.${assignmentStatus}`)}</p> : null}
    </section>
  );
}

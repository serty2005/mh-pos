import { useI18n } from '../../shared/i18n/I18nProvider';
import type { UnassignedEdgeNode } from '../../shared/api/schemas';
import SafeErrorBanner from '../../shared/ui/SafeErrorBanner';
import EmptyState from '../../shared/ui/EmptyState';
import { Cpu, RefreshCw, ShieldCheck } from 'lucide-react';

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
  const selectedPendingDevice = devices.some((device) => device.node_device_id === selectedDeviceId);

  return (
    <section className="space-y-4 rounded-2xl border border-slate-200 bg-white p-5 sm:p-6">
      <div className="flex flex-wrap items-start justify-between gap-3">
        <div className="flex items-start gap-3">
          <div className="flex h-10 w-10 shrink-0 items-center justify-center rounded-xl border border-slate-200 bg-slate-50 text-slate-600">
            <Cpu className="h-4 w-4" />
          </div>
          <h3 className="text-lg font-semibold tracking-tight text-slate-950">{t('edge.devicesTitle')}</h3>
        </div>
        <button type="button" onClick={() => { void onRefresh(); }} className="inline-flex items-center gap-2 rounded-xl border border-slate-300 px-3 py-2 text-xs font-semibold" disabled={refreshLoading}>
          <RefreshCw className={refreshLoading ? 'h-3.5 w-3.5 animate-spin' : 'h-3.5 w-3.5'} />
          {refreshLoading ? t('edge.refreshing') : t('ui.retry')}
        </button>
      </div>

      {status === 'blocked' ? <SafeErrorBanner error={error} /> : null}
      {actionError ? <SafeErrorBanner error={actionError} /> : null}
      {status === 'ready' && devices.length === 0 ? (
        <EmptyState title={t('edge.emptyDevicesTitle')} description={t('edge.emptyDevicesDescription')} />
      ) : null}

      {devices.length > 0 ? (
        <div className="grid gap-3 lg:grid-cols-2">
          {devices.map((device) => (
            <label key={device.node_device_id} className="flex cursor-pointer items-start justify-between gap-3 rounded-2xl border border-slate-200 p-4 text-sm">
              <span className="min-w-0">
                <span className="block truncate font-semibold text-slate-900">{device.display_name || device.node_device_id}</span>
                <span className="mt-1 block truncate font-mono text-xs text-slate-500">{device.node_device_id}</span>
              </span>
              <span className="flex shrink-0 items-center gap-2">
                <span className="rounded-full border border-slate-200 bg-slate-100 px-2 py-1 text-xs font-semibold text-slate-700">{t(`edge.status.${device.status}`)}</span>
                <input type="radio" name="edge-device" checked={selectedDeviceId === device.node_device_id} onChange={() => onSelectDevice(device.node_device_id)} />
              </span>
            </label>
          ))}
        </div>
      ) : null}

      <button
        type="button"
        onClick={() => { void onAssign(); }}
        className="inline-flex items-center gap-2 rounded-lg bg-slate-900 px-3 py-2 text-sm font-medium text-white disabled:opacity-50"
        disabled={!restaurantSelected || !selectedPendingDevice || assignLoading}
      >
        <ShieldCheck className="h-4 w-4" />
        {assignLoading ? t('edge.refreshing') : t('edge.assignAction')}
      </button>
      {assignmentStatus ? <p className="text-xs text-slate-600">{t('edge.assignmentStatus')}: {t(`edge.status.${assignmentStatus}`)}</p> : null}
    </section>
  );
}

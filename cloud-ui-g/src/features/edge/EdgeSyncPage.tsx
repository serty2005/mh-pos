import { useI18n } from '../../shared/i18n/I18nProvider';
import PairingCodePanel from './PairingCodePanel';
import UnassignedDevicesPanel from './UnassignedDevicesPanel';
import EdgeEventsPanel from './EdgeEventsPanel';
import { useEdgeDevices } from './useEdgeDevices';

type EdgeSyncPageProps = {
  restaurantId: string;
};

export default function EdgeSyncPage({ restaurantId }: EdgeSyncPageProps) {
  const { t } = useI18n();
  const {
    devices,
    status,
    error,
    selectedDeviceId,
    setSelectedDeviceId,
    reload,
    assign,
    requestPairingCode,
    assignLoading,
    pairingLoading,
    actionError,
    assignmentStatus,
    pairingCode,
    clearPairingCode,
  } = useEdgeDevices(restaurantId);

  return (
    <div className="space-y-4">
      <section className="rounded-2xl border border-slate-200 bg-white p-6">
        <h3 className="text-base font-semibold text-slate-900">{t('edge.pageTitle')}</h3>
        <p className="mt-1 text-sm text-slate-600">{t('edge.pageDescription')}</p>
      </section>

      <UnassignedDevicesPanel
        devices={devices}
        status={status}
        error={error}
        selectedDeviceId={selectedDeviceId}
        onSelectDevice={setSelectedDeviceId}
        onAssign={assign}
        onRefresh={reload}
        refreshLoading={status === 'loading'}
        assignLoading={assignLoading}
        restaurantSelected={Boolean(restaurantId)}
        assignmentStatus={assignmentStatus}
        actionError={actionError}
      />

      <PairingCodePanel
        restaurantSelected={Boolean(restaurantId)}
        loading={pairingLoading}
        pairingCode={pairingCode}
        onGenerate={requestPairingCode}
        onClear={clearPairingCode}
        actionError={actionError}
      />

      <EdgeEventsPanel restaurantId={restaurantId} />
    </div>
  );
}

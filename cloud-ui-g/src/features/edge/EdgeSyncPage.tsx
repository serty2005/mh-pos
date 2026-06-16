import { RadioTower } from 'lucide-react';
import { useI18n } from '../../shared/i18n/I18nProvider';
import PairingCodePanel from './PairingCodePanel';
import UnassignedDevicesPanel from './UnassignedDevicesPanel';
import RestaurantDevicesPanel from './RestaurantDevicesPanel';
import EdgeEventsPanel from './EdgeEventsPanel';
import MasterDataPackagesPanel from './MasterDataPackagesPanel';
import { useEdgeDevices } from './useEdgeDevices';

type EdgeSyncPageProps = {
  restaurantId: string;
};

export default function EdgeSyncPage({ restaurantId }: EdgeSyncPageProps) {
  const { t } = useI18n();
  const {
    devices,
    restaurantDevices,
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
      <section className="rounded-2xl border border-slate-200 bg-white p-5 sm:p-6">
        <div className="flex items-start gap-3">
          <div className="flex h-10 w-10 shrink-0 items-center justify-center rounded-xl border border-blue-100 bg-blue-50 text-blue-700">
            <RadioTower className="h-4 w-4" />
          </div>
          <div>
            <h3 className="text-lg font-semibold tracking-tight text-slate-950">{t('edge.pageTitle')}</h3>
            <p className="mt-1 max-w-3xl text-sm leading-6 text-slate-600">{t('edge.pageDescription')}</p>
          </div>
        </div>
      </section>

      <div className="grid gap-4 xl:grid-cols-[1.1fr_0.9fr]">
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

        <RestaurantDevicesPanel
          devices={restaurantDevices}
          selectedDeviceId={selectedDeviceId}
          onSelectDevice={setSelectedDeviceId}
          onRefresh={reload}
          refreshLoading={status === 'loading'}
        />
      </div>

      <PairingCodePanel
        restaurantSelected={Boolean(restaurantId)}
        loading={pairingLoading}
        pairingCode={pairingCode}
        onGenerate={requestPairingCode}
        onClear={clearPairingCode}
        actionError={actionError}
      />

      <div className="grid gap-4 xl:grid-cols-[1.1fr_0.9fr]">
        <EdgeEventsPanel restaurantId={restaurantId} deviceId={selectedDeviceId} />
        <MasterDataPackagesPanel nodeDeviceId={selectedDeviceId} />
      </div>
    </div>
  );
}

import React, { useMemo, useState } from 'react';
import { Cloud, KeyRound, RefreshCw, ShieldCheck, Wifi } from 'lucide-react';

import { usePOS } from '../context/POSContext';
import { t } from '../shared/i18n';
import { PosBanner, PosButton, PosSelectableChip } from '../shared/ui';

export const PairingScreen: React.FC = () => {
  const {
    authSnapshot,
    provisioningStatus,
    provisioningLoading,
    provisioningError,
    refreshProvisioningStatus,
    registerCloudProvisioning,
    pairViaLicense,
  } = usePOS();
  const [mode, setMode] = useState<'cloud' | 'license'>('cloud');
  const [licenseCode, setLicenseCode] = useState('');

  const statusLabel = useMemo(() => {
    const status = provisioningStatus?.status ?? 'not_configured';
    return t.pair.status[status];
  }, [provisioningStatus?.status]);

  const submitLicense = () => {
    const normalized = licenseCode.trim().toUpperCase();
    if (!normalized) return;
    void pairViaLicense(normalized);
  };

  return (
    <div className="fixed inset-0 z-50 flex flex-col lg:flex-row bg-[var(--pos-bg)] text-[var(--pos-text-primary)]">
      <section className="flex-1 flex flex-col justify-between p-8 md:p-12 border-b lg:border-b-0 lg:border-r border-[var(--pos-border)] bg-[var(--pos-surface-raised)]">
        <div className="space-y-8">
          <div className="flex items-center gap-3">
            <div className="w-10 h-10 border border-[var(--pos-border-strong)] flex items-center justify-center bg-[var(--pos-surface)]">
              <ShieldCheck className="w-4 h-4 text-[var(--pos-text-secondary)]" />
            </div>
            <span className="font-sans text-base text-[var(--pos-text-secondary)] font-semibold">MyHoreca POS</span>
          </div>

          <div className="max-w-xl space-y-3">
            <h1 className="font-sans text-2xl md:text-4xl font-semibold tracking-normal text-[var(--pos-text-secondary)]">
              {t.pair.title}
            </h1>
            <p className="font-sans text-sm text-[var(--pos-text-muted)] leading-relaxed max-w-[60ch]">
              {t.pair.subtitle}
            </p>
          </div>
        </div>

        <div className="font-mono text-[10px] text-[var(--pos-text-muted)] uppercase tracking-widest space-y-1">
          <div>{t.pair.clientDevice}: {authSnapshot.clientDeviceId}</div>
          <div>{t.pair.nodeDevice}: {provisioningStatus?.node_device_id || authSnapshot.nodeDeviceId || t.common.loading}</div>
        </div>
      </section>

      <section className="flex-1 flex items-center justify-center p-6 md:p-10 bg-[var(--pos-surface)] overflow-y-auto">
        <div className="w-full max-w-[560px] border border-[var(--pos-border)] bg-[var(--pos-bg)]">
          <div className="grid grid-cols-2 border-b border-[var(--pos-border)]">
            <PosSelectableChip
              id="pair-mode-cloud-btn"
              active={mode === 'cloud'}
              onClick={() => setMode('cloud')}
              className="h-14 flex items-center justify-center gap-2 border-y-0 border-l-0 border-r border-[var(--pos-border)]"
            >
              <Cloud className="w-4 h-4" />
              {t.pair.cloudMode}
            </PosSelectableChip>
            <PosSelectableChip
              id="pair-mode-license-btn"
              active={mode === 'license'}
              onClick={() => setMode('license')}
              className="h-14 flex items-center justify-center gap-2 border-0"
            >
              <KeyRound className="w-4 h-4" />
              {t.pair.licenseMode}
            </PosSelectableChip>
          </div>

          <div className="p-6 space-y-6">
            {provisioningError && (
              <PosBanner type="danger" message={provisioningError} />
            )}

            <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
              <StatusField label={t.pair.nodeDevice} value={provisioningStatus?.node_device_id || authSnapshot.nodeDeviceId || t.common.loading} />
              <StatusField label={t.pair.restaurant} value={provisioningStatus?.restaurant_id || authSnapshot.restaurantId || t.common.none} />
              <StatusField label={t.pair.cloudUrl} value={provisioningStatus?.cloud_url || t.pair.cloudUrlEmpty} />
              <StatusField label={t.pair.statusTitle} value={statusLabel} strong />
            </div>

            {provisioningStatus?.paired ? (
              <PosBanner type="success" message={t.pair.paired} />
            ) : (
              <PosBanner type="warning" message={t.pair.notPaired} />
            )}

            {mode === 'cloud' ? (
              <div className="space-y-4">
                <p className="font-sans text-sm text-[var(--pos-text-secondary)] leading-relaxed">
                  {t.pair.pendingCopy}
                </p>
                <div className="flex flex-col sm:flex-row gap-3">
                  <PosButton
                    id="pair-register-cloud-btn"
                    variant="primary"
                    size="md"
                    fullWidth
                    disabled={provisioningLoading}
                    onClick={() => void registerCloudProvisioning(provisioningStatus?.cloud_url ?? '')}
                    icon={<Wifi className="w-4 h-4" />}
                  >
                    {t.pair.retryRegister}
                  </PosButton>
                  <PosButton
                    id="pair-refresh-status-btn"
                    variant="secondary"
                    size="md"
                    fullWidth
                    disabled={provisioningLoading}
                    onClick={() => void refreshProvisioningStatus()}
                    icon={<RefreshCw className="w-4 h-4" />}
                  >
                    {t.pair.refreshStatus}
                  </PosButton>
                </div>
              </div>
            ) : (
              <div className="space-y-4">
                <label className="block space-y-2">
                  <span className="font-mono text-xs font-bold uppercase text-[var(--pos-text-secondary)]">{t.pair.licenseCode}</span>
                  <input
                    id="pair-license-code-input"
                    value={licenseCode}
                    onChange={(event) => setLicenseCode(event.target.value.toUpperCase())}
                    onKeyDown={(event) => {
                      if (event.key === 'Enter') submitLicense();
                    }}
                    placeholder={t.pair.licensePlaceholder}
                    className="w-full h-12 border border-[var(--pos-border)] bg-[var(--pos-surface)] px-4 font-mono text-sm uppercase tracking-wider text-[var(--pos-text-primary)] focus:outline-none focus:border-[var(--pos-border-strong)]"
                    autoComplete="off"
                  />
                </label>
                <PosButton
                  id="pair-license-submit-btn"
                  variant="primary"
                  size="md"
                  fullWidth
                  disabled={provisioningLoading || !licenseCode.trim()}
                  onClick={submitLicense}
                  icon={<KeyRound className="w-4 h-4" />}
                >
                  {t.pair.pairByLicense}
                </PosButton>
              </div>
            )}
          </div>
        </div>
      </section>
    </div>
  );
};

function StatusField({ label, value, strong = false }: { label: string; value: string; strong?: boolean }) {
  return (
    <div className="border border-[var(--pos-border)] bg-[var(--pos-surface)] p-3 min-w-0">
      <div className="font-mono text-[9px] text-[var(--pos-text-muted)] uppercase tracking-widest mb-1">
        {label}
      </div>
      <div className={`font-mono text-xs truncate ${strong ? 'font-black text-[var(--pos-text-primary)] uppercase' : 'font-semibold text-[var(--pos-text-secondary)]'}`}>
        {value}
      </div>
    </div>
  );
}

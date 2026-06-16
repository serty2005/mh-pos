import { useI18n } from '../../shared/i18n/I18nProvider';
import type { PairingCodeResult } from '../../shared/api/schemas';
import SafeErrorBanner from '../../shared/ui/SafeErrorBanner';
import { KeyRound, RefreshCw, X } from 'lucide-react';

type PairingCodePanelProps = {
  restaurantSelected: boolean;
  loading: boolean;
  pairingCode: PairingCodeResult | null;
  onGenerate: () => Promise<void>;
  onClear: () => void;
  actionError: unknown;
};

export default function PairingCodePanel({
  restaurantSelected,
  loading,
  pairingCode,
  onGenerate,
  onClear,
  actionError,
}: PairingCodePanelProps) {
  const { t } = useI18n();

  return (
    <section className="space-y-4 rounded-2xl border border-slate-200 bg-white p-5 sm:p-6">
      <div className="flex items-start gap-3">
        <div className="flex h-10 w-10 shrink-0 items-center justify-center rounded-xl border border-blue-100 bg-blue-50 text-blue-700">
          <KeyRound className="h-4 w-4" />
        </div>
        <div>
          <h3 className="text-lg font-semibold tracking-tight text-slate-950">{t('edge.pairingTitle')}</h3>
          <p className="mt-1 text-sm leading-6 text-slate-600">{t('edge.pairingHint')}</p>
        </div>
      </div>
      {actionError ? <SafeErrorBanner error={actionError} /> : null}
      <button type="button" onClick={() => { void onGenerate(); }} className="inline-flex items-center gap-2 rounded-lg bg-slate-900 px-3 py-2 text-sm font-medium text-white disabled:opacity-50" disabled={!restaurantSelected || loading}>
        <RefreshCw className={loading ? 'h-4 w-4 animate-spin' : 'h-4 w-4'} />
        {loading ? t('edge.refreshing') : t('edge.generatePairing')}
      </button>

      {pairingCode ? (
        <div className="rounded-2xl border border-emerald-200 bg-emerald-50 p-4">
          <p className="text-xs text-emerald-700">{t('edge.pairingVisibleNow')}</p>
          <p className="mt-2 break-all font-mono text-2xl font-semibold tracking-tight text-emerald-950">{pairingCode.pairing_code}</p>
          <p className="mt-1 break-all font-mono text-xs text-emerald-800">{t('edge.pairingId')}: {pairingCode.pairing_id}</p>
          <p className="mt-1 text-xs text-emerald-800">{t('edge.expiresAt')}: {pairingCode.expires_at}</p>
          <button type="button" className="mt-3 inline-flex items-center gap-1.5 rounded border border-emerald-300 px-2 py-1 text-xs text-emerald-900" onClick={onClear}>
            <X className="h-3.5 w-3.5" />
            {t('edge.hidePairing')}
          </button>
        </div>
      ) : null}
    </section>
  );
}

import { useI18n } from '../../shared/i18n/I18nProvider';
import type { PairingCodeResult } from '../../shared/api/schemas';
import SafeErrorBanner from '../../shared/ui/SafeErrorBanner';

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
    <section className="space-y-3 rounded-2xl border border-slate-200 bg-white p-5">
      <h3 className="text-base font-semibold text-slate-900">{t('edge.pairingTitle')}</h3>
      <p className="text-sm text-slate-600">{t('edge.pairingHint')}</p>
      {actionError ? <SafeErrorBanner error={actionError} /> : null}
      <button type="button" onClick={() => { void onGenerate(); }} className="rounded-lg bg-slate-900 px-3 py-2 text-sm font-medium text-white disabled:opacity-50" disabled={!restaurantSelected || loading}>
        {loading ? t('edge.refreshing') : t('edge.generatePairing')}
      </button>

      {pairingCode ? (
        <div className="rounded-lg border border-emerald-200 bg-emerald-50 p-3">
          <p className="text-xs text-emerald-700">{t('edge.pairingVisibleNow')}</p>
          <p className="mt-1 font-mono text-lg text-emerald-900">{pairingCode.pairing_code}</p>
          <p className="mt-1 text-xs text-emerald-800">{t('edge.expiresAt')}: {pairingCode.expires_at}</p>
          <button type="button" className="mt-2 rounded border border-emerald-300 px-2 py-1 text-xs text-emerald-900" onClick={onClear}>{t('edge.hidePairing')}</button>
        </div>
      ) : null}
    </section>
  );
}

import React, { useState, useEffect } from 'react';
import { t } from '../../shared/i18n';
import { PosButton, PosDialog, PosFormRow, PosSelectableChip } from '../../shared/ui';
import { ShieldAlert, Unlock } from 'lucide-react';

interface PrecheckCancelDialogProps {
  isOpen: boolean;
  onClose: () => void;
  onConfirm: (pin: string, reason: string) => Promise<boolean>;
}

export const PrecheckCancelDialog: React.FC<PrecheckCancelDialogProps> = ({
  isOpen,
  onClose,
  onConfirm
}) => {
  const [pin, setPin] = useState<string>('');
  const [reason, setReason] = useState<string>('');
  const [errorMsg, setErrorMsg] = useState<string | null>(null);

  useEffect(() => {
    if (isOpen) {
      setPin('');
      setReason('');
      setErrorMsg(null);
    }
  }, [isOpen]);

  const handleSubmit = async () => {
    if (!pin) {
      setErrorMsg(t.modals.precheckPinRequired);
      return;
    }
    if (!reason || reason.trim().length < 4) {
      setErrorMsg(t.modals.precheckReasonRequired);
      return;
    }

    const success = await onConfirm(pin, reason);
    if (success) {
      onClose();
    } else {
      setErrorMsg(t.modals.precheckManagerDenied);
    }
  };

  const handleQuickReason = (text: string) => {
    setReason(text);
    setErrorMsg(null);
  };

  return (
    <PosDialog
      isOpen={isOpen}
      onClose={onClose}
      title={t.modals.precheckCancelTitle}
      footer={
        <>
          <PosButton variant="secondary" size="sm" onClick={onClose}>
            {t.common.cancel}
          </PosButton>
          <PosButton
            id="precheck-override-btn"
            variant="danger"
            size="sm"
            onClick={handleSubmit}
            icon={<Unlock className="w-4 h-4" />}
          >
            {t.modals.precheckCancelConfirm}
          </PosButton>
        </>
      }
    >
      <div className="flex flex-col gap-4 select-none">
        
        {/* Warning Indicator */}
        <div className="p-4 bg-amber-50 dark:bg-amber-950/20 border border-amber-200 dark:border-amber-900 text-amber-800 dark:text-amber-100 flex gap-3 text-xs md:text-sm">
          <ShieldAlert className="w-5 h-5 shrink-0 text-amber-500" />
          <div className="space-y-1">
            <span className="font-bold uppercase tracking-wider block font-mono text-xs">{t.modals.precheckCancelRequired}</span>
            <span>{t.modals.precheckCancelDesc}</span>
          </div>
        </div>

        {errorMsg && (
          <span className="font-sans text-xs font-bold text-[var(--pos-status-danger)] bg-red-50 dark:bg-red-950/20 py-1.5 px-4 block text-center uppercase tracking-wide">
            {errorMsg}
          </span>
        )}

        {/* PIN Entry Area */}
        <PosFormRow
          label={t.modals.precheckManagerPin}
          id="precheck-manager-pin"
        >
          <input
            id="precheck-manager-pin"
            type="password"
            maxLength={4}
            className="w-full h-12 border border-[var(--pos-border)] px-4 font-mono text-center text-lg tracking-widest bg-[var(--pos-surface)] text-[var(--pos-text-primary)] rounded-none outline-none focus:border-[var(--pos-border-strong)]"
            placeholder="••••"
            value={pin}
            onChange={(e) => {
              setPin(e.target.value.replace(/\D/g, ''));
              setErrorMsg(null);
            }}
          />
        </PosFormRow>

        {/* Cancellation Reason Entry Area */}
        <PosFormRow
          label={t.modals.precheckCancelReason}
          id="precheck-cancel-reason"
        >
          <input
            id="precheck-cancel-reason"
            type="text"
            className="w-full h-12 border border-[var(--pos-border)] px-4 font-sans text-sm bg-[var(--pos-surface)] text-[var(--pos-text-primary)] rounded-none outline-none focus:border-[var(--pos-border-strong)]"
            placeholder={t.modals.precheckCancelReasonPlaceholder}
            value={reason}
            onChange={(e) => {
              setReason(e.target.value);
              setErrorMsg(null);
            }}
          />
        </PosFormRow>

        {/* Quick Reasons Chips list */}
        <div className="space-y-1.5">
          <span className="font-mono text-[10px] font-bold uppercase tracking-widest text-[var(--pos-text-muted)]">{t.modals.precheckQuickReason}</span>
          <div className="flex flex-wrap gap-1.5">
            {t.modals.precheckQuickReasons.map((text) => (
              <PosSelectableChip
                key={text}
                id={`quick-reason-btn-${text.replace(/\s+/g, '-')}`}
                onClick={() => handleQuickReason(text)}
                className="h-8 font-mono text-[10px] px-2.5 py-1.5 tracking-tight"
              >
                {text}
              </PosSelectableChip>
            ))}
          </div>
        </div>

      </div>
    </PosDialog>
  );
};

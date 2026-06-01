import React, { useState, useEffect } from 'react';
import { t } from '../../shared/i18n';
import { PosBanner, PosButton, PosDialog, PosFormRow, PosSelectableChip } from '../../shared/ui';
import { Banknote, Check } from 'lucide-react';

interface CashDrawerEventDialogProps {
  isOpen: boolean;
  onClose: () => void;
  onSubmit: (type: 'in' | 'out', amount: number, reason: string) => void;
}

export const CashDrawerEventDialog: React.FC<CashDrawerEventDialogProps> = ({
  isOpen,
  onClose,
  onSubmit
}) => {
  const [type, setType] = useState<'in' | 'out'>('in');
  const [amount, setAmount] = useState<string>('');
  const [reason, setReason] = useState<string>('');
  const [errorMsg, setErrorMsg] = useState<string | null>(null);

  useEffect(() => {
    if (isOpen) {
      setType('in');
      setAmount('');
      setReason('');
      setErrorMsg(null);
    }
  }, [isOpen]);

  const handleReasonClick = (text: string) => {
    setReason(text);
    setErrorMsg(null);
  };

  const handleFormSubmit = () => {
    setErrorMsg(null);
    const numAmount = parseFloat(amount) || 0;
    
    if (numAmount <= 0) {
      setErrorMsg(t.modals.cashEventAmountRequired);
      return;
    }
    if (!reason || reason.trim().length < 4) {
      setErrorMsg(t.modals.cashEventReasonRequired);
      return;
    }

    onSubmit(type, numAmount, reason);
    onClose();
  };

  return (
    <PosDialog
      isOpen={isOpen}
      onClose={onClose}
      title={t.modals.cashEventTitle}
      footer={
        <>
          <PosButton variant="secondary" size="sm" onClick={onClose}>
            {t.common.cancel}
          </PosButton>
          <PosButton
            id="cash-event-submit-btn"
            variant="primary"
            size="sm"
            onClick={handleFormSubmit}
            icon={<Check className="w-4 h-4" />}
          >
            {t.modals.cashEventSubmit}
          </PosButton>
        </>
      }
    >
      <div className="flex flex-col gap-4 select-none">
        
        {errorMsg && (
          <PosBanner type="danger" message={errorMsg} />
        )}

        {/* Cash In vs Cash Out selection rows */}
        <div className="space-y-1.5">
          <span className="font-mono text-[10px] font-bold uppercase tracking-widest text-[var(--pos-text-muted)]">{t.modals.cashEventType}</span>
          <div className="grid grid-cols-2 gap-2">
            <PosSelectableChip
              id="cash-event-type-in-btn"
              active={type === 'in'}
              onClick={() => setType('in')}
              className="h-11 px-4"
            >
              {t.modals.cashEventInFull}
            </PosSelectableChip>
            <PosSelectableChip
              id="cash-event-type-out-btn"
              active={type === 'out'}
              onClick={() => setType('out')}
              className="h-11 px-4"
            >
              {t.modals.cashEventOutFull}
            </PosSelectableChip>
          </div>
        </div>

        {/* Amount input row */}
        <PosFormRow
          label={t.modals.cashEventAmount}
          id="cash-event-input-amount"
        >
          <input
            id="cash-event-input-amount"
            type="text"
            className="w-full h-12 border border-[var(--pos-border)] px-4 font-mono text-lg text-right bg-[var(--pos-surface)] text-[var(--pos-text-primary)] rounded-none outline-none focus:border-[var(--pos-border-strong)]"
            placeholder={t.modals.cashEventAmountPlaceholder}
            value={amount}
            onChange={(e) => {
              setAmount(e.target.value.replace(/[^\d.]/g, ''));
              setErrorMsg(null);
            }}
          />
        </PosFormRow>

        {/* Text justification reasons */}
        <PosFormRow
          label={t.modals.cashEventReason}
          id="cash-event-input-reason"
        >
          <input
            id="cash-event-input-reason"
            type="text"
            className="w-full h-12 border border-[var(--pos-border)] px-4 font-sans text-sm bg-[var(--pos-surface)] text-[var(--pos-text-primary)] rounded-none outline-none focus:border-[var(--pos-border-strong)]"
            placeholder={t.modals.cashEventReasonPlaceholder}
            value={reason}
            onChange={(e) => {
              setReason(e.target.value);
              setErrorMsg(null);
            }}
          />
        </PosFormRow>

        {/* Quick reasons hints list */}
        <div className="flex flex-wrap gap-1.5">
          {t.modals.cashEventQuickReasons.map((text) => (
            <PosSelectableChip
              key={text}
              id={`cash-reason-btn-${text.replace(/\s+/g, '-')}`}
              onClick={() => handleReasonClick(text)}
              className="h-8 font-mono text-[9px] px-2.5 py-1.5"
            >
              {text}
            </PosSelectableChip>
          ))}
        </div>

      </div>
    </PosDialog>
  );
};

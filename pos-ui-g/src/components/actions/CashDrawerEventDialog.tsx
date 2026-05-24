import React, { useState, useEffect } from 'react';
import { t } from '../../shared/i18n';
import { PosButton, PosDialog, PosFormRow } from '../../shared/ui';
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
      setErrorMsg('Укажите корректную сумму операции больше нуля.');
      return;
    }
    if (!reason || reason.trim().length < 4) {
      setErrorMsg('Укажите обоснование операции (минимум 4 символа).');
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
            Фондировать
          </PosButton>
        </>
      }
    >
      <div className="flex flex-col gap-4 select-none">
        
        {errorMsg && (
          <span className="font-sans text-xs font-bold text-[var(--pos-status-danger)] bg-red-50 dark:bg-red-950/20 py-1.5 px-4 block text-center uppercase">
            {errorMsg}
          </span>
        )}

        {/* Cash In vs Cash Out selection rows */}
        <div className="space-y-1.5">
          <span className="font-mono text-[10px] font-bold uppercase tracking-widest text-[var(--pos-text-muted)]">Тип кассовой операции</span>
          <div className="grid grid-cols-2 gap-2">
            <button
              id="cash-event-type-in-btn"
              type="button"
              onClick={() => setType('in')}
              className={`h-11 px-4 font-mono text-xs uppercase font-bold border cursor-pointer select-none rounded-none transition-all
                ${type === 'in' 
                  ? 'bg-[var(--pos-action-primary)] text-[var(--pos-surface)] border-[var(--pos-action-primary)]' 
                  : 'bg-[var(--pos-surface)] border-[var(--pos-border)] text-[var(--pos-text-primary)] hover:bg-[var(--pos-bg)]'
                }`}
            >
              Внесение наличности (Cash In)
            </button>
            <button
              id="cash-event-type-out-btn"
              type="button"
              onClick={() => setType('out')}
              className={`h-11 px-4 font-mono text-xs uppercase font-bold border cursor-pointer select-none rounded-none transition-all
                ${type === 'out' 
                  ? 'bg-[var(--pos-action-primary)] text-[var(--pos-surface)] border-[var(--pos-action-primary)]' 
                  : 'bg-[var(--pos-surface)] border-[var(--pos-border)] text-[var(--pos-text-primary)] hover:bg-[var(--pos-bg)]'
                }`}
            >
              Извлечение наличности (Cash Out)
            </button>
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
            placeholder="0.00"
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
            placeholder="Опишите цель операции (инкассация, размен и т.д.)"
            value={reason}
            onChange={(e) => {
              setReason(e.target.value);
              setErrorMsg(null);
            }}
          />
        </PosFormRow>

        {/* Quick reasons hints list */}
        <div className="flex flex-wrap gap-1.5">
          {['Разменная монета на утро', 'Инкассация торговой выручки', 'Приобретение хозтоваров', 'Обед персонала'].map((text) => (
            <button
              key={text}
              id={`cash-reason-btn-${text.replace(/\s+/g, '-')}`}
              type="button"
              onClick={() => handleReasonClick(text)}
              className="font-mono text-[9px] px-2.5 py-1.5 border border-[var(--pos-border)] hover:bg-[var(--pos-surface-raised)] cursor-pointer rounded-none bg-[var(--pos-surface)] text-[var(--pos-text-secondary)]"
            >
              {text}
            </button>
          ))}
        </div>

      </div>
    </PosDialog>
  );
};

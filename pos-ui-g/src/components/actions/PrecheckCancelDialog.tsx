import React, { useState, useEffect } from 'react';
import { t } from '../../shared/i18n';
import { PosButton, PosDialog, PosFormRow } from '../../shared/ui';
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
      setErrorMsg('Необходимо заполнить PIN-код менеджера.');
      return;
    }
    if (!reason || reason.trim().length < 4) {
      setErrorMsg('Причина отмены должна содержать минимум 4 символа.');
      return;
    }

    const success = await onConfirm(pin, reason);
    if (success) {
      onClose();
    } else {
      setErrorMsg('Неверный PIN-код или отсутствие прав менеджера.');
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
            <span className="font-bold uppercase tracking-wider block font-mono text-xs">Действие требует подтверждения:</span>
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
          label="PIN-код Менеджера"
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
          label="Причина отмены пречека"
          id="precheck-cancel-reason"
        >
          <input
            id="precheck-cancel-reason"
            type="text"
            className="w-full h-12 border border-[var(--pos-border)] px-4 font-sans text-sm bg-[var(--pos-surface)] text-[var(--pos-text-primary)] rounded-none outline-none focus:border-[var(--pos-border-strong)]"
            placeholder="Например: дозаказ блюда, ошибка официанта"
            value={reason}
            onChange={(e) => {
              setReason(e.target.value);
              setErrorMsg(null);
            }}
          />
        </PosFormRow>

        {/* Quick Reasons Chips list */}
        <div className="space-y-1.5">
          <span className="font-mono text-[10px] font-bold uppercase tracking-widest text-[var(--pos-text-muted)]">Быстрый шаблон причины</span>
          <div className="flex flex-wrap gap-1.5">
            {['Дозаказ блюда', 'Ошибка официанта', 'Разделение чека', 'Гость передумал'].map((text) => (
              <button
                key={text}
                id={`quick-reason-btn-${text.replace(/\s+/g, '-')}`}
                type="button"
                onClick={() => handleQuickReason(text)}
                className="font-mono text-[10px] px-2.5 py-1.5 border border-[var(--pos-border)] hover:bg-[var(--pos-surface-raised)] cursor-pointer tracking-tight rounded-none bg-[var(--pos-surface)] text-[var(--pos-text-secondary)]"
              >
                {text}
              </button>
            ))}
          </div>
        </div>

      </div>
    </PosDialog>
  );
};

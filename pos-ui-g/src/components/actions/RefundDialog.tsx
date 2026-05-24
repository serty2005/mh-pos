import React, { useState, useEffect } from 'react';
import { ClosedOrder, OrderLine } from '../../types';
import { t } from '../../shared/i18n';
import { PosButton, PosDialog, PosFormRow, PosQuantityStepper } from '../../shared/ui';
import { ShieldAlert, RefreshCw, Trash2 } from 'lucide-react';

interface RefundDialogProps {
  isOpen: boolean;
  onClose: () => void;
  check: ClosedOrder | null;
  onFullRefund: (reason: string, disposition: 'waste' | 'return') => void;
  onPartialRefund: (lineId: string, qty: number, reason: string, disposition: 'waste' | 'return') => void;
}

export const RefundDialog: React.FC<RefundDialogProps> = ({
  isOpen,
  onClose,
  check,
  onFullRefund,
  onPartialRefund
}) => {
  const [refundMode, setRefundMode] = useState<'full' | 'partial'>('full');
  const [reason, setReason] = useState<string>('');
  const [selectedLineId, setSelectedLineId] = useState<string>('');
  const [qtyToRefund, setQtyToRefund] = useState<number>(1);
  const [disposition, setDisposition] = useState<'waste' | 'return'>('return');
  const [errorMsg, setErrorMsg] = useState<string | null>(null);

  useEffect(() => {
    if (isOpen && check) {
      setRefundMode('full');
      setReason('');
      setDisposition('return');
      setErrorMsg(null);
      if (check.lines.length > 0) {
        setSelectedLineId(check.lines[0].id);
        setQtyToRefund(1);
      }
    }
  }, [isOpen, check]);

  if (!check) return null;

  const targetLine = check.lines.find(l => l.id === selectedLineId);
  const maxQtyAllowed = targetLine ? targetLine.quantity : 1;

  const handleSubmit = () => {
    setErrorMsg(null);
    if (!reason || reason.trim().length < 4) {
      setErrorMsg('Укажите причину возврата (минимум 4 символа).');
      return;
    }

    if (refundMode === 'full') {
      onFullRefund(reason, disposition);
      onClose();
    } else {
      if (!selectedLineId) {
        setErrorMsg('Выберите позицию для частичного возврата.');
        return;
      }
      onPartialRefund(selectedLineId, qtyToRefund, reason, disposition);
      onClose();
    }
  };

  return (
    <PosDialog
      isOpen={isOpen}
      onClose={onClose}
      title={t.modals.refundTitle}
      footer={
        <>
          <PosButton variant="secondary" size="sm" onClick={onClose}>
            {t.common.cancel}
          </PosButton>
          <PosButton
            id="refund-confirm-btn"
            variant="danger"
            size="sm"
            onClick={handleSubmit}
            icon={<RefreshCw className="w-4 h-4" />}
          >
            {refundMode === 'full' ? 'Оформить полный возврат' : 'Вернуть выбранную часть'}
          </PosButton>
        </>
      }
    >
      <div className="flex flex-col gap-4 select-none">
        
        {/* Warning Badge */}
        <div className="p-3 bg-red-50 dark:bg-red-950/20 border border-red-200 dark:border-red-900 text-red-850 dark:text-red-100 flex gap-2.5 text-xs">
          <ShieldAlert className="w-[18px] h-[18px] text-red-500 shrink-0" />
          <span>Это действие регистрирует фискальный возврат и уменьшает кассовый баланс. Позиция списывается согласно выбранному типу инвентаря.</span>
        </div>

        {errorMsg && (
          <span className="font-sans text-xs font-semibold text-[var(--pos-status-danger)] bg-red-50 dark:bg-red-950/20 py-1.5 px-4 block text-center">
            {errorMsg}
          </span>
        )}

        {/* Refund Mode Selectors */}
        <div className="space-y-1.5">
          <span className="font-mono text-[10px] font-bold uppercase tracking-widest text-[var(--pos-text-muted)]">Режим возврата оплат</span>
          <div className="grid grid-cols-2 gap-2">
            <button
              id="refund-mode-full-btn"
              type="button"
              onClick={() => {
                setRefundMode('full');
                setErrorMsg(null);
              }}
              className={`h-11 px-4 font-mono text-xs uppercase font-bold border cursor-pointer select-none rounded-none transition-all
                ${refundMode === 'full' 
                  ? 'bg-[var(--pos-action-primary)] text-[var(--pos-surface)] border-[var(--pos-action-primary)]' 
                  : 'bg-[var(--pos-surface)] border-[var(--pos-border)] text-[var(--pos-text-primary)] hover:bg-[var(--pos-bg)]'
                }`}
            >
              Полный возврат чека
            </button>
            <button
              id="refund-mode-part-btn"
              type="button"
              onClick={() => {
                setRefundMode('partial');
                setErrorMsg(null);
              }}
              className={`h-11 px-4 font-mono text-xs uppercase font-bold border cursor-pointer select-none rounded-none transition-all
                ${refundMode === 'partial' 
                  ? 'bg-[var(--pos-action-primary)] text-[var(--pos-surface)] border-[var(--pos-action-primary)]' 
                  : 'bg-[var(--pos-surface)] border-[var(--pos-border)] text-[var(--pos-text-primary)] hover:bg-[var(--pos-bg)]'
                }`}
            >
              Частичный по позициям
            </button>
          </div>
        </div>

        {/* Selected target line if Partial refund mode selected */}
        {refundMode === 'partial' && (
          <div className="space-y-3 p-4 bg-[var(--pos-surface-raised)] border border-[var(--pos-border)]">
            <PosFormRow
              label="Выберите товар к возврату"
              id="refund-target-item"
            >
              <select
                id="refund-target-item"
                className="w-full h-11 border border-[var(--pos-border)] bg-[var(--pos-surface)] text-[var(--pos-text-primary)] font-sans text-xs px-3 rounded-none outline-none focus:border-[var(--pos-border-strong)]"
                value={selectedLineId}
                onChange={(e) => {
                  setSelectedLineId(e.target.value);
                  setQtyToRefund(1);
                  setErrorMsg(null);
                }}
              >
                {check.lines.map((line) => (
                  <option key={line.id} value={line.id}>
                    {line.name} ({line.price} ₽ × {line.quantity} шт.)
                  </option>
                ))}
              </select>
            </PosFormRow>

            <div className="flex items-center justify-between">
              <span className="font-mono text-xs font-bold uppercase tracking-wider text-[var(--pos-text-secondary)]">Количество к возврату</span>
              <PosQuantityStepper
                id="refund-qty-stepper"
                value={qtyToRefund}
                onChange={(val) => setQtyToRefund(val)}
                min={1}
                max={maxQtyAllowed}
              />
            </div>
          </div>
        )}

        {/* Inventory Disposition rows */}
        <div className="space-y-1.5">
          <span className="font-mono text-[10px] font-bold uppercase tracking-widest text-[var(--pos-text-muted)]">Движение товара на складе</span>
          <div className="grid grid-cols-2 gap-2">
            <button
              id="dispo-return-btn"
              type="button"
              onClick={() => setDisposition('return')}
              className={`h-11 px-4 font-sans text-xs font-semibold border cursor-pointer select-none rounded-none transition-all
                ${disposition === 'return' 
                  ? 'bg-[var(--pos-action-primary)] text-[var(--pos-surface)] border-[var(--pos-action-primary)]' 
                  : 'bg-[var(--pos-surface)] border-[var(--pos-border)] text-[var(--pos-text-primary)] hover:bg-[var(--pos-bg)]'
                }`}
            >
              {t.modals.inventoryReturn}
            </button>
            <button
              id="dispo-waste-btn"
              type="button"
              onClick={() => setDisposition('waste')}
              className={`h-11 px-4 font-sans text-xs font-semibold border cursor-pointer select-none rounded-none transition-all
                ${disposition === 'waste' 
                  ? 'bg-[var(--pos-action-primary)] text-[var(--pos-surface)] border-[var(--pos-action-primary)]' 
                  : 'bg-[var(--pos-surface)] border-[var(--pos-border)] text-[var(--pos-text-primary)] hover:bg-[var(--pos-bg)]'
                }`}
            >
              {t.modals.inventoryWaste}
            </button>
          </div>
        </div>

        {/* Cancellation Reason Entry Input */}
        <PosFormRow
          label="Причина отмены/возврата чека"
          id="refund-text-reason"
        >
          <input
            id="refund-text-reason"
            type="text"
            className="w-full h-12 border border-[var(--pos-border)] px-4 font-sans text-sm bg-[var(--pos-surface)] text-[var(--pos-text-primary)] rounded-none outline-none focus:border-[var(--pos-border-strong)]"
            placeholder="Например: Ошибка кассира, Возврат гостя при отказе"
            value={reason}
            onChange={(e) => {
              setReason(e.target.value);
              setErrorMsg(null);
            }}
          />
        </PosFormRow>

        {/* Quick template triggers for reasons */}
        <div className="flex flex-wrap gap-1.5">
          {['Ошибка при оплате', 'Отвергнуто гостем', 'Плохое качество блюд', 'Ошибка при дозаказе'].map((text) => (
            <button
              key={text}
              id={`refund-qreason-${text.replace(/\s+/g, '-')}`}
              type="button"
              onClick={() => {
                setReason(text);
                setErrorMsg(null);
              }}
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

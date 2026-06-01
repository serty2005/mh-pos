import React, { useState, useEffect } from 'react';
import { ClosedOrder } from '../../types';
import { t } from '../../shared/i18n';
import { PosButton, PosDialog, PosFormRow, PosQuantityStepper, PosSelectableChip } from '../../shared/ui';
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
      setErrorMsg(t.modals.refundReasonRequired);
      return;
    }

    if (refundMode === 'full') {
      onFullRefund(reason, disposition);
      onClose();
    } else {
      if (!selectedLineId) {
        setErrorMsg(t.modals.refundLineRequired);
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
            {refundMode === 'full' ? t.modals.refundConfirmFull : t.modals.refundConfirmPartial}
          </PosButton>
        </>
      }
    >
      <div className="flex flex-col gap-4 select-none">
        
        {/* Warning Badge */}
        <div className="p-3 bg-red-50 dark:bg-red-950/20 border border-red-200 dark:border-red-900 text-red-850 dark:text-red-100 flex gap-2.5 text-xs">
          <ShieldAlert className="w-[18px] h-[18px] text-red-500 shrink-0" />
          <span>{t.modals.refundFiscalWarning}</span>
        </div>

        {errorMsg && (
          <span className="font-sans text-xs font-semibold text-[var(--pos-status-danger)] bg-red-50 dark:bg-red-950/20 py-1.5 px-4 block text-center">
            {errorMsg}
          </span>
        )}

        {/* Refund Mode Selectors */}
        <div className="space-y-1.5">
          <span className="font-mono text-[10px] font-bold uppercase tracking-widest text-[var(--pos-text-muted)]">{t.modals.refundMode}</span>
          <div className="grid grid-cols-2 gap-2">
            <PosSelectableChip
              id="refund-mode-full-btn"
              active={refundMode === 'full'}
              onClick={() => {
                setRefundMode('full');
                setErrorMsg(null);
              }}
              className="h-11 px-4"
            >
              {t.modals.refundModeFull}
            </PosSelectableChip>
            <PosSelectableChip
              id="refund-mode-part-btn"
              active={refundMode === 'partial'}
              onClick={() => {
                setRefundMode('partial');
                setErrorMsg(null);
              }}
              className="h-11 px-4"
            >
              {t.modals.refundModePartial}
            </PosSelectableChip>
          </div>
        </div>

        {/* Selected target line if Partial refund mode selected */}
        {refundMode === 'partial' && (
          <div className="space-y-3 p-4 bg-[var(--pos-surface-raised)] border border-[var(--pos-border)]">
            <PosFormRow
              label={t.modals.refundTargetItem}
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
                    {line.name} ({line.price} {t.common.ruble} × {line.quantity} {t.common.pieces})
                  </option>
                ))}
              </select>
            </PosFormRow>

            <div className="flex items-center justify-between">
              <span className="font-mono text-xs font-bold uppercase tracking-wider text-[var(--pos-text-secondary)]">{t.modals.refundQuantity}</span>
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
          <span className="font-mono text-[10px] font-bold uppercase tracking-widest text-[var(--pos-text-muted)]">{t.modals.refundInventoryFlow}</span>
          <div className="grid grid-cols-2 gap-2">
            <PosSelectableChip
              id="dispo-return-btn"
              active={disposition === 'return'}
              onClick={() => setDisposition('return')}
              className="h-11 px-4 font-sans normal-case tracking-normal"
            >
              {t.modals.inventoryReturn}
            </PosSelectableChip>
            <PosSelectableChip
              id="dispo-waste-btn"
              active={disposition === 'waste'}
              onClick={() => setDisposition('waste')}
              className="h-11 px-4 font-sans normal-case tracking-normal"
            >
              {t.modals.inventoryWaste}
            </PosSelectableChip>
          </div>
        </div>

        {/* Cancellation Reason Entry Input */}
        <PosFormRow
          label={t.modals.refundReasonLabel}
          id="refund-text-reason"
        >
          <input
            id="refund-text-reason"
            type="text"
            className="w-full h-12 border border-[var(--pos-border)] px-4 font-sans text-sm bg-[var(--pos-surface)] text-[var(--pos-text-primary)] rounded-none outline-none focus:border-[var(--pos-border-strong)]"
            placeholder={t.modals.refundReasonPlaceholder}
            value={reason}
            onChange={(e) => {
              setReason(e.target.value);
              setErrorMsg(null);
            }}
          />
        </PosFormRow>

        {/* Quick template triggers for reasons */}
        <div className="flex flex-wrap gap-1.5">
          {t.modals.refundQuickReasons.map((text) => (
            <PosSelectableChip
              key={text}
              id={`refund-qreason-${text.replace(/\s+/g, '-')}`}
              onClick={() => {
                setReason(text);
                setErrorMsg(null);
              }}
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

import React, { useState, useEffect } from 'react';
import { usePOS } from '../../context/POSContext';
import { t } from '../../shared/i18n';
import { PosButton, PosDialog, PosBanner, PosSelectableChip } from '../../shared/ui';
import { CreditCard, Banknote, CheckCircle } from 'lucide-react';

interface PaymentDialogProps {
  isOpen: boolean;
  onClose: () => void;
}

export const PaymentDialog: React.FC<PaymentDialogProps> = ({
  isOpen,
  onClose
}) => {
  const { currentOrder, payOrder, cashSession } = usePOS();
  
  const [method, setMethod] = useState<'cash' | 'card'>('cash');
  const [inputVal, setInputVal] = useState<string>('');
  const [paymentReport, setPaymentReport] = useState<{ amountToPay: number; inputAmount: number; change: number; success: boolean } | null>(null);

  useEffect(() => {
    if (isOpen && currentOrder) {
      setInputVal(currentOrder.total.toString());
      setPaymentReport(null);
      setMethod('cash');
    }
  }, [isOpen, currentOrder]);

  if (!currentOrder && !paymentReport) return null;

  const handleQuickSum = (amountToAdd: number) => {
    const current = parseFloat(inputVal) || 0;
    setInputVal((current + amountToAdd).toString());
  };

  const handleExactSum = () => {
    if (!currentOrder) return;
    setInputVal(currentOrder.total.toString());
  };

  const handleSubmit = async () => {
    if (!currentOrder) return;
    const numericAmount = parseFloat(inputVal) || 0;
    const amountToPay = currentOrder.total;
    
    // Process payment mutation via context
    const report = await payOrder(method, numericAmount);
    
    if (report.success) {
      setPaymentReport({
        amountToPay,
        inputAmount: numericAmount,
        change: report.change,
        success: true,
      });
    }
  };

  const handleKeypadPress = (key: string) => {
    if (key === 'C') {
      setInputVal('');
    } else if (key === 'DEL') {
      setInputVal(prev => prev.slice(0, -1));
    } else if (key === '.') {
      if (!inputVal.includes('.')) {
        setInputVal(prev => prev + '.');
      }
    } else {
      setInputVal(prev => prev + key);
    }
  };

  const amountToPay = currentOrder?.total ?? paymentReport?.amountToPay ?? 0;
  const inputtedAmount = paymentReport?.inputAmount ?? (parseFloat(inputVal) || 0);
  const changeValue = Math.max(0, inputtedAmount - amountToPay);
  const isInputSufficient = inputtedAmount >= amountToPay;

  // Render checkout result if payment successfully captured
  if (paymentReport?.success) {
    return (
      <PosDialog
        isOpen={isOpen}
        onClose={onClose}
        title={t.modals.paymentTitle}
      >
        <div className="flex flex-col items-center justify-center p-6 text-center select-none">
          <CheckCircle className="w-16 h-16 text-[var(--pos-status-success)] mb-4 animate-bounce" />
          <h3 className="font-mono text-base font-bold uppercase tracking-wider text-[var(--pos-text-primary)] mb-2">
            {t.modals.paySuccess}
          </h3>
          
          <div className="w-full bg-[var(--pos-surface-raised)] border border-[var(--pos-border)] p-4 mt-4 space-y-2">
            <div className="flex justify-between text-xs font-mono text-[var(--pos-text-secondary)]">
              <span>{t.modals.paymentAmountDue}:</span>
              <span className="font-bold">{amountToPay} {t.common.ruble}</span>
            </div>
            <div className="flex justify-between text-xs font-mono text-[var(--pos-text-secondary)]">
              <span>{t.modals.paymentAmountReceived}:</span>
              <span className="font-bold">{inputtedAmount} {t.common.ruble}</span>
            </div>
            {paymentReport.change > 0 && (
              <div className="flex justify-between text-sm font-mono text-[var(--pos-status-success)] border-t border-[var(--pos-border)] pt-2 font-bold">
                <span>{t.modals.paymentChange}:</span>
                <span>{paymentReport.change} {t.common.ruble}</span>
              </div>
            )}
          </div>

          <PosButton 
            id="payment-success-close-btn"
            variant="primary" 
            size="md" 
            fullWidth 
            className="mt-8 font-mono"
            onClick={onClose}
          >
            {t.common.close}
          </PosButton>
        </div>
      </PosDialog>
    );
  }

  return (
    <PosDialog
      isOpen={isOpen}
      onClose={onClose}
      title={t.modals.paymentTitle}
      footer={
        <>
          <PosButton variant="secondary" size="sm" onClick={onClose}>
            {t.common.cancel}
          </PosButton>
          <PosButton
            id="payment-submit-btn"
            variant="financial"
            size="md"
            onClick={handleSubmit}
            disabled={!cashSession || !isInputSufficient}
            icon={<Banknote className="w-5 h-5" />}
          >
            {t.modals.paymentSubmit} ({amountToPay} {t.common.ruble})
          </PosButton>
        </>
      }
    >
      <div className="flex flex-col gap-6 select-none">
        
        {/* Verification of Cash KKM Session block */}
        {!cashSession && (
          <PosBanner
            type="danger"
            message={t.errors.noCashSession}
          />
        )}

        {/* Totals Review banner */}
        <div className="grid grid-cols-2 gap-4 bg-[var(--pos-surface-raised)] border border-[var(--pos-border)] p-4">
          <div className="flex flex-col">
            <span className="font-mono text-[10px] text-[var(--pos-text-muted)] uppercase tracking-wider">{t.modals.paymentDueTotal}:</span>
            <span className="font-mono text-xl font-black text-[var(--pos-text-primary)]">{amountToPay} {t.common.ruble}</span>
          </div>
          <div className="flex flex-col items-end">
            <span className="font-mono text-[10px] text-[var(--pos-text-muted)] uppercase tracking-wider">{t.modals.paymentChange}:</span>
            <span className={`font-mono text-xl font-black ${changeValue > 0 ? 'text-[var(--pos-status-success)]' : 'text-[var(--pos-text-muted)]'}`}>
              {changeValue} {t.common.ruble}
            </span>
          </div>
        </div>

        {/* Method Picker row */}
        <div className="space-y-2">
          <span className="font-mono text-xs font-bold uppercase tracking-wider text-[var(--pos-text-secondary)]">{t.modals.paymentMethod}</span>
          <div className="grid grid-cols-2 gap-3">
            <PosSelectableChip
              id="pay-method-cash-btn"
              active={method === 'cash'}
              onClick={() => setMethod('cash')}
              className="h-[56px] flex items-center justify-center gap-2"
            >
              <Banknote className="w-5 h-5 shrink-0" />
              <span>{t.modals.paymentMethodCash}</span>
            </PosSelectableChip>
            <PosSelectableChip
              id="pay-method-card-btn"
              active={method === 'card'}
              onClick={() => setMethod('card')}
              className="h-[56px] flex items-center justify-center gap-2"
            >
              <CreditCard className="w-5 h-5 shrink-0" />
              <span>{t.modals.paymentMethodCard}</span>
            </PosSelectableChip>
          </div>
        </div>

        {/* Input box row */}
        <div className="space-y-2">
          <div className="flex justify-between items-baseline">
            <span className="font-mono text-xs font-bold uppercase tracking-wider text-[var(--pos-text-secondary)]">{t.modals.paymentInputAmount}</span>
            <PosButton
              id="pay-exact-btn"
              type="button" 
              onClick={handleExactSum} 
              variant="ghost"
              size="sm"
              className="h-7 px-2 font-mono text-[10px] text-[var(--pos-status-info)]"
            >
              {t.modals.paymentExactAmount}
            </PosButton>
          </div>
          <input
            id="payment-amount-input"
            type="text"
            readOnly
            value={inputVal}
            className="w-full h-14 border border-[var(--pos-border-strong)] bg-[var(--pos-surface-raised)] font-mono text-xl font-bold px-4 text-right tracking-tight text-[var(--pos-text-primary)] focus:outline-none"
          />
        </div>

        {/* Numeric keypad + Quick sums builder */}
        <div className="flex flex-col sm:flex-row gap-4">
          
          {/* Pad buttons */}
          <div className="flex-1 grid grid-cols-3 gap-2">
            {['1', '2', '3', '4', '5', '6', '7', '8', '9', '0', '.', 'C'].map((key) => (
              <PosButton
                key={key}
                id={`pay-pad-${key === 'C' ? 'clear' : key === '.' ? 'dot' : key}`}
                type="button"
                onClick={() => handleKeypadPress(key)}
                variant="secondary"
                size="sm"
                className="h-12 px-0 bg-[var(--pos-surface)] font-mono font-bold text-center hover:bg-[var(--pos-surface-raised)] text-[var(--pos-text-primary)]"
              >
                {key}
              </PosButton>
            ))}
          </div>

          {/* Quick Sum Buttons Column */}
          <div className="w-full sm:w-[120px] flex sm:flex-col gap-2 shrink-0">
            {[100, 500, 1000, 5000].map((sum) => (
              <PosButton
                key={sum}
                id={`pay-quick-${sum}`}
                type="button"
                onClick={() => handleQuickSum(sum)}
                variant="secondary"
                size="sm"
                className="flex-1 h-12 px-0 bg-[var(--pos-surface-raised)] font-mono text-xs font-bold text-center hover:bg-[var(--pos-border)] text-[var(--pos-text-secondary)]"
              >
                +{sum} {t.common.ruble}
              </PosButton>
            ))}
          </div>
        </div>

      </div>
    </PosDialog>
  );
};

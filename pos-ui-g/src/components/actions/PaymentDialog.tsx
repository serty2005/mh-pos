import React, { useState, useEffect } from 'react';
import { usePOS } from '../../context/POSContext';
import { t } from '../../shared/i18n';
import { PosButton, PosDialog, PosBanner } from '../../shared/ui';
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
  const [paymentReport, setPaymentReport] = useState<{ change: number; success: boolean } | null>(null);

  useEffect(() => {
    if (isOpen && currentOrder) {
      setInputVal(currentOrder.total.toString());
      setPaymentReport(null);
      setMethod('cash');
    }
  }, [isOpen, currentOrder]);

  if (!currentOrder) return null;

  const handleQuickSum = (amountToAdd: number) => {
    const current = parseFloat(inputVal) || 0;
    setInputVal((current + amountToAdd).toString());
  };

  const handleExactSum = () => {
    setInputVal(currentOrder.total.toString());
  };

  const handleSubmit = async () => {
    const numericAmount = parseFloat(inputVal) || 0;
    
    // Process payment mutation via context
    const report = await payOrder(method, numericAmount);
    
    if (report.success) {
      setPaymentReport({
        success: true,
        change: report.change
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

  const amountToPay = currentOrder.total;
  const inputtedAmount = parseFloat(inputVal) || 0;
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
              <span>Сумма к оплате:</span>
              <span className="font-bold">{amountToPay} ₽</span>
            </div>
            <div className="flex justify-between text-xs font-mono text-[var(--pos-text-secondary)]">
              <span>Внесено:</span>
              <span className="font-bold">{inputtedAmount} ₽</span>
            </div>
            {paymentReport.change > 0 && (
              <div className="flex justify-between text-sm font-mono text-[var(--pos-status-success)] border-t border-[var(--pos-border)] pt-2 font-bold">
                <span>{t.modals.paymentChange}:</span>
                <span>{paymentReport.change} ₽</span>
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
            Провести оплату ({amountToPay} ₽)
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
            <span className="font-mono text-[10px] text-[var(--pos-text-muted)] uppercase tracking-wider">К оплате (Итого):</span>
            <span className="font-mono text-xl font-black text-[var(--pos-text-primary)]">{amountToPay} ₽</span>
          </div>
          <div className="flex flex-col items-end">
            <span className="font-mono text-[10px] text-[var(--pos-text-muted)] uppercase tracking-wider">{t.modals.paymentChange}:</span>
            <span className={`font-mono text-xl font-black ${changeValue > 0 ? 'text-[var(--pos-status-success)]' : 'text-[var(--pos-text-muted)]'}`}>
              {changeValue} ₽
            </span>
          </div>
        </div>

        {/* Method Picker row */}
        <div className="space-y-2">
          <span className="font-mono text-xs font-bold uppercase tracking-wider text-[var(--pos-text-secondary)]">Способ оплаты</span>
          <div className="grid grid-cols-2 gap-3">
            <button
              id="pay-method-cash-btn"
              type="button"
              onClick={() => setMethod('cash')}
              className={`h-[56px] border flex items-center justify-center gap-2 font-mono text-xs uppercase font-bold cursor-pointer select-none rounded-none transition-all
                ${method === 'cash' 
                  ? 'bg-[var(--pos-action-primary)] border-[var(--pos-action-primary)] text-[var(--pos-surface)]' 
                  : 'bg-[var(--pos-surface)] border-[var(--pos-border)] text-[var(--pos-text-primary)] hover:bg-[var(--pos-surface-raised)]'
                }`}
            >
              <Banknote className="w-5 h-5 shrink-0" />
              <span>{t.modals.paymentMethodCash}</span>
            </button>
            <button
              id="pay-method-card-btn"
              type="button"
              onClick={() => setMethod('card')}
              className={`h-[56px] border flex items-center justify-center gap-2 font-mono text-xs uppercase font-bold cursor-pointer select-none rounded-none transition-all
                ${method === 'card' 
                  ? 'bg-[var(--pos-action-primary)] border-[var(--pos-action-primary)] text-[var(--pos-surface)]' 
                  : 'bg-[var(--pos-surface)] border-[var(--pos-border)] text-[var(--pos-text-primary)] hover:bg-[var(--pos-surface-raised)]'
                }`}
            >
              <CreditCard className="w-5 h-5 shrink-0" />
              <span>{t.modals.paymentMethodCard}</span>
            </button>
          </div>
        </div>

        {/* Input box row */}
        <div className="space-y-2">
          <div className="flex justify-between items-baseline">
            <span className="font-mono text-xs font-bold uppercase tracking-wider text-[var(--pos-text-secondary)]">{t.modals.paymentInputAmount}</span>
            <button 
              id="pay-exact-btn"
              type="button" 
              onClick={handleExactSum} 
              className="font-mono text-[10px] text-[var(--pos-status-info)] font-bold uppercase cursor-pointer hover:underline border-none bg-none"
            >
              Точная сумма
            </button>
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
              <button
                key={key}
                id={`pay-pad-${key === 'C' ? 'clear' : key === '.' ? 'dot' : key}`}
                type="button"
                onClick={() => handleKeypadPress(key)}
                className="h-12 border border-[var(--pos-border)] bg-[var(--pos-surface)] font-mono font-bold text-center hover:bg-[var(--pos-surface-raised)] cursor-pointer select-none rounded-none text-[var(--pos-text-primary)]"
              >
                {key}
              </button>
            ))}
          </div>

          {/* Quick Sum Buttons Column */}
          <div className="w-full sm:w-[120px] flex sm:flex-col gap-2 shrink-0">
            {[100, 500, 1000, 5000].map((sum) => (
              <button
                key={sum}
                id={`pay-quick-${sum}`}
                type="button"
                onClick={() => handleQuickSum(sum)}
                className="flex-1 h-12 border border-[var(--pos-border)] bg-[var(--pos-surface-raised)] font-mono text-xs font-bold text-center flex items-center justify-center hover:bg-[var(--pos-border)] cursor-pointer select-none text-[var(--pos-text-secondary)] rounded-none"
              >
                +{sum} ₽
              </button>
            ))}
          </div>
        </div>

      </div>
    </PosDialog>
  );
};

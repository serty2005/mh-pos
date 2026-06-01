import React, { useState } from 'react';
import { usePOS } from '../context/POSContext';
import { t } from '../shared/i18n';
import { PosButton } from '../shared/ui';
import { Lock, Delete, ArrowRight } from 'lucide-react';
import { appendPinDigit, canSubmitPin, pinIndicatorCount } from './pinInput';

export const PinLogin: React.FC = () => {
  const { pinLogin, appVersion } = usePOS();
  const [pin, setPin] = useState<string>('');
  const [errorMsg, setErrorMsg] = useState<string | null>(null);

  const handleKeyPress = (num: string) => {
    setErrorMsg(null);
    setPin(prev => appendPinDigit(prev, num));
  };

  const handleDelete = () => {
    setErrorMsg(null);
    setPin(prev => prev.slice(0, -1));
  };

  const handleClear = () => {
    setErrorMsg(null);
    setPin('');
  };

  const handlePinSubmit = async () => {
    if (!canSubmitPin(pin)) return;
    const success = await pinLogin(pin);
    if (!success) {
      setErrorMsg(t.auth.invalidPin);
      setPin('');
    }
  };

  return (
    <div className="fixed inset-0 z-50 flex flex-col md:flex-row bg-[var(--pos-bg)] text-[var(--pos-text-primary)]">
      {/* Terminal context column */}
      <div className="flex-1 flex flex-col justify-between p-8 md:p-12 border-b md:border-b-0 md:border-r border-[var(--pos-border)] bg-[var(--pos-surface-raised)] select-none">
        <div className="flex items-center gap-3">
          <div className="w-10 h-10 border border-[var(--pos-border-strong)] flex items-center justify-center bg-[var(--pos-surface)]">
            <Lock className="w-4 h-4 text-[var(--pos-text-secondary)]" />
          </div>
          <span className="font-sans text-base tracking-normal text-[var(--pos-text-secondary)] font-semibold">MyHoreca POS</span>
        </div>

        <div className="my-[40px] max-w-md">
          <h1 className="font-sans text-xl md:text-3xl font-semibold tracking-normal mb-3 text-[var(--pos-text-secondary)]">
            {t.auth.pinTitle}
          </h1>
          <p className="font-sans text-xs md:text-sm text-[var(--pos-text-muted)] leading-relaxed font-light">
            {t.auth.pinSessionCopy}
          </p>
        </div>

        <div className="font-mono text-[10px] text-[var(--pos-text-muted)] uppercase tracking-widest">
          {t.auth.dbVersion}: {appVersion}
        </div>
      </div>

      {/* Keypad Authenticator Column */}
      <div className="flex-1 flex flex-col justify-center items-center p-8 bg-[var(--pos-surface)] select-none">
        <div className="w-full max-w-[320px] flex flex-col items-center">
          {/* PIN Indicators */}
          <div className="flex flex-wrap justify-center gap-1.5 mb-6 max-w-full">
            {Array.from({ length: pinIndicatorCount(pin) }, (_, pos) => {
              const filled = pin.length > pos;
              return (
                <div 
                  key={pos} 
                  className={`w-5 h-5 border-2 flex items-center justify-center transition-all duration-100 rounded-none
                    ${filled 
                      ? 'border-[var(--pos-action-primary)] bg-[var(--pos-action-primary)] font-bold text-xs' 
                      : 'border-[var(--pos-border-strong)] bg-transparent'
                    }`}
                >
                  {filled && <span className="text-[var(--pos-surface)]">•</span>}
                </div>
              );
            })}
          </div>

          {/* Display Lock icon or Error messages */}
          <div className="min-h-[44px] flex items-center justify-center mb-6 text-center">
            {errorMsg ? (
              <span className="font-sans text-xs font-semibold text-[var(--pos-status-danger)] bg-red-50 dark:bg-red-950/20 py-1.5 px-4 animate-shake">
                {errorMsg}
              </span>
            ) : (
              <div className="flex items-center gap-2 text-[var(--pos-text-muted)] font-mono text-[11px] uppercase tracking-wider">
                <Lock className="w-4 h-4" />
                <span>{t.auth.lockedTerm}</span>
              </div>
            )}
          </div>

          {/* Keypad Grid cells */}
          <div className="grid grid-cols-3 gap-3 w-full mb-6">
            {['1', '2', '3', '4', '5', '6', '7', '8', '9'].map((num) => (
              <PosButton
                key={num}
                id={`pin-btn-${num}`}
                type="button"
                variant="secondary"
                size="lg"
                className="h-[64px] px-0 bg-[var(--pos-surface-raised)] hover:bg-[var(--pos-border)] active:bg-[var(--pos-border-strong)] font-mono text-xl font-bold text-[var(--pos-text-primary)]"
                onClick={() => handleKeyPress(num)}
              >
                {num}
              </PosButton>
            ))}
            
            {/* Clear Button */}
            <PosButton
              id={`pin-btn-clear`}
              type="button"
              variant="secondary"
              size="lg"
              aria-label={t.common.clear}
              className="h-[64px] px-0 bg-[var(--pos-surface-raised)] hover:bg-[var(--pos-border)] font-mono text-xs font-bold text-[var(--pos-status-danger)]"
              onClick={handleClear}
            >
              C
            </PosButton>
            
            {/* Zero */}
            <PosButton
              id={`pin-btn-0`}
              type="button"
              variant="secondary"
              size="lg"
              className="h-[64px] px-0 bg-[var(--pos-surface-raised)] hover:bg-[var(--pos-border)] font-mono text-xl font-bold text-[var(--pos-text-primary)]"
              onClick={() => handleKeyPress('0')}
            >
              0
            </PosButton>

            {/* Backspace */}
            <PosButton
              id={`pin-btn-delete`}
              type="button"
              variant="secondary"
              size="lg"
              className="h-[64px] px-0 bg-[var(--pos-surface-raised)] hover:bg-[var(--pos-border)] font-mono text-xl font-bold text-[var(--pos-text-secondary)]"
              onClick={handleDelete}
              aria-label={t.auth.deleteDigit}
            >
              <Delete className="w-5 h-5" />
            </PosButton>
          </div>

          {/* Submit Action Button */}
          <PosButton
            id="pin-submit-btn"
            variant="primary"
            size="md"
            fullWidth
            onClick={handlePinSubmit}
            disabled={!canSubmitPin(pin)}
            className="font-mono text-xs uppercase tracking-widest"
            icon={<ArrowRight className="w-4 h-4" />}
          >
            {t.auth.login}
          </PosButton>
        </div>
      </div>
    </div>
  );
};

import React, { useState } from 'react';
import { usePOS } from '../context/POSContext';
import { t } from '../shared/i18n';
import { PosButton } from '../shared/ui';
import { Lock, Delete, ArrowRight } from 'lucide-react';
import { appendPinDigit, canSubmitPin, pinIndicatorCount } from './pinInput';

export const PinLogin: React.FC = () => {
  const { pinLogin } = usePOS();
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
      {/* Branding Column / Info Column */}
      <div className="flex-1 flex flex-col justify-between p-8 md:p-12 border-b md:border-b-0 md:border-r border-[var(--pos-border)] bg-[var(--pos-surface-raised)] select-none">
        <div className="flex items-center gap-3">
          <div className="w-10 h-10 border border-[var(--pos-border-strong)] flex items-center justify-center bg-[#050505]">
            <span className="text-lg font-serif italic text-[var(--pos-text-secondary)] font-semibold">E</span>
          </div>
          <span className="font-serif text-base tracking-widest uppercase text-[var(--pos-text-secondary)] font-medium">L'ESSENCE POS</span>
        </div>

        <div className="my-[40px] max-w-md">
          <h1 className="font-serif text-xl md:text-3xl font-medium tracking-wider italic uppercase mb-3 text-[var(--pos-text-secondary)]">
            {t.auth.pinTitle}
          </h1>
          <p className="font-sans text-xs md:text-sm text-[var(--pos-text-muted)] leading-relaxed font-light">
            Добро пожаловать в премиальный POS-терминал L'Essence. Введите ваш персональный PIN-код для открытия смены и доступа к столам.
          </p>

          {/* Test Hint Panel */}
          <div className="mt-8 p-4 bg-[var(--pos-surface)] border border-[var(--pos-border)]">
            <span className="font-mono text-[10px] uppercase font-bold text-[var(--pos-text-secondary)] tracking-widest block mb-2">
              Тестовые PIN-коды авторизации (RBAC):
            </span>
            <div className="grid grid-cols-1 gap-2 font-mono text-xs text-[var(--pos-text-muted)]">
              <div><strong className="text-[var(--pos-text-primary)]">1111</strong> • Старший Кассир (Все права)</div>
              <div><strong className="text-[var(--pos-text-primary)]">2222</strong> • Официант (Права ограничены)</div>
              <div><strong className="text-[var(--pos-text-primary)]">3333</strong> • Администратор (Сервисный ключ)</div>
            </div>
          </div>
        </div>

        <div className="font-mono text-[10px] text-[var(--pos-text-muted)] uppercase tracking-widest">
          © {new Date().getFullYear()} L'Essence Group • Edge Node v2.0.7p
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
              <button
                key={num}
                id={`pin-btn-${num}`}
                className="h-[64px] border border-[var(--pos-border)] bg-[var(--pos-surface-raised)] hover:bg-[var(--pos-border)] active:bg-[var(--pos-border-strong)] font-mono text-xl font-bold cursor-pointer transition-colors text-[var(--pos-text-primary)] rounded-none"
                onClick={() => handleKeyPress(num)}
              >
                {num}
              </button>
            ))}
            
            {/* Clear Button */}
            <button
              id={`pin-btn-clear`}
              className="h-[64px] border border-[var(--pos-border)] bg-[var(--pos-surface-raised)] hover:bg-[var(--pos-border)] font-mono text-xs uppercase tracking-wider font-bold cursor-pointer transition-colors text-[var(--pos-status-danger)] rounded-none"
              onClick={handleClear}
            >
              C
            </button>
            
            {/* Zero */}
            <button
              id={`pin-btn-0`}
              className="h-[64px] border border-[var(--pos-border)] bg-[var(--pos-surface-raised)] hover:bg-[var(--pos-border)] font-mono text-xl font-bold cursor-pointer transition-colors text-[var(--pos-text-primary)] rounded-none"
              onClick={() => handleKeyPress('0')}
            >
              0
            </button>

            {/* Backspace */}
            <button
              id={`pin-btn-delete`}
              className="h-[64px] border border-[var(--pos-border)] bg-[var(--pos-surface-raised)] hover:bg-[var(--pos-border)] flex items-center justify-center font-mono text-xl font-bold cursor-pointer transition-colors text-[var(--pos-text-secondary)] rounded-none"
              onClick={handleDelete}
              aria-label="Delete"
            >
              <Delete className="w-5 h-5" />
            </button>
          </div>

          {/* Submit Action Button */}
          <PosButton
            variant="primary"
            size="md"
            fullWidth
            onClick={handlePinSubmit}
            disabled={!canSubmitPin(pin)}
            className="font-mono text-xs uppercase tracking-widest"
            icon={<ArrowRight className="w-4 h-4" />}
          >
            Войти
          </PosButton>
        </div>
      </div>
    </div>
  );
};

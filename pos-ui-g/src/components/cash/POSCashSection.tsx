import React, { useState } from 'react';
import { usePOS } from '../../context/POSContext';
import { t } from '../../shared/i18n';
import { 
  PosButton, 
  PosDialog, 
  PosEmptyState, 
  PosStatusStrip, 
  PosActionRail,
  PosFormRow,
  PosBanner
} from '../../shared/ui';
import { 
  CashDrawerEventDialog 
} from '../actions/CashDrawerEventDialog';
import { 
  Wallet, 
  Clock, 
  RefreshCw, 
  Check, 
  ArrowUpRight, 
  Wifi, 
  WifiOff, 
  Settings,
  Terminal
} from 'lucide-react';

export const POSCashSection: React.FC = () => {
  const {
    currentOperator,
    openEmployeeShift,
    closeEmployeeShift,
    cashSession,
    openCashSession,
    closeCashSession,
    cashDrawerEvents,
    addCashDrawerEvent,
    outboxCount,
    syncOutbox,
    syncStatus,
    logEvents,
  } = usePOS();

  const [isEventModalOpen, setEventModalOpen] = useState<boolean>(false);
  
  // States for opening KKM session
  const [initialCash, setInitialCash] = useState<string>('5000');
  const [sessionError, setSessionError] = useState<string | null>(null);

  const handleOpenSessionSubmit = () => {
    setSessionError(null);
    const floatVal = parseFloat(initialCash) || 0;
    if (floatVal < 0) {
      setSessionError('Инвентарный размен на открытие кассы не может быть отрицательным.');
      return;
    }

    openCashSession(floatVal);
  };

  const handleSyncOutboxClick = () => {
    syncOutbox();
  };

  return (
    <div className="flex flex-col lg:flex-row flex-1 min-h-0 select-none bg-[var(--pos-bg)]">
      
      {/* 3/4 Main Operations Columns */}
      <div className="flex-1 p-6 pos-scrollarea-y pos-scrollbar-thin overflow-y-auto space-y-6">
        
        {/* Header Title */}
        <div className="flex items-center gap-2 select-none">
          <Wallet className="w-5 h-5 text-[var(--pos-text-secondary)]" />
          <h2 className="font-mono text-base font-bold uppercase tracking-wider text-[var(--pos-text-primary)]">
            {t.cash.shiftTitle}
          </h2>
        </div>

        {/* Sync Status Banner */}
        {syncStatus === 'offline' && (
          <PosBanner
            type="warning"
            message="Внимание: терминал находится в автономном режиме работы. Платежи и смены сохраняются локально."
          />
        )}

        <div className="grid grid-cols-1 md:grid-cols-2 gap-6 select-none">
          
          {/* Personal shift Employee card */}
          <div className="border border-[var(--pos-border)] bg-[var(--pos-surface)] p-5 flex flex-col justify-between space-y-4">
            <div className="space-y-1">
              <span className="font-mono text-[10px] font-bold uppercase tracking-widest text-[var(--pos-text-muted)]">
                Шаг 1: Смена оператора
              </span>
              <h3 className="font-sans text-sm md:text-base font-bold text-[var(--pos-text-primary)]">
                {t.cash.personalShiftStatus}
              </h3>
            </div>

            <div className="space-y-2 py-2">
              <PosStatusStrip
                title="Текущая сессия"
                message={currentOperator ? currentOperator.employeeName : 'Личная смена не открыта'}
                variant={currentOperator ? 'success' : 'danger'}
              />
              {currentOperator && (
                <div className="font-mono text-[10px] text-[var(--pos-text-secondary)] space-y-0.5">
                  <div>Смена открыта: <strong className="text-[var(--pos-text-primary)]">{currentOperator.openTime}</strong></div>
                  <div>Роль RBAC: <strong className="text-[var(--pos-text-primary)] font-bold uppercase">{currentOperator.role}</strong></div>
                </div>
              )}
            </div>

            {currentOperator ? (
              <PosButton
                id="cash-close-shift-btn"
                variant="danger"
                size="sm"
                fullWidth
                onClick={closeEmployeeShift}
              >
                {t.cash.closeShift}
              </PosButton>
            ) : (
              <PosButton
                id="cash-open-shift-btn"
                variant="primary"
                size="sm"
                fullWidth
                onClick={openEmployeeShift}
              >
                {t.cash.openShift}
              </PosButton>
            )}
          </div>

          {/* Cash KKM Session open/close card */}
          <div className="border border-[var(--pos-border)] bg-[var(--pos-surface)] p-5 flex flex-col justify-between space-y-4">
            <div className="space-y-1">
              <span className="font-mono text-[10px] font-bold uppercase tracking-widest text-[var(--pos-text-muted)]">
                Шаг 2: Кассовая ККМ смена
              </span>
              <h3 className="font-sans text-sm md:text-base font-bold text-[var(--pos-text-primary)]">
                {t.cash.cashSessionStatus}
              </h3>
            </div>

            <div className="space-y-2 py-2 min-h-[96px] flex flex-col justify-center">
              {cashSession ? (
                <>
                  <PosStatusStrip
                    title="Кассовая сессия"
                    message={`ОТКРЫТА • Баланс: ${cashSession.currentAmount} ₽`}
                    variant="success"
                  />
                  <div className="font-mono text-[10px] text-[var(--pos-text-secondary)] space-y-0.5 mt-2">
                    <div>Открыл: <strong className="text-[var(--pos-text-primary)]">{cashSession.openedBy}</strong></div>
                    <div>Время открытия: <strong className="text-[var(--pos-text-primary)]">{cashSession.openedAt}</strong></div>
                    <div>Начальный размен: <strong className="text-[var(--pos-text-primary)]">{cashSession.initialAmount} ₽</strong></div>
                  </div>
                </>
              ) : (
                <div className="space-y-3">
                  <div className="flex gap-2 items-end">
                    <PosFormRow
                      label="Стартовый баланс кассы (Размен)"
                      id="kkm-float-input"
                      error={sessionError || undefined}
                    >
                      <input
                        id="kkm-float-input"
                        type="text"
                        className="h-10 border border-[var(--pos-border)] px-3 font-mono text-sm bg-[var(--pos-surface)] text-[var(--pos-text-primary)] focus:outline-none w-[120px] rounded-none focus:border-[var(--pos-border-strong)]"
                        value={initialCash}
                        onChange={(e) => {
                          setInitialCash(e.target.value.replace(/\D/g, ''));
                          setSessionError(null);
                        }}
                      />
                    </PosFormRow>
                    
                    <PosButton
                      id="cash-open-session-btn"
                      variant="primary"
                      size="sm"
                      onClick={handleOpenSessionSubmit}
                      disabled={!currentOperator}
                      className="h-10 text-xs shrink-0"
                    >
                      {t.cash.openSession}
                    </PosButton>
                  </div>
                </div>
              )}
            </div>

            {cashSession && (
              <PosButton
                id="cash-close-session-btn"
                variant="secondary"
                size="sm"
                fullWidth
                onClick={closeCashSession}
              >
                {t.cash.closeSession}
              </PosButton>
            )}
          </div>

        </div>

        {/* Cash Drawer Transactions Ledger */}
        <div className="border border-[var(--pos-border)] bg-[var(--pos-surface)] p-6 space-y-4 select-none">
          <div className="flex items-center gap-1.5 shrink-0">
            <Wallet className="w-4 h-4 text-[var(--pos-text-secondary)] shrink-0" />
            <h3 className="font-mono text-xs font-bold uppercase tracking-widest text-[var(--pos-text-secondary)]">
              Журнал операций кассового ящика ({cashDrawerEvents.length})
            </h3>
          </div>

          {cashDrawerEvents.length === 0 ? (
            <div className="py-12 border border-dashed border-[var(--pos-border)] text-center text-xs text-[var(--pos-text-muted)] p-4 flex flex-col items-center justify-center">
              <Terminal className="w-6 h-6 opacity-30 mb-2 shrink-0 animate-pulse" />
              <span>Движение наличных средств не фиксировалось за текущую смену.</span>
            </div>
          ) : (
            <div className="border border-[var(--pos-border)] divide-y divide-[var(--pos-border)] max-h-[300px] overflow-y-auto pos-scrollarea-y pos-scrollbar-thin">
              {cashDrawerEvents.map((evt) => (
                <div key={evt.id} id={`drawer-log-item-${evt.id}`} className="p-3 flex items-center justify-between text-xs hover:bg-[var(--pos-surface-raised)]/30 transition-colors">
                  <div className="space-y-0.5">
                    <div className="flex items-center gap-2">
                      <span className={`px-1.5 py-0.5 font-mono text-[9px] font-bold uppercase ${evt.type === 'in' ? 'bg-emerald-50 text-emerald-600' : 'bg-red-50 text-red-600'}`}>
                        {evt.type === 'in' ? 'Внесение' : 'Изъятие'}
                      </span>
                      <span className="font-sans font-semibold text-[var(--pos-text-secondary)]">
                        {evt.reason}
                      </span>
                    </div>
                    <div className="font-mono text-[9px] text-[var(--pos-text-muted)]">
                      {evt.timestamp} • Оператор: {evt.operator}
                    </div>
                  </div>

                  <span className={`font-mono text-sm font-bold ${evt.type === 'in' ? 'text-[var(--pos-status-success)]' : 'text-[var(--pos-status-danger)]'}`}>
                    {evt.type === 'in' ? '+' : '-'}{evt.amount} ₽
                  </span>
                </div>
              ))}
            </div>
          )}
        </div>

      </div>

      {/* 1/4 Column actions and diagnostics list panel */}
      <div className="w-full lg:w-[320px] shrink-0">
        <PosActionRail className="h-full">
          <div className="h-14 border-b border-[var(--pos-border)] px-4 flex items-center justify-between bg-[var(--pos-surface-raised)] select-none shrink-0">
            <span className="font-mono text-xs font-bold uppercase tracking-wider text-[var(--pos-text-secondary)]">
              Инструменты кассы
            </span>
          </div>

          {/* Call register cash event */}
          <div className="p-4 border-b border-[var(--pos-border)] bg-[var(--pos-surface-raised)]/10 select-none space-y-3">
            <span className="font-mono text-[10px] font-bold uppercase tracking-widest text-[var(--pos-text-muted)] block">
              Внеочередные внесения/изъятия
            </span>
            <PosButton
              id="cash-drawer-event-btn"
              variant="secondary"
              size="md"
              fullWidth
              onClick={() => setEventModalOpen(true)}
              // Waiter permission bypass guard
              disabled={currentOperator?.role === 'waiter' || !cashSession}
              icon={<ArrowUpRight className="w-4 h-4" />}
            >
              Внести / Изъять Cash
            </PosButton>
          </div>

          {/* Sync status diagnostics */}
          <div className="p-4 border-b border-[var(--pos-border)] bg-[var(--pos-surface-raised)]/10 select-none space-y-3">
            <span className="font-mono text-[10px] font-bold uppercase tracking-widest text-[var(--pos-text-muted)] block">
              Синхронизация с backend
            </span>
            <div className="flex gap-2">
              <div className="h-10 flex-1 border border-[var(--pos-border)] bg-[var(--pos-surface)] px-3 flex items-center gap-2 font-mono text-[10px] font-bold uppercase text-[var(--pos-text-secondary)]">
                {syncStatus === 'online' ? <Wifi className="w-4 h-4 text-emerald-500" /> : <WifiOff className="w-4 h-4 text-red-500" />}
                <span>{syncStatus === 'online' ? 'Связь активна' : 'Есть проблемы sync'}</span>
              </div>

              <PosButton
                id="cash-sync-flush-btn"
                variant="primary"
                size="sm"
                className="text-xs"
                onClick={handleSyncOutboxClick}
                disabled={outboxCount === 0 || syncStatus === 'offline'}
                icon={<RefreshCw className="w-4 h-4" />}
              >
                Сброс ({outboxCount})
              </PosButton>
            </div>
          </div>

          {/* Logger diagnostics console feed */}
          <div className="flex-1 p-4 flex flex-col min-h-0 select-none">
            <span className="font-mono text-[10px] font-bold uppercase tracking-widest text-[var(--pos-text-muted)] block mb-2 shrink-0">
              Логи консоли Edge-сервера
            </span>
            
            <div className="flex-1 border border-[var(--pos-border)] bg-gray-950 text-emerald-400 font-mono text-[10px] p-3 overflow-y-auto space-y-1.5 pos-scrollarea-y pos-scrollbar-thin max-h-[220px]">
              {logEvents.map((evt) => (
                <div key={evt.id} className="leading-relaxed">
                  <span className="text-zinc-500 select-none">[{evt.time}]</span>{' '}
                  <span className={`
                    ${evt.type === 'warn' ? 'text-amber-400' : evt.type === 'success' ? 'text-emerald-300 font-bold' : 'text-zinc-300'}
                  `}>
                    {evt.msg}
                  </span>
                </div>
              ))}
            </div>
          </div>

        </PosActionRail>
      </div>

      {/* Subdialog CashDrawerEventModal */}
      <CashDrawerEventDialog
        isOpen={isEventModalOpen}
        onClose={() => setEventModalOpen(false)}
        onSubmit={addCashDrawerEvent}
      />

    </div>
  );
};

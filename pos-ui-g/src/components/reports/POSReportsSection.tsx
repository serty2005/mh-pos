import React from 'react';
import { usePOS } from '../../context/POSContext';
import { t } from '../../shared/i18n';
import { 
  PosMetricCard, 
  PosStatusStrip, 
  PosBanner,
  PosActionRail,
  PosDataRow
} from '../../shared/ui';
import { BarChart3, Clock, Wallet, Info, ArrowUpRight } from 'lucide-react';

export const POSReportsSection: React.FC = () => {
  const { 
    closedOrders, 
    currentOperator, 
    cashSession, 
    syncStatus, 
    outboxCount,
    cashDrawerEvents
  } = usePOS();

  // Aggregate amounts
  const totalClosedSalesCount = closedOrders.length;
  
  // Exclude cancelled/refunded sums for direct performance reporting
  const successfulClosedOrders = closedOrders.filter(c => c.status !== 'refunded' && c.status !== 'cancelled');
  const shiftSalesSum = successfulClosedOrders.reduce((sum, c) => sum + c.total, 0);

  const cashSalesSum = successfulClosedOrders
    .filter(c => c.paymentMethod === 'cash')
    .reduce((sum, c) => sum + c.total, 0);

  const cardSalesSum = successfulClosedOrders
    .filter(c => c.paymentMethod === 'card')
    .reduce((sum, c) => sum + c.total, 0);

  // Compute total unexpected cash drawer transactions (adjusting cash session initial sum if session open)
  const initialCashAmount = cashSession ? cashSession.initialAmount : 0;
  
  const totalCashInTransactions = cashDrawerEvents
    .filter(e => e.type === 'in')
    .reduce((sum, e) => sum + e.amount, 0);

  const totalCashOutTransactions = cashDrawerEvents
    .filter(e => e.type === 'out')
    .reduce((sum, e) => sum + e.amount, 0);

  const finalExpectedCash = initialCashAmount + cashSalesSum + totalCashInTransactions - totalCashOutTransactions;

  return (
    <div className="flex flex-col lg:flex-row flex-1 min-h-0 select-none bg-[var(--pos-bg)]">
      
      {/* 3/4 Column: Metrics Summaries and details panels */}
      <div className="flex-1 p-6 pos-scrollarea-y pos-scrollbar-thin overflow-y-auto space-y-6">
        
        {/* Subheader Title */}
        <div className="flex items-center gap-2 mb-2 select-none">
          <BarChart3 className="w-5 h-5 text-[var(--pos-text-secondary)]" />
          <h2 className="font-mono text-base font-bold uppercase tracking-wider text-[var(--pos-text-primary)]">
            {t.reports.summaryTitle}
          </h2>
        </div>

        {/* Offline Outbox Buffer alert indicators */}
        {syncStatus === 'offline' && (
          <PosBanner
            type="warning"
            message={t.reports.offlineNotice}
          />
        )}

        {/* Metrics Grid */}
        <div className="grid grid-cols-1 sm:grid-cols-2 md:grid-cols-3 gap-6">
          <PosMetricCard
            title={t.reports.shiftSales}
            value={`${shiftSalesSum} ₽`}
            statusText="В РЕАЛЕ"
            statusVariant="success"
          />
          <PosMetricCard
            title={t.reports.salesCount}
            value={totalClosedSalesCount}
            statusText="ЧЕКИ"
            statusVariant="info"
          />
          <PosMetricCard
            title={t.reports.expectedCash}
            value={`${finalExpectedCash} ₽`}
            statusText="КАССОВЫЙ БАЛАНС"
            statusVariant="warning"
          />
        </div>

        {/* Breakdown by Payment method table */}
        <div className="border border-[var(--pos-border)] bg-[var(--pos-surface)] p-6 space-y-4">
          <h3 className="font-mono text-xs font-bold uppercase tracking-widest text-[var(--pos-text-secondary)]">
            {t.reports.methodsBreakdown}
          </h3>

          <div className="divide-y divide-[var(--pos-border)]">
            <PosDataRow
              title="Оплаты Наличными (Cash captures)"
              subtitle="Фискальный ящик и разменная монета"
              value={`${cashSalesSum} ₽`}
              highlightValue
            />
            <PosDataRow
              title="Оплаты Картой (Manual bank transfers)"
              subtitle="Оплачено через банковские эквайринг-терминалы"
              value={`${cardSalesSum} ₽`}
            />
            <PosDataRow
              title="Стартовый баланс сейфа"
              subtitle="Начальное внесение при открытии кассовой смены"
              value={`${initialCashAmount} ₽`}
            />
          </div>
        </div>

        {/* Informative Cloud reporting boundary notice */}
        <div className="p-4 border border-blue-200 bg-blue-50/10 text-blue-800 dark:text-blue-100 flex gap-3 text-xs md:text-sm">
          <Info className="w-5 h-5 text-[var(--pos-status-info)] shrink-0 mt-0.5" />
          <span className="leading-relaxed font-sans">
            <strong>Обратите внимание:</strong> Текущий POS-терминал хранит только операционную историю за текущую бизнес-дату. Детальные OLAP-аналитические отчеты, ABC-анализ меню, инвентарные техкарты и расходы сырья выводятся на управляющем веб-интерфейсе <strong>Cloud Manager Workspace</strong>.
          </span>
        </div>

      </div>

      {/* 1/4 Column Right Rail: Technical details and shifts active status */}
      <div className="w-full lg:w-[320px] shrink-0">
        <PosActionRail className="h-full">
          <div className="h-14 border-b border-[var(--pos-border)] px-4 flex items-center justify-between bg-[var(--pos-surface-raised)] select-none shrink-0">
            <span className="font-mono text-xs font-bold uppercase tracking-wider text-[var(--pos-text-secondary)]">
              Операционный профиль
            </span>
          </div>

          <div className="p-4 space-y-4">
            
            {/* Operator status */}
            <div className="space-y-1.5 select-none">
              <span className="font-mono text-[10px] font-bold uppercase tracking-widest text-[var(--pos-text-muted)]">Личная сессия</span>
              <PosStatusStrip
                title={t.reports.personalShift}
                message={currentOperator ? currentOperator.employeeName : 'Нет открытой смены'}
                variant={currentOperator ? 'success' : 'danger'}
              />
            </div>

            {/* Cash KKM status */}
            <div className="space-y-1.5 select-none">
              <span className="font-mono text-[10px] font-bold uppercase tracking-widest text-[var(--pos-text-muted)]">Сессия кассы</span>
              <PosStatusStrip
                title={t.reports.cashSession}
                message={cashSession ? `Открыта в ${cashSession.openedAt.split(',')[1] || cashSession.openedAt}` : 'Кассовая сессия закрыта'}
                variant={cashSession ? 'warning' : 'neutral'}
              />
            </div>

            {/* Sync status indicators */}
            <div className="space-y-1.5 select-none">
              <span className="font-mono text-[10px] font-bold uppercase tracking-widest text-[var(--pos-text-muted)]">Соединение с Cloud</span>
              <PosStatusStrip
                title="Очередь транзакций"
                message={syncStatus === 'online' ? `Связь активна · ${outboxCount} в outbox` : `Есть проблемы sync · ${outboxCount} в outbox`}
                variant={syncStatus === 'online' ? 'success' : 'danger'}
              />
            </div>

          </div>
        </PosActionRail>
      </div>

    </div>
  );
};

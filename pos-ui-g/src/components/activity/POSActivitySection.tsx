import React, { useState } from 'react';
import { usePOS } from '../../context/POSContext';
import { t } from '../../shared/i18n';
import { 
  PosButton, 
  PosEmptyState, 
  PosPagination, 
  PosActionRail,
  PosDataRow,
  PosSearchInput,
  PosStatusBadge
} from '../../shared/ui';
import { 
  RefundDialog 
} from '../actions/RefundDialog';
import { 
  ReceiptText, 
  Search, 
  Printer, 
  RefreshCw, 
  Clock, 
  FileText,
  ShieldAlert
} from 'lucide-react';
import { ClosedOrder } from '../../types';

export const POSActivitySection: React.FC = () => {
  const { 
    closedOrders, 
    refundCheck, 
    partialRefundCheck, 
    reprintCheck,
    currentOperator,
    cashSession
  } = usePOS();

  const [searchQuery, setSearchQuery] = useState<string>('');
  const [selectedCheckId, setSelectedCheckId] = useState<string | null>(null);
  const [isRefundModalOpen, setRefundModalOpen] = useState<boolean>(false);

  // Pagination states
  const [page, setPage] = useState<number>(1);
  const itemsPerPage = 6;

  // Filter local closed orders
  const filteredChecks = closedOrders.filter(check => {
    const matchId = check.shortId.includes(searchQuery) || check.id.includes(searchQuery);
    const matchTable = check.tableName?.toLowerCase().includes(searchQuery.toLowerCase());
    const matchOperator = check.operator.toLowerCase().includes(searchQuery.toLowerCase());
    return matchId || matchTable || matchOperator;
  });

  const totalPages = Math.max(1, Math.ceil(filteredChecks.length / itemsPerPage));
  const paginatedChecks = filteredChecks.slice((page - 1) * itemsPerPage, page * itemsPerPage);

  const selectedCheck = closedOrders.find(c => c.id === selectedCheckId) || null;

  const handleFullRefundSubmit = (reason: string, disposition: 'waste' | 'return') => {
    if (selectedCheckId) {
      refundCheck(selectedCheckId, reason, disposition);
    }
  };

  const handlePartialRefundSubmit = (lineId: string, qty: number, reason: string, disposition: 'waste' | 'return') => {
    if (selectedCheckId) {
      partialRefundCheck(selectedCheckId, lineId, qty, reason, disposition);
    }
  };

  const getStatusVariant = (status: ClosedOrder['status']): 'success' | 'warning' | 'danger' | 'neutral' => {
    switch (status) {
      case 'refunded':
        return 'danger';
      case 'partially_refunded':
        return 'warning';
      case 'cancelled':
        return 'neutral';
      case 'closed':
      default:
        return 'success';
    }
  };

  const getStatusLabel = (status: ClosedOrder['status']) => {
    switch (status) {
      case 'refunded': return t.status.refunded;
      case 'partially_refunded': return t.status.partially_refunded;
      case 'cancelled': return t.status.cancelled;
      case 'closed':
      default: return t.status.closed;
    }
  };

  return (
    <div className="flex flex-1 flex-col lg:flex-row min-h-0 select-none bg-[var(--pos-bg)]">
      
      {/* 3/4 Column: Paginated listing checklist */}
      <div className="flex-1 flex flex-col min-h-0">
        
        {/* Search Bar Subheader */}
        <div className="border-b border-[var(--pos-border)] bg-[var(--pos-surface)] p-3 md:p-4 flex justify-between items-center shrink-0">
          <PosSearchInput
            id="activity-search-input"
            value={searchQuery}
            onChange={(value) => {
              setSearchQuery(value);
              setPage(1);
            }}
            onClear={() => setPage(1)}
            placeholder={t.activity.searchPlaceholder}
            clearLabel={t.common.clearSearch}
            className="sm:w-[320px]"
          />
        </div>

        {/* Closed Checks List container */}
        <div className="flex-1 p-6 pos-scrollarea-y pos-scrollbar-thin overflow-y-auto">
          {closedOrders.length === 0 ? (
            <PosEmptyState
              title={t.activity.emptyChecksTitle}
              description={t.activity.emptyChecksDesc}
              icon={<ReceiptText className="w-12 h-12" />}
            />
          ) : paginatedChecks.length === 0 ? (
            <PosEmptyState
              title={t.activity.noSearchResultsTitle}
              description={t.activity.noSearchResultsDesc}
              icon={<Search className="w-12 h-12" />}
            />
          ) : (
            <div className="border border-[var(--pos-border)] bg-[var(--pos-surface)] divide-y divide-[var(--pos-border)]">
              {paginatedChecks.map((check) => {
                const isSelected = check.id === selectedCheckId;
                
                return (
                  <button
                    key={check.id}
                    id={`activity-check-row-${check.id}`}
                    type="button"
                    onClick={() => setSelectedCheckId(check.id)}
                    className={`w-full text-left p-4 hover:bg-[var(--pos-surface-raised)]/50 select-none transition-colors border-none flex items-center justify-between gap-4 cursor-pointer
                      ${isSelected ? 'bg-[var(--pos-selected-order)] border-l-4 border-l-[var(--pos-action-primary)]' : ''}`}
                  >
                    <div className="flex flex-col gap-1 min-w-0">
                      <div className="flex items-center gap-2">
                        <span className="font-mono text-sm font-bold text-[var(--pos-text-primary)]">
                          {check.tableName || t.activity.noTable} • {t.activity.check} #{check.shortId}
                        </span>
                        <PosStatusBadge variant={getStatusVariant(check.status)}>
                          {getStatusLabel(check.status)}
                        </PosStatusBadge>
                      </div>
                      <span className="font-sans text-xs text-[var(--pos-text-muted)] truncate">
                        {t.activity.cashier}: {check.operator} • {check.closedAt}
                      </span>
                    </div>

                    <div className="flex flex-col items-end shrink-0">
                      <span className="font-mono text-base font-black text-[var(--pos-status-success)]">
                        {check.total} ₽
                      </span>
                      <span className="font-mono text-[9px] text-[var(--pos-text-muted)] uppercase">
                        {check.paymentMethod === 'cash' ? t.modals.paymentMethodCash : t.modals.paymentMethodCard}
                      </span>
                    </div>
                  </button>
                );
              })}
            </div>
          )}
        </div>

        {/* Footer Paginated Control Panel */}
        {filteredChecks.length > itemsPerPage && (
          <div className="p-4 shrink-0">
            <PosPagination
              currentPage={page}
              isFirstPage={page === 1}
              isLastPage={page === totalPages}
              onPrev={() => setPage(p => Math.max(1, p - 1))}
              onNext={() => setPage(p => Math.min(totalPages, p + 1))}
              label={`${t.activity.pageLabelPrefix} ${(page-1)*itemsPerPage+1} ${t.activity.pageLabelMiddle} ${Math.min(filteredChecks.length, page*itemsPerPage)} ${t.activity.pageLabelSuffix} ${filteredChecks.length}`}
            />
          </div>
        )}

      </div>

      {/* 1/4 Column Check Detail Panel */}
      <div className="w-full lg:w-[320px] shrink-0">
        <PosActionRail className="h-full">
          <div className="h-14 border-b border-[var(--pos-border)] px-4 flex items-center justify-between bg-[var(--pos-surface-raised)] select-none shrink-0">
            <span className="font-mono text-xs font-bold uppercase tracking-wider text-[var(--pos-text-secondary)]">
              {t.activity.checkDetails}
            </span>
          </div>

          <div className="flex-1 pos-scrollarea-y pos-scrollbar-thin overflow-y-auto">
            {!selectedCheck ? (
              <div className="p-8 text-center text-[var(--pos-text-muted)] h-full flex flex-col items-center justify-center">
                <FileText className="w-8 h-8 opacity-40 mb-2 shrink-0 animate-pulse" />
                <span className="font-sans text-xs">{t.activity.selectCheckHint}</span>
              </div>
            ) : (
              <div className="divide-y divide-[var(--pos-border)] select-none">
                
                {/* Meta details */}
                <div className="p-4 space-y-2 bg-[var(--pos-surface-raised)]/10">
                  <PosDataRow title={t.activity.checkId} value={selectedCheck.shortId} />
                  <PosDataRow title={t.activity.closedDate} value={selectedCheck.closedAt.split(',')[0]} />
                  <PosDataRow title={t.common.time} value={selectedCheck.closedAt.split(',')[1] || ''} />
                  <PosDataRow title={t.activity.paymentType} value={selectedCheck.paymentMethod === 'cash' ? t.modals.paymentMethodCash : t.modals.paymentMethodCard} />
                  {selectedCheck.refundReason && (
                    <div className="p-3 bg-red-50 dark:bg-red-950/10 border border-red-200 dark:border-red-900 border-l-4 border-l-red-500">
                      <span className="font-mono text-[9px] uppercase font-bold text-[var(--pos-status-danger)] block">{t.activity.refundReason}:</span>
                      <span className="font-sans text-xs text-red-800 dark:text-red-200 mt-0.5 block">"{selectedCheck.refundReason}"</span>
                    </div>
                  )}
                </div>

                {/* Ordered Items rows */}
                <div className="p-4 space-y-3">
                  <span className="font-mono text-[10px] font-bold uppercase tracking-widest text-[var(--pos-text-muted)] block">{t.activity.checkLines}</span>
                  <div className="space-y-2">
                    {selectedCheck.lines.map((line) => (
                      <div key={line.id} className="flex justify-between text-xs text-[var(--pos-text-secondary)]">
                        <div className="min-w-0 pr-2">
                          <span className="font-sans font-semibold hover:underline truncate block">{line.name}</span>
                          {line.selectedModifiers.map(m => (
                            <span key={m.optionId} className="font-mono text-[9px] text-[var(--pos-text-muted)] block ml-1.5">• {m.optionName}</span>
                          ))}
                        </div>
                        <span className="font-mono shrink-0 font-bold">{line.price} ₽ × {line.quantity}</span>
                      </div>
                    ))}
                  </div>
                </div>

                {/* Financial Ledger Operations timeline */}
                <div className="p-4 space-y-3">
                  <span className="font-mono text-[10px] font-bold uppercase tracking-widest text-[var(--pos-text-muted)] block">{t.activity.financialLog}</span>
                  <div className="space-y-4">
                    {selectedCheck.operations.map((op) => (
                      <div key={op.id} className="relative pl-4 border-l border-[var(--pos-border-strong)] flex flex-col gap-0.5">
                        <div className="absolute w-2 h-2 rounded-full bg-[var(--pos-border-strong)] -left-[4.5px] top-[4px]" />
                        <div className="flex justify-between items-baseline">
                          <span className="font-mono text-[10px] font-extrabold uppercase tracking-tight text-[var(--pos-text-primary)]">
                            {op.kind}
                          </span>
                          <span className="font-mono text-[10px] font-black text-[var(--pos-text-muted)]">{op.amount} ₽</span>
                        </div>
                        <span className="font-sans text-[10px] text-[var(--pos-text-muted)]">
                          {t.activity.performedBy}: {op.employee} • {op.timestamp}
                        </span>
                        {op.reason && (
                          <span className="font-sans text-[10px] text-[var(--pos-text-secondary)] font-medium italic">
                            "{op.reason}"
                          </span>
                        )}
                      </div>
                    ))}
                  </div>
                </div>

              </div>
            )}
          </div>

          {/* Locked indicators warning based on permissions */}
          {selectedCheck && currentOperator?.role === 'waiter' && (
            <div className="p-3 bg-red-50 dark:bg-neutral-900 border-t border-[var(--pos-border)]">
              <div className="flex items-start gap-2 text-red-800 dark:text-red-250 font-sans text-xs p-1 select-none">
                <ShieldAlert className="w-4 h-4 text-red-500 shrink-0 mt-0.5" />
                <span>{t.activity.waiterRefundBlocked}</span>
              </div>
            </div>
          )}

          {/* Reprint & Refund actions in footer */}
          {selectedCheck && (
            <div className="p-4 border-t border-[var(--pos-border)] bg-[var(--pos-surface-raised)] select-none grid grid-cols-2 gap-2 shrink-0">
              <PosButton
                id="activity-reprint-check-btn"
                variant="secondary"
                size="md"
                onClick={() => reprintCheck(selectedCheck.id)}
                icon={<Printer className="w-4 h-4" />}
              >
                {t.activity.print}
              </PosButton>

              <PosButton
                id="activity-refund-check-btn"
                variant="danger"
                size="md"
                onClick={() => setRefundModalOpen(true)}
                // Waiter permissions bypass guard
                disabled={currentOperator?.role === 'waiter' || !cashSession || selectedCheck.status === 'refunded'}
                icon={<RefreshCw className="w-4 h-4 animate-spin-slow" />}
              >
                {t.activity.refund}
              </PosButton>
            </div>
          )}
        </PosActionRail>
      </div>

      {/* Refund and return manager subdialog */}
      <RefundDialog
        isOpen={isRefundModalOpen}
        onClose={() => setRefundModalOpen(false)}
        check={selectedCheck}
        onFullRefund={handleFullRefundSubmit}
        onPartialRefund={handlePartialRefundSubmit}
      />

    </div>
  );
};

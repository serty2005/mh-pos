import React, { useState } from 'react';
import { usePOS } from '../../context/POSContext';
import { t } from '../../shared/i18n';
import { 
  PosButton, 
  PosDialog, 
  PosEmptyState, 
  PosQuantityStepper, 
  PosActionRail,
  PosStatusBadge,
  PosTabs
} from '../../shared/ui';
import { Armchair, Users, Plus, ReceiptText } from 'lucide-react';

export const POSFloorSection: React.FC = () => {
  const { 
    tables, 
    activeHallId, 
    setActiveHallId, 
    setSelectedTableId, 
    createOrderForTable, 
    activeOrders, 
    setCurrentSection,
    currentOperator,
    halls
  } = usePOS();

  const [isGuestModalOpen, setGuestModalOpen] = useState<boolean>(false);
  const [clickedTableId, setClickedTableId] = useState<string | null>(null);
  const [guestsCount, setGuestsCount] = useState<number>(2);

  const currentHallTables = tables.filter(t => t.hallId === activeHallId);

  const handleTableClick = (tableId: string) => {
    const table = tables.find(t => t.id === tableId);
    if (!table) return;

    if (table.status === 'occupied') {
      // Connect and auto route to order section
      setSelectedTableId(tableId);
      setCurrentSection('order');
    } else {
      // Free table, ask guests count first
      setClickedTableId(tableId);
      setGuestsCount(2); // standard fallback
      setGuestModalOpen(true);
    }
  };

  const handleCreateOrderSubmit = () => {
    if (clickedTableId) {
      createOrderForTable(clickedTableId, guestsCount);
      setSelectedTableId(clickedTableId);
      setGuestModalOpen(false);
      setClickedTableId(null);
      setCurrentSection('order');
    }
  };

  // Quick select order from the active orders list on the right side
  const handleActiveOrderClick = (tableId: string) => {
    setSelectedTableId(tableId);
    setCurrentSection('order');
  };

  return (
    <div className="flex flex-1 flex-col lg:flex-row min-h-0 select-none bg-[var(--pos-bg)]">
      
      {/* 3/4 Room Layout & Grid View */}
      <div className="flex-1 flex flex-col min-h-0">
        
        {/* Hall Tabs Header */}
        <PosTabs
          id="hall-tabs"
          idPrefix="hall-tab"
          items={halls.map((hall) => ({ id: hall.id, label: hall.name }))}
          activeId={activeHallId}
          onChange={setActiveHallId}
        />

        {/* Tables Matrix */}
        <div className="flex-1 p-6 overflow-y-auto pos-scrollarea-y pos-scrollbar-thin">
          {currentHallTables.length === 0 ? (
            <PosEmptyState
              title={t.floor.noTablesTitle}
              description={t.floor.noTablesDesc}
              icon={<Armchair className="w-12 h-12" />}
            />
          ) : (
            <div className="grid grid-cols-2 sm:grid-cols-3 md:grid-cols-4 xl:grid-cols-5 gap-6">
              {currentHallTables.map((table) => {
                const isActiveOrder = table.status === 'occupied';
                
                return (
                  <button
                    key={table.id}
                    id={`table-card-${table.id}`}
                    onClick={() => handleTableClick(table.id)}
                    className={`h-[120px] p-4 text-left border rounded-none flex flex-col justify-between cursor-pointer transition-all select-none group focus:ring-2 focus:ring-[var(--pos-focus-ring)] outline-none
                      ${isActiveOrder 
                        ? 'bg-[var(--pos-action-primary)] border-[var(--pos-action-primary)] text-[var(--pos-surface)] shadow-md' 
                        : 'bg-[var(--pos-surface)] border-[var(--pos-border)] hover:bg-[var(--pos-surface-raised)] text-[var(--pos-text-primary)]'
                      }`}
                  >
                    {/* Table Name and Guests Icon */}
                    <div className="flex items-start justify-between w-full">
                      <span className="font-mono text-base font-bold uppercase tracking-tight">
                        {t.common.table} {table.number}
                      </span>
                      {isActiveOrder && table.guestsCount ? (
                        <div className="flex items-center gap-1 font-mono text-xs opacity-75">
                          <Users className="w-3.5 h-3.5" />
                          <span>{table.guestsCount}</span>
                        </div>
                      ) : (
                        <Armchair className={`w-4 h-4 ${isActiveOrder ? 'text-[var(--pos-surface)] opacity-40' : 'text-[var(--pos-text-muted)] group-hover:text-[var(--pos-text-primary)]'}`} />
                      )}
                    </div>

                    {/* Waiter Assignee / Sum */}
                    <div className="flex flex-col gap-0.5 mt-auto">
                      {isActiveOrder ? (
                        <>
                          <span className="font-mono text-sm font-extrabold text-[var(--pos-surface)] tracking-tight">
                            {table.activeOrderSum || 0} ₽
                          </span>
                          <span className="font-sans text-[10px] uppercase tracking-wider opacity-60 truncate">
                            {table.waiter?.split(' ')[0] || t.floor.waiterFallback}
                          </span>
                        </>
                      ) : (
                        <span className="font-mono text-[10px] uppercase font-semibold text-[var(--pos-text-muted)] tracking-wider">
                          {t.status.free}
                        </span>
                      )}
                    </div>
                  </button>
                );
              })}
            </div>
          )}
        </div>
      </div>

      {/* 1/4 Column Right Rail: Active Orders list view */}
      <div className="w-full lg:w-[320px] border-t lg:border-t-0 border-l border-[var(--pos-border)] bg-[var(--pos-surface)] flex flex-col min-h-0 select-none shrink-0">
        <div className="h-14 border-b border-[var(--pos-border)] px-4 flex items-center bg-[var(--pos-surface-raised)] select-none shrink-0">
          <span className="font-mono text-xs font-bold uppercase tracking-wider text-[var(--pos-text-secondary)]">
            {t.floor.activeOrders} ({activeOrders.length})
          </span>
        </div>

        <div className="flex-1 pos-scrollarea-y pos-scrollbar-thin overflow-y-auto">
          {activeOrders.length === 0 ? (
            <div className="p-8 text-center text-[var(--pos-text-muted)] flex flex-col items-center justify-center h-full">
              <ReceiptText className="w-8 h-8 opacity-40 mb-2 shrink-0" />
              <span className="font-sans text-xs font-medium">{t.floor.noActiveOrders}</span>
            </div>
          ) : (
            <div className="divide-y divide-[var(--pos-border)]">
              {activeOrders.map((ord) => (
                <button
                  key={ord.id}
                  id={`active-order-item-${ord.id}`}
                  onClick={() => ord.tableId && handleActiveOrderClick(ord.tableId)}
                  className="w-full text-left p-4 hover:bg-[var(--pos-surface-raised)] active:bg-[var(--pos-border)] select-none transition-colors border-none flex flex-col gap-1 cursor-pointer"
                >
                  <div className="flex items-center justify-between">
                    <span className="font-mono text-xs font-bold text-[var(--pos-text-primary)]">
                      {ord.tableName || `${t.common.order} #${ord.shortId}`}
                    </span>
                    <span className="font-mono text-xs font-black text-[var(--pos-status-success)]">
                      {ord.total} ₽
                    </span>
                  </div>
                  <div className="flex items-center justify-between font-mono text-[10px] text-[var(--pos-text-muted)]">
                    <span>{ord.openedAt} • {ord.lines.length} {t.floor.positionsShort}</span>
                    <PosStatusBadge variant={ord.status === 'precheck_issued' ? 'warning' : 'info'}>
                      {ord.status === 'precheck_issued' ? t.status.precheck_issued : t.status.open}
                    </PosStatusBadge>
                  </div>
                </button>
              ))}
            </div>
          )}
        </div>
      </div>

      {/* Guests guest count numeric picker Modal dialog */}
      <PosDialog
        isOpen={isGuestModalOpen}
        onClose={() => setGuestModalOpen(false)}
        title={t.floor.newOrderTitle}
        footer={
          <>
            <PosButton variant="secondary" size="sm" onClick={() => setGuestModalOpen(false)}>
              {t.common.cancel}
            </PosButton>
            <PosButton 
              id="confirm-guests-btn"
              variant="primary" 
              size="sm" 
              onClick={handleCreateOrderSubmit}
              icon={<Plus className="w-4 h-4" />}
            >
              {t.floor.addOrder}
            </PosButton>
          </>
        }
      >
        <div className="flex flex-col items-center justify-center py-4">
          <span className="font-sans text-sm font-semibold text-[var(--pos-text-secondary)] mb-4 text-center">
            {t.floor.selectGuests}
          </span>

          <PosQuantityStepper
            id="guests-stepper"
            value={guestsCount}
            onChange={(val) => setGuestsCount(val)}
            min={1}
            max={12}
          />
          
          <div className="flex items-center gap-1 text-[var(--pos-text-muted)] font-mono text-[10px] uppercase tracking-wider mt-6">
            <Armchair className="w-3 md:w-4 h-3 md:h-4" />
            <span>{t.floor.waiter}: {currentOperator?.employeeName || t.floor.notAuthorized}</span>
          </div>
        </div>
      </PosDialog>
    </div>
  );
};

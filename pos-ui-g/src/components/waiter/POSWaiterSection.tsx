import React, { useMemo, useState } from 'react';
import {
  Armchair,
  Lock,
  Plus,
  Printer,
  ReceiptText,
  RotateCcw,
  ShieldAlert,
  Trash2,
  Users,
  Utensils,
} from 'lucide-react';

import { usePOS } from '../../context/POSContext';
import { canUsePermission } from '../../context/posContextHelpers';
import { t } from '../../shared/i18n';
import {
  PosBanner,
  PosButton,
  PosDialog,
  PosEmptyState,
  PosInlineStatusBadge,
  PosQuantityStepper,
  PosSearchInput,
  PosSelectableChip,
  PosSelectableTile,
} from '../../shared/ui';
import type { MenuItem, OrderLine, SelectedModifier, Table } from '../../types';
import { permissions } from '../../types';
import { ModifierSelectionDialog } from '../actions/ModifierSelectionDialog';

export const POSWaiterSection: React.FC = () => {
  const {
    activeHallId,
    activeOrders,
    changeLineQuantity,
    createOrderForTable,
    currentOperator,
    currentOrder,
    halls,
    issuePrecheck,
    menuItems,
    addMenuItemToOrder,
    removeOrderLine,
    reprintPrecheck,
    selectedTableId,
    setActiveHallId,
    setSelectedTableId,
    tables,
  } = usePOS();

  const [guestCount, setGuestCount] = useState<number>(2);
  const [searchQuery, setSearchQuery] = useState<string>('');
  const [selectedItemForModifiers, setSelectedItemForModifiers] = useState<MenuItem | null>(null);
  const [lineForVoid, setLineForVoid] = useState<OrderLine | null>(null);
  const [voidReason, setVoidReason] = useState<string>('');
  const [voidError, setVoidError] = useState<string>('');
  const [localNotice, setLocalNotice] = useState<string>('');

  const operatorPermissions = currentOperator?.permissions ?? [];
  const canCreateOrder = canUsePermission(operatorPermissions, permissions.ORDER_CREATE);
  const canAddLine = canUsePermission(operatorPermissions, permissions.ORDER_ADD_LINE);
  const canChangeQuantity = canUsePermission(operatorPermissions, permissions.ORDER_CHANGE_QTY);
  const canVoidLine = canUsePermission(operatorPermissions, permissions.ORDER_VOID_LINE);
  const canIssuePrecheck = canUsePermission(operatorPermissions, permissions.PRECHECK_ISSUE);
  const canReprintPrecheck = canUsePermission(operatorPermissions, permissions.PRECHECK_REPRINT);
  const isOrderLocked = currentOrder?.status === 'precheck_issued';
  const selectedTable = tables.find((table) => table.id === selectedTableId) ?? null;
  const visibleTables = tables.filter((table) => table.hallId === activeHallId);

  const filteredMenuItems = useMemo(() => {
    const query = searchQuery.trim().toLowerCase();
    return menuItems.filter((item) => !query || item.name.toLowerCase().includes(query));
  }, [menuItems, searchQuery]);

  const handleHallSelect = (hallId: string) => {
    setActiveHallId(hallId);
    setLocalNotice('');
  };

  const handleTableSelect = (table: Table) => {
    setSelectedTableId(table.id);
    setLocalNotice('');
  };

  const handleCreateOrder = async () => {
    if (!selectedTable || !canCreateOrder) return;
    await createOrderForTable(selectedTable.id, guestCount);
  };

  const handleMenuItemClick = async (item: MenuItem) => {
    setLocalNotice('');
    if (!currentOrder) {
      setLocalNotice(t.waiter.noOrderDesc);
      return;
    }
    if (isOrderLocked) {
      setLocalNotice(t.waiter.lockedReason);
      return;
    }
    if (!canAddLine) {
      setLocalNotice(t.waiter.permissionDenied);
      return;
    }
    if (!item.isAvailable || item.stopListBlocked) {
      setLocalNotice(t.waiter.stopListBlocked);
      return;
    }
    if (item.modifierGroups?.length) {
      setSelectedItemForModifiers(item);
      return;
    }
    await addMenuItemToOrder(item, []);
  };

  const handleModifierSubmit = async (selections: SelectedModifier[]) => {
    if (!selectedItemForModifiers) return;
    await addMenuItemToOrder(selectedItemForModifiers, selections);
    setSelectedItemForModifiers(null);
  };

  const openVoidDialog = (line: OrderLine) => {
    setLineForVoid(line);
    setVoidReason('');
    setVoidError('');
  };

  const submitVoidLine = async () => {
    const reason = voidReason.trim();
    if (reason.length < 4) {
      setVoidError(t.waiter.voidReasonRequired);
      return;
    }
    if (!lineForVoid) return;
    await removeOrderLine(lineForVoid.id, reason);
    setLineForVoid(null);
    setVoidReason('');
    setVoidError('');
  };

  return (
    <div id="waiter-mobile-runtime" className="flex-1 min-h-0 bg-[var(--pos-bg)] overflow-y-auto pos-scrollarea-y pos-scrollbar-thin">
      <div className="mx-auto flex w-full max-w-5xl flex-col gap-3 p-3 sm:p-4">
        <section className="border border-[var(--pos-border)] bg-[var(--pos-surface)]">
          <div className="flex items-start justify-between gap-3 border-b border-[var(--pos-border)] p-3">
            <div className="min-w-0">
              <div className="flex items-center gap-2">
                <SmartWaiterMark />
                <h2 className="font-mono text-sm font-black uppercase tracking-widest text-[var(--pos-text-primary)]">
                  {t.waiter.title}
                </h2>
              </div>
              <p className="mt-1 font-sans text-xs text-[var(--pos-text-muted)]">{t.waiter.subtitle}</p>
            </div>
            <PosInlineStatusBadge variant={isOrderLocked ? 'warning' : 'info'} className="shrink-0">
              {isOrderLocked ? t.status.locked : t.status.open}
            </PosInlineStatusBadge>
          </div>

          <div className="grid grid-cols-3 divide-x divide-[var(--pos-border)] text-[10px] font-mono font-bold uppercase tracking-wider text-[var(--pos-text-muted)]">
            <ContextCell label={t.waiter.tableStep} value={selectedTable ? `${t.common.table} ${selectedTable.number}` : t.common.none} />
            <ContextCell label={t.waiter.orderStep} value={currentOrder ? `#${currentOrder.shortId}` : t.common.none} />
            <ContextCell label={t.common.total} value={currentOrder ? `${currentOrder.total} ${t.common.ruble}` : `0 ${t.common.ruble}`} />
          </div>
        </section>

        <PosBanner type="info" message={t.waiter.noFinancialAuthority} />
        {isOrderLocked && <PosBanner type="warning" message={t.waiter.lockedReason} />}
        {localNotice && <PosBanner type="warning" message={localNotice} />}

        {!currentOperator ? (
          <PosEmptyState
            title={t.waiter.noShiftTitle}
            description={t.waiter.noShiftDesc}
            icon={<ShieldAlert className="h-12 w-12" />}
          />
        ) : (
          <>
            <section className="border border-[var(--pos-border)] bg-[var(--pos-surface)]">
              <WaiterSectionHeader icon={<Armchair className="h-4 w-4" />} title={t.waiter.selectHall} />
              <div className="p-3">
                <div className="flex gap-2 overflow-x-auto pos-scrollarea-x pos-scrollbar-thin pb-1">
                  {halls.map((hall) => (
                    <PosSelectableChip
                      key={hall.id}
                      id={`waiter-hall-${hall.id}`}
                      active={hall.id === activeHallId}
                      onClick={() => handleHallSelect(hall.id)}
                      className="shrink-0"
                    >
                      {hall.name}
                    </PosSelectableChip>
                  ))}
                </div>
              </div>
              <div className="grid grid-cols-2 gap-2 border-t border-[var(--pos-border)] p-3 sm:grid-cols-3">
                {visibleTables.map((table) => (
                  <PosSelectableTile
                    key={table.id}
                    id={`waiter-table-${table.id}`}
                    active={table.id === selectedTableId}
                    onClick={() => handleTableSelect(table)}
                    className="min-h-[96px] p-3"
                  >
                    <div className="flex h-full flex-col justify-between gap-3">
                      <div className="flex items-center justify-between gap-2">
                        <span className="font-mono text-sm font-black uppercase text-[var(--pos-text-primary)]">
                          {t.common.table} {table.number}
                        </span>
                        <Armchair className="h-4 w-4 text-[var(--pos-text-muted)]" />
                      </div>
                      <div className="flex items-center justify-between gap-2">
                        <PosInlineStatusBadge variant={table.status === 'occupied' ? 'info' : 'neutral'} className="text-[8px]">
                          {table.status === 'occupied' ? t.waiter.tableOccupied : t.waiter.tableFree}
                        </PosInlineStatusBadge>
                        {table.guestsCount ? (
                          <span className="inline-flex items-center gap-1 font-mono text-[10px] text-[var(--pos-text-muted)]">
                            <Users className="h-3 w-3" />
                            {table.guestsCount}
                          </span>
                        ) : null}
                      </div>
                    </div>
                  </PosSelectableTile>
                ))}
              </div>
            </section>

            <section className="grid grid-cols-1 gap-3 lg:grid-cols-[minmax(0,1fr)_360px]">
              <div className="space-y-3">
                <section className="border border-[var(--pos-border)] bg-[var(--pos-surface)]">
                  <WaiterSectionHeader
                    icon={<ReceiptText className="h-4 w-4" />}
                    title={t.waiter.activeOrders}
                    badge={String(activeOrders.length)}
                  />
                  <div className="divide-y divide-[var(--pos-border)]">
                    {activeOrders.length === 0 ? (
                      <div className="p-5 text-center font-sans text-xs text-[var(--pos-text-muted)]">
                        {t.waiter.noActiveOrders}
                      </div>
                    ) : (
                      activeOrders.map((order) => (
                        <PosSelectableTile
                          key={order.id}
                          id={`waiter-active-order-${order.id}`}
                          active={order.tableId === selectedTableId}
                          onClick={() => order.tableId && setSelectedTableId(order.tableId)}
                          className="border-none p-3"
                        >
                          <div className="flex items-center justify-between gap-3">
                            <div className="min-w-0">
                              <div className="truncate font-sans text-sm font-bold text-[var(--pos-text-primary)]">
                                {order.tableName || `${t.common.order} #${order.shortId}`}
                              </div>
                              <div className="font-mono text-[10px] uppercase text-[var(--pos-text-muted)]">
                                {order.lines.length} {t.floor.positionsShort} · {order.openedAt}
                              </div>
                            </div>
                            <div className="text-right">
                              <div className="font-mono text-sm font-black text-[var(--pos-text-primary)]">
                                {order.total} {t.common.ruble}
                              </div>
                              <PosInlineStatusBadge variant={order.status === 'precheck_issued' ? 'warning' : 'info'} className="mt-1 text-[8px]">
                                {order.status === 'precheck_issued' ? t.status.precheck_issued : t.status.open}
                              </PosInlineStatusBadge>
                            </div>
                          </div>
                        </PosSelectableTile>
                      ))
                    )}
                  </div>
                </section>

                <section className="border border-[var(--pos-border)] bg-[var(--pos-surface)]">
                  <WaiterSectionHeader icon={<Plus className="h-4 w-4" />} title={t.waiter.newOrder} />
                  <div className="grid grid-cols-1 gap-3 p-3 sm:grid-cols-[1fr_auto] sm:items-end">
                    <div className="space-y-2">
                      <div className="font-mono text-[10px] font-bold uppercase tracking-wider text-[var(--pos-text-muted)]">
                        {t.waiter.selectedTable}: {selectedTable ? `${t.common.table} ${selectedTable.number}` : t.common.none}
                      </div>
                      <div className="flex items-center justify-between gap-3 border border-[var(--pos-border)] bg-[var(--pos-surface-raised)] p-2">
                        <span className="font-mono text-xs font-bold uppercase text-[var(--pos-text-secondary)]">{t.waiter.guests}</span>
                        <PosQuantityStepper id="waiter-guests" value={guestCount} min={1} max={12} onChange={setGuestCount} />
                      </div>
                    </div>
                    <PosButton
                      id="waiter-create-order-btn"
                      variant="primary"
                      size="md"
                      onClick={handleCreateOrder}
                      disabled={!selectedTable || selectedTable.status === 'occupied' || !canCreateOrder}
                      icon={<Plus className="h-4 w-4" />}
                    >
                      {t.waiter.createOrder}
                    </PosButton>
                  </div>
                </section>

                <section className="border border-[var(--pos-border)] bg-[var(--pos-surface)]">
                  <WaiterSectionHeader icon={<Utensils className="h-4 w-4" />} title={t.waiter.menuStep} />
                  <div className="border-b border-[var(--pos-border)] p-3">
                    <PosSearchInput
                      id="waiter-menu-search"
                      value={searchQuery}
                      onChange={setSearchQuery}
                      placeholder={t.waiter.searchMenu}
                      clearLabel={t.common.clearSearch}
                    />
                  </div>
                  <div className="grid grid-cols-1 gap-2 p-3 sm:grid-cols-2">
                    {filteredMenuItems.length === 0 ? (
                      <div className="col-span-full p-5 text-center font-sans text-xs text-[var(--pos-text-muted)]">
                        {t.waiter.noMenuItems}
                      </div>
                    ) : (
                      filteredMenuItems.map((item) => {
                        const disabled = !currentOrder || isOrderLocked || !canAddLine || !item.isAvailable || item.stopListBlocked;
                        return (
                          <PosSelectableTile
                            key={item.id}
                            id={`waiter-menu-item-${item.id}`}
                            disabled={disabled}
                            onClick={() => handleMenuItemClick(item)}
                            className="min-h-[88px] p-3"
                          >
                            <div className="flex h-full items-start justify-between gap-3">
                              <div className="min-w-0">
                                <span className="line-clamp-2 font-sans text-sm font-semibold leading-snug text-[var(--pos-text-primary)]">
                                  {item.name}
                                </span>
                                <span className="mt-2 block font-mono text-xs font-black text-[var(--pos-text-secondary)]">
                                  {item.price} {t.common.ruble}
                                </span>
                              </div>
                              <div className="flex shrink-0 flex-col items-end gap-1">
                                {item.modifierGroups?.length ? (
                                  <PosInlineStatusBadge variant="neutral" className="text-[8px]">{t.menu.modifierShort}</PosInlineStatusBadge>
                                ) : null}
                                {isOrderLocked ? <Lock className="h-4 w-4 text-amber-500" /> : <Plus className="h-4 w-4 text-[var(--pos-text-muted)]" />}
                              </div>
                            </div>
                          </PosSelectableTile>
                        );
                      })
                    )}
                  </div>
                </section>
              </div>

              <aside className="border border-[var(--pos-border)] bg-[var(--pos-surface)] lg:sticky lg:top-3 lg:self-start">
                <WaiterSectionHeader
                  icon={<ReceiptText className="h-4 w-4" />}
                  title={currentOrder ? `${currentOrder.tableName} #${currentOrder.shortId}` : t.menu.currentOrder}
                  badge={currentOrder ? String(currentOrder.lines.length) : undefined}
                />
                {!currentOrder ? (
                  <PosEmptyState
                    title={t.waiter.noOrderTitle}
                    description={t.waiter.noOrderDesc}
                    icon={<ReceiptText className="h-12 w-12" />}
                  />
                ) : (
                  <>
                    <div className="divide-y divide-[var(--pos-border)]">
                      {currentOrder.lines.length === 0 ? (
                        <div className="p-5 text-center font-sans text-xs text-[var(--pos-text-muted)]">
                          {t.waiter.orderEmpty}
                        </div>
                      ) : (
                        currentOrder.lines.map((line) => (
                          <WaiterOrderLine
                            key={line.id}
                            line={line}
                            locked={isOrderLocked}
                            canChangeQuantity={canChangeQuantity}
                            canVoidLine={canVoidLine}
                            onQuantityChange={(value) => changeLineQuantity(line.id, value)}
                            onVoid={() => openVoidDialog(line)}
                          />
                        ))
                      )}
                    </div>

                    <div className="space-y-2 border-t border-[var(--pos-border)] bg-[var(--pos-surface-raised)]/30 p-3">
                      <TotalRow label={t.common.subtotal} value={currentOrder.subtotal} />
                      <TotalRow label={t.common.tax} value={currentOrder.tax} />
                      <TotalRow label={t.common.discount} value={currentOrder.discount} />
                      <div className="flex items-baseline justify-between border-t border-[var(--pos-border)] pt-2">
                        <span className="font-mono text-xs font-bold uppercase text-[var(--pos-text-muted)]">{t.common.total}</span>
                        <span className="font-mono text-xl font-black text-[var(--pos-status-success)]">
                          {currentOrder.total} {t.common.ruble}
                        </span>
                      </div>
                    </div>

                    <div className="grid grid-cols-1 gap-2 border-t border-[var(--pos-border)] p-3">
                      {isOrderLocked ? (
                        <PosButton
                          id="waiter-reprint-precheck-btn"
                          variant="secondary"
                          size="md"
                          onClick={reprintPrecheck}
                          disabled={!canReprintPrecheck}
                          icon={<RotateCcw className="h-4 w-4" />}
                        >
                          {t.waiter.reprintPrecheck}
                        </PosButton>
                      ) : (
                        <PosButton
                          id="waiter-issue-precheck-btn"
                          variant="primary"
                          size="md"
                          onClick={issuePrecheck}
                          disabled={currentOrder.lines.length === 0 || !canIssuePrecheck}
                          icon={<Printer className="h-4 w-4" />}
                        >
                          {t.waiter.issuePrecheck}
                        </PosButton>
                      )}
                      <div className="flex items-start gap-2 text-[10px] font-mono uppercase leading-snug text-[var(--pos-text-muted)]">
                        <ShieldAlert className="mt-0.5 h-3.5 w-3.5 shrink-0" />
                        <span>{t.waiter.backendAuthority}</span>
                      </div>
                    </div>
                  </>
                )}
              </aside>
            </section>
          </>
        )}
      </div>

      <ModifierSelectionDialog
        isOpen={Boolean(selectedItemForModifiers)}
        onClose={() => setSelectedItemForModifiers(null)}
        item={selectedItemForModifiers}
        onSubmit={handleModifierSubmit}
      />

      <PosDialog
        isOpen={Boolean(lineForVoid)}
        onClose={() => setLineForVoid(null)}
        title={t.waiter.voidTitle}
        footer={
          <>
            <PosButton variant="secondary" size="sm" onClick={() => setLineForVoid(null)}>
              {t.common.cancel}
            </PosButton>
            <PosButton id="waiter-confirm-void-btn" variant="danger" size="sm" onClick={submitVoidLine} icon={<Trash2 className="h-4 w-4" />}>
              {t.waiter.lineVoid}
            </PosButton>
          </>
        }
      >
        <div className="space-y-4">
          <p className="font-sans text-sm text-[var(--pos-text-secondary)]">{t.waiter.voidDesc}</p>
          <label className="block space-y-2">
            <span className="font-mono text-xs font-bold uppercase text-[var(--pos-text-secondary)]">{t.common.reason}</span>
            <input
              id="waiter-void-reason-input"
              className="h-12 w-full border border-[var(--pos-border)] bg-[var(--pos-surface)] px-3 font-sans text-sm text-[var(--pos-text-primary)] outline-none focus:border-[var(--pos-border-strong)]"
              value={voidReason}
              onChange={(event) => {
                setVoidReason(event.target.value);
                setVoidError('');
              }}
              placeholder={t.waiter.voidReasonPlaceholder}
            />
          </label>
          {voidError && <div className="font-sans text-xs font-semibold text-[var(--pos-status-danger)]">{voidError}</div>}
        </div>
      </PosDialog>
    </div>
  );
};

function WaiterSectionHeader({ icon, title, badge }: { icon: React.ReactNode; title: string; badge?: string }) {
  return (
    <div className="flex h-12 items-center justify-between gap-3 border-b border-[var(--pos-border)] bg-[var(--pos-surface-raised)] px-3">
      <div className="flex min-w-0 items-center gap-2">
        <span className="text-[var(--pos-text-secondary)]">{icon}</span>
        <h3 className="truncate font-mono text-xs font-black uppercase tracking-widest text-[var(--pos-text-primary)]">
          {title}
        </h3>
      </div>
      {badge ? <PosInlineStatusBadge variant="neutral">{badge}</PosInlineStatusBadge> : null}
    </div>
  );
}

function ContextCell({ label, value }: { label: string; value: string }) {
  return (
    <div className="min-w-0 p-3">
      <div className="truncate text-[var(--pos-text-muted)]">{label}</div>
      <div className="mt-1 truncate text-[var(--pos-text-primary)]">{value}</div>
    </div>
  );
}

function WaiterOrderLine({
  line,
  locked,
  canChangeQuantity,
  canVoidLine,
  onQuantityChange,
  onVoid,
}: {
  line: OrderLine;
  locked: boolean;
  canChangeQuantity: boolean;
  canVoidLine: boolean;
  onQuantityChange: (value: number) => void;
  onVoid: () => void;
}) {
  const controlsDisabled = locked || !canChangeQuantity;
  return (
    <div id={`waiter-order-line-${line.id}`} className="space-y-3 p-3">
      <div className="flex items-start justify-between gap-3">
        <div className="min-w-0">
          <div className="truncate font-sans text-sm font-bold text-[var(--pos-text-primary)]">{line.name}</div>
          {line.selectedModifiers.length > 0 ? (
            <div className="mt-1 space-y-0.5 border-l border-[var(--pos-border)] pl-2">
              {line.selectedModifiers.map((modifier) => (
                <div key={`${modifier.groupId}-${modifier.optionId}`} className="font-mono text-[10px] text-[var(--pos-text-muted)]">
                  + {modifier.optionName}
                </div>
              ))}
            </div>
          ) : null}
        </div>
        <div className="shrink-0 text-right font-mono text-xs font-black text-[var(--pos-text-primary)]">
          {line.totalPrice} {t.common.ruble}
        </div>
      </div>
      <div className="flex items-center justify-between gap-3">
        <PosQuantityStepper
          id={`waiter-line-qty-${line.id}`}
          value={line.quantity}
          onChange={onQuantityChange}
          disabled={controlsDisabled}
        />
        <PosButton
          id={`waiter-void-line-${line.id}`}
          variant="danger"
          size="sm"
          onClick={onVoid}
          disabled={locked || !canVoidLine}
          icon={<Trash2 className="h-4 w-4" />}
        >
          {t.waiter.lineVoid}
        </PosButton>
      </div>
      {locked ? (
        <div className="flex items-center gap-2 font-mono text-[10px] font-bold uppercase text-amber-500">
          <Lock className="h-3.5 w-3.5" />
          {t.waiter.lockedReason}
        </div>
      ) : null}
    </div>
  );
}

function TotalRow({ label, value }: { label: string; value: number }) {
  return (
    <div className="flex items-center justify-between font-mono text-xs text-[var(--pos-text-muted)]">
      <span>{label}</span>
      <span>{value} {t.common.ruble}</span>
    </div>
  );
}

function SmartWaiterMark() {
  return (
    <span className="inline-flex h-8 w-8 items-center justify-center border border-[var(--pos-border)] bg-[var(--pos-action-secondary)] text-[var(--pos-text-secondary)]">
      <Utensils className="h-4 w-4" />
    </span>
  );
}

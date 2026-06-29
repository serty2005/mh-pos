import React, { useEffect, useMemo, useState } from 'react';
import { usePOS } from '../../context/POSContext';
import { t } from '../../shared/i18n';
import { 
  PosButton, 
  PosEmptyState, 
  PosQuantityStepper, 
  PosSkeleton, 
  PosBanner,
  PosActionRail,
  PosStatusStrip,
  PosDialog,
  PosIconButton,
  PosInlineStatusBadge,
  PosRailHeader,
  PosSearchInput,
  PosSegmentedControl,
  PosSelectableTile,
  PosSelectableChip
} from '../../shared/ui';
import { 
  ModifierSelectionDialog 
} from '../actions/ModifierSelectionDialog';
import { 
  PaymentDialog 
} from '../actions/PaymentDialog';
import { 
  ActionsDialog 
} from '../actions/ActionsDialog';
import { 
  PrecheckCancelDialog 
} from '../actions/PrecheckCancelDialog';
import { 
  Search, 
  SlidersHorizontal, 
  Utensils, 
  Printer, 
  Banknote, 
  Lock,
  MessageSquare,
  BadgePercent,
  Plus,
  ReceiptText,
  CreditCard
} from 'lucide-react';
import { canUseAnyPermission, canUsePermission } from '../../context/posContextHelpers';
import { ClosedOrder, MenuItem, Order, OrderLine, PricingPolicy, SelectedModifier, permissions } from '../../types';

export const POSOrderSection: React.FC = () => {
  const {
    currentOrder,
    addMenuItemToOrder,
    editOrderLineModifiers,
    changeLineQuantity,
    removeOrderLine,
    updateCommentAndCourse,
    issuePrecheck,
    cancelPrecheck,
    reprintPrecheck,
    currentOperator,
    setCurrentSection,
    menuItems,
    pricingPolicies,
    applyPricingPolicy,
    tableModeEnabled,
    createCounterOrder,
    closedOrders,
  } = usePOS();

  // Search and Category filters
  const [activeCategory, setActiveCategory] = useState<string>('');
  const [searchQuery, setSearchQuery] = useState<string>('');
  const [hideStopListed, setHideStopListed] = useState<boolean>(false);

  // Skeletons loader simulated transition
  const [loading, setLoading] = useState<boolean>(false);

  // Submodals triggers
  const [selectedItemForMod, setSelectedItemForMod] = useState<MenuItem | null>(null);
  const [selectedLineForMod, setSelectedLineForMod] = useState<OrderLine | null>(null);
  const [isModModalOpen, setModModalOpen] = useState<boolean>(false);

  const [isPaymentModalOpen, setPaymentModalOpen] = useState<boolean>(false);
  const [clickedLineForActions, setClickedLineForActions] = useState<OrderLine | null>(null);
  const [isActionsModalOpen, setActionsModalOpen] = useState<boolean>(false);

  const [isPrecheckCancelOpen, setPrecheckCancelOpen] = useState<boolean>(false);
  const [isPricingModalOpen, setPricingModalOpen] = useState<boolean>(false);
  const [isTotalsBreakdownOpen, setTotalsBreakdownOpen] = useState<boolean>(false);
  const [selectedPricingPolicyId, setSelectedPricingPolicyId] = useState<string>('');
  const [pricingReason, setPricingReason] = useState<string>('');
  const [pricingLineId, setPricingLineId] = useState<string>('');
  const [selectedClosedOrder, setSelectedClosedOrder] = useState<ClosedOrder | null>(null);

  // Local notification banner for stop list alerts
  const [stopListAlertProduct, setStopListAlertProduct] = useState<string | null>(null);
  const operatorPermissions = currentOperator?.permissions ?? [];
  const canIssuePrecheck = canUsePermission(operatorPermissions, permissions.PRECHECK_ISSUE);
  const canRequestPrecheckCancel = canUsePermission(operatorPermissions, permissions.PRECHECK_CANCEL_REQUEST);
  const canCapturePayment = canUseAnyPermission(operatorPermissions, [
    permissions.PAYMENT_CASH,
    permissions.PAYMENT_CARD,
    permissions.PAYMENT_OTHER,
  ]);
  const canApplyDiscount = canUsePermission(operatorPermissions, permissions.PRICING_DISCOUNT_APPLY);
  const canApplySurcharge = canUsePermission(operatorPermissions, permissions.PRICING_SURCHARGE_APPLY);
  const canApplyPolicy = (policy: PricingPolicy): boolean => {
    const canApplyByKind = policy.kind === 'discount' ? canApplyDiscount : canApplySurcharge;
    if (!canApplyByKind) return false;
    return !policy.requiresPermission || canUsePermission(operatorPermissions, policy.requiresPermission);
  };
  const availablePricingPolicies = pricingPolicies.filter(canApplyPolicy);

  const categories = useMemo(() => {
    const unique = Array.from(new Set(menuItems.map((item) => item.category))).filter((id): id is string => Boolean(id));
    return unique.map((id) => ({ id, label: categoryLabel(id) }));
  }, [menuItems]);

  useEffect(() => {
    if (!activeCategory && categories[0]) {
      setActiveCategory(categories[0].id);
    }
  }, [activeCategory, categories]);

  // Filter Catalog Items
  const filteredItems = menuItems.filter(item => {
    const matchCat = !activeCategory || item.category === activeCategory;
    const matchSearch = item.name.toLowerCase().includes(searchQuery.toLowerCase());
    const matchStopList = !hideStopListed || !item.stopListActive;
    return matchCat && matchSearch && matchStopList;
  });

  const handleCategoryChange = (catId: string) => {
    setLoading(true);
    setActiveCategory(catId);
    setTimeout(() => {
      setLoading(false);
    }, 150);
  };

  const handleItemTileClick = (item: MenuItem) => {
    setStopListAlertProduct(null);

    // Stop list handling
    if (!item.isAvailable || item.stopListBlocked) {
      setStopListAlertProduct(item.name);
      return;
    }

    if (item.modifierGroups && item.modifierGroups.length > 0) {
      setSelectedItemForMod(item);
      setModModalOpen(true);
    } else {
      addMenuItemToOrder(item, []);
    }
  };

  const handleModifiersSubmit = (selections: SelectedModifier[]) => {
    if (selectedItemForMod && selectedLineForMod) {
      editOrderLineModifiers(selectedLineForMod.id, selections);
      setModModalOpen(false);
      setSelectedItemForMod(null);
      setSelectedLineForMod(null);
      return;
    }
    if (selectedItemForMod) {
      addMenuItemToOrder(selectedItemForMod, selections);
      setModModalOpen(false);
      setSelectedItemForMod(null);
    }
  };

  const openLineModifierEditor = (line: OrderLine) => {
    const item = menuItems.find((menuItem) => menuItem.id === line.itemId);
    if (!item?.modifierGroups?.length || currentOrder?.status !== 'open') return;
    setSelectedLineForMod(line);
    setSelectedItemForMod(item);
    setModModalOpen(true);
  };

  const selectedPricingPolicy = availablePricingPolicies.find((policy) => policy.id === selectedPricingPolicyId) ?? availablePricingPolicies[0];
  const selectedActionMenuItem = clickedLineForActions ? menuItems.find((menuItem) => menuItem.id === clickedLineForActions.itemId) : null;

  const openPricingDialog = () => {
    const firstPolicy = availablePricingPolicies[0];
    if (!firstPolicy) return;
    setSelectedPricingPolicyId(firstPolicy?.id ?? '');
    setPricingLineId(currentOrder?.lines[0]?.id ?? '');
    setPricingReason('');
    setPricingModalOpen(true);
  };

  const submitPricingPolicy = async () => {
    const policy = availablePricingPolicies.find((item) => item.id === selectedPricingPolicyId);
    if (!policy) return;
    const result = await applyPricingPolicy(policy.id, policy.scope === 'line' ? pricingLineId : '', pricingReason);
    if (result.success) {
      setPricingModalOpen(false);
    }
  };

  const handleLineClick = (line: OrderLine) => {
    setClickedLineForActions(line);
    setActionsModalOpen(true);
  };

  return (
    <div className="flex flex-1 flex-col lg:flex-row min-h-0 select-none bg-[var(--pos-bg)]">
      
      {/* 3/4 Left Hand Main Menu Content Workspace */}
      <div className="flex-1 flex flex-col min-h-0">
        
        {/* Subheader: Category switcher & Search input */}
        <div className="border-b border-[var(--pos-border)] bg-[var(--pos-surface)] p-3 md:p-4 flex flex-col sm:flex-row sm:items-center justify-between gap-3 shrink-0">
          
          {/* Swaps */}
          <PosSegmentedControl
            items={categories}
            activeId={activeCategory}
            onChange={handleCategoryChange}
            idPrefix="cat-chip"
            className="pb-1 sm:pb-0"
            itemClassName="px-6 font-extrabold"
          />

          {/* Search Bar Input */}
          <PosSearchInput
            id="dish-search-input"
            value={searchQuery}
            onChange={setSearchQuery}
            placeholder={t.common.search}
            clearLabel={t.common.clearSearch}
            className="sm:w-[240px] shrink-0"
          />
          <PosSelectableChip
            onClick={() => setHideStopListed((value) => !value)}
            active={hideStopListed}
            className="text-[10px] font-extrabold shrink-0"
          >
            {t.menu.notInStopList}
          </PosSelectableChip>
        </div>

        {/* Local warning panels if any alerts triggered */}
        {stopListAlertProduct && (
          <div className="px-6 pt-4 shrink-0">
            <PosBanner
              type="danger"
              message={`${t.menu.stopListAlertPrefix} "${stopListAlertProduct}" ${t.menu.stopListAlertSuffix}`}
              action={
                <PosIconButton
                  onClick={() => setStopListAlertProduct(null)} 
                  variant="ghost"
                  size="sm"
                  label={t.common.close}
                  icon={<span className="font-mono text-base leading-none">×</span>}
                />
              }
            />
          </div>
        )}

        {/* Menu Tile Grid Room */}
        <div className="flex-1 p-6 overflow-y-auto pos-scrollarea-y pos-scrollbar-thin">
          {!currentOrder ? (
            !tableModeEnabled ? (
              <div className="h-full min-h-[360px] flex flex-col items-center justify-center gap-5 text-center">
                <PosButton
                  id="counter-start-order-plus-btn"
                  variant="primary"
                  size="critical"
                  onClick={createCounterOrder}
                  aria-label={t.menu.startCounterOrder}
                  className="h-28 w-28 rounded-none px-0 text-5xl"
                >
                  <Plus className="h-14 w-14" aria-hidden="true" />
                </PosButton>
                <div className="space-y-1">
                  <div className="font-mono text-sm font-black uppercase tracking-wider text-[var(--pos-text-primary)]">
                    {t.menu.noSaleOrderTitle}
                  </div>
                  <div className="font-sans text-xs text-[var(--pos-text-muted)]">
                    {t.menu.noSaleOrderDesc}
                  </div>
                </div>
              </div>
            ) : (
              <PosEmptyState
                title={t.blocks.noOrderSelected}
                description={t.blocks.noOrderSelectedDesc}
                buttonLabel={t.blocks.toTables}
                onAction={() => setCurrentSection('floor')}
                icon={<Utensils className="w-12 h-12" />}
              />
            )
          ) : loading ? (
            <PosSkeleton type="grid" />
          ) : filteredItems.length === 0 ? (
            <PosEmptyState
              title={t.menu.noItemsTitle}
              description={t.menu.noItemsDesc}
              icon={<Search className="w-12 h-12" />}
            />
          ) : (
            <div className="grid grid-cols-2 md:grid-cols-3 xl:grid-cols-4 gap-4 p-1">
              {filteredItems.map((item) => {
                const isAvailable = item.isAvailable && !item.stopListBlocked;
                const hasModifiers = item.modifierGroups && item.modifierGroups.length > 0;
                
                return (
                  <PosSelectableTile
                    key={item.id}
                    id={`menu-tile-${item.id}`}
                    onClick={() => handleItemTileClick(item)}
                    className={`h-[116px] p-3 text-left border rounded-none flex flex-col justify-between cursor-pointer select-none relative group transition-all text-[var(--pos-text-primary)] focus:ring-2 focus:ring-[var(--pos-focus-ring)] outline-none
                      ${isAvailable 
                        ? 'bg-[var(--pos-surface)] border-[var(--pos-border)] hover:bg-[var(--pos-surface-raised)] hover:border-[var(--pos-border-strong)]' 
                        : 'bg-zinc-100 dark:bg-zinc-850/50 border-dashed border-[var(--pos-border)] opacity-40 cursor-not-allowed filter saturate-50'
                      }`}
                  >
                    {/* Item Title */}
                    <span id={`menu-tile-title-${item.id}`} className="font-sans text-xs md:text-sm font-semibold tracking-tight text-[var(--pos-text-secondary)] line-clamp-2 pr-1 select-none leading-snug">
                      {item.name}
                    </span>

                    {/* Price Tag & Badges */}
                    <div className="flex items-center justify-between w-full mt-auto">
                      <span className="font-mono text-sm font-extrabold text-[var(--pos-text-primary)]">
                        {item.price} {t.common.ruble}
                      </span>
                      {hasModifiers && isAvailable && (
                        <PosInlineStatusBadge variant="neutral" className="text-[9px] px-1 py-0 relative bg-[var(--pos-surface)] shrink-0">
                          {t.menu.modifierShort}
                        </PosInlineStatusBadge>
                      )}
                      {!isAvailable && (
                        <PosInlineStatusBadge variant="danger" className="text-[8px] px-1 py-0 shrink-0 bg-white">
                          ×
                        </PosInlineStatusBadge>
                      )}
                      {item.stopListActive && !item.stopListBlocked && item.stopListAvailableQuantity !== undefined && (
                        <PosInlineStatusBadge variant="warning" className="text-[8px] px-1 py-0 shrink-0 bg-white">
                          {t.menu.stopListQty}: {item.stopListAvailableQuantity}
                        </PosInlineStatusBadge>
                      )}
                    </div>
                  </PosSelectableTile>
                );
              })}
            </div>
          )}
        </div>
      </div>

      {/* 1/4 Right Column Workspace: Current Active Order Rail */}
      <div className="w-full lg:w-[320px] shrink-0">
        <PosActionRail className="h-full">
          
          {/* Header descriptor */}
          <PosRailHeader
            title={currentOrder ? `${currentOrder.tableName || t.activity.noTable} (${t.common.order} #${currentOrder.shortId})` : (!tableModeEnabled ? t.activity.recentOrders : t.menu.currentOrder)}
            badge={currentOrder?.status === 'precheck_issued' ? (
              <PosInlineStatusBadge variant="warning" className="text-[9px] px-1.5 py-0.5 animate-pulse bg-[var(--pos-status-warning)] text-zinc-950">
                {t.status.locked}
              </PosInlineStatusBadge>
            ) : undefined}
          />

          {/* Lines iterator */}
          <div className="flex-1 pos-scrollarea-y pos-scrollbar-thin overflow-y-auto">
            {!currentOrder ? (
              !tableModeEnabled ? (
                <RecentClosedOrdersRail
                  orders={closedOrders}
                  onSelect={setSelectedClosedOrder}
                />
              ) : (
                <div className="p-8 text-center text-[var(--pos-text-muted)] h-full flex flex-col items-center justify-center gap-4">
                  <SlidersHorizontal className="w-8 h-8 opacity-35 shrink-0" />
                  <span className="font-mono text-xs font-bold uppercase tracking-wider">{t.menu.orderEmpty}</span>
                </div>
              )
            ) : currentOrder.lines.length === 0 ? (
              <div className="p-8 text-center text-[var(--pos-text-muted)] h-full flex flex-col items-center justify-center">
                <Utensils className="w-8 h-8 opacity-35 mb-2 shrink-0 animate-pulse" />
                <span className="font-sans text-xs">{t.menu.addDishesHint}</span>
              </div>
            ) : (
              <div className="divide-y divide-[var(--pos-border)]">
                {currentOrder.lines.map((line) => {
                  const isLocked = currentOrder.status === 'precheck_issued';
                  const lineMenuItem = menuItems.find((menuItem) => menuItem.id === line.itemId);
                  const maxQty = lineMenuItem?.singleUnitPerLine ? 1 : undefined;

                  return (
                    <div 
                      key={line.id} 
                      id={`order-line-${line.id}`}
                      data-order-line-id={line.id}
                      onClick={() => {
                        if (!isLocked) handleLineClick(line);
                      }}
                      className="p-4 flex flex-col gap-2 bg-[var(--pos-surface)] hover:bg-[var(--pos-surface-raised)]/30 transition-colors cursor-pointer"
                    >
                      {/* Name of item and selections */}
                      <div className="flex items-start justify-between gap-1.5 select-none text-[var(--pos-text-primary)]">
                        <div 
                          className="flex-1 min-w-0"
                        >
                          <span className="font-sans text-xs font-semibold leading-relaxed hover:underline block truncate">
                            {line.name}
                          </span>
                          
                          {/* Modifiers selected lists nested */}
                          {line.selectedModifiers.length > 0 && (
                            <div className="font-mono text-[9px] text-[var(--pos-text-muted)] font-medium mt-0.5 space-y-0.5 pl-1.5 border-l border-[var(--pos-border-strong)]">
                              {line.selectedModifiers.map(sel => (
                                <div key={sel.optionId} className="flex justify-between">
                                  <span>+ {sel.optionName}</span>
                                  <span>{sel.price > 0 ? `(${sel.price} ${t.common.ruble})` : ''}</span>
                                </div>
                              ))}
                            </div>
                          )}

                          {/* Comment or Course indicators */}
                          {(line.comment || (line.course && line.course > 1)) && (
                            <div className="flex flex-wrap items-center gap-1.5 font-mono text-[9px] text-[var(--pos-status-info)] font-bold mt-1">
                              {line.course && line.course > 1 && (
                                <span className="px-1 border border-blue-200 bg-blue-50/50 uppercase">{t.common.course} {line.course}</span>
                              )}
                              {line.comment && (
                                <span className="flex items-center gap-1">
                                  <MessageSquare className="w-2.5 h-2.5 shrink-0" />
                                  <span className="max-w-[120px] truncate">"{line.comment}"</span>
                                </span>
                              )}
                            </div>
                          )}
                        </div>
                        <span className="font-mono text-xs font-black tracking-tight shrink-0">{line.price} {t.common.ruble}</span>
                      </div>

                      {/* Line Counts controls / Quantities */}
                      <div className="flex items-center justify-between mt-1" onClick={(event) => event.stopPropagation()}>
                        <PosQuantityStepper
                          id={`line-qty-${line.id}`}
                          value={line.quantity}
                          onChange={(val) => changeLineQuantity(line.id, val)}
                          disabled={isLocked}
                          max={maxQty}
                        />

                        <span className="font-mono text-xs font-black text-[var(--pos-text-primary)]">
                          {line.totalPrice} {t.common.ruble}
                        </span>
                      </div>
                    </div>
                  );
                })}
              </div>
            )}
          </div>

          {/* Locked / Prechecked state banner indicators */}
          {currentOrder?.status === 'precheck_issued' && (
            <div className="p-3 shrink-0">
              <div className="p-3 border border-amber-200 bg-amber-50/10 text-amber-500 rounded-none flex items-center gap-2 select-none">
                <Lock className="w-4 h-4 shrink-0" />
                <span className="font-mono text-[10px] font-bold uppercase tracking-wider leading-snug">
                  {t.menu.precheckLocked}
                </span>
              </div>
            </div>
          )}

          {/* Totals summaries container */}
          {currentOrder && (
            <button
              type="button"
              className="w-full border-t border-[var(--pos-border)] p-4 bg-[var(--pos-surface-raised)]/20 select-none space-y-2 shrink-0 text-left outline-none hover:bg-[var(--pos-surface-raised)] focus:ring-2 focus:ring-[var(--pos-focus-ring)]"
              onClick={() => setTotalsBreakdownOpen(true)}
            >
              <div className="flex justify-between items-baseline font-mono text-xs text-[var(--pos-text-muted)]">
                <span>{t.common.subtotal}:</span>
                <span>{currentOrder.subtotal} {t.common.ruble}</span>
              </div>
              <div className="flex justify-between items-baseline font-mono text-xs text-[var(--pos-text-muted)]">
                <span>{t.common.tax} (10%):</span>
                <span>{currentOrder.tax} {t.common.ruble}</span>
              </div>
              {(currentOrder.discount > 0 || availablePricingPolicies.length > 0) && (
                <div className="flex justify-between items-baseline font-mono text-xs text-[var(--pos-text-muted)]">
                  <span>{t.common.discount}:</span>
                  <span>{currentOrder.discount} {t.common.ruble}</span>
                </div>
              )}
              <div className="flex justify-between items-baseline font-mono text-base font-black text-[var(--pos-text-primary)] border-t border-[var(--pos-border)] pt-2 uppercase tracking-wide">
                <span>{t.common.total}:</span>
                <span className="text-sm md:text-xl font-black text-[var(--pos-status-success)]">{currentOrder.total} {t.common.ruble}</span>
              </div>
            </button>
          )}

          {/* Action buttons footer */}
          {currentOrder && (
            <div className="p-4 border-t border-[var(--pos-border)] bg-[var(--pos-surface-raised)] select-none grid grid-cols-2 gap-2 shrink-0">
              {currentOrder.status === 'open' ? (
                <>
                  {/* Actions dialog */}
                  <PosButton
                    id="order-actions-btn"
                    variant="secondary"
                    size="md"
                    onClick={() => {
                      if (currentOrder?.lines && currentOrder.lines.length > 0) {
                        setClickedLineForActions(currentOrder.lines[0]);
                        setActionsModalOpen(true);
                      }
                    }}
                    disabled={currentOrder.lines.length === 0}
                  >
                    {t.common.actions}
                  </PosButton>

                  <PosButton
                    id="order-pricing-btn"
                    variant="secondary"
                    size="md"
                    onClick={openPricingDialog}
                    disabled={currentOrder.lines.length === 0 || availablePricingPolicies.length === 0}
                    icon={<BadgePercent className="w-4 h-4" />}
                  >
                    {t.pricing.action}
                  </PosButton>

                  {/* Counter-mode: direct payment without precheck; table-mode: issue precheck first */}
                  {tableModeEnabled ? (
                    <PosButton
                      id="order-precheck-btn"
                      variant="primary"
                      size="md"
                      onClick={issuePrecheck}
                      disabled={currentOrder.lines.length === 0 || !canIssuePrecheck}
                      icon={<Printer className="w-4 h-4" />}
                    >
                      {t.menu.precheck}
                    </PosButton>
                  ) : (
                    <PosButton
                      id="order-counter-pay-btn"
                      variant="primary"
                      size="md"
                      onClick={() => setPaymentModalOpen(true)}
                      disabled={currentOrder.lines.length === 0 || !canCapturePayment}
                      icon={<Banknote className="w-4 h-4" />}
                    >
                      {t.menu.payCounter}
                    </PosButton>
                  )}
                </>
              ) : (
                <>
                  {/* Checkout Payment modal triggers */}
                  <PosButton
                    id="order-checkout-btn"
                    variant="primary"
                    size="md"
                    onClick={() => setPaymentModalOpen(true)}
                    disabled={!canCapturePayment}
                    icon={<Banknote className="w-4 h-4" />}
                  >
                    {t.sections.cash}
                  </PosButton>

                  {/* Override locked cancel precheck trigger code */}
                  <PosButton
                    id="order-precheck-cancel-btn"
                    variant="danger"
                    size="md"
                    onClick={() => setPrecheckCancelOpen(true)}
                    disabled={!canRequestPrecheckCancel}
                    icon={<SlidersHorizontal className="w-4 h-4" />}
                  >
                    {t.common.cancel}
                  </PosButton>
                </>
              )}
            </div>
          )}

        </PosActionRail>
      </div>

      {/* Sub Modals components declarations */}
      <ModifierSelectionDialog
        isOpen={isModModalOpen}
        onClose={() => {
          setModModalOpen(false);
          setSelectedLineForMod(null);
          setSelectedItemForMod(null);
        }}
        item={selectedItemForMod}
        initialSelections={selectedLineForMod?.selectedModifiers ?? []}
        mode={selectedLineForMod ? 'edit' : 'add'}
        onSubmit={handleModifiersSubmit}
      />

      <PricingPolicyDialog
        isOpen={isPricingModalOpen}
        onClose={() => setPricingModalOpen(false)}
        policies={availablePricingPolicies}
        selectedPolicyId={selectedPricingPolicyId}
        selectedPolicy={selectedPricingPolicy}
        onPolicyChange={setSelectedPricingPolicyId}
        lines={currentOrder?.lines ?? []}
        selectedLineId={pricingLineId}
        onLineChange={setPricingLineId}
        reason={pricingReason}
        onReasonChange={setPricingReason}
        onSubmit={submitPricingPolicy}
      />

      <PaymentDialog
        isOpen={isPaymentModalOpen}
        onClose={() => setPaymentModalOpen(false)}
      />

      <ClosedOrderDialog
        order={selectedClosedOrder}
        onClose={() => setSelectedClosedOrder(null)}
      />

      <TotalsBreakdownDialog
        isOpen={isTotalsBreakdownOpen}
        onClose={() => setTotalsBreakdownOpen(false)}
        order={currentOrder}
      />

      <ActionsDialog
        isOpen={isActionsModalOpen}
        onClose={() => setActionsModalOpen(false)}
        line={clickedLineForActions}
        menuItem={selectedActionMenuItem}
        onUpdateLine={(comment, course, modifiers) => {
          if (clickedLineForActions) {
            updateCommentAndCourse(clickedLineForActions.id, comment, course);
            if (selectedActionMenuItem?.modifierGroups?.length) {
              editOrderLineModifiers(clickedLineForActions.id, modifiers);
            }
          }
        }}
        onDeleteLine={() => {
          if (clickedLineForActions) {
            removeOrderLine(clickedLineForActions.id);
          }
        }}
      />

      <PrecheckCancelDialog
        isOpen={isPrecheckCancelOpen}
        onClose={() => setPrecheckCancelOpen(false)}
        onConfirm={cancelPrecheck}
      />

    </div>
  );
};

function RecentClosedOrdersRail({
  orders,
  onSelect,
}: {
  orders: ClosedOrder[];
  onSelect: (order: ClosedOrder) => void;
}) {
  if (orders.length === 0) {
    return (
      <div className="p-8 text-center text-[var(--pos-text-muted)] h-full flex flex-col items-center justify-center">
        <ReceiptText className="w-8 h-8 opacity-40 mb-2 shrink-0" />
        <span className="font-sans text-xs">{t.activity.emptyChecksTitle}</span>
      </div>
    );
  }

  return (
    <div className="divide-y divide-[var(--pos-border)]">
      {orders.map((order) => {
        const PaymentIcon = order.paymentMethod === 'card' ? CreditCard : Banknote;
        return (
          <PosSelectableTile
            key={order.id}
            id={`recent-order-row-${order.id}`}
            onClick={() => onSelect(order)}
            className="w-full border-none p-4 text-left hover:bg-[var(--pos-surface-raised)] active:bg-[var(--pos-border)] transition-colors flex items-center justify-between gap-3"
          >
            <div className="min-w-0 space-y-1">
              <div className="flex items-center gap-2">
                <span className="font-mono text-xs font-black text-[var(--pos-text-primary)]">
                  {t.activity.check} #{order.shortId}
                </span>
                <PosInlineStatusBadge variant="success" className="text-[9px] px-1.5 py-0.5">
                  {order.paymentMethod === 'cash' ? t.modals.paymentMethodCash : t.modals.paymentMethodCard}
                </PosInlineStatusBadge>
              </div>
              <span className="block truncate font-sans text-[11px] text-[var(--pos-text-muted)]">
                {order.closedAt}
              </span>
            </div>
            <div className="flex shrink-0 items-center gap-2">
              <PaymentIcon className="h-4 w-4 text-[var(--pos-text-muted)]" />
              <span className="font-mono text-sm font-black text-[var(--pos-status-success)]">
                {order.total} {t.common.ruble}
              </span>
            </div>
          </PosSelectableTile>
        );
      })}
    </div>
  );
}

function ClosedOrderDialog({
  order,
  onClose,
}: {
  order: ClosedOrder | null;
  onClose: () => void;
}) {
  return (
    <PosDialog
      isOpen={Boolean(order)}
      onClose={onClose}
      title={order ? `${t.activity.check} #${order.shortId}` : t.activity.checkDetails}
      footer={
        <PosButton variant="primary" size="sm" onClick={onClose}>
          {t.common.close}
        </PosButton>
      }
    >
      {order && (
        <div className="space-y-4">
          <div className="grid grid-cols-2 gap-3 border border-[var(--pos-border)] bg-[var(--pos-surface-raised)] p-4">
            <div className="space-y-1">
              <span className="block font-mono text-[10px] font-bold uppercase text-[var(--pos-text-muted)]">
                {t.activity.paymentType}
              </span>
              <span className="font-sans text-sm font-semibold text-[var(--pos-text-primary)]">
                {order.paymentMethod === 'cash' ? t.modals.paymentMethodCash : t.modals.paymentMethodCard}
              </span>
            </div>
            <div className="space-y-1 text-right">
              <span className="block font-mono text-[10px] font-bold uppercase text-[var(--pos-text-muted)]">
                {t.common.total}
              </span>
              <span className="font-mono text-lg font-black text-[var(--pos-status-success)]">
                {order.total} {t.common.ruble}
              </span>
            </div>
          </div>

          <div className="space-y-2">
            <span className="font-mono text-[10px] font-bold uppercase tracking-widest text-[var(--pos-text-muted)]">
              {t.activity.checkLines}
            </span>
            <div className="divide-y divide-[var(--pos-border)] border border-[var(--pos-border)]">
              {order.lines.map((line) => (
                <div key={line.id} className="flex justify-between gap-3 p-3 text-xs">
                  <div className="min-w-0">
                    <span className="block truncate font-sans font-semibold text-[var(--pos-text-primary)]">
                      {line.name}
                    </span>
                    {line.selectedModifiers.map((modifier) => (
                      <span key={modifier.optionId} className="block truncate font-mono text-[9px] text-[var(--pos-text-muted)]">
                        + {modifier.optionName}
                      </span>
                    ))}
                  </div>
                  <span className="shrink-0 font-mono font-bold text-[var(--pos-text-secondary)]">
                    {line.price} {t.common.ruble} x {line.quantity}
                  </span>
                </div>
              ))}
            </div>
          </div>
        </div>
      )}
    </PosDialog>
  );
}

function TotalsBreakdownDialog({
  isOpen,
  onClose,
  order,
}: {
  isOpen: boolean;
  onClose: () => void;
  order: Order | null;
}) {
  return (
    <PosDialog
      isOpen={isOpen}
      onClose={onClose}
      title={t.pricing.title}
      footer={
        <PosButton variant="primary" size="sm" onClick={onClose}>
          {t.common.close}
        </PosButton>
      }
    >
      {order && (
        <div className="space-y-2">
          <div className="flex justify-between border-b border-[var(--pos-border)] py-2 font-mono text-sm">
            <span className="text-[var(--pos-text-muted)]">{t.common.subtotal}</span>
            <span className="font-bold text-[var(--pos-text-primary)]">{order.subtotal} {t.common.ruble}</span>
          </div>
          <div className="flex justify-between border-b border-[var(--pos-border)] py-2 font-mono text-sm">
            <span className="text-[var(--pos-text-muted)]">{t.common.discount}</span>
            <span className="font-bold text-[var(--pos-text-primary)]">-{order.discount} {t.common.ruble}</span>
          </div>
          <div className="flex justify-between border-b border-[var(--pos-border)] py-2 font-mono text-sm">
            <span className="text-[var(--pos-text-muted)]">{t.common.tax}</span>
            <span className="font-bold text-[var(--pos-text-primary)]">{order.tax} {t.common.ruble}</span>
          </div>
          <div className="flex justify-between pt-3 font-mono text-base font-black uppercase">
            <span className="text-[var(--pos-text-primary)]">{t.common.total}</span>
            <span className="text-[var(--pos-status-success)]">{order.total} {t.common.ruble}</span>
          </div>
        </div>
      )}
    </PosDialog>
  );
}

function categoryLabel(id: string) {
  switch (id) {
    case 'dish':
      return t.menu.categories.dish;
    case 'good':
      return t.menu.categories.good;
    case 'semi_finished':
      return t.menu.categories.semiFinished;
    case 'service':
    case 'services':
      return t.menu.categories.service;
    default:
      return id;
  }
}

function PricingPolicyDialog({
  isOpen,
  onClose,
  policies,
  selectedPolicyId,
  selectedPolicy,
  onPolicyChange,
  lines,
  selectedLineId,
  onLineChange,
  reason,
  onReasonChange,
  onSubmit,
}: {
  isOpen: boolean;
  onClose: () => void;
  policies: PricingPolicy[];
  selectedPolicyId: string;
  selectedPolicy?: PricingPolicy;
  onPolicyChange: (value: string) => void;
  lines: OrderLine[];
  selectedLineId: string;
  onLineChange: (value: string) => void;
  reason: string;
  onReasonChange: (value: string) => void;
  onSubmit: () => void;
}) {
  return (
    <PosDialog
      isOpen={isOpen}
      onClose={onClose}
      title={t.pricing.title}
      footer={
        <>
          <PosButton variant="secondary" size="sm" onClick={onClose}>
            {t.common.cancel}
          </PosButton>
          <PosButton variant="primary" size="sm" onClick={onSubmit} disabled={!selectedPolicyId || (selectedPolicy?.scope === 'line' && !selectedLineId)}>
            {t.pricing.apply}
          </PosButton>
        </>
      }
    >
      {policies.length === 0 ? (
        <PosEmptyState title={t.pricing.noPolicies} description={t.pricing.noPoliciesDesc} />
      ) : (
        <div className="space-y-5">
          <div className="grid grid-cols-1 gap-2">
            {policies.map((policy) => {
              const active = policy.id === selectedPolicyId;
              return (
                <PosSelectableTile
                  key={policy.id}
                  type="button"
                  onClick={() => onPolicyChange(policy.id)}
                  active={active}
                  className={`p-3 border text-left rounded-none ${active ? 'bg-[var(--pos-action-primary)] text-[var(--pos-surface)] border-[var(--pos-action-primary)]' : 'bg-[var(--pos-surface)] border-[var(--pos-border)] text-[var(--pos-text-primary)]'}`}
                >
                  <span className="font-sans text-sm font-bold block">{policy.name}</span>
                  <span className="font-mono text-[10px] uppercase opacity-80">
                    {policy.kind === 'discount' ? t.pricing.discount : t.pricing.surcharge} · {policy.scope === 'line' ? t.pricing.lineScope : t.pricing.orderScope} · {policy.amountKind === 'percentage' ? `${policy.valueBasisPoints / 100}%` : `${policy.amount} ${t.common.ruble}`}
                  </span>
                </PosSelectableTile>
              );
            })}
          </div>

          {selectedPolicy?.scope === 'line' && (
            <label className="block space-y-2">
              <span className="font-mono text-xs font-bold uppercase text-[var(--pos-text-secondary)]">{t.pricing.lineScope}</span>
              <select
                className="w-full h-11 border border-[var(--pos-border)] bg-[var(--pos-surface)] px-3 font-sans text-sm text-[var(--pos-text-primary)]"
                value={selectedLineId}
                onChange={(event) => onLineChange(event.target.value)}
              >
                {lines.map((line) => (
                  <option key={line.id} value={line.id}>{line.name}</option>
                ))}
              </select>
            </label>
          )}

          <label className="block space-y-2">
            <span className="font-mono text-xs font-bold uppercase text-[var(--pos-text-secondary)]">{t.pricing.reason}</span>
            <input
              className="w-full h-11 border border-[var(--pos-border)] bg-[var(--pos-surface)] px-3 font-sans text-sm text-[var(--pos-text-primary)]"
              value={reason}
              onChange={(event) => onReasonChange(event.target.value)}
            />
          </label>
        </div>
      )}
    </PosDialog>
  );
}

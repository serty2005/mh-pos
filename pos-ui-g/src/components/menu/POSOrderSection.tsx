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
  PosDialog
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
  X,
  Lock,
  MessageSquare,
  BadgePercent
} from 'lucide-react';
import { MenuItem, OrderLine, PricingPolicy, SelectedModifier } from '../../types';

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
    applyPricingPolicy
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
  const [selectedPricingPolicyId, setSelectedPricingPolicyId] = useState<string>('');
  const [pricingReason, setPricingReason] = useState<string>('');
  const [pricingLineId, setPricingLineId] = useState<string>('');

  // Local notification banner for stop list alerts
  const [stopListAlertProduct, setStopListAlertProduct] = useState<string | null>(null);

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

  const selectedPricingPolicy = pricingPolicies.find((policy) => policy.id === selectedPricingPolicyId) ?? pricingPolicies[0];
  const selectedActionMenuItem = clickedLineForActions ? menuItems.find((menuItem) => menuItem.id === clickedLineForActions.itemId) : null;

  const openPricingDialog = () => {
    const firstPolicy = pricingPolicies[0];
    setSelectedPricingPolicyId(firstPolicy?.id ?? '');
    setPricingLineId(currentOrder?.lines[0]?.id ?? '');
    setPricingReason('');
    setPricingModalOpen(true);
  };

  const submitPricingPolicy = async () => {
    const policy = pricingPolicies.find((item) => item.id === selectedPricingPolicyId);
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
          <div className="flex gap-1.5 overflow-x-auto pos-scrollbar-thin pb-1 sm:pb-0">
            {categories.map((cat) => {
              const active = cat.id === activeCategory;
              return (
                <button
                  key={cat.id}
                  id={`cat-chip-${cat.id}`}
                  onClick={() => handleCategoryChange(cat.id)}
                  className={`h-11 px-6 font-mono text-xs uppercase font-extrabold cursor-pointer select-none transition-all rounded-none border
                    ${active 
                      ? 'bg-[var(--pos-action-primary)] text-[var(--pos-surface)] border-[var(--pos-action-primary)] font-black' 
                      : 'bg-[var(--pos-surface)] text-[var(--pos-text-secondary)] border-[var(--pos-border)] hover:bg-[var(--pos-bg)]'
                    }`}
                >
                  {cat.label}
                </button>
              );
            })}
          </div>

          {/* Search Bar Input */}
          <div className="relative w-full sm:w-[240px] shrink-0">
            <span className="absolute inset-y-0 left-3 flex items-center text-[var(--pos-text-muted)] pointer-events-none">
              <Search className="w-4 h-4" />
            </span>
            <input
              id="dish-search-input"
              type="text"
              placeholder={t.common.search}
              className="w-full h-11 pl-10 pr-4 border border-[var(--pos-border)] bg-[var(--pos-surface)] text-[var(--pos-text-primary)] font-sans text-xs focus:outline-none focus:border-[var(--pos-border-strong)] rounded-none"
              value={searchQuery}
              onChange={(e) => setSearchQuery(e.target.value)}
            />
            {searchQuery && (
              <button 
                type="button" 
                onClick={() => setSearchQuery('')}
                className="absolute inset-y-0 right-3 flex items-center text-[var(--pos-text-muted)] hover:text-[var(--pos-text-primary)] cursor-pointer outline-none border-none bg-none"
              >
                <X className="w-3.5 h-3.5" />
              </button>
            )}
          </div>
          <button
            type="button"
            onClick={() => setHideStopListed((value) => !value)}
            className={`h-11 px-4 border font-mono text-[10px] uppercase font-extrabold shrink-0 rounded-none ${
              hideStopListed
                ? 'bg-[var(--pos-action-primary)] text-[var(--pos-surface)] border-[var(--pos-action-primary)]'
                : 'bg-[var(--pos-surface)] text-[var(--pos-text-secondary)] border-[var(--pos-border)]'
            }`}
          >
            {t.menu.notInStopList}
          </button>
        </div>

        {/* Local warning panels if any alerts triggered */}
        {stopListAlertProduct && (
          <div className="px-6 pt-4 shrink-0">
            <PosBanner
              type="danger"
              message={`Позиция "${stopListAlertProduct}" находится в стоп-листе! Добавление заблокировано по остаткам кухни.`}
              action={
                <button 
                  type="button" 
                  onClick={() => setStopListAlertProduct(null)} 
                  className="w-8 h-8 border border-red-200 outline-none text-red-800 hover:bg-red-100 flex items-center justify-center cursor-pointer rounded-none"
                >
                  ×
                </button>
              }
            />
          </div>
        )}

        {/* Menu Tile Grid Room */}
        <div className="flex-1 p-6 overflow-y-auto pos-scrollarea-y pos-scrollbar-thin">
          {!currentOrder ? (
            <PosEmptyState
              title={t.blocks.noOrderSelected}
              description={t.blocks.noOrderSelectedDesc}
              buttonLabel={t.blocks.toTables}
              onAction={() => setCurrentSection('floor')}
              icon={<Utensils className="w-12 h-12" />}
            />
          ) : loading ? (
            <PosSkeleton type="grid" />
          ) : filteredItems.length === 0 ? (
            <PosEmptyState
              title="Ничего не найдено"
              description="Позиции каталога не найдены или закрыты из-за отсутствия связи."
              icon={<Search className="w-12 h-12" />}
            />
          ) : (
            <div className="grid grid-cols-2 md:grid-cols-3 xl:grid-cols-4 gap-4 p-1">
              {filteredItems.map((item) => {
                const isAvailable = item.isAvailable && !item.stopListBlocked;
                const hasModifiers = item.modifierGroups && item.modifierGroups.length > 0;
                
                return (
                  <button
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
                        {item.price} ₽
                      </span>
                      {hasModifiers && isAvailable && (
                        <span className="font-mono text-[9px] uppercase font-bold tracking-widest text-[var(--pos-text-muted)] border border-[var(--pos-border)] px-1 relative bg-[var(--pos-surface)] shrink-0">
                          +Мод
                        </span>
                      )}
                      {!isAvailable && (
                        <span className="font-mono text-[8px] uppercase font-mono px-1 border border-red-500 text-red-500 font-bold shrink-0 bg-white">
                          ×
                        </span>
                      )}
                      {item.stopListActive && !item.stopListBlocked && item.stopListAvailableQuantity !== undefined && (
                        <span className="font-mono text-[8px] uppercase px-1 border border-amber-500 text-amber-600 font-bold shrink-0 bg-white">
                          {t.menu.stopListQty}: {item.stopListAvailableQuantity}
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

      {/* 1/4 Right Column Workspace: Current Active Order Rail */}
      <div className="w-full lg:w-[320px] shrink-0">
        <PosActionRail className="h-full">
          
          {/* Header descriptor */}
          <div className="h-14 border-b border-[var(--pos-border)] px-4 flex items-center justify-between bg-[var(--pos-surface-raised)] select-none shrink-0">
            <span className="font-mono text-xs font-bold uppercase tracking-wider text-[var(--pos-text-secondary)]">
              {currentOrder ? `${currentOrder.tableName} (Заказ #${currentOrder.shortId})` : 'Текущий заказ'}
            </span>
            {currentOrder?.status === 'precheck_issued' && (
              <span className="font-mono bg-[var(--pos-status-warning)] text-zinc-950 font-bold text-[9px] uppercase px-1.5 py-0.5 animate-pulse">
                Locked
              </span>
            )}
          </div>

          {/* Lines iterator */}
          <div className="flex-1 pos-scrollarea-y pos-scrollbar-thin overflow-y-auto">
            {!currentOrder ? (
              <div className="p-8 text-center text-[var(--pos-text-muted)] h-full flex flex-col items-center justify-center">
                <SlidersHorizontal className="w-8 h-8 opacity-35 mb-2 shrink-0" />
                <span className="font-mono text-xs font-bold uppercase tracking-wider">Заказ пуст</span>
              </div>
            ) : currentOrder.lines.length === 0 ? (
              <div className="p-8 text-center text-[var(--pos-text-muted)] h-full flex flex-col items-center justify-center">
                <Utensils className="w-8 h-8 opacity-35 mb-2 shrink-0 animate-pulse" />
                <span className="font-sans text-xs">Добавьте блюда из меню слева</span>
              </div>
            ) : (
              <div className="divide-y divide-[var(--pos-border)]">
                {currentOrder.lines.map((line) => {
                  const isLocked = currentOrder.status === 'precheck_issued';
                  
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
                                  <span>{sel.price > 0 ? `(${sel.price} ₽)` : ''}</span>
                                </div>
                              ))}
                            </div>
                          )}

                          {/* Comment or Course indicators */}
                          {(line.comment || (line.course && line.course > 1)) && (
                            <div className="flex flex-wrap items-center gap-1.5 font-mono text-[9px] text-[var(--pos-status-info)] font-bold mt-1">
                              {line.course && line.course > 1 && (
                                <span className="px-1 border border-blue-200 bg-blue-50/50 uppercase">Курс {line.course}</span>
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
                        <span className="font-mono text-xs font-black tracking-tight shrink-0">{line.price} ₽</span>
                      </div>

                      {/* Line Counts controls / Quantities */}
                      <div className="flex items-center justify-between mt-1" onClick={(event) => event.stopPropagation()}>
                        <PosQuantityStepper
                          id={`line-qty-${line.id}`}
                          value={line.quantity}
                          onChange={(val) => changeLineQuantity(line.id, val)}
                          disabled={isLocked}
                        />

                        <span className="font-mono text-xs font-black text-[var(--pos-text-primary)]">
                          {line.totalPrice} ₽
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
                  Заблокировано выпущен пречек
                </span>
              </div>
            </div>
          )}

          {/* Totals summaries container */}
          {currentOrder && (
            <div className="border-t border-[var(--pos-border)] p-4 bg-[var(--pos-surface-raised)]/20 select-none space-y-2 shrink-0">
              <div className="flex justify-between items-baseline font-mono text-xs text-[var(--pos-text-muted)]">
                <span>{t.common.subtotal}:</span>
                <span>{currentOrder.subtotal} ₽</span>
              </div>
              <div className="flex justify-between items-baseline font-mono text-xs text-[var(--pos-text-muted)]">
                <span>НДС (10%):</span>
                <span>{currentOrder.tax} ₽</span>
              </div>
              {(currentOrder.discount > 0 || pricingPolicies.length > 0) && (
                <div className="flex justify-between items-baseline font-mono text-xs text-[var(--pos-text-muted)]">
                  <span>{t.common.discount}:</span>
                  <span>{currentOrder.discount} ₽</span>
                </div>
              )}
              <div className="flex justify-between items-baseline font-mono text-base font-black text-[var(--pos-text-primary)] border-t border-[var(--pos-border)] pt-2 uppercase tracking-wide">
                <span>{t.common.total}:</span>
                <span className="text-sm md:text-xl font-black text-[var(--pos-status-success)]">{currentOrder.total} ₽</span>
              </div>
            </div>
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
                    Действия
                  </PosButton>

                  <PosButton
                    id="order-pricing-btn"
                    variant="secondary"
                    size="md"
                    onClick={openPricingDialog}
                    disabled={currentOrder.lines.length === 0 || pricingPolicies.length === 0}
                    icon={<BadgePercent className="w-4 h-4" />}
                  >
                    {t.pricing.action}
                  </PosButton>

                  {/* Precheck issuance triggers */}
                  <PosButton
                    id="order-precheck-btn"
                    variant="primary"
                    size="md"
                    onClick={issuePrecheck}
                    disabled={currentOrder.lines.length === 0}
                    icon={<Printer className="w-4 h-4" />}
                  >
                    Пречек
                  </PosButton>
                </>
              ) : (
                <>
                  {/* Checkout Payment modal triggers */}
                  <PosButton
                    id="order-checkout-btn"
                    variant="primary"
                    size="md"
                    onClick={() => setPaymentModalOpen(true)}
                    // Limited waiter permissions safeguard check
                    disabled={currentOperator?.role === 'waiter'}
                    icon={<Banknote className="w-4 h-4" />}
                  >
                    Касса
                  </PosButton>

                  {/* Override locked cancel precheck trigger code */}
                  <PosButton
                    id="order-precheck-cancel-btn"
                    variant="danger"
                    size="md"
                    onClick={() => setPrecheckCancelOpen(true)}
                    icon={<SlidersHorizontal className="w-4 h-4" />}
                  >
                    Отмена
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
        policies={pricingPolicies}
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

function categoryLabel(id: string) {
  switch (id) {
    case 'dish':
      return 'Меню';
    case 'good':
      return 'Товары';
    case 'semi_finished':
      return 'Полуфабрикаты';
    case 'service':
    case 'services':
      return 'Сервис';
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
        <PosEmptyState title={t.pricing.noPolicies} />
      ) : (
        <div className="space-y-5">
          <div className="grid grid-cols-1 gap-2">
            {policies.map((policy) => {
              const active = policy.id === selectedPolicyId;
              return (
                <button
                  key={policy.id}
                  type="button"
                  onClick={() => onPolicyChange(policy.id)}
                  className={`p-3 border text-left rounded-none ${active ? 'bg-[var(--pos-action-primary)] text-[var(--pos-surface)] border-[var(--pos-action-primary)]' : 'bg-[var(--pos-surface)] border-[var(--pos-border)] text-[var(--pos-text-primary)]'}`}
                >
                  <span className="font-sans text-sm font-bold block">{policy.name}</span>
                  <span className="font-mono text-[10px] uppercase opacity-80">
                    {policy.kind === 'discount' ? t.pricing.discount : t.pricing.surcharge} · {policy.scope === 'line' ? t.pricing.lineScope : t.pricing.orderScope} · {policy.amountKind === 'percentage' ? `${policy.valueBasisPoints / 100}%` : `${policy.amount} ₽`}
                  </span>
                </button>
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

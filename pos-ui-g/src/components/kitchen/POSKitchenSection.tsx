import React, { useEffect, useMemo, useState } from 'react';

import { ApiError, createApiClient, type KitchenTicketAction } from '../../shared/api';
import { t } from '../../shared/i18n';
import { PosBanner, PosButton, PosSectionHeader, PosTabs } from '../../shared/ui';
import { usePOS } from '../../context/POSContext';
import type { BackendCatalogItem, BackendKitchenOrderQueueItem, BackendKitchenProposal, BackendKitchenRecipe, BackendKitchenTicket } from '../../shared/schemas';

type KitchenBottomSection = 'orders' | 'stock' | 'kitchen';
type OrdersTab = 'queue' | 'ready';
type StockTab = 'receipt' | 'count' | 'writeoff' | 'production';
type KitchenTab = 'recipes' | 'suggestions' | 'my_proposals';

const orderActions: Record<string, KitchenTicketAction[]> = {
  new: ['accept', 'cancel'],
  accepted: ['start', 'hold', 'cancel'],
  in_progress: ['hold', 'ready', 'cancel'],
  hold: ['start', 'cancel'],
  ready: ['serve', 'recall'],
  recall: ['start', 'cancel'],
};

export function POSKitchenSection({ section }: { section: KitchenBottomSection }) {
  const { authSnapshot } = usePOS();
  const api = useMemo(() => createApiClient(() => authSnapshot), [authSnapshot]);

  const [ordersTab, setOrdersTab] = useState<OrdersTab>('queue');
  const [stockTab, setStockTab] = useState<StockTab>('receipt');
  const [kitchenTab, setKitchenTab] = useState<KitchenTab>('recipes');

  const [orders, setOrders] = useState<BackendKitchenOrderQueueItem[]>([]);
  const [tickets, setTickets] = useState<BackendKitchenTicket[]>([]);
  const [catalog, setCatalog] = useState<BackendCatalogItem[]>([]);
  const [recipe, setRecipe] = useState<BackendKitchenRecipe | null>(null);
  const [proposals, setProposals] = useState<BackendKitchenProposal[]>([]);
  const [recipeItemId, setRecipeItemId] = useState('');
  const [busy, setBusy] = useState(false);
  const [safeError, setSafeError] = useState('');

  const [itemId, setItemId] = useState('');
  const [qty, setQty] = useState('1.000');
  const [cost, setCost] = useState('0');
  const [reason, setReason] = useState('');
  const [catalogSuggestionName, setCatalogSuggestionName] = useState('');
  const [recipeSuggestionReason, setRecipeSuggestionReason] = useState('');

  const clearError = () => setSafeError('');
  const safeErrorText = useMemo(() => {
    if (safeError === 'errors.validation') return t.errors.conflict;
    if (safeError === 'errors.permission') return t.errors.noPermission;
    if (safeError === 'errors.not_found') return t.common.empty;
    if (safeError === 'errors.conflict') return t.errors.conflict;
    return safeError ? t.errors.unknown : '';
  }, [safeError]);

  const withGuard = async (fn: () => Promise<void>) => {
    clearError();
    setBusy(true);
    try {
      await fn();
    } catch (error) {
      if (error instanceof ApiError) {
        setSafeError(error.messageKey || 'errors.unknown');
      } else {
        setSafeError('errors.unknown');
      }
    } finally {
      setBusy(false);
    }
  };

  const refreshOrders = () => withGuard(async () => {
    const [queue, nextTickets] = await Promise.all([
      api.listKitchenOrderQueue({ limit: 100 }),
      api.listKitchenTickets({ limit: 100 }),
    ]);
    setOrders(queue.orders);
    setTickets(nextTickets);
  });

  const refreshCatalog = () => withGuard(async () => {
    const items = await api.listCatalogItems();
    setCatalog(items.filter((item) => item.active !== false));
  });

  const refreshProposals = () => withGuard(async () => {
    const items = await api.listKitchenProposals({ limit: 100 });
    setProposals(items);
  });

  useEffect(() => {
    void refreshOrders();
    void refreshCatalog();
    void refreshProposals();
  }, []);

  const ordersForTab = useMemo(() => {
    if (ordersTab === 'ready') {
      return orders.filter((order) => ['ready', 'partially_ready', 'partially_served'].includes(order.kitchen_order_status));
    }
    return orders.filter((order) => !['ready', 'served', 'cancelled'].includes(order.kitchen_order_status));
  }, [orders, ordersTab]);

  const runTicketAction = (ticketId: string, action: KitchenTicketAction) => withGuard(async () => {
    await api.changeKitchenTicketStatus(ticketId, action);
    await refreshOrders();
  });

  const submitStock = () => withGuard(async () => {
    if (!itemId) {
      setSafeError('errors.validation');
      return;
    }
    const now = new Date();
    const day = now.toISOString().slice(0, 10);
    const lineTotal = Number.parseInt(cost, 10) * Number.parseFloat(qty);
    if (stockTab === 'receipt') {
      await api.submitKitchenStockReceipt({
        command_id: `cmd-pos-ui-g-${Date.now()}-receipt`,
        receipt_id: `receipt-${Date.now()}`,
        document_date: day,
        received_at: now.toISOString(),
        business_date_local: day,
        currency: 'RUB',
        items: [{
          line_id: `line-${Date.now()}`,
          catalog_item_id: itemId,
          quantity: qty,
          unit_code: 'KG',
          unit_cost_minor: Number.parseInt(cost, 10) || 0,
          line_total_minor: Number.isFinite(lineTotal) ? Math.round(lineTotal) : 0,
          currency: 'RUB',
        }],
      });
    }
    if (stockTab === 'count') {
      await api.submitKitchenInventoryCount({
        command_id: `cmd-pos-ui-g-${Date.now()}-count`,
        count_id: `count-${Date.now()}`,
        counted_at: now.toISOString(),
        business_date_local: day,
        items: [{ line_id: `line-${Date.now()}`, catalog_item_id: itemId, counted_quantity: qty, unit_code: 'KG' }],
      });
    }
    if (stockTab === 'writeoff') {
      await api.submitKitchenWriteOff({
        command_id: `cmd-pos-ui-g-${Date.now()}-writeoff`,
        write_off_id: `writeoff-${Date.now()}`,
        written_off_at: now.toISOString(),
        business_date_local: day,
        reason_code: 'manual',
        reason,
        items: [{ line_id: `line-${Date.now()}`, catalog_item_id: itemId, quantity: qty, unit_code: 'KG' }],
      });
    }
    if (stockTab === 'production') {
      await api.submitKitchenProduction({
        command_id: `cmd-pos-ui-g-${Date.now()}-production`,
        production_id: `production-${Date.now()}`,
        semi_finished_catalog_item_id: itemId,
        quantity: qty,
        unit_code: 'KG',
        completed_at: now.toISOString(),
        business_date_local: day,
      });
    }
  });

  const loadRecipe = () => withGuard(async () => {
    if (!recipeItemId) return;
    const data = await api.getKitchenRecipe(recipeItemId);
    setRecipe(data);
  });

  const submitCatalogSuggestion = () => withGuard(async () => {
    await api.submitCatalogSuggestion({
      command_id: `cmd-pos-ui-g-${Date.now()}-catalog-suggest`,
      action: 'create_item',
      kind: 'good',
      name: catalogSuggestionName,
      reason: 'kitchen_suggestion',
    });
    setCatalogSuggestionName('');
    await refreshProposals();
  });

  const submitRecipeSuggestion = () => withGuard(async () => {
    if (!recipeItemId) {
      setSafeError('errors.validation');
      return;
    }
    await api.submitRecipeSuggestion({
      command_id: `cmd-pos-ui-g-${Date.now()}-recipe-suggest`,
      recipe_version_id: recipe?.recipe_version_id,
      owner_catalog_item_id: recipeItemId,
      action: 'update_recipe',
      prep_time_delta_minutes: 0,
      changes: [],
      reason: recipeSuggestionReason || 'kitchen_suggestion',
    });
    setRecipeSuggestionReason('');
    await refreshProposals();
  });

  return (
    <div className="flex-1 min-h-0 flex flex-col bg-[var(--pos-bg)]">
      <PosSectionHeader title={t.modes.kds} actions={<PosButton size="sm" onClick={() => void refreshOrders()}>{t.kitchen.refresh}</PosButton>} />
      {safeErrorText && <div className="p-4"><PosBanner type="danger" message={safeErrorText} /></div>}

      {section === 'orders' && (
        <div className="flex-1 min-h-0 p-4 space-y-3 overflow-auto">
          <PosTabs id="orders-tabs" activeId={ordersTab} onChange={(id) => setOrdersTab(id as OrdersTab)} items={[{ id: 'queue', label: t.kitchen.tabQueue }, { id: 'ready', label: t.kitchen.tabReady }]} />
          <div className="grid gap-3 md:grid-cols-2 xl:grid-cols-3">
            {ordersForTab.map((order) => (
              <article key={order.order_id} className="border border-[var(--pos-border)] bg-[var(--pos-surface)] p-3 space-y-2">
                <div className="flex items-center justify-between">
                  <strong className="font-mono text-xs">{order.table_name || order.edge_order_id || order.order_id}</strong>
                  <span className="text-xs">{order.kitchen_order_status}</span>
                </div>
                <div className="text-xs text-[var(--pos-text-muted)]">{Math.max(0, Math.floor((order.elapsed_seconds || 0) / 60))} {t.kitchen.minutes}</div>
                {(order.tickets || []).map((ticket) => (
                  <div key={ticket.id} className="border border-[var(--pos-border)] p-2 space-y-1">
                    <div className="text-sm font-semibold">{ticket.name}</div>
                    <div className="text-xs">{ticket.quantity} {ticket.unit_code} · {ticket.status}</div>
                    <div className="flex flex-wrap gap-1">
                      {(orderActions[ticket.status] || []).map((action) => (
                        <PosButton key={`${ticket.id}-${action}`} size="sm" disabled={busy} onClick={() => void runTicketAction(ticket.id, action)}>{action}</PosButton>
                      ))}
                    </div>
                  </div>
                ))}
              </article>
            ))}
          </div>
        </div>
      )}

      {section === 'stock' && (
        <div className="flex-1 min-h-0 p-4 space-y-3 overflow-auto">
          <PosTabs id="stock-tabs" activeId={stockTab} onChange={(id) => setStockTab(id as StockTab)} items={[
            { id: 'receipt', label: t.kitchen.tabReceipt },
            { id: 'count', label: t.kitchen.tabCount },
            { id: 'writeoff', label: t.kitchen.tabWriteOff },
            { id: 'production', label: t.kitchen.tabProduction },
          ]} />
          <div className="grid gap-3 max-w-2xl">
            <select className="h-12 border px-3 bg-[var(--pos-surface)]" value={itemId} onChange={(e) => setItemId(e.target.value)}>
              <option value="">{t.kitchen.selectCatalogItem}</option>
              {catalog.map((item) => <option key={item.id} value={item.id}>{item.name} ({item.kind || item.item_type || 'item'})</option>)}
            </select>
            <input className="h-12 border px-3 bg-[var(--pos-surface)]" value={qty} onChange={(e) => setQty(e.target.value)} placeholder={t.kitchen.quantityPlaceholder} />
            {stockTab === 'receipt' && <input className="h-12 border px-3 bg-[var(--pos-surface)]" value={cost} onChange={(e) => setCost(e.target.value)} placeholder={t.kitchen.unitCostPlaceholder} />}
            {stockTab === 'writeoff' && <input className="h-12 border px-3 bg-[var(--pos-surface)]" value={reason} onChange={(e) => setReason(e.target.value)} placeholder={t.kitchen.writeOffReasonPlaceholder} />}
            <PosButton onClick={() => void submitStock()} disabled={busy}>{t.common.execute}</PosButton>
          </div>
        </div>
      )}

      {section === 'kitchen' && (
        <div className="flex-1 min-h-0 p-4 space-y-3 overflow-auto">
          <PosTabs id="kitchen-tabs" activeId={kitchenTab} onChange={(id) => setKitchenTab(id as KitchenTab)} items={[
            { id: 'recipes', label: t.kitchen.tabRecipes },
            { id: 'suggestions', label: t.kitchen.tabSuggestions },
            { id: 'my_proposals', label: t.kitchen.tabMyProposals },
          ]} />

          {kitchenTab === 'recipes' && (
            <div className="space-y-3 max-w-3xl">
              <select className="h-12 border px-3 bg-[var(--pos-surface)]" value={recipeItemId} onChange={(e) => setRecipeItemId(e.target.value)}>
                <option value="">{t.kitchen.selectDishOrSemi}</option>
                {catalog.filter((item) => ['dish', 'semi_finished'].includes((item.kind || item.item_type || '').toLowerCase())).map((item) => <option key={item.id} value={item.id}>{item.name}</option>)}
              </select>
              <PosButton onClick={() => void loadRecipe()} disabled={busy}>{t.kitchen.loadRecipe}</PosButton>
              {recipe && (
                <div className="border border-[var(--pos-border)] bg-[var(--pos-surface)] p-3 space-y-2">
                  {(recipe.lines || []).map((line, idx) => (
                    <div key={`${line.line_id || idx}`} className="text-sm">{line.ingredient_name || line.catalog_item_id} · {line.quantity} {line.unit_code}</div>
                  ))}
                </div>
              )}
            </div>
          )}

          {kitchenTab === 'suggestions' && (
            <div className="grid gap-4 md:grid-cols-2">
              <div className="border border-[var(--pos-border)] p-3 space-y-2">
                <h4 className="font-semibold">{t.kitchen.suggestCatalogItem}</h4>
                <input className="h-12 border px-3 w-full bg-[var(--pos-surface)]" value={catalogSuggestionName} onChange={(e) => setCatalogSuggestionName(e.target.value)} placeholder={t.kitchen.namePlaceholder} />
                <PosButton onClick={() => void submitCatalogSuggestion()} disabled={busy}>{t.kitchen.submit}</PosButton>
              </div>
              <div className="border border-[var(--pos-border)] p-3 space-y-2">
                <h4 className="font-semibold">{t.kitchen.suggestRecipeChange}</h4>
                <textarea className="border px-3 py-2 w-full bg-[var(--pos-surface)]" rows={3} value={recipeSuggestionReason} onChange={(e) => setRecipeSuggestionReason(e.target.value)} placeholder={t.kitchen.reasonPlaceholder} />
                <PosButton onClick={() => void submitRecipeSuggestion()} disabled={busy}>{t.kitchen.submit}</PosButton>
              </div>
            </div>
          )}

          {kitchenTab === 'my_proposals' && (
            <div className="space-y-2">
              {proposals.map((proposal) => (
                <div key={proposal.id} className="border border-[var(--pos-border)] bg-[var(--pos-surface)] p-3">
                  <div className="font-mono text-xs">{proposal.kind}</div>
                  <div className="text-sm">{proposal.status}</div>
                  <div className="text-xs text-[var(--pos-text-muted)]">{proposal.created_at}</div>
                </div>
              ))}
            </div>
          )}
        </div>
      )}
    </div>
  );
}

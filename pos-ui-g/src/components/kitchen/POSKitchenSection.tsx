import React, { useEffect, useMemo, useState } from 'react';
import {
  AlertTriangle,
  CheckCircle2,
  ClipboardList,
  Clock3,
  PackageCheck,
  RefreshCcw,
  Search,
  Send,
  Utensils,
} from 'lucide-react';

import { usePOS } from '../../context/POSContext';
import { ApiError, createApiClient, type KitchenTicketAction } from '../../shared/api';
import { t } from '../../shared/i18n';
import { PosBanner, PosButton, PosEmptyState, PosFormRow, PosSectionHeader, PosSkeleton, PosTabs } from '../../shared/ui';
import type {
  BackendCatalogItem,
  BackendKitchenOrderQueueItem,
  BackendKitchenProposal,
  BackendKitchenRecipe,
  BackendKitchenTicket,
} from '../../shared/schemas';

type KitchenBottomSection = 'orders' | 'stock' | 'kitchen';
type OrdersTab = 'queue' | 'ready';
type StockTab = 'receipt' | 'count' | 'writeoff' | 'production';
type KitchenTab = 'recipes' | 'suggestions' | 'my_proposals';
type CatalogKindFilter = 'all' | 'dish' | 'good' | 'semi_finished' | 'service';
type RecipeSuggestionAction =
  | 'change_prep_time'
  | 'add_ingredient'
  | 'remove_ingredient'
  | 'replace_ingredient'
  | 'change_quantity'
  | 'change_loss_percent';

type StockFormState = {
  itemId: string;
  quantity: string;
  unitCode: string;
  supplierName: string;
  documentNumber: string;
  documentDate: string;
  businessDate: string;
  unitCostMinor: string;
  lineTotalMinor: string;
  reasonCode: string;
  reason: string;
};

type CatalogSuggestionState = {
  kind: 'dish' | 'good' | 'semi_finished' | 'service';
  name: string;
  sku: string;
  baseUnit: string;
  kitchenType: string;
  accountingCategory: string;
  reason: string;
};

type RecipeSuggestionState = {
  action: RecipeSuggestionAction;
  lineId: string;
  ingredientItemId: string;
  quantity: string;
  unitCode: string;
  lossPercent: string;
  prepTimeDeltaMinutes: string;
  reason: string;
};

const orderActions: Record<string, KitchenTicketAction[]> = {
  new: ['accept', 'cancel'],
  accepted: ['start', 'hold', 'cancel'],
  in_progress: ['hold', 'ready', 'cancel'],
  hold: ['start', 'cancel'],
  ready: ['serve', 'recall'],
  served: ['recall'],
  recall: ['start', 'cancel'],
};

const activeOrderStatuses = ['queued', 'accepted', 'in_progress', 'partially_ready', 'mixed'];
const readyOrderStatuses = ['ready', 'partially_ready', 'partially_served'];
const catalogKinds: CatalogKindFilter[] = ['all', 'dish', 'good', 'semi_finished', 'service'];
const recipeSuggestionActions: RecipeSuggestionAction[] = [
  'change_prep_time',
  'add_ingredient',
  'remove_ingredient',
  'replace_ingredient',
  'change_quantity',
  'change_loss_percent',
];

function todayLocalDate() {
  return new Date().toISOString().slice(0, 10);
}

function createInitialStockForm(): StockFormState {
  const today = todayLocalDate();
  return {
    itemId: '',
    quantity: '1.000',
    unitCode: 'KG',
    supplierName: '',
    documentNumber: '',
    documentDate: today,
    businessDate: today,
    unitCostMinor: '0',
    lineTotalMinor: '0',
    reasonCode: 'manual',
    reason: '',
  };
}

function catalogKind(item: BackendCatalogItem) {
  return (item.type || item.kind || item.item_type || '').toLowerCase();
}

function catalogUnit(item: BackendCatalogItem | undefined, fallback = 'KG') {
  return item?.base_unit || fallback;
}

function catalogKindLabel(kind: string) {
  switch (kind) {
    case 'dish':
      return t.kitchen.itemKindDish;
    case 'good':
      return t.kitchen.itemKindGood;
    case 'semi_finished':
      return t.kitchen.itemKindSemiFinished;
    case 'service':
      return t.kitchen.itemKindService;
    default:
      return kind || t.common.none;
  }
}

function localizedError(error: unknown) {
  if (error instanceof Error && error.message === 'validation') return t.errors.validation;
  if (!(error instanceof ApiError)) return t.errors.unknown;
  switch (error.messageKey) {
    case 'errors.validation':
      return t.errors.validation;
    case 'errors.permission':
      return t.errors.noPermission;
    case 'errors.not_found':
      return t.errors.notFound;
    case 'errors.conflict':
      return t.errors.conflict;
    case 'errors.rateLimit':
      return t.errors.rateLimit;
    case 'errors.server':
      return t.errors.server;
    case 'errors.session.required':
      return t.errors.sessionRequired;
    case 'errors.network.unavailable':
      return t.errors.networkUnavailable;
    case 'errors.network.timeout':
      return t.errors.networkTimeout;
    case 'errors.response.invalid':
      return t.errors.invalidResponse;
    default:
      return t.errors.unknown;
  }
}

function formatMinutes(seconds: number) {
  return `${Math.max(0, Math.floor(seconds / 60))} ${t.kitchen.minutes}`;
}

function formatDateTime(value = '') {
  if (!value) return t.common.none;
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return value;
  return date.toLocaleString('ru-RU', { day: '2-digit', month: '2-digit', hour: '2-digit', minute: '2-digit' });
}

function proposalStatusLabel(status: string) {
  return t.kitchen.proposalStatus[status as keyof typeof t.kitchen.proposalStatus] ?? status;
}

function proposalKindLabel(kind: string) {
  return t.kitchen.proposalKind[kind as keyof typeof t.kitchen.proposalKind] ?? kind;
}

function ticketStatusLabel(status: string) {
  return t.kitchen.ticketStatus[status as keyof typeof t.kitchen.ticketStatus] ?? status;
}

function orderStatusLabel(status: string) {
  return t.kitchen.orderStatus[status as keyof typeof t.kitchen.orderStatus] ?? status;
}

function actionLabel(action: KitchenTicketAction) {
  return t.kitchen.actions[action];
}

function recipeActionLabel(action: RecipeSuggestionAction) {
  return t.kitchen.recipeActions[action];
}

function safeNumber(value: string) {
  const parsed = Number.parseInt(value, 10);
  return Number.isFinite(parsed) ? parsed : 0;
}

function isPositiveDecimal(value: string) {
  return Number.parseFloat(value) > 0;
}

function getRecipeIngredients(recipe: BackendKitchenRecipe | null) {
  if (!recipe) return [];
  return recipe.ingredients.length > 0 ? recipe.ingredients : recipe.lines;
}

function selectedRecipeVersionId(recipe: BackendKitchenRecipe | null) {
  return recipe?.recipe_version_id || recipe?.recipe_version?.id || '';
}

function createRecipeChange(state: RecipeSuggestionState) {
  if (state.action === 'change_prep_time') return [];
  const change: Record<string, string> = { action: state.action };
  if (state.lineId) change.line_id = state.lineId;
  if (state.ingredientItemId) {
    if (state.action === 'replace_ingredient') change.to_catalog_item_id = state.ingredientItemId;
    if (state.action === 'add_ingredient') change.to_catalog_item_id = state.ingredientItemId;
  }
  if (state.quantity) change.quantity = state.quantity;
  if (state.unitCode) change.unit_code = state.unitCode;
  if (state.lossPercent) change.loss_percent = state.lossPercent;
  return [change];
}

export function POSKitchenSection({ section }: { section: KitchenBottomSection }) {
  const { authSnapshot } = usePOS();
  const api = useMemo(() => createApiClient(() => authSnapshot), [authSnapshot]);

  const [ordersTab, setOrdersTab] = useState<OrdersTab>('queue');
  const [stockTab, setStockTab] = useState<StockTab>('receipt');
  const [kitchenTab, setKitchenTab] = useState<KitchenTab>('recipes');
  const [orders, setOrders] = useState<BackendKitchenOrderQueueItem[]>([]);
  const [catalog, setCatalog] = useState<BackendCatalogItem[]>([]);
  const [recipe, setRecipe] = useState<BackendKitchenRecipe | null>(null);
  const [proposals, setProposals] = useState<BackendKitchenProposal[]>([]);
  const [recipeItemId, setRecipeItemId] = useState('');
  const [loading, setLoading] = useState(true);
  const [busy, setBusy] = useState(false);
  const [safeError, setSafeError] = useState('');
  const [successMessage, setSuccessMessage] = useState('');
  const [catalogSearch, setCatalogSearch] = useState('');
  const [catalogFilter, setCatalogFilter] = useState<CatalogKindFilter>('all');
  const [stockForm, setStockForm] = useState<StockFormState>(() => createInitialStockForm());
  const [catalogSuggestion, setCatalogSuggestion] = useState<CatalogSuggestionState>({
    kind: 'good',
    name: '',
    sku: '',
    baseUnit: 'KG',
    kitchenType: '',
    accountingCategory: '',
    reason: '',
  });
  const [recipeSuggestion, setRecipeSuggestion] = useState<RecipeSuggestionState>({
    action: 'change_prep_time',
    lineId: '',
    ingredientItemId: '',
    quantity: '1.000',
    unitCode: 'KG',
    lossPercent: '0',
    prepTimeDeltaMinutes: '0',
    reason: '',
  });

  const selectedStockItem = useMemo(
    () => catalog.find((item) => item.id === stockForm.itemId),
    [catalog, stockForm.itemId],
  );

  const activeCatalog = useMemo(() => catalog.filter((item) => item.active !== false), [catalog]);
  const stockCatalog = useMemo(
    () => activeCatalog.filter((item) => catalogKind(item) !== 'service'),
    [activeCatalog],
  );
  const dishCatalog = useMemo(
    () => activeCatalog.filter((item) => ['dish', 'semi_finished'].includes(catalogKind(item))),
    [activeCatalog],
  );
  const ingredientCatalog = useMemo(
    () => activeCatalog.filter((item) => ['good', 'semi_finished'].includes(catalogKind(item))),
    [activeCatalog],
  );

  const filteredCatalog = useMemo(() => {
    const query = catalogSearch.trim().toLowerCase();
    return activeCatalog.filter((item) => {
      const kindMatches = catalogFilter === 'all' || catalogKind(item) === catalogFilter;
      const text = `${item.name} ${item.sku ?? ''} ${item.base_unit ?? ''}`.toLowerCase();
      return kindMatches && (!query || text.includes(query));
    });
  }, [activeCatalog, catalogFilter, catalogSearch]);

  const ordersForTab = useMemo(() => {
    if (ordersTab === 'ready') {
      return orders.filter((order) => readyOrderStatuses.includes(order.kitchen_order_status));
    }
    return orders.filter((order) => activeOrderStatuses.includes(order.kitchen_order_status));
  }, [orders, ordersTab]);

  const loadOrders = async () => {
    const queue = await api.listKitchenOrderQueue({ limit: 100 });
    setOrders(queue.orders);
  };

  const loadCatalog = async () => {
    const items = await api.listCatalogItems();
    setCatalog(items);
  };

  const loadProposals = async () => {
    const items = await api.listKitchenProposals({ limit: 100 });
    setProposals(items);
  };

  const runSafe = async (fn: () => Promise<void>, showSuccess = false) => {
    setSafeError('');
    setSuccessMessage('');
    setBusy(true);
    try {
      await fn();
      if (showSuccess) setSuccessMessage(t.kitchen.submitted);
    } catch (error) {
      setSafeError(localizedError(error));
    } finally {
      setBusy(false);
    }
  };

  const refreshAll = async () => {
    setSafeError('');
    setLoading(true);
    try {
      await Promise.all([loadOrders(), loadCatalog(), loadProposals()]);
    } catch (error) {
      setSafeError(localizedError(error));
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    void refreshAll();
  }, []);

  const updateStockForm = (patch: Partial<StockFormState>) => {
    setStockForm((current) => ({ ...current, ...patch }));
  };

  const updateCatalogSuggestion = (patch: Partial<CatalogSuggestionState>) => {
    setCatalogSuggestion((current) => ({ ...current, ...patch }));
  };

  const updateRecipeSuggestion = (patch: Partial<RecipeSuggestionState>) => {
    setRecipeSuggestion((current) => ({ ...current, ...patch }));
  };

  const selectStockItem = (itemId: string) => {
    const item = catalog.find((candidate) => candidate.id === itemId);
    updateStockForm({ itemId, unitCode: catalogUnit(item, stockForm.unitCode) });
  };

  const runTicketAction = (ticketId: string, action: KitchenTicketAction) => runSafe(async () => {
    await api.changeKitchenTicketStatus(ticketId, action);
    await loadOrders();
  });

  const submitStock = () => runSafe(async () => {
    if (!stockForm.itemId || !isPositiveDecimal(stockForm.quantity)) throw new Error('validation');
    const now = new Date().toISOString();
    const commonLine = {
      catalog_item_id: stockForm.itemId,
      quantity: stockForm.quantity,
      unit_code: stockForm.unitCode || catalogUnit(selectedStockItem),
    };

    if (stockTab === 'receipt') {
      if (
        !stockForm.supplierName.trim()
        || !stockForm.documentDate
        || !stockForm.businessDate
        || safeNumber(stockForm.lineTotalMinor) <= 0
      ) {
        throw new Error('validation');
      }
      await api.submitKitchenStockReceipt({
        command_id: `cmd-pos-ui-g-${Date.now()}-receipt`,
        supplier_name_snapshot: stockForm.supplierName,
        document_number: stockForm.documentNumber,
        document_date: stockForm.documentDate,
        received_at: now,
        business_date_local: stockForm.businessDate,
        currency: 'RUB',
        items: [{
          ...commonLine,
          name_snapshot: selectedStockItem?.name ?? '',
          unit_cost_minor: safeNumber(stockForm.unitCostMinor),
          line_total_minor: safeNumber(stockForm.lineTotalMinor),
          currency: 'RUB',
        }],
      });
    }

    if (stockTab === 'count') {
      await api.submitKitchenInventoryCount({
        command_id: `cmd-pos-ui-g-${Date.now()}-count`,
        counted_at: now,
        business_date_local: stockForm.businessDate,
        items: [{
          catalog_item_id: stockForm.itemId,
          counted_quantity: stockForm.quantity,
          unit_code: stockForm.unitCode || catalogUnit(selectedStockItem),
        }],
      });
    }

    if (stockTab === 'writeoff') {
      if (!stockForm.reason.trim() && !stockForm.reasonCode.trim()) throw new Error('validation');
      await api.submitKitchenWriteOff({
        command_id: `cmd-pos-ui-g-${Date.now()}-writeoff`,
        written_off_at: now,
        business_date_local: stockForm.businessDate,
        reason_code: stockForm.reasonCode,
        reason: stockForm.reason,
        items: [commonLine],
      });
    }

    if (stockTab === 'production') {
      await api.submitKitchenProduction({
        command_id: `cmd-pos-ui-g-${Date.now()}-production`,
        semi_finished_catalog_item_id: stockForm.itemId,
        quantity: stockForm.quantity,
        unit_code: stockForm.unitCode || catalogUnit(selectedStockItem),
        completed_at: now,
        business_date_local: stockForm.businessDate,
      });
    }

    await Promise.all([loadCatalog(), loadProposals()]);
  }, true);

  const loadRecipe = () => runSafe(async () => {
    if (!recipeItemId) throw new Error('validation');
    const data = await api.getKitchenRecipe(recipeItemId);
    setRecipe(data);
    setRecipeSuggestion((current) => ({
      ...current,
      lineId: '',
      ingredientItemId: '',
      unitCode: catalogUnit(activeCatalog.find((item) => item.id === recipeItemId), current.unitCode),
    }));
  });

  const submitCatalogSuggestion = () => runSafe(async () => {
    if (!catalogSuggestion.name.trim() || !catalogSuggestion.baseUnit.trim() || !catalogSuggestion.reason.trim()) {
      throw new Error('validation');
    }
    await api.submitCatalogSuggestion({
      command_id: `cmd-pos-ui-g-${Date.now()}-catalog-suggest`,
      action: 'create',
      kind: catalogSuggestion.kind,
      name: catalogSuggestion.name,
      sku: catalogSuggestion.sku,
      base_unit: catalogSuggestion.baseUnit,
      kitchen_type: catalogSuggestion.kitchenType,
      accounting_category: catalogSuggestion.accountingCategory,
      reason: catalogSuggestion.reason,
    });
    setCatalogSuggestion({
      kind: 'good',
      name: '',
      sku: '',
      baseUnit: 'KG',
      kitchenType: '',
      accountingCategory: '',
      reason: '',
    });
    await loadProposals();
  }, true);

  const submitRecipeSuggestion = () => runSafe(async () => {
    if (!recipeItemId || !recipeSuggestion.reason.trim()) throw new Error('validation');
    const changes = createRecipeChange(recipeSuggestion);
    if (recipeSuggestion.action !== 'change_prep_time' && changes.length === 0) throw new Error('validation');
    await api.submitRecipeSuggestion({
      command_id: `cmd-pos-ui-g-${Date.now()}-recipe-suggest`,
      recipe_version_id: selectedRecipeVersionId(recipe),
      owner_catalog_item_id: recipeItemId,
      action: recipeSuggestion.action,
      prep_time_delta_minutes: safeNumber(recipeSuggestion.prepTimeDeltaMinutes),
      changes,
      reason: recipeSuggestion.reason,
    });
    setRecipeSuggestion({
      action: 'change_prep_time',
      lineId: '',
      ingredientItemId: '',
      quantity: '1.000',
      unitCode: 'KG',
      lossPercent: '0',
      prepTimeDeltaMinutes: '0',
      reason: '',
    });
    const [nextRecipe] = await Promise.all([
      api.getKitchenRecipe(recipeItemId),
      loadProposals(),
    ]);
    setRecipe(nextRecipe);
  }, true);

  const sectionTitle = section === 'orders' ? t.kitchen.navOrders : section === 'stock' ? t.kitchen.navStock : t.kitchen.navKitchen;

  return (
    <div className="flex-1 min-h-0 flex flex-col bg-[var(--pos-bg)]">
      <PosSectionHeader
        title={t.kitchen.title}
        badge={sectionTitle}
        actions={(
          <PosButton
            size="sm"
            onClick={() => void refreshAll()}
            disabled={busy || loading}
            icon={<RefreshCcw className="w-4 h-4" />}
          >
            {t.kitchen.refresh}
          </PosButton>
        )}
      />
      {safeError && <div className="px-4 pt-4"><PosBanner type="danger" message={safeError} /></div>}
      {successMessage && <div className="px-4 pt-4"><PosBanner type="success" message={successMessage} /></div>}

      {section === 'orders' && (
        <div className="flex-1 min-h-0 overflow-hidden flex flex-col">
          <PosTabs
            id="orders-tabs"
            activeId={ordersTab}
            onChange={(id) => setOrdersTab(id as OrdersTab)}
            items={[
              { id: 'queue', label: t.kitchen.tabQueue, count: orders.filter((order) => activeOrderStatuses.includes(order.kitchen_order_status)).length },
              { id: 'ready', label: t.kitchen.tabReady, count: orders.filter((order) => readyOrderStatuses.includes(order.kitchen_order_status)).length },
            ]}
          />
          <div className="flex-1 min-h-0 p-4 overflow-auto pos-scrollbar-thin">
            {loading ? (
              <PosSkeleton type="grid" />
            ) : ordersForTab.length === 0 ? (
              <PosEmptyState
                title={ordersTab === 'ready' ? t.kitchen.emptyReady : t.kitchen.emptyQueue}
                description={t.kitchen.refresh}
                icon={ordersTab === 'ready' ? <PackageCheck className="w-10 h-10" /> : <ClipboardList className="w-10 h-10" />}
              />
            ) : (
              <div className="grid gap-4 md:grid-cols-2 xl:grid-cols-3">
                {ordersForTab.map((order) => (
                  <React.Fragment key={order.order_id}>
                    <OrderTile
                      order={order}
                      busy={busy}
                      onAction={(ticketId, action) => void runTicketAction(ticketId, action)}
                    />
                  </React.Fragment>
                ))}
              </div>
            )}
          </div>
        </div>
      )}

      {section === 'stock' && (
        <div className="flex-1 min-h-0 overflow-hidden flex flex-col">
          <PosTabs
            id="stock-tabs"
            activeId={stockTab}
            onChange={(id) => setStockTab(id as StockTab)}
            items={[
              { id: 'receipt', label: t.kitchen.tabReceipt },
              { id: 'count', label: t.kitchen.tabCount },
              { id: 'writeoff', label: t.kitchen.tabWriteOff },
              { id: 'production', label: t.kitchen.tabProduction },
            ]}
          />
          <div className="flex-1 min-h-0 grid gap-4 p-4 overflow-auto pos-scrollbar-thin xl:grid-cols-[minmax(0,1fr)_420px]">
            <CatalogPicker
              filteredItems={filteredCatalog.filter((item) => stockTab === 'production' ? catalogKind(item) === 'semi_finished' : catalogKind(item) !== 'service')}
              selectedId={stockForm.itemId}
              search={catalogSearch}
              filter={catalogFilter}
              onSearch={setCatalogSearch}
              onFilter={setCatalogFilter}
              onSelect={selectStockItem}
            />
            <StockForm
              tab={stockTab}
              form={stockForm}
              selectedItem={selectedStockItem}
              busy={busy}
              onChange={updateStockForm}
              onSubmit={() => void submitStock()}
            />
          </div>
        </div>
      )}

      {section === 'kitchen' && (
        <div className="flex-1 min-h-0 overflow-hidden flex flex-col">
          <PosTabs
            id="kitchen-tabs"
            activeId={kitchenTab}
            onChange={(id) => setKitchenTab(id as KitchenTab)}
            items={[
              { id: 'recipes', label: t.kitchen.tabRecipes },
              { id: 'suggestions', label: t.kitchen.tabSuggestions },
              { id: 'my_proposals', label: t.kitchen.tabMyProposals, count: proposals.length },
            ]}
          />
          <div className="flex-1 min-h-0 p-4 overflow-auto pos-scrollbar-thin">
            {kitchenTab === 'recipes' && (
              <RecipeWorkspace
                catalog={dishCatalog}
                ingredientCatalog={ingredientCatalog}
                recipe={recipe}
                recipeItemId={recipeItemId}
                suggestion={recipeSuggestion}
                busy={busy}
                onRecipeItemChange={setRecipeItemId}
                onLoadRecipe={() => void loadRecipe()}
                onSuggestionChange={updateRecipeSuggestion}
                onSubmitSuggestion={() => void submitRecipeSuggestion()}
              />
            )}

            {kitchenTab === 'suggestions' && (
              <div className="max-w-4xl">
                <CatalogSuggestionForm
                  form={catalogSuggestion}
                  busy={busy}
                  onChange={updateCatalogSuggestion}
                  onSubmit={() => void submitCatalogSuggestion()}
                />
              </div>
            )}

            {kitchenTab === 'my_proposals' && (
              <ProposalList proposals={proposals} />
            )}
          </div>
        </div>
      )}
    </div>
  );
}

function OrderTile({
  order,
  busy,
  onAction,
}: {
  order: BackendKitchenOrderQueueItem;
  busy: boolean;
  onAction: (ticketId: string, action: KitchenTicketAction) => void;
}) {
  return (
    <article className="border border-[var(--pos-border)] bg-[var(--pos-surface)] min-h-[260px] flex flex-col">
      <div className="p-4 border-b border-[var(--pos-border)] flex items-start justify-between gap-3">
        <div className="min-w-0">
          <div className="font-mono text-[10px] uppercase tracking-widest text-[var(--pos-text-muted)]">
            {t.common.order}
          </div>
          <h3 className="font-mono text-base font-black text-[var(--pos-text-primary)] truncate">
            {order.table_name || order.edge_order_id || order.order_id || t.kitchen.tableFallback}
          </h3>
        </div>
        <span className="shrink-0 border border-[var(--pos-border-strong)] px-2 py-1 font-mono text-[10px] font-bold uppercase text-[var(--pos-text-secondary)]">
          {orderStatusLabel(order.kitchen_order_status)}
        </span>
      </div>

      <div className="grid grid-cols-3 border-b border-[var(--pos-border)] divide-x divide-[var(--pos-border)]">
        <TileMetric icon={<Clock3 className="w-3.5 h-3.5" />} label={t.kitchen.elapsed} value={formatMinutes(order.elapsed_seconds || 0)} />
        <TileMetric label={t.kitchen.openedAt} value={formatDateTime(order.created_at)} />
        <TileMetric label={t.kitchen.changedAt} value={formatDateTime(order.last_status_changed_at)} />
      </div>

      <div className="flex-1 divide-y divide-[var(--pos-border)]">
        {(order.tickets || []).map((ticket) => (
          <React.Fragment key={ticket.id}>
            <TicketRow ticket={ticket} busy={busy} onAction={onAction} />
          </React.Fragment>
        ))}
      </div>
    </article>
  );
}

function TileMetric({ icon, label, value }: { icon?: React.ReactNode; label: string; value: string }) {
  return (
    <div className="p-3 min-w-0">
      <div className="flex items-center gap-1.5 font-mono text-[9px] uppercase tracking-widest text-[var(--pos-text-muted)]">
        {icon}
        {label}
      </div>
      <div className="mt-1 font-mono text-xs font-bold text-[var(--pos-text-primary)] truncate">{value}</div>
    </div>
  );
}

function TicketRow({
  ticket,
  busy,
  onAction,
}: {
  ticket: BackendKitchenTicket;
  busy: boolean;
  onAction: (ticketId: string, action: KitchenTicketAction) => void;
}) {
  const actions = orderActions[ticket.status] || [];
  return (
    <div className="p-3 space-y-3">
      <div className="flex items-start justify-between gap-3">
        <div className="min-w-0">
          <div className="font-sans text-sm font-semibold text-[var(--pos-text-primary)] break-words">
            {ticket.name}
          </div>
          <div className="mt-1 font-mono text-[10px] uppercase tracking-wider text-[var(--pos-text-muted)]">
            {ticket.quantity} {ticket.unit_code}
            {ticket.course ? ` · ${t.common.course} ${ticket.course}` : ''}
          </div>
          {ticket.comment && (
            <div className="mt-1 text-xs text-[var(--pos-text-secondary)] break-words">{ticket.comment}</div>
          )}
        </div>
        <span className="shrink-0 bg-[var(--pos-action-secondary)] border border-[var(--pos-border)] px-2 py-1 font-mono text-[10px] font-bold uppercase text-[var(--pos-text-secondary)]">
          {ticketStatusLabel(ticket.status)}
        </span>
      </div>
      {actions.length > 0 && (
        <div className="flex flex-wrap gap-2">
          {actions.map((action) => (
            <PosButton
              key={`${ticket.id}-${action}`}
              size="sm"
              variant={action === 'cancel' ? 'danger' : action === 'ready' || action === 'serve' ? 'success' : 'secondary'}
              disabled={busy}
              onClick={() => onAction(ticket.id, action)}
            >
              {actionLabel(action)}
            </PosButton>
          ))}
        </div>
      )}
    </div>
  );
}

function CatalogPicker({
  filteredItems,
  selectedId,
  search,
  filter,
  onSearch,
  onFilter,
  onSelect,
}: {
  filteredItems: BackendCatalogItem[];
  selectedId: string;
  search: string;
  filter: CatalogKindFilter;
  onSearch: (value: string) => void;
  onFilter: (value: CatalogKindFilter) => void;
  onSelect: (value: string) => void;
}) {
  return (
    <div className="border border-[var(--pos-border)] bg-[var(--pos-surface)] min-h-[420px] flex flex-col">
      <div className="p-4 border-b border-[var(--pos-border)] grid gap-3 md:grid-cols-[minmax(0,1fr)_180px]">
        <label className="relative block">
          <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-[var(--pos-text-muted)]" />
          <input
            className="h-12 w-full border border-[var(--pos-border)] bg-[var(--pos-surface-raised)] pl-10 pr-3 text-sm outline-none focus:ring-2 focus:ring-[var(--pos-focus-ring)]"
            value={search}
            onChange={(event) => onSearch(event.target.value)}
            placeholder={t.kitchen.searchCatalog}
          />
        </label>
        <select
          className="h-12 border border-[var(--pos-border)] bg-[var(--pos-surface-raised)] px-3 text-sm outline-none focus:ring-2 focus:ring-[var(--pos-focus-ring)]"
          value={filter}
          onChange={(event) => onFilter(event.target.value as CatalogKindFilter)}
        >
          {catalogKinds.map((kind) => (
            <option key={kind} value={kind}>{kind === 'all' ? t.kitchen.allKinds : catalogKindLabel(kind)}</option>
          ))}
        </select>
      </div>
      <div className="flex-1 min-h-0 overflow-auto pos-scrollbar-thin divide-y divide-[var(--pos-border)]">
        {filteredItems.length === 0 ? (
          <PosEmptyState title={t.kitchen.noCatalogItems} description={t.kitchen.searchCatalog} icon={<Search className="w-9 h-9" />} />
        ) : filteredItems.map((item) => {
          const active = selectedId === item.id;
          return (
            <button
              key={item.id}
              type="button"
              className={`w-full p-3 text-left grid gap-1 cursor-pointer transition-colors ${
                active ? 'bg-[var(--pos-action-secondary)] text-[var(--pos-text-primary)]' : 'hover:bg-[var(--pos-surface-raised)]'
              }`}
              onClick={() => onSelect(item.id)}
            >
              <span className="font-sans text-sm font-semibold">{item.name}</span>
              <span className="font-mono text-[10px] uppercase tracking-wider text-[var(--pos-text-muted)]">
                {catalogKindLabel(catalogKind(item))} · {item.base_unit || t.common.none}{item.sku ? ` · ${item.sku}` : ''}
              </span>
            </button>
          );
        })}
      </div>
    </div>
  );
}

function StockForm({
  tab,
  form,
  selectedItem,
  busy,
  onChange,
  onSubmit,
}: {
  tab: StockTab;
  form: StockFormState;
  selectedItem?: BackendCatalogItem;
  busy: boolean;
  onChange: (patch: Partial<StockFormState>) => void;
  onSubmit: () => void;
}) {
  const submitLabel = {
    receipt: t.kitchen.captureReceipt,
    count: t.kitchen.captureCount,
    writeoff: t.kitchen.captureWriteOff,
    production: t.kitchen.completeProduction,
  }[tab];

  return (
    <form className="border border-[var(--pos-border)] bg-[var(--pos-surface)] p-4" onSubmit={(event) => { event.preventDefault(); onSubmit(); }}>
      <div className="mb-4 border border-[var(--pos-border)] bg-[var(--pos-surface-raised)] p-3">
        <div className="font-mono text-[10px] uppercase tracking-widest text-[var(--pos-text-muted)]">{t.kitchen.selectedItem}</div>
        <div className="mt-1 font-sans text-sm font-semibold text-[var(--pos-text-primary)]">{selectedItem?.name || t.kitchen.selectCatalogItem}</div>
      </div>

      <PosFormRow id="stock-quantity" label={tab === 'count' ? t.kitchen.countedQuantity : t.kitchen.quantity}>
        <input id="stock-quantity" className={inputClassName} value={form.quantity} onChange={(event) => onChange({ quantity: event.target.value })} />
      </PosFormRow>
      <PosFormRow id="stock-unit" label={t.kitchen.unit}>
        <input id="stock-unit" className={inputClassName} value={form.unitCode} onChange={(event) => onChange({ unitCode: event.target.value })} />
      </PosFormRow>
      <PosFormRow id="stock-business-date" label={t.kitchen.businessDate}>
        <input id="stock-business-date" type="date" className={inputClassName} value={form.businessDate} onChange={(event) => onChange({ businessDate: event.target.value })} />
      </PosFormRow>

      {tab === 'receipt' && (
        <>
          <PosFormRow id="stock-supplier" label={t.kitchen.supplierName}>
            <input id="stock-supplier" className={inputClassName} value={form.supplierName} onChange={(event) => onChange({ supplierName: event.target.value })} />
          </PosFormRow>
          <PosFormRow id="stock-document-number" label={t.kitchen.documentNumber}>
            <input id="stock-document-number" className={inputClassName} value={form.documentNumber} onChange={(event) => onChange({ documentNumber: event.target.value })} />
          </PosFormRow>
          <PosFormRow id="stock-document-date" label={t.kitchen.documentDate}>
            <input id="stock-document-date" type="date" className={inputClassName} value={form.documentDate} onChange={(event) => onChange({ documentDate: event.target.value })} />
          </PosFormRow>
          <PosFormRow id="stock-unit-cost" label={t.kitchen.unitCostMinor}>
            <input id="stock-unit-cost" inputMode="numeric" className={inputClassName} value={form.unitCostMinor} onChange={(event) => onChange({ unitCostMinor: event.target.value })} />
          </PosFormRow>
          <PosFormRow id="stock-line-total" label={t.kitchen.lineTotalMinor}>
            <input id="stock-line-total" inputMode="numeric" className={inputClassName} value={form.lineTotalMinor} onChange={(event) => onChange({ lineTotalMinor: event.target.value })} />
          </PosFormRow>
        </>
      )}

      {tab === 'writeoff' && (
        <>
          <PosFormRow id="stock-reason-code" label={t.kitchen.writeOffReasonCode}>
            <input id="stock-reason-code" className={inputClassName} value={form.reasonCode} onChange={(event) => onChange({ reasonCode: event.target.value })} />
          </PosFormRow>
          <PosFormRow id="stock-reason" label={t.kitchen.writeOffReason}>
            <textarea id="stock-reason" rows={3} className={textareaClassName} value={form.reason} onChange={(event) => onChange({ reason: event.target.value })} />
          </PosFormRow>
        </>
      )}

      <PosButton fullWidth variant="primary" disabled={busy || !form.itemId} icon={<PackageCheck className="w-4 h-4" />}>
        {submitLabel}
      </PosButton>
    </form>
  );
}

function RecipeWorkspace({
  catalog,
  ingredientCatalog,
  recipe,
  recipeItemId,
  suggestion,
  busy,
  onRecipeItemChange,
  onLoadRecipe,
  onSuggestionChange,
  onSubmitSuggestion,
}: {
  catalog: BackendCatalogItem[];
  ingredientCatalog: BackendCatalogItem[];
  recipe: BackendKitchenRecipe | null;
  recipeItemId: string;
  suggestion: RecipeSuggestionState;
  busy: boolean;
  onRecipeItemChange: (value: string) => void;
  onLoadRecipe: () => void;
  onSuggestionChange: (patch: Partial<RecipeSuggestionState>) => void;
  onSubmitSuggestion: () => void;
}) {
  const ingredients = getRecipeIngredients(recipe);
  return (
    <div className="grid gap-4 xl:grid-cols-[minmax(0,1fr)_420px]">
      <div className="border border-[var(--pos-border)] bg-[var(--pos-surface)] min-h-[420px]">
        <div className="p-4 border-b border-[var(--pos-border)] grid gap-3 md:grid-cols-[minmax(0,1fr)_auto]">
          <select className={inputClassName} value={recipeItemId} onChange={(event) => onRecipeItemChange(event.target.value)}>
            <option value="">{t.kitchen.selectDishOrSemi}</option>
            {catalog.map((item) => <option key={item.id} value={item.id}>{item.name}</option>)}
          </select>
          <PosButton type="button" onClick={onLoadRecipe} disabled={busy || !recipeItemId} icon={<Utensils className="w-4 h-4" />}>
            {t.kitchen.loadRecipe}
          </PosButton>
        </div>
        {!recipe ? (
          <PosEmptyState title={t.kitchen.recipeEmpty} description={t.kitchen.loadRecipe} icon={<ClipboardList className="w-10 h-10" />} />
        ) : (
          <div className="p-4 space-y-4">
            <div>
              <div className="font-mono text-[10px] uppercase tracking-widest text-[var(--pos-text-muted)]">{t.kitchen.ingredients}</div>
              <h3 className="mt-1 font-sans text-lg font-bold text-[var(--pos-text-primary)]">
                {recipe.catalog_item?.name || catalog.find((item) => item.id === recipeItemId)?.name}
              </h3>
            </div>
            <div className="divide-y divide-[var(--pos-border)] border border-[var(--pos-border)]">
              {ingredients.map((line, index) => (
                <div key={line.line_id || `${line.catalog_item_id}-${index}`} className="p-3 grid gap-1 md:grid-cols-[minmax(0,1fr)_120px_100px]">
                  <div className="font-sans text-sm font-semibold">{line.catalog_item_name || line.ingredient_name || line.catalog_item_id}</div>
                  <div className="font-mono text-xs text-[var(--pos-text-secondary)]">{line.quantity} {line.unit_code}</div>
                  <div className="font-mono text-xs text-[var(--pos-text-muted)]">{line.loss_percent || '0'}%</div>
                </div>
              ))}
            </div>
            {recipe.proposals.length > 0 && (
              <div className="border border-[var(--pos-border)] bg-[var(--pos-surface-raised)] p-3">
                <div className="font-mono text-[10px] uppercase tracking-widest text-[var(--pos-text-muted)]">{t.kitchen.pendingProposals}</div>
                <div className="mt-2 flex flex-wrap gap-2">
                  {recipe.proposals.map((proposal) => (
                    <span key={proposal.id} className="border border-[var(--pos-border)] px-2 py-1 font-mono text-[10px] uppercase text-[var(--pos-text-secondary)]">
                      {proposalStatusLabel(proposal.status)}
                    </span>
                  ))}
                </div>
              </div>
            )}
          </div>
        )}
      </div>
      <RecipeSuggestionForm
        ingredients={ingredients}
        ingredientCatalog={ingredientCatalog}
        suggestion={suggestion}
        busy={busy || !recipeItemId}
        onChange={onSuggestionChange}
        onSubmit={onSubmitSuggestion}
      />
    </div>
  );
}

function RecipeSuggestionForm({
  ingredients,
  ingredientCatalog,
  suggestion,
  busy,
  onChange,
  onSubmit,
}: {
  ingredients: ReturnType<typeof getRecipeIngredients>;
  ingredientCatalog: BackendCatalogItem[];
  suggestion: RecipeSuggestionState;
  busy: boolean;
  onChange: (patch: Partial<RecipeSuggestionState>) => void;
  onSubmit: () => void;
}) {
  const needsLine = !['change_prep_time', 'add_ingredient'].includes(suggestion.action);
  const needsIngredient = ['add_ingredient', 'replace_ingredient'].includes(suggestion.action);
  return (
    <form className="border border-[var(--pos-border)] bg-[var(--pos-surface)] p-4" onSubmit={(event) => { event.preventDefault(); onSubmit(); }}>
      <h3 className="font-mono text-sm font-black uppercase tracking-widest text-[var(--pos-text-primary)] mb-4">{t.kitchen.suggestRecipeChange}</h3>
      <PosFormRow id="recipe-action" label={t.kitchen.recipeAction}>
        <select id="recipe-action" className={inputClassName} value={suggestion.action} onChange={(event) => onChange({ action: event.target.value as RecipeSuggestionAction })}>
          {recipeSuggestionActions.map((action) => <option key={action} value={action}>{recipeActionLabel(action)}</option>)}
        </select>
      </PosFormRow>
      {needsLine && (
        <PosFormRow id="recipe-line" label={t.kitchen.recipeLine}>
          <select id="recipe-line" className={inputClassName} value={suggestion.lineId} onChange={(event) => onChange({ lineId: event.target.value })}>
            <option value="">{t.common.none}</option>
            {ingredients.map((line, index) => (
              <option key={line.line_id || index} value={line.line_id || ''}>{line.catalog_item_name || line.ingredient_name || line.catalog_item_id}</option>
            ))}
          </select>
        </PosFormRow>
      )}
      {needsIngredient && (
        <PosFormRow id="recipe-ingredient" label={suggestion.action === 'add_ingredient' ? t.kitchen.newIngredient : t.kitchen.replacementIngredient}>
          <select id="recipe-ingredient" className={inputClassName} value={suggestion.ingredientItemId} onChange={(event) => onChange({ ingredientItemId: event.target.value })}>
            <option value="">{t.kitchen.selectCatalogItem}</option>
            {ingredientCatalog.map((item) => <option key={item.id} value={item.id}>{item.name}</option>)}
          </select>
        </PosFormRow>
      )}
      {suggestion.action === 'change_prep_time' && (
        <PosFormRow id="recipe-prep-time" label={t.kitchen.prepTimeDelta}>
          <input id="recipe-prep-time" inputMode="numeric" className={inputClassName} value={suggestion.prepTimeDeltaMinutes} onChange={(event) => onChange({ prepTimeDeltaMinutes: event.target.value })} />
        </PosFormRow>
      )}
      {['add_ingredient', 'change_quantity'].includes(suggestion.action) && (
        <>
          <PosFormRow id="recipe-quantity" label={t.kitchen.quantity}>
            <input id="recipe-quantity" className={inputClassName} value={suggestion.quantity} onChange={(event) => onChange({ quantity: event.target.value })} />
          </PosFormRow>
          <PosFormRow id="recipe-unit" label={t.kitchen.unit}>
            <input id="recipe-unit" className={inputClassName} value={suggestion.unitCode} onChange={(event) => onChange({ unitCode: event.target.value })} />
          </PosFormRow>
        </>
      )}
      {suggestion.action === 'change_loss_percent' && (
        <PosFormRow id="recipe-loss" label={t.kitchen.lossPercent}>
          <input id="recipe-loss" inputMode="numeric" className={inputClassName} value={suggestion.lossPercent} onChange={(event) => onChange({ lossPercent: event.target.value })} />
        </PosFormRow>
      )}
      <PosFormRow id="recipe-reason" label={t.kitchen.reason}>
        <textarea id="recipe-reason" rows={4} className={textareaClassName} value={suggestion.reason} onChange={(event) => onChange({ reason: event.target.value })} />
      </PosFormRow>
      <PosButton fullWidth variant="primary" disabled={busy} icon={<Send className="w-4 h-4" />}>{t.kitchen.submit}</PosButton>
    </form>
  );
}

function CatalogSuggestionForm({
  form,
  busy,
  onChange,
  onSubmit,
}: {
  form: CatalogSuggestionState;
  busy: boolean;
  onChange: (patch: Partial<CatalogSuggestionState>) => void;
  onSubmit: () => void;
}) {
  return (
    <form className="border border-[var(--pos-border)] bg-[var(--pos-surface)] p-4 grid gap-1 md:grid-cols-2 md:gap-x-4" onSubmit={(event) => { event.preventDefault(); onSubmit(); }}>
      <div className="md:col-span-2 flex items-center gap-2 mb-2">
        <CheckCircle2 className="w-5 h-5 text-[var(--pos-status-success)]" />
        <h3 className="font-mono text-sm font-black uppercase tracking-widest">{t.kitchen.suggestCatalogItem}</h3>
      </div>
      <PosFormRow id="catalog-kind" label={t.kitchen.suggestionKind}>
        <select id="catalog-kind" className={inputClassName} value={form.kind} onChange={(event) => onChange({ kind: event.target.value as CatalogSuggestionState['kind'] })}>
          {catalogKinds.filter((kind) => kind !== 'all').map((kind) => <option key={kind} value={kind}>{catalogKindLabel(kind)}</option>)}
        </select>
      </PosFormRow>
      <PosFormRow id="catalog-name" label={t.kitchen.name}>
        <input id="catalog-name" className={inputClassName} value={form.name} onChange={(event) => onChange({ name: event.target.value })} />
      </PosFormRow>
      <PosFormRow id="catalog-sku" label={t.kitchen.sku}>
        <input id="catalog-sku" className={inputClassName} value={form.sku} onChange={(event) => onChange({ sku: event.target.value })} />
      </PosFormRow>
      <PosFormRow id="catalog-base-unit" label={t.kitchen.baseUnit}>
        <input id="catalog-base-unit" className={inputClassName} value={form.baseUnit} onChange={(event) => onChange({ baseUnit: event.target.value })} />
      </PosFormRow>
      <PosFormRow id="catalog-kitchen-type" label={t.kitchen.kitchenType}>
        <input id="catalog-kitchen-type" className={inputClassName} value={form.kitchenType} onChange={(event) => onChange({ kitchenType: event.target.value })} />
      </PosFormRow>
      <PosFormRow id="catalog-accounting-category" label={t.kitchen.accountingCategory}>
        <input id="catalog-accounting-category" className={inputClassName} value={form.accountingCategory} onChange={(event) => onChange({ accountingCategory: event.target.value })} />
      </PosFormRow>
      <div className="md:col-span-2">
        <PosFormRow id="catalog-reason" label={t.kitchen.reason}>
          <textarea id="catalog-reason" rows={4} className={textareaClassName} value={form.reason} onChange={(event) => onChange({ reason: event.target.value })} />
        </PosFormRow>
      </div>
      <div className="md:col-span-2">
        <PosButton fullWidth variant="primary" disabled={busy} icon={<Send className="w-4 h-4" />}>{t.kitchen.submit}</PosButton>
      </div>
    </form>
  );
}

function ProposalList({ proposals }: { proposals: BackendKitchenProposal[] }) {
  if (proposals.length === 0) {
    return (
      <PosEmptyState
        title={t.kitchen.emptyProposals}
        description={t.kitchen.tabSuggestions}
        icon={<AlertTriangle className="w-10 h-10" />}
      />
    );
  }
  return (
    <div className="grid gap-3 md:grid-cols-2 xl:grid-cols-3">
      {proposals.map((proposal) => (
        <article key={proposal.id} className="border border-[var(--pos-border)] bg-[var(--pos-surface)] p-4 space-y-3">
          <div className="flex items-start justify-between gap-3">
            <div>
              <div className="font-mono text-[10px] uppercase tracking-widest text-[var(--pos-text-muted)]">{proposalKindLabel(proposal.kind)}</div>
              <h3 className="font-sans text-sm font-bold text-[var(--pos-text-primary)]">{proposal.action || proposal.outbox_event_type}</h3>
            </div>
            <span className="border border-[var(--pos-border)] bg-[var(--pos-action-secondary)] px-2 py-1 font-mono text-[10px] uppercase text-[var(--pos-text-secondary)]">
              {proposalStatusLabel(proposal.status)}
            </span>
          </div>
          <div className="font-mono text-[10px] uppercase tracking-wider text-[var(--pos-text-muted)]">
            {formatDateTime(proposal.created_at)}
          </div>
        </article>
      ))}
    </div>
  );
}

const inputClassName = 'h-12 w-full border border-[var(--pos-border)] bg-[var(--pos-surface-raised)] px-3 text-sm text-[var(--pos-text-primary)] outline-none focus:ring-2 focus:ring-[var(--pos-focus-ring)]';
const textareaClassName = 'w-full border border-[var(--pos-border)] bg-[var(--pos-surface-raised)] px-3 py-2 text-sm text-[var(--pos-text-primary)] outline-none focus:ring-2 focus:ring-[var(--pos-focus-ring)] resize-none';

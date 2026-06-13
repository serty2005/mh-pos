import { useEffect, useMemo, useState } from 'react';
import { RefreshCcw } from 'lucide-react';

import { canUseAnyPermission, canUsePermission } from '../../context/posContextHelpers';
import { usePOS } from '../../context/POSContext';
import { createApiClient, type KitchenTicketAction } from '../../shared/api';
import { t } from '../../shared/i18n';
import { PosBanner, PosButton, PosSectionHeader, PosTabs } from '../../shared/ui';
import { permissions } from '../../types';
import { KitchenCatalogSuggestionForm } from './KitchenCatalogSuggestionForm';
import { KitchenOrdersTab, type OrdersTab } from './KitchenOrdersTab';
import { KitchenProposalList } from './KitchenProposalList';
import { KitchenRecipeTab } from './KitchenRecipeTab';
import { KitchenStockTab, type StockTab } from './KitchenStockTab';
import { KitchenStopListTab } from './KitchenStopListTab';
import type {
  BackendCatalogItem,
  BackendKitchenOrderQueueItem,
  BackendKitchenProposal,
  BackendKitchenRecipe,
  BackendKitchenStopListState,
} from '../../shared/schemas';
import {
  catalogKind,
  catalogUnit,
  createInitialStockForm,
  createRecipeChange,
  isPositiveDecimal,
  localizedError,
  safeNumber,
  selectedRecipeVersionId,
  type CatalogKindFilter,
  type CatalogSuggestionState,
  type RecipeSuggestionState,
  type StockFormState,
  type StopListFormState,
} from './kitchenHelpers';

type KitchenBottomSection = 'orders' | 'stock' | 'kitchen';
type KitchenTab = 'recipes' | 'suggestions' | 'stop_list' | 'my_proposals';

export function POSKitchenSection({ section }: { section: KitchenBottomSection }) {
  const { authSnapshot, currentOperator } = usePOS();
  const api = useMemo(() => createApiClient(() => authSnapshot), [authSnapshot]);

  const [ordersTab, setOrdersTab] = useState<OrdersTab>('queue');
  const [stockTab, setStockTab] = useState<StockTab>('receipt');
  const [kitchenTab, setKitchenTab] = useState<KitchenTab>('recipes');
  const [orders, setOrders] = useState<BackendKitchenOrderQueueItem[]>([]);
  const [catalog, setCatalog] = useState<BackendCatalogItem[]>([]);
  const [recipe, setRecipe] = useState<BackendKitchenRecipe | null>(null);
  const [proposals, setProposals] = useState<BackendKitchenProposal[]>([]);
  const [stopList, setStopList] = useState<BackendKitchenStopListState[]>([]);
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
  const [stopListForm, setStopListForm] = useState<StopListFormState>({
    itemId: '',
    action: 'stop',
    reason: '',
  });
  const operatorPermissions = currentOperator?.permissions ?? [];
  const canViewKitchen = canUsePermission(operatorPermissions, permissions.KITCHEN_VIEW);
  const canChangeKitchenStatus = canUsePermission(operatorPermissions, permissions.KITCHEN_STATUS_CHANGE);
  const canViewKitchenCatalog = canUseAnyPermission(operatorPermissions, [
    permissions.CATALOG_VIEW,
    permissions.KITCHEN_CATALOG_VIEW,
  ]);
  const canViewRecipe = canUsePermission(operatorPermissions, permissions.KITCHEN_RECIPE_VIEW);
  const canSuggestRecipe = canUsePermission(operatorPermissions, permissions.KITCHEN_RECIPE_SUGGEST);
  const canSuggestCatalog = canUsePermission(operatorPermissions, permissions.KITCHEN_CATALOG_SUGGEST);
  const canViewStopList = canUsePermission(operatorPermissions, permissions.KITCHEN_STOP_LIST_VIEW);
  const canUpdateStopList = canUsePermission(operatorPermissions, permissions.KITCHEN_STOP_LIST_UPDATE);
  const canUseStock = canUseAnyPermission(operatorPermissions, [
    permissions.KITCHEN_STOCK_RECEIPT,
    permissions.KITCHEN_STOCK_INVENTORY_COUNT,
    permissions.KITCHEN_STOCK_WRITE_OFF,
    permissions.KITCHEN_PRODUCTION_COMPLETE,
  ]);

  const selectedStockItem = useMemo(
    () => catalog.find((item) => item.id === stockForm.itemId),
    [catalog, stockForm.itemId],
  );
  const selectedStopListItem = useMemo(
    () => catalog.find((item) => item.id === stopListForm.itemId),
    [catalog, stopListForm.itemId],
  );
  const stopListByCatalogItem = useMemo(() => {
    const map = new Map<string, BackendKitchenStopListState>();
    stopList.forEach((item) => map.set(item.catalog_item_id, item));
    return map;
  }, [stopList]);
  const canSubmitCurrentStock = useMemo(() => {
    switch (stockTab) {
      case 'receipt':
        return canUsePermission(operatorPermissions, permissions.KITCHEN_STOCK_RECEIPT);
      case 'count':
        return canUsePermission(operatorPermissions, permissions.KITCHEN_STOCK_INVENTORY_COUNT);
      case 'writeoff':
        return canUsePermission(operatorPermissions, permissions.KITCHEN_STOCK_WRITE_OFF);
      case 'production':
        return canUsePermission(operatorPermissions, permissions.KITCHEN_PRODUCTION_COMPLETE);
      default:
        return false;
    }
  }, [operatorPermissions, stockTab]);

  const activeCatalog = useMemo(() => catalog.filter((item) => item.active !== false), [catalog]);
  const dishCatalog = useMemo(
    () => activeCatalog.filter((item) => ['dish', 'semi_finished'].includes(catalogKind(item))),
    [activeCatalog],
  );
  const ingredientCatalog = useMemo(
    () => activeCatalog.filter((item) => ['good', 'semi_finished'].includes(catalogKind(item))),
    [activeCatalog],
  );
  const stopListCatalog = useMemo(
    () => activeCatalog.filter((item) => catalogKind(item) !== 'service'),
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

  const loadOrders = async () => {
    if (!canViewKitchen) return;
    const queue = await api.listKitchenOrderQueue({ limit: 100 });
    setOrders(queue.orders);
  };

  const loadCatalog = async () => {
    if (!canViewKitchenCatalog) return;
    const items = await api.listCatalogItems();
    setCatalog(items);
  };

  const loadProposals = async () => {
    if (!canViewKitchen) return;
    const items = await api.listKitchenProposals({ limit: 100 });
    setProposals(items);
  };

  const loadStopList = async () => {
    if (!canViewStopList) return;
    const items = await api.listKitchenStopList();
    setStopList(items);
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
      const tasks: Array<Promise<void>> = [];
      if (canViewKitchen) tasks.push(loadOrders(), loadProposals());
      if (canViewKitchenCatalog) tasks.push(loadCatalog());
      if (canViewStopList) tasks.push(loadStopList());
      if (tasks.length === 0) setSafeError(t.errors.noPermission);
      await Promise.all(tasks);
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

  const updateStopListForm = (patch: Partial<StopListFormState>) => {
    setStopListForm((current) => ({ ...current, ...patch }));
  };

  const selectStockItem = (itemId: string) => {
    const item = catalog.find((candidate) => candidate.id === itemId);
    updateStockForm({ itemId, unitCode: catalogUnit(item, stockForm.unitCode) });
  };

  const showNoPermission = () => {
    setSafeError(t.errors.noPermission);
    setSuccessMessage('');
  };

  const runTicketAction = (ticketId: string, action: KitchenTicketAction) => {
    if (!canChangeKitchenStatus) {
      showNoPermission();
      return;
    }
    void runSafe(async () => {
      await api.changeKitchenTicketStatus(ticketId, action);
      await loadOrders();
    });
  };

  const submitStock = () => {
    if (!canSubmitCurrentStock) {
      showNoPermission();
      return;
    }
    void runSafe(async () => {
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
  };

  const loadRecipe = () => {
    if (!canViewRecipe) {
      showNoPermission();
      return;
    }
    void runSafe(async () => {
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
  };

  const submitCatalogSuggestion = () => {
    if (!canSuggestCatalog) {
      showNoPermission();
      return;
    }
    void runSafe(async () => {
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
  };

  const submitRecipeSuggestion = () => {
    if (!canSuggestRecipe) {
      showNoPermission();
      return;
    }
    void runSafe(async () => {
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
  };

  const submitStopListUpdate = () => {
    if (!canUpdateStopList) {
      showNoPermission();
      return;
    }
    void runSafe(async () => {
      if (!stopListForm.itemId || !stopListForm.reason.trim()) throw new Error('validation');
      const existing = stopListByCatalogItem.get(stopListForm.itemId);
      await api.submitKitchenStopListUpdate({
        stop_list_id: existing?.id,
        catalog_item_id: stopListForm.itemId,
        available_quantity: stopListForm.action === 'stop' ? 0 : undefined,
        active: stopListForm.action === 'stop',
        reason: stopListForm.reason.trim(),
      });
      setStopListForm({ itemId: '', action: 'stop', reason: '' });
      await Promise.all([loadStopList(), loadCatalog()]);
    }, true);
  };

  const sectionTitle = section === 'orders' ? t.kitchen.navOrders : section === 'stock' ? t.kitchen.navStock : t.kitchen.navKitchen;
  const canUseCurrentKitchenSection = section === 'orders'
    ? canViewKitchen
    : section === 'stock'
      ? canViewKitchenCatalog && canUseStock
      : canViewKitchenCatalog && (
        canViewRecipe
        || canSuggestRecipe
        || canSuggestCatalog
        || canViewStopList
        || canViewKitchen
      );

  return (
    <div className="flex-1 min-h-0 flex flex-col bg-[var(--pos-bg)]">
      <PosSectionHeader
        title={t.kitchen.title}
        badge={sectionTitle}
        actions={(
          <PosButton
            size="sm"
            onClick={() => void refreshAll()}
            disabled={busy || loading || !canUseCurrentKitchenSection}
            icon={<RefreshCcw className="w-4 h-4" />}
          >
            {t.kitchen.refresh}
          </PosButton>
        )}
      />
      {safeError && <div className="px-4 pt-4"><PosBanner type="danger" message={safeError} /></div>}
      {successMessage && <div className="px-4 pt-4"><PosBanner type="success" message={successMessage} /></div>}
      {!canUseCurrentKitchenSection && <div className="px-4 pt-4"><PosBanner type="warning" message={t.errors.noPermission} /></div>}

      {canUseCurrentKitchenSection && section === 'orders' && (
        <KitchenOrdersTab
          orders={orders}
          ordersTab={ordersTab}
          loading={loading}
          busy={busy || !canChangeKitchenStatus}
          onTabChange={setOrdersTab}
          onAction={runTicketAction}
        />
      )}

      {canUseCurrentKitchenSection && section === 'stock' && (
        <KitchenStockTab
          stockTab={stockTab}
          filteredCatalog={filteredCatalog}
          selectedItem={selectedStockItem}
          stockForm={stockForm}
          catalogSearch={catalogSearch}
          catalogFilter={catalogFilter}
          busy={busy || !canSubmitCurrentStock}
          onTabChange={setStockTab}
          onCatalogSearch={setCatalogSearch}
          onCatalogFilter={setCatalogFilter}
          onSelectItem={selectStockItem}
          onFormChange={updateStockForm}
          onSubmit={submitStock}
        />
      )}

      {canUseCurrentKitchenSection && section === 'kitchen' && (
        <div className="flex-1 min-h-0 overflow-hidden flex flex-col">
          <PosTabs
            id="kitchen-tabs"
            activeId={kitchenTab}
            onChange={(id) => setKitchenTab(id as KitchenTab)}
            items={[
              { id: 'recipes', label: t.kitchen.tabRecipes },
              { id: 'suggestions', label: t.kitchen.tabSuggestions },
              { id: 'stop_list', label: t.kitchen.tabStopList, count: stopList.filter((item) => item.active).length },
              { id: 'my_proposals', label: t.kitchen.tabMyProposals, count: proposals.length },
            ]}
          />
          <div className="flex-1 min-h-0 p-4 overflow-auto pos-scrollbar-thin">
            {kitchenTab === 'recipes' && (
              <KitchenRecipeTab
                catalog={dishCatalog}
                ingredientCatalog={ingredientCatalog}
                recipe={recipe}
                recipeItemId={recipeItemId}
                suggestion={recipeSuggestion}
                busy={busy}
                canLoadRecipe={canViewRecipe}
                canSubmitSuggestion={canSuggestRecipe}
                onRecipeItemChange={setRecipeItemId}
                onLoadRecipe={loadRecipe}
                onSuggestionChange={updateRecipeSuggestion}
                onSubmitSuggestion={submitRecipeSuggestion}
              />
            )}

            {kitchenTab === 'suggestions' && (
              <div className="max-w-4xl">
                <KitchenCatalogSuggestionForm
                  form={catalogSuggestion}
                  busy={busy || !canSuggestCatalog}
                  onChange={updateCatalogSuggestion}
                  onSubmit={submitCatalogSuggestion}
                />
              </div>
            )}

            {kitchenTab === 'stop_list' && (
              <KitchenStopListTab
                catalog={stopListCatalog}
                stopList={stopList}
                selectedItem={selectedStopListItem}
                form={stopListForm}
                busy={busy || !canUpdateStopList}
                onChange={updateStopListForm}
                onSubmit={submitStopListUpdate}
              />
            )}

            {kitchenTab === 'my_proposals' && (
              <KitchenProposalList proposals={proposals} />
            )}
          </div>
        </div>
      )}
    </div>
  );
}

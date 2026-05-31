import { ApiError, type KitchenTicketAction } from '../../shared/api';
import { t } from '../../shared/i18n';
import type { BackendCatalogItem, BackendKitchenRecipe } from '../../shared/schemas';

export type CatalogKindFilter = 'all' | 'dish' | 'good' | 'semi_finished' | 'service';
export type StopListAction = 'stop' | 'resume';
export type RecipeSuggestionAction =
  | 'change_prep_time'
  | 'add_ingredient'
  | 'remove_ingredient'
  | 'replace_ingredient'
  | 'change_quantity'
  | 'change_loss_percent';

export type StockFormState = {
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

export type CatalogSuggestionState = {
  kind: 'dish' | 'good' | 'semi_finished' | 'service';
  name: string;
  sku: string;
  baseUnit: string;
  kitchenType: string;
  accountingCategory: string;
  reason: string;
};

export type RecipeSuggestionState = {
  action: RecipeSuggestionAction;
  lineId: string;
  ingredientItemId: string;
  quantity: string;
  unitCode: string;
  lossPercent: string;
  prepTimeDeltaMinutes: string;
  reason: string;
};

export type StopListFormState = {
  itemId: string;
  action: StopListAction;
  reason: string;
};

export const activeOrderStatuses = ['queued', 'accepted', 'in_progress', 'partially_ready', 'mixed'];
export const readyOrderStatuses = ['ready', 'partially_ready', 'partially_served'];
export const catalogKinds: CatalogKindFilter[] = ['all', 'dish', 'good', 'semi_finished', 'service'];
export const recipeSuggestionActions: RecipeSuggestionAction[] = [
  'change_prep_time',
  'add_ingredient',
  'remove_ingredient',
  'replace_ingredient',
  'change_quantity',
  'change_loss_percent',
];
export const stopListActions: StopListAction[] = ['stop', 'resume'];

export function todayLocalDate() {
  return new Date().toISOString().slice(0, 10);
}

export function createInitialStockForm(): StockFormState {
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

export function catalogKind(item: BackendCatalogItem) {
  return (item.type || item.kind || item.item_type || '').toLowerCase();
}

export function catalogUnit(item: BackendCatalogItem | undefined, fallback = 'KG') {
  return item?.base_unit || fallback;
}

export function catalogKindLabel(kind: string) {
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

export function localizedError(error: unknown) {
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

export function formatMinutes(seconds: number) {
  return `${Math.max(0, Math.floor(seconds / 60))} ${t.kitchen.minutes}`;
}

export function formatDateTime(value = '') {
  if (!value) return t.common.none;
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return value;
  return date.toLocaleString('ru-RU', { day: '2-digit', month: '2-digit', hour: '2-digit', minute: '2-digit' });
}

export function proposalStatusLabel(status: string) {
  return t.kitchen.proposalStatus[status as keyof typeof t.kitchen.proposalStatus] ?? status;
}

export function proposalKindLabel(kind: string) {
  return t.kitchen.proposalKind[kind as keyof typeof t.kitchen.proposalKind] ?? kind;
}

export function ticketStatusLabel(status: string) {
  return t.kitchen.ticketStatus[status as keyof typeof t.kitchen.ticketStatus] ?? status;
}

export function orderStatusLabel(status: string) {
  return t.kitchen.orderStatus[status as keyof typeof t.kitchen.orderStatus] ?? status;
}

export function actionLabel(action: KitchenTicketAction) {
  return t.kitchen.actions[action];
}

export function recipeActionLabel(action: RecipeSuggestionAction) {
  return t.kitchen.recipeActions[action];
}

export function stopListActionLabel(action: StopListAction) {
  return t.kitchen.stopListActions[action];
}

export function stopListSyncLabel(state: string) {
  return t.kitchen.stopListSync[state as keyof typeof t.kitchen.stopListSync] ?? state;
}

export function safeNumber(value: string) {
  const parsed = Number.parseInt(value, 10);
  return Number.isFinite(parsed) ? parsed : 0;
}

export function isPositiveDecimal(value: string) {
  return Number.parseFloat(value) > 0;
}

export function getRecipeIngredients(recipe: BackendKitchenRecipe | null) {
  if (!recipe) return [];
  return recipe.ingredients.length > 0 ? recipe.ingredients : recipe.lines;
}

export function selectedRecipeVersionId(recipe: BackendKitchenRecipe | null) {
  return recipe?.recipe_version_id || recipe?.recipe_version?.id || '';
}

export function createRecipeChange(state: RecipeSuggestionState) {
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

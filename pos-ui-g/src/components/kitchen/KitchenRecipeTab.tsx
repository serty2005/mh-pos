import { ClipboardList, Send, Utensils } from 'lucide-react';

import { t } from '../../shared/i18n';
import { PosButton, PosEmptyState, PosFormRow, PosInlineStatusBadge } from '../../shared/ui';
import type { BackendCatalogItem, BackendKitchenRecipe } from '../../shared/schemas';
import {
  getRecipeIngredients,
  proposalStatusLabel,
  recipeActionLabel,
  recipeSuggestionActions,
  type RecipeSuggestionAction,
  type RecipeSuggestionState,
} from './kitchenHelpers';

export function KitchenRecipeTab({
  catalog,
  ingredientCatalog,
  recipe,
  recipeItemId,
  suggestion,
  busy,
  canLoadRecipe,
  canSubmitSuggestion,
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
  canLoadRecipe: boolean;
  canSubmitSuggestion: boolean;
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
          <select
            className={inputClassName}
            value={recipeItemId}
            onChange={(event) => onRecipeItemChange(event.target.value)}
          >
            <option value="">{t.kitchen.selectDishOrSemi}</option>
            {catalog.map((item) => <option key={item.id} value={item.id}>{item.name}</option>)}
          </select>
          <PosButton type="button" onClick={onLoadRecipe} disabled={busy || !canLoadRecipe || !recipeItemId} icon={<Utensils className="w-4 h-4" />}>
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
                    <PosInlineStatusBadge key={proposal.id} variant="neutral">
                      {proposalStatusLabel(proposal.status)}
                    </PosInlineStatusBadge>
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
        busy={busy || !canSubmitSuggestion || !recipeItemId}
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

const inputClassName = 'h-12 w-full border border-[var(--pos-border)] bg-[var(--pos-surface-raised)] px-3 text-sm text-[var(--pos-text-primary)] outline-none focus:ring-2 focus:ring-[var(--pos-focus-ring)]';
const textareaClassName = 'w-full border border-[var(--pos-border)] bg-[var(--pos-surface-raised)] px-3 py-2 text-sm text-[var(--pos-text-primary)] outline-none focus:ring-2 focus:ring-[var(--pos-focus-ring)] resize-none';

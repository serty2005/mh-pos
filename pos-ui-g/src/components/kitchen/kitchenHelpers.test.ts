import { describe, expect, it } from 'vitest';

import { ApiError } from '../../shared/api';
import { t } from '../../shared/i18n';
import {
  createRecipeChange,
  formatDateTime,
  isPositiveDecimal,
  localizedError,
  safeNumber,
  type RecipeSuggestionState,
} from './kitchenHelpers';

function recipeState(patch: Partial<RecipeSuggestionState>): RecipeSuggestionState {
  return {
    action: 'change_prep_time',
    lineId: '',
    ingredientItemId: '',
    quantity: '1.000',
    unitCode: 'KG',
    lossPercent: '0',
    prepTimeDeltaMinutes: '0',
    reason: 'audit',
    ...patch,
  };
}

describe('kitchen helpers', () => {
  it('falls back to zero for invalid safeNumber input', () => {
    expect(safeNumber('42')).toBe(42);
    expect(safeNumber('42.9')).toBe(42);
    expect(safeNumber('abc')).toBe(0);
  });

  it('checks positive decimal input without accepting zero or invalid text', () => {
    expect(isPositiveDecimal('0.001')).toBe(true);
    expect(isPositiveDecimal('0')).toBe(false);
    expect(isPositiveDecimal('-1')).toBe(false);
    expect(isPositiveDecimal('bad')).toBe(false);
  });

  it('creates recipe change payloads for supported actions', () => {
    expect(createRecipeChange(recipeState({ action: 'change_prep_time' }))).toEqual([]);
    expect(createRecipeChange(recipeState({
      action: 'replace_ingredient',
      lineId: 'line-1',
      ingredientItemId: 'ingredient-2',
      quantity: '2.500',
      unitCode: 'KG',
      lossPercent: '4',
    }))).toEqual([{
      action: 'replace_ingredient',
      line_id: 'line-1',
      to_catalog_item_id: 'ingredient-2',
      quantity: '2.500',
      unit_code: 'KG',
      loss_percent: '4',
    }]);
  });

  it('keeps invalid date values stable for display fallback', () => {
    expect(formatDateTime('')).toBe(t.common.none);
    expect(formatDateTime('not-a-date')).toBe('not-a-date');
  });

  it('maps safe ApiError message keys to localized copy', () => {
    expect(localizedError(new Error('validation'))).toBe(t.errors.validation);
    expect(localizedError(new ApiError({
      status: 403,
      code: 'FORBIDDEN',
      messageKey: 'errors.permission',
      category: 'permission',
    }))).toBe(t.errors.noPermission);
  });
});

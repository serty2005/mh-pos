import { describe, expect, it } from 'vitest';

import {
  defaultModifierSelections,
  modifierSelectionTotal,
  toggleModifierSelection,
} from './modifierSelection';
import type { MenuItem, ModifierGroup } from '../../types';

const milkGroup: ModifierGroup = {
  id: 'milk',
  name: 'Молоко',
  minRequired: 0,
  maxAllowed: 1,
  options: [
    { id: 'regular', name: 'Обычное', price: 0 },
    { id: 'lactose-free', name: 'Lactose Free', price: 2000 },
  ],
};

describe('modifier selection state', () => {
  it('keeps a selected single-choice modifier and includes it in the displayed total', () => {
    const selected = toggleModifierSelection([], milkGroup, milkGroup.options[1]);

    expect(selected).toEqual([
      {
        groupId: 'milk',
        groupName: 'Молоко',
        optionId: 'lactose-free',
        optionName: 'Lactose Free',
        price: 2000,
      },
    ]);
    expect(modifierSelectionTotal(12900, selected)).toBe(14900);
  });

  it('auto-selects only required single-choice defaults for add mode', () => {
    const item: MenuItem = {
      id: 'espresso',
      catalogItemId: 'catalog-espresso',
      name: 'Espresso',
      price: 12900,
      category: 'dish',
      isAvailable: true,
      modifierGroups: [
        { ...milkGroup, minRequired: 0 },
        {
          id: 'size',
          name: 'Размер',
          minRequired: 1,
          maxAllowed: 1,
          options: [{ id: 'std', name: 'Стандарт', price: 0 }],
        },
      ],
    };

    expect(defaultModifierSelections(item, 'add')).toEqual([
      {
        groupId: 'size',
        groupName: 'Размер',
        optionId: 'std',
        optionName: 'Стандарт',
        price: 0,
      },
    ]);
  });
});

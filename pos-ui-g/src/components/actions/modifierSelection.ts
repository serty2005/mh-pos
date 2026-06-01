import type { MenuItem, ModifierGroup, ModifierOption, SelectedModifier } from '../../types';

const EMPTY_SELECTIONS: SelectedModifier[] = [];

export function initialSelectionsForMode(
  item: MenuItem | null,
  mode: 'add' | 'edit',
  initialSelections: SelectedModifier[] = EMPTY_SELECTIONS,
) {
  if (!item) return EMPTY_SELECTIONS;
  if (mode === 'edit') return initialSelections;
  return defaultModifierSelections(item, mode);
}

export function defaultModifierSelections(item: MenuItem, mode: 'add' | 'edit') {
  if (mode !== 'add') return EMPTY_SELECTIONS;
  const selections: SelectedModifier[] = [];
  item.modifierGroups?.forEach((group) => {
    if (group.minRequired === 1 && group.maxAllowed === 1 && group.options.length > 0) {
      selections.push(selectedModifierFromOption(group, group.options[0]));
    }
  });
  return selections;
}

export function toggleModifierSelection(
  current: SelectedModifier[],
  group: ModifierGroup,
  option: ModifierOption,
) {
  const isCurrentlySelected = current.some((selection) => (
    selection.groupId === group.id && selection.optionId === option.id
  ));

  if (group.maxAllowed === 1) {
    const withoutGroup = current.filter((selection) => selection.groupId !== group.id);
    if (isCurrentlySelected && group.minRequired === 0) return withoutGroup;
    return [...withoutGroup, selectedModifierFromOption(group, option)];
  }

  if (isCurrentlySelected) {
    return current.filter((selection) => !(selection.groupId === group.id && selection.optionId === option.id));
  }

  const groupSelections = current.filter((selection) => selection.groupId === group.id);
  if (groupSelections.length >= group.maxAllowed) return current;
  return [...current, selectedModifierFromOption(group, option)];
}

export function modifierSelectionTotal(basePrice: number, selections: SelectedModifier[]) {
  return basePrice + selections.reduce((sum, selection) => sum + selection.price, 0);
}

function selectedModifierFromOption(group: ModifierGroup, option: ModifierOption): SelectedModifier {
  return {
    groupId: group.id,
    groupName: group.name,
    optionId: option.id,
    optionName: option.name,
    price: option.price,
  };
}

import React, { useState, useEffect } from 'react';
import { MenuItem, SelectedModifier, ModifierGroup } from '../../types';
import { t } from '../../shared/i18n';
import { PosButton, PosDialog } from '../../shared/ui';
import { Check } from 'lucide-react';

interface ModifierSelectionDialogProps {
  isOpen: boolean;
  onClose: () => void;
  item: MenuItem | null;
  initialSelections?: SelectedModifier[];
  mode?: 'add' | 'edit';
  onSubmit: (selections: SelectedModifier[]) => void;
}

export const ModifierSelectionDialog: React.FC<ModifierSelectionDialogProps> = ({
  isOpen,
  onClose,
  item,
  initialSelections = [],
  mode = 'add',
  onSubmit
}) => {
  const [selections, setSelections] = useState<SelectedModifier[]>([]);
  const [errors, setErrors] = useState<Record<string, string>>({});

  useEffect(() => {
    if (item) {
      setSelections(initialSelections);
      setErrors({});
      
      // Auto-select standard default options for groups requiring exactly 1 choice
      if (mode === 'add' && item.modifierGroups) {
        const initialSelections: SelectedModifier[] = [];
        item.modifierGroups.forEach(group => {
          if (group.minRequired === 1 && group.options.length > 0) {
            const first = group.options[0];
            initialSelections.push({
              groupId: group.id,
              groupName: group.name,
              optionId: first.id,
              optionName: first.name,
              price: first.price
            });
          }
        });
        setSelections(initialSelections);
      }
    }
  }, [initialSelections, item, isOpen, mode]);

  if (!item || !item.modifierGroups) return null;

  const handleOptionToggle = (group: ModifierGroup, optionId: string, optionName: string, optionPrice: number) => {
    setSelections(prev => {
      const filteredGroup = prev.filter(sel => sel.groupId !== group.id);
      const isCurrentlySelected = prev.some(sel => sel.optionId === optionId);

      if (group.maxAllowed === 1) {
        // Single choice selection
        return [
          ...filteredGroup,
          { groupId: group.id, groupName: group.name, optionId, optionName, price: optionPrice }
        ];
      } else {
        // Multi choice selection
        const groupSelections = prev.filter(sel => sel.groupId === group.id);
        if (isCurrentlySelected) {
          // Remove selected option
          return prev.filter(sel => sel.optionId !== optionId);
        } else {
          // Add if size remains within constraints
          if (groupSelections.length < group.maxAllowed) {
            return [
              ...prev,
              { groupId: group.id, groupName: group.name, optionId, optionName, price: optionPrice }
            ];
          }
          return prev;
        }
      }
    });

    // Clear group errors
    if (errors[group.id]) {
      setErrors(prev => {
        const next = { ...prev };
        delete next[group.id];
        return next;
      });
    }
  };

  const handleValidationAndSubmit = () => {
    const nextErrors: Record<string, string> = {};
    let valid = true;

    item.modifierGroups?.forEach(group => {
      const groupSelections = selections.filter(sel => sel.groupId === group.id);
      if (groupSelections.length < group.minRequired) {
        nextErrors[group.id] = `Необходимо выбрать минимум: ${group.minRequired}`;
        valid = false;
      }
    });

    if (!valid) {
      setErrors(nextErrors);
      return;
    }

    onSubmit(selections);
  };

  const calculatedTotal = item.price + selections.reduce((sum, sel) => sum + sel.price, 0);

  return (
    <PosDialog
      isOpen={isOpen}
      onClose={onClose}
      title={mode === 'edit' ? 'Изменение модификаторов' : 'Настройка модификаторов'}
      footer={
        <>
          <PosButton variant="secondary" size="sm" onClick={onClose}>
            {t.common.cancel}
          </PosButton>
          <PosButton 
            id="modifier-submit-btn"
            variant="primary" 
            size="sm" 
            onClick={handleValidationAndSubmit}
            icon={<Check className="w-4 h-4" />}
          >
            {mode === 'edit' ? t.common.save : t.common.confirm} ({calculatedTotal} ₽)
          </PosButton>
        </>
      }
    >
      <div className="flex flex-col gap-6 select-none">
        
        {/* Core Item Header */}
        <div className="border-b border-[var(--pos-border)] pb-4">
          <h4 className="font-sans text-base font-bold text-[var(--pos-text-primary)]">{item.name}</h4>
          <span className="font-mono text-xs font-semibold text-[var(--pos-text-muted)]">Базовая цена: {item.price} ₽</span>
        </div>

        {/* Modifier groups iterator */}
        {item.modifierGroups.map((group) => {
          const groupSelections = selections.filter(sel => sel.groupId === group.id);
          const hasError = errors[group.id];

          return (
            <div key={group.id} id={`mod-group-${group.id}`} className="space-y-3">
              <div className="flex items-center justify-between">
                <span className="font-mono text-xs font-bold uppercase tracking-wider text-[var(--pos-text-secondary)]">
                  {group.name}
                  {group.minRequired > 0 && <span className="text-[var(--pos-status-danger)] ml-0.5">*</span>}
                </span>
                <span className="font-mono text-[10px] text-[var(--pos-text-muted)] uppercase">
                  (Мин: {group.minRequired} / Макс: {group.maxAllowed})
                </span>
              </div>

              {/* Selector buttons list */}
              <div className="grid grid-cols-1 sm:grid-cols-2 gap-2">
                {group.options.map((opt) => {
                  const isSelected = selections.some(sel => sel.optionId === opt.id);
                  
                  return (
                    <button
                      key={opt.id}
                      id={`mod-opt-${opt.id}`}
                      type="button"
                      onClick={() => handleOptionToggle(group, opt.id, opt.name, opt.price)}
                      className={`h-[48px] px-4 font-sans text-xs font-semibold border flex items-center justify-between cursor-pointer select-none transition-all rounded-none
                        ${isSelected
                          ? 'bg-[var(--pos-action-primary)] text-[var(--pos-surface)] border-[var(--pos-action-primary)] font-bold'
                          : 'bg-[var(--pos-surface)] text-[var(--pos-text-primary)] border-[var(--pos-border)] hover:bg-[var(--pos-surface-raised)]'
                        }`}
                    >
                      <span className="truncate">{opt.name}</span>
                      <span className="font-mono font-bold shrink-0 ml-2">
                        {opt.price > 0 ? `+${opt.price} ₽` : '0 ₽'}
                      </span>
                    </button>
                  );
                })}
              </div>

              {hasError && (
                <span className="font-sans text-[11px] font-semibold text-[var(--pos-status-danger)] block select-none">
                  {hasError}
                </span>
              )}
            </div>
          );
        })}
      </div>
    </PosDialog>
  );
};

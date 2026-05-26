import React, { useState, useEffect } from 'react';
import { MenuItem, ModifierGroup, OrderLine, SelectedModifier } from '../../types';
import { t } from '../../shared/i18n';
import { PosButton, PosDialog, PosFormRow } from '../../shared/ui';
import { Trash2, Check } from 'lucide-react';

interface ActionsDialogProps {
  isOpen: boolean;
  onClose: () => void;
  line: OrderLine | null;
  menuItem?: MenuItem | null;
  onUpdateLine: (comment: string, course: number, modifiers: SelectedModifier[]) => void;
  onDeleteLine: () => void;
}

export const ActionsDialog: React.FC<ActionsDialogProps> = ({
  isOpen,
  onClose,
  line,
  menuItem,
  onUpdateLine,
  onDeleteLine
}) => {
  const [comment, setComment] = useState<string>('');
  const [course, setCourse] = useState<number>(1);
  const [modifierSelections, setModifierSelections] = useState<SelectedModifier[]>([]);
  const [modifierErrors, setModifierErrors] = useState<Record<string, string>>({});

  useEffect(() => {
    if (isOpen && line) {
      setComment(line.comment || '');
      setCourse(line.course || 1);
      setModifierSelections(line.selectedModifiers);
      setModifierErrors({});
    }
  }, [isOpen, line]);

  if (!line) return null;

  const handleSubmit = () => {
    const nextErrors: Record<string, string> = {};
    let valid = true;
    menuItem?.modifierGroups?.forEach((group) => {
      const count = modifierSelections.filter((selection) => selection.groupId === group.id).length;
      if (count < group.minRequired) {
        nextErrors[group.id] = `${t.modals.modifierMinRequired}: ${group.minRequired}`;
        valid = false;
      }
    });
    if (!valid) {
      setModifierErrors(nextErrors);
      return;
    }
    onUpdateLine(comment, course, modifierSelections);
    onClose();
  };

  const handleModifierToggle = (group: ModifierGroup, optionId: string, optionName: string, optionPrice: number) => {
    setModifierSelections((prev) => {
      const isSelected = prev.some((selection) => selection.groupId === group.id && selection.optionId === optionId);
      if (group.maxAllowed === 1) {
        if (isSelected && group.minRequired === 0) {
          return prev.filter((selection) => selection.groupId !== group.id);
        }
        return [
          ...prev.filter((selection) => selection.groupId !== group.id),
          { groupId: group.id, groupName: group.name, optionId, optionName, price: optionPrice },
        ];
      }
      const groupSelections = prev.filter((selection) => selection.groupId === group.id);
      if (isSelected) {
        return prev.filter((selection) => !(selection.groupId === group.id && selection.optionId === optionId));
      }
      if (group.maxAllowed > 0 && groupSelections.length >= group.maxAllowed) {
        return prev;
      }
      return [...prev, { groupId: group.id, groupName: group.name, optionId, optionName, price: optionPrice }];
    });
    if (modifierErrors[group.id]) {
      setModifierErrors((prev) => {
        const next = { ...prev };
        delete next[group.id];
        return next;
      });
    }
  };

  return (
    <PosDialog
      isOpen={isOpen}
      onClose={onClose}
      title={t.modals.lineActionsTitle}
      footer={
        <div className="flex w-full justify-between items-center shrink-0">
          {/* Delete/Void Line Trigger */}
          <button
            id="action-void-line-btn"
            type="button"
            onClick={() => {
              onDeleteLine();
              onClose();
            }}
            className="flex items-center gap-1.5 font-mono text-xs font-bold text-[var(--pos-status-danger)] bg-transparent hover:bg-red-50 dark:hover:bg-red-950/20 px-4 h-11 border border-transparent select-none cursor-pointer rounded-none"
          >
            <Trash2 className="w-4 h-4 shrink-0" />
            <span>{t.modals.deleteLine}</span>
          </button>

          <div className="flex gap-2">
            <PosButton variant="secondary" size="sm" onClick={onClose}>
              {t.common.cancel}
            </PosButton>
            <PosButton 
              id="action-save-line-btn"
              variant="primary" 
              size="sm" 
              onClick={handleSubmit}
              icon={<Check className="w-4 h-4" />}
            >
              {t.common.execute}
            </PosButton>
          </div>
        </div>
      }
    >
      <div className="flex flex-col gap-5 select-none">
        
        {/* Selected dish descriptor */}
        <div className="p-4 bg-[var(--pos-surface-raised)] border border-[var(--pos-border)]">
          <span className="font-mono text-[10px] text-[var(--pos-text-muted)] uppercase tracking-wider">{t.modals.orderLine}</span>
          <h4 className="font-sans text-sm md:text-base font-bold text-[var(--pos-text-primary)] mt-1">{line.name}</h4>
          <span className="font-mono text-xs text-[var(--pos-text-secondary)] font-semibold mt-0.5">
            {t.modals.linePrice}: {line.price} {t.common.ruble} × {line.quantity} {t.common.pieces}
          </span>
        </div>

        {/* Course Option Button selectors */}
        <div className="space-y-2">
          <span className="font-mono text-xs font-bold uppercase tracking-wider text-[var(--pos-text-secondary)]">
            {t.modals.selectCourse}
          </span>
          <div className="grid grid-cols-3 gap-2">
            {[1, 2, 3].map((num) => {
              const active = course === num;
              return (
                <button
                  key={num}
                  id={`course-btn-${num}`}
                  type="button"
                  onClick={() => setCourse(num)}
                  className={`h-[48px] font-mono text-xs uppercase font-bold border cursor-pointer select-none transition-all rounded-none
                    ${active
                      ? 'bg-[var(--pos-action-primary)] border-[var(--pos-action-primary)] text-[var(--pos-surface)]'
                      : 'bg-[var(--pos-surface)] text-[var(--pos-text-primary)] border-[var(--pos-border)] hover:bg-[var(--pos-surface-raised)]'
                    }`}
                >
                  {num}-й Курс
                </button>
              );
            })}
          </div>
        </div>

        {/* Free text commentaries input row */}
        <PosFormRow
          label={t.common.comment}
          id="item-comment-input"
        >
          <input
            id="item-comment-input"
            type="text"
            className="w-full h-12 border border-[var(--pos-border)] px-4 font-sans text-sm bg-[var(--pos-surface)] text-[var(--pos-text-primary)] rounded-none outline-none focus:border-[var(--pos-border-strong)]"
            placeholder={t.modals.editComment}
            value={comment}
            onChange={(e) => setComment(e.target.value)}
          />
        </PosFormRow>

        {menuItem?.modifierGroups?.length ? (
          <div className="space-y-4">
            <span className="font-mono text-xs font-bold uppercase tracking-wider text-[var(--pos-text-secondary)]">
              {t.modals.modifiers}
            </span>
            {menuItem.modifierGroups.map((group) => {
              const hasError = modifierErrors[group.id];
              return (
                <div key={group.id} className="space-y-2">
                  <div className="flex items-center justify-between gap-3">
                    <span className="font-mono text-[11px] font-bold uppercase text-[var(--pos-text-secondary)]">
                      {group.name}
                    </span>
                    <span className="font-mono text-[10px] text-[var(--pos-text-muted)]">
                      {group.minRequired} / {group.maxAllowed || '∞'}
                    </span>
                  </div>
                  <div className="grid grid-cols-1 sm:grid-cols-2 gap-2">
                    {group.options.map((option) => {
                      const isSelected = modifierSelections.some((selection) => selection.groupId === group.id && selection.optionId === option.id);
                      return (
                        <button
                          key={option.id}
                          type="button"
                          onClick={() => handleModifierToggle(group, option.id, option.name, option.price)}
                          className={`h-11 px-3 border flex items-center justify-between text-left font-sans text-xs font-semibold rounded-none ${
                            isSelected
                              ? 'bg-[var(--pos-action-primary)] border-[var(--pos-action-primary)] text-[var(--pos-surface)]'
                              : 'bg-[var(--pos-surface)] border-[var(--pos-border)] text-[var(--pos-text-primary)]'
                          }`}
                        >
                          <span className="truncate">{option.name}</span>
                          <span className="font-mono font-bold shrink-0 ml-2">{option.price > 0 ? `+${option.price} ₽` : '0 ₽'}</span>
                        </button>
                      );
                    })}
                  </div>
                  {hasError && <span className="text-[11px] font-semibold text-[var(--pos-status-danger)]">{hasError}</span>}
                </div>
              );
            })}
          </div>
        ) : null}

      </div>
    </PosDialog>
  );
};

import React, { useState, useEffect } from 'react';
import { OrderLine } from '../../types';
import { t } from '../../shared/i18n';
import { PosButton, PosDialog, PosFormRow } from '../../shared/ui';
import { Trash2, Check } from 'lucide-react';

interface ActionsDialogProps {
  isOpen: boolean;
  onClose: () => void;
  line: OrderLine | null;
  onUpdateLine: (comment: string, course: number) => void;
  onDeleteLine: () => void;
}

export const ActionsDialog: React.FC<ActionsDialogProps> = ({
  isOpen,
  onClose,
  line,
  onUpdateLine,
  onDeleteLine
}) => {
  const [comment, setComment] = useState<string>('');
  const [course, setCourse] = useState<number>(1);

  useEffect(() => {
    if (isOpen && line) {
      setComment(line.comment || '');
      setCourse(line.course || 1);
    }
  }, [isOpen, line]);

  if (!line) return null;

  const handleSubmit = () => {
    onUpdateLine(comment, course);
    onClose();
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
            <span>Удалить позицию</span>
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
              Исполнить
            </PosButton>
          </div>
        </div>
      }
    >
      <div className="flex flex-col gap-5 select-none">
        
        {/* Selected dish descriptor */}
        <div className="p-4 bg-[var(--pos-surface-raised)] border border-[var(--pos-border)]">
          <span className="font-mono text-[10px] text-[var(--pos-text-muted)] uppercase tracking-wider">Позиция заказа</span>
          <h4 className="font-sans text-sm md:text-base font-bold text-[var(--pos-text-primary)] mt-1">{line.name}</h4>
          <span className="font-mono text-xs text-[var(--pos-text-secondary)] font-semibold mt-0.5">Цена: {line.price} ₽ × {line.quantity} шт.</span>
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

      </div>
    </PosDialog>
  );
};

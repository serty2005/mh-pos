import { FileText } from 'lucide-react';
import type { ReceiptTemplate, ReceiptTemplateDocumentType } from '../../shared/api/schemas';
import { useI18n } from '../../shared/i18n/I18nProvider';
import { DOCUMENT_TYPES } from './receiptTemplateForms';

type Props = {
  templates: ReceiptTemplate[];
  selectedId: string | null;
  filterDocType: ReceiptTemplateDocumentType | '';
  onSelect: (template: ReceiptTemplate) => void;
  onNew: () => void;
  onFilterChange: (docType: ReceiptTemplateDocumentType | '') => void;
};

export default function TemplateListPanel({
  templates,
  selectedId,
  filterDocType,
  onSelect,
  onNew,
  onFilterChange,
}: Props) {
  const { t } = useI18n();

  const filtered = filterDocType
    ? templates.filter((tpl) => tpl.document_type === filterDocType)
    : templates;

  return (
    <div className="flex h-full flex-col">
      <div className="flex items-center justify-between border-b border-slate-200 px-4 py-3">
        <h2 className="text-sm font-semibold text-slate-800">{t('receiptTemplates.listTitle')}</h2>
        <button
          type="button"
          onClick={onNew}
          className="rounded-lg bg-slate-900 px-3 py-1.5 text-xs font-medium text-white hover:bg-slate-700"
        >
          {t('receiptTemplates.newTemplate')}
        </button>
      </div>

      <div className="border-b border-slate-200 px-4 py-2">
        <select
          value={filterDocType}
          onChange={(e) => onFilterChange(e.target.value as ReceiptTemplateDocumentType | '')}
          className="w-full rounded-lg border border-slate-300 bg-white px-2 py-1.5 text-xs text-slate-700"
        >
          <option value="">— все типы —</option>
          {DOCUMENT_TYPES.map((dt) => (
            <option key={dt} value={dt}>
              {t(`receiptTemplates.documentTypes.${dt}`)}
            </option>
          ))}
        </select>
      </div>

      <ul className="min-h-0 flex-1 overflow-y-auto divide-y divide-slate-100">
        {filtered.length === 0 ? (
          <li className="px-4 py-6 text-center text-sm text-slate-400">{t('receiptTemplates.empty')}</li>
        ) : (
          filtered.map((tpl) => {
            const isSelected = tpl.id === selectedId;
            return (
              <li key={tpl.id}>
                <button
                  type="button"
                  onClick={() => onSelect(tpl)}
                  className={[
                    'flex w-full items-start gap-3 px-4 py-3 text-left transition-colors',
                    isSelected
                      ? 'bg-blue-50 border-l-4 border-blue-500'
                      : 'hover:bg-slate-50 border-l-4 border-transparent',
                  ].join(' ')}
                >
                  <FileText className="mt-0.5 h-4 w-4 shrink-0 text-slate-400" />
                  <div className="min-w-0">
                    <p className="truncate text-sm font-medium text-slate-800">{tpl.name}</p>
                    <p className="mt-0.5 text-xs text-slate-500">
                      {t(`receiptTemplates.documentTypes.${tpl.document_type}`)}
                      {' · '}
                      {tpl.cpl} CPL
                      {tpl.is_default ? (
                        <span className="ml-1.5 rounded bg-blue-100 px-1 py-0.5 text-[10px] font-semibold text-blue-700">
                          {t('receiptTemplates.defaultBadge')}
                        </span>
                      ) : null}
                      {!tpl.restaurant_id ? (
                        <span className="ml-1.5 rounded bg-slate-100 px-1 py-0.5 text-[10px] font-semibold text-slate-500">
                          {t('receiptTemplates.tenantBadge')}
                        </span>
                      ) : (
                        <span className="ml-1.5 rounded bg-green-100 px-1 py-0.5 text-[10px] font-semibold text-green-700">
                          {t('receiptTemplates.restaurantBadge')}
                        </span>
                      )}
                    </p>
                  </div>
                </button>
              </li>
            );
          })
        )}
      </ul>
    </div>
  );
}

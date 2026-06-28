import { useEffect, useState } from 'react';
import { FileText } from 'lucide-react';
import { listReceiptTemplates } from '../../shared/api/endpoints';
import type { ReceiptTemplate, ReceiptTemplateDocumentType } from '../../shared/api/schemas';
import { useI18n } from '../../shared/i18n/I18nProvider';
import SafeErrorBanner from '../../shared/ui/SafeErrorBanner';
import TemplateListPanel from './TemplateListPanel';
import TemplateEditorPanel from './TemplateEditorPanel';

type RouteStatus = 'loading' | 'ready' | 'blocked';

export default function ReceiptTemplatesPage() {
  const { t } = useI18n();
  const [templates, setTemplates] = useState<ReceiptTemplate[]>([]);
  const [status, setStatus] = useState<RouteStatus>('loading');
  const [error, setError] = useState<unknown>(null);
  const [selectedTemplate, setSelectedTemplate] = useState<ReceiptTemplate | null>(null);
  const [isNew, setIsNew] = useState(false);
  const [filterDocType, setFilterDocType] = useState<ReceiptTemplateDocumentType | ''>('');

  const reload = async () => {
    setStatus('loading');
    setError(null);
    try {
      const list = await listReceiptTemplates();
      setTemplates(list);
      setStatus('ready');
    } catch (err) {
      setError(err);
      setStatus('blocked');
    }
  };

  useEffect(() => { void reload(); }, []);

  const handleSelect = (tpl: ReceiptTemplate) => {
    setIsNew(false);
    setSelectedTemplate(tpl);
  };

  const handleNew = () => {
    setIsNew(true);
    setSelectedTemplate(null);
  };

  const handleSaved = (saved: ReceiptTemplate) => {
    setTemplates((prev) => {
      const idx = prev.findIndex((t) => t.id === saved.id);
      if (idx >= 0) {
        const next = [...prev];
        next[idx] = saved;
        return next;
      }
      return [saved, ...prev];
    });
    setSelectedTemplate(saved);
    setIsNew(false);
  };

  const handleDeactivated = (updated: ReceiptTemplate) => {
    setTemplates((prev) => prev.map((t) => (t.id === updated.id ? updated : t)));
    setSelectedTemplate(updated);
  };

  const editorVisible = isNew || selectedTemplate !== null;

  return (
    <section className="flex flex-col gap-4">
      <div className="flex items-center gap-3">
        <div className="flex h-9 w-9 items-center justify-center rounded-xl bg-slate-900 text-white">
          <FileText className="h-4 w-4" />
        </div>
        <div>
          <h1 className="text-base font-semibold text-slate-900">{t('receiptTemplates.pageTitle')}</h1>
          <p className="text-xs text-slate-500">{t('receiptTemplates.pageDescription')}</p>
        </div>
      </div>

      {error ? <SafeErrorBanner error={error} /> : null}

      {status === 'loading' ? (
        <div className="flex items-center gap-2 rounded-xl border border-slate-200 bg-white p-6">
          <span className="h-4 w-4 animate-spin rounded-full border-2 border-slate-300 border-t-slate-700" />
          <span className="text-sm text-slate-500">{t('status.loading')}</span>
        </div>
      ) : (
        <div className="flex h-[calc(100vh-12rem)] overflow-hidden rounded-2xl border border-slate-200 bg-white shadow-sm">
          <div className="w-64 shrink-0 border-r border-slate-200 overflow-hidden">
            <TemplateListPanel
              templates={templates}
              selectedId={selectedTemplate?.id ?? null}
              filterDocType={filterDocType}
              onSelect={handleSelect}
              onNew={handleNew}
              onFilterChange={setFilterDocType}
            />
          </div>
          <div className="min-w-0 flex-1 overflow-hidden">
            {editorVisible ? (
              <TemplateEditorPanel
                template={isNew ? null : selectedTemplate}
                onSaved={handleSaved}
                onDeactivated={handleDeactivated}
              />
            ) : (
              <div className="flex h-full items-center justify-center">
                <p className="text-sm text-slate-400">{t('receiptTemplates.selectPrompt')}</p>
              </div>
            )}
          </div>
        </div>
      )}
    </section>
  );
}

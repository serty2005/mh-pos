import { useEffect, useRef, useState } from 'react';
import { RefreshCw } from 'lucide-react';
import { previewReceiptTemplate, createReceiptTemplate, updateReceiptTemplate, deactivateReceiptTemplate } from '../../shared/api/endpoints';
import type { ReceiptTemplate } from '../../shared/api/schemas';
import { useI18n } from '../../shared/i18n/I18nProvider';
import SafeErrorBanner from '../../shared/ui/SafeErrorBanner';
import {
  buildCreateTemplatePayload,
  buildUpdateTemplatePayload,
  CPL_OPTIONS,
  defaultTemplateFormValues,
  DOCUMENT_TYPES,
  toTemplateFormValues,
  validateTemplateForm,
  type ReceiptTemplateFormValues,
} from './receiptTemplateForms';

type Props = {
  template: ReceiptTemplate | null;
  onSaved: (template: ReceiptTemplate) => void;
  onDeactivated: (template: ReceiptTemplate) => void;
};

const PREVIEW_DEBOUNCE_MS = 600;

export default function TemplateEditorPanel({ template, onSaved, onDeactivated }: Props) {
  const { t } = useI18n();
  const [values, setValues] = useState<ReceiptTemplateFormValues>(defaultTemplateFormValues);
  const [previewSvg, setPreviewSvg] = useState<string>('');
  const [previewLoading, setPreviewLoading] = useState(false);
  const [previewError, setPreviewError] = useState<unknown>(null);
  const [saving, setSaving] = useState(false);
  const [saveError, setSaveError] = useState<unknown>(null);
  const [savedFlash, setSavedFlash] = useState(false);
  const debounceRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const isNew = template === null;

  useEffect(() => {
    if (template) {
      setValues(toTemplateFormValues(template));
    } else {
      setValues(defaultTemplateFormValues);
    }
    setPreviewSvg('');
    setPreviewError(null);
    setSaveError(null);
  }, [template]);

  useEffect(() => {
    if (!values.content.trim()) {
      setPreviewSvg('');
      return;
    }
    if (debounceRef.current) clearTimeout(debounceRef.current);
    debounceRef.current = setTimeout(() => {
      setPreviewLoading(true);
      setPreviewError(null);
      previewReceiptTemplate({
        content: values.content,
        document_type: values.document_type,
        cpl: values.cpl,
      })
        .then((svg) => { setPreviewSvg(svg); setPreviewLoading(false); })
        .catch((err) => { setPreviewError(err); setPreviewLoading(false); });
    }, PREVIEW_DEBOUNCE_MS);

    return () => { if (debounceRef.current) clearTimeout(debounceRef.current); };
  }, [values.content, values.document_type, values.cpl]);

  const set = <K extends keyof ReceiptTemplateFormValues>(key: K, val: ReceiptTemplateFormValues[K]) =>
    setValues((prev) => ({ ...prev, [key]: val }));

  const fieldErrors = validateTemplateForm(values);
  const canSave = fieldErrors.length === 0 && !saving;

  const handleSave = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!canSave) return;
    setSaving(true);
    setSaveError(null);
    try {
      const saved = isNew
        ? await createReceiptTemplate(buildCreateTemplatePayload(values))
        : await updateReceiptTemplate(template.id, buildUpdateTemplatePayload(values));
      onSaved(saved);
      setSavedFlash(true);
      setTimeout(() => setSavedFlash(false), 2000);
    } catch (err) {
      setSaveError(err);
    } finally {
      setSaving(false);
    }
  };

  const handleDeactivate = async () => {
    if (isNew || !template) return;
    setSaving(true);
    setSaveError(null);
    try {
      const updated = await deactivateReceiptTemplate(template.id);
      onDeactivated(updated);
    } catch (err) {
      setSaveError(err);
    } finally {
      setSaving(false);
    }
  };

  return (
    <div className="flex h-full flex-col overflow-hidden">
      <div className="border-b border-slate-200 px-4 py-3">
        <h2 className="text-sm font-semibold text-slate-800">
          {isNew ? t('receiptTemplates.newTemplate') : t('receiptTemplates.editorTitle')}
        </h2>
      </div>

      <div className="flex min-h-0 flex-1 gap-0 overflow-hidden">
        {/* Form */}
        <form
          className="flex w-[420px] shrink-0 flex-col gap-4 overflow-y-auto border-r border-slate-200 p-4"
          onSubmit={(e) => { void handleSave(e); }}
        >
          <div>
            <label className="mb-1 block text-xs font-medium text-slate-700">
              {t('receiptTemplates.form.name')}
            </label>
            <input
              value={values.name}
              onChange={(e) => set('name', e.target.value)}
              className="w-full rounded-lg border border-slate-300 bg-white px-3 py-2 text-sm"
              disabled={saving}
              required
            />
          </div>

          <div>
            <label className="mb-1 block text-xs font-medium text-slate-700">
              {t('receiptTemplates.form.description')}
            </label>
            <input
              value={values.description}
              onChange={(e) => set('description', e.target.value)}
              className="w-full rounded-lg border border-slate-300 bg-white px-3 py-2 text-sm"
              disabled={saving}
            />
          </div>

          <div className="grid grid-cols-2 gap-3">
            <div>
              <label className="mb-1 block text-xs font-medium text-slate-700">
                {t('receiptTemplates.form.documentType')}
              </label>
              <select
                value={values.document_type}
                onChange={(e) => set('document_type', e.target.value as ReceiptTemplateFormValues['document_type'])}
                className="w-full rounded-lg border border-slate-300 bg-white px-2 py-2 text-sm"
                disabled={saving}
              >
                {DOCUMENT_TYPES.map((dt) => (
                  <option key={dt} value={dt}>
                    {t(`receiptTemplates.documentTypes.${dt}`)}
                  </option>
                ))}
              </select>
            </div>

            <div>
              <label className="mb-1 block text-xs font-medium text-slate-700">
                {t('receiptTemplates.form.cpl')}
              </label>
              <select
                value={values.cpl}
                onChange={(e) => set('cpl', Number(e.target.value))}
                className="w-full rounded-lg border border-slate-300 bg-white px-2 py-2 text-sm"
                disabled={saving}
              >
                {CPL_OPTIONS.map((cpl) => (
                  <option key={cpl} value={cpl}>{cpl}</option>
                ))}
              </select>
            </div>
          </div>

          <div>
            <label className="mb-1 block text-xs font-medium text-slate-700">
              {t('receiptTemplates.form.content')}
            </label>
            <textarea
              value={values.content}
              onChange={(e) => set('content', e.target.value)}
              rows={18}
              className="w-full rounded-lg border border-slate-300 bg-white px-3 py-2 font-mono text-xs"
              disabled={saving}
              spellCheck={false}
            />
          </div>

          {saveError ? <SafeErrorBanner error={saveError} /> : null}
          {savedFlash ? (
            <p className="text-xs font-medium text-green-600">{t('receiptTemplates.saved')}</p>
          ) : null}

          <div className="flex flex-wrap gap-2">
            <button
              type="submit"
              disabled={!canSave}
              className="rounded-lg bg-slate-900 px-4 py-2 text-sm font-medium text-white disabled:opacity-50"
            >
              {saving ? '...' : t('receiptTemplates.saveTemplate')}
            </button>
            {!isNew && template?.is_active ? (
              <button
                type="button"
                onClick={() => { void handleDeactivate(); }}
                disabled={saving}
                className="rounded-lg border border-red-300 px-4 py-2 text-sm font-medium text-red-600 hover:bg-red-50 disabled:opacity-50"
              >
                {t('receiptTemplates.deactivate')}
              </button>
            ) : null}
          </div>
        </form>

        {/* SVG Preview */}
        <div className="flex min-w-0 flex-1 flex-col overflow-hidden">
          <div className="flex items-center justify-between border-b border-slate-200 px-4 py-2">
            <h3 className="text-xs font-semibold uppercase tracking-wider text-slate-500">
              {t('receiptTemplates.previewTitle')}
            </h3>
            {previewLoading ? (
              <RefreshCw className="h-3.5 w-3.5 animate-spin text-slate-400" />
            ) : null}
          </div>
          <div className="flex min-h-0 flex-1 items-start justify-center overflow-y-auto bg-slate-100 p-6">
            {previewError ? (
              <div className="w-full max-w-sm rounded-xl border border-red-200 bg-red-50 p-4">
                <p className="text-xs font-medium text-red-700">{t('receiptTemplates.previewError')}</p>
                <p className="mt-1 font-mono text-[10px] text-red-500">
                  {previewError instanceof Error ? previewError.message : String(previewError)}
                </p>
              </div>
            ) : previewSvg ? (
              <div
                className="rounded-lg bg-white shadow-md"
                dangerouslySetInnerHTML={{ __html: previewSvg }}
              />
            ) : (
              <p className="text-sm text-slate-400">{t('receiptTemplates.previewEmpty')}</p>
            )}
          </div>
        </div>
      </div>
    </div>
  );
}

import { useEffect, useState } from 'react';
import { Printer } from 'lucide-react';
import { listPrinters, createPrinter, updatePrinter, deactivatePrinter } from '../../shared/api/endpoints';
import type { Printer as PrinterModel } from '../../shared/api/schemas';
import { useI18n } from '../../shared/i18n/I18nProvider';
import SafeErrorBanner from '../../shared/ui/SafeErrorBanner';
import {
  toPrinterFormValues,
  buildCreatePrinterPayload,
  buildUpdatePrinterPayload,
  validatePrinterForm,
  defaultPrinterFormValues,
  PRINTER_DOCUMENT_TYPES,
  PRINTER_CPL_OPTIONS,
  type PrinterFormValues,
} from './printerForms';

type PageStatus = 'loading' | 'ready' | 'blocked';

type PrintersPageProps = {
  restaurantId: string;
};

export default function PrintersPage({ restaurantId }: PrintersPageProps) {
  const { t } = useI18n();
  const [printers, setPrinters] = useState<PrinterModel[]>([]);
  const [status, setStatus] = useState<PageStatus>('loading');
  const [error, setError] = useState<unknown>(null);
  const [selected, setSelected] = useState<PrinterModel | null>(null);
  const [isNew, setIsNew] = useState(false);
  const [form, setForm] = useState<PrinterFormValues>({ ...defaultPrinterFormValues, restaurant_id: restaurantId });
  const [saving, setSaving] = useState(false);
  const [saveError, setSaveError] = useState<unknown>(null);
  const [fieldErrors, setFieldErrors] = useState<string[]>([]);
  const [successMsg, setSuccessMsg] = useState('');

  const reload = async () => {
    setStatus('loading');
    setError(null);
    try {
      const list = await listPrinters(restaurantId);
      setPrinters(list);
      setStatus('ready');
    } catch (err) {
      setError(err);
      setStatus('blocked');
    }
  };

  useEffect(() => {
    void reload();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [restaurantId]);

  const handleSelect = (p: PrinterModel) => {
    setIsNew(false);
    setSelected(p);
    setForm(toPrinterFormValues(p));
    setSaveError(null);
    setFieldErrors([]);
    setSuccessMsg('');
  };

  const handleNew = () => {
    setIsNew(true);
    setSelected(null);
    setForm({ ...defaultPrinterFormValues, restaurant_id: restaurantId });
    setSaveError(null);
    setFieldErrors([]);
    setSuccessMsg('');
  };

  const handleSave = async () => {
    const errors = validatePrinterForm(form);
    if (errors.length > 0) {
      setFieldErrors(errors);
      return;
    }
    setFieldErrors([]);
    setSaving(true);
    setSaveError(null);
    setSuccessMsg('');
    try {
      let saved: PrinterModel;
      if (isNew) {
        saved = await createPrinter(buildCreatePrinterPayload(form));
        setPrinters((prev) => [saved, ...prev]);
      } else if (selected) {
        saved = await updatePrinter(selected.id, buildUpdatePrinterPayload(form));
        setPrinters((prev) => prev.map((p) => (p.id === saved.id ? saved : p)));
      } else {
        return;
      }
      setSelected(saved);
      setIsNew(false);
      setForm(toPrinterFormValues(saved));
      setSuccessMsg(t('printers.saved'));
    } catch (err) {
      setSaveError(err);
    } finally {
      setSaving(false);
    }
  };

  const handleDeactivate = async () => {
    if (!selected) return;
    setSaving(true);
    setSaveError(null);
    setSuccessMsg('');
    try {
      const updated = await deactivatePrinter(selected.id);
      setPrinters((prev) => prev.map((p) => (p.id === updated.id ? updated : p)));
      setSelected(updated);
      setForm(toPrinterFormValues(updated));
      setSuccessMsg(t('printers.deactivated'));
    } catch (err) {
      setSaveError(err);
    } finally {
      setSaving(false);
    }
  };

  const updateField = <K extends keyof PrinterFormValues>(key: K, value: PrinterFormValues[K]) => {
    setForm((prev) => ({ ...prev, [key]: value }));
    setFieldErrors((prev) => prev.filter((e) => e !== key));
    setSuccessMsg('');
  };

  const toggleDocType = (dt: PrinterFormValues['document_types'][number]) => {
    setForm((prev) => {
      const has = prev.document_types.includes(dt);
      return {
        ...prev,
        document_types: has
          ? prev.document_types.filter((d) => d !== dt)
          : [...prev.document_types, dt],
      };
    });
    setFieldErrors((prev) => prev.filter((e) => e !== 'document_types'));
    setSuccessMsg('');
  };

  const editorVisible = isNew || selected !== null;

  return (
    <section className="flex flex-col gap-4">
      <div className="flex items-center gap-3">
        <div className="flex h-9 w-9 items-center justify-center rounded-xl bg-slate-900 text-white">
          <Printer className="h-4 w-4" />
        </div>
        <div>
          <h1 className="text-base font-semibold text-slate-900">{t('printers.pageTitle')}</h1>
          <p className="text-xs text-slate-500">{t('printers.pageDescription')}</p>
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
          {/* Список принтеров */}
          <div className="flex w-64 shrink-0 flex-col border-r border-slate-200 overflow-hidden">
            <div className="flex items-center justify-between border-b border-slate-100 px-4 py-3">
              <span className="text-xs font-semibold uppercase tracking-wider text-slate-500">
                {t('printers.listTitle')}
              </span>
              <button
                type="button"
                onClick={handleNew}
                className="rounded-lg bg-slate-900 px-2.5 py-1 text-xs font-semibold text-white hover:bg-slate-700"
              >
                + {t('printers.newPrinter')}
              </button>
            </div>
            <div className="min-h-0 flex-1 overflow-y-auto">
              {printers.length === 0 ? (
                <p className="px-4 py-6 text-center text-sm text-slate-400">{t('printers.empty')}</p>
              ) : (
                <ul className="py-1">
                  {printers.map((p) => {
                    const isActive = p.id === selected?.id;
                    return (
                      <li key={p.id}>
                        <button
                          type="button"
                          onClick={() => handleSelect(p)}
                          className={[
                            'flex w-full flex-col gap-0.5 border-l-4 px-4 py-3 text-left transition-all',
                            isActive
                              ? 'border-blue-500 bg-blue-50'
                              : 'border-transparent hover:bg-slate-50',
                            !p.is_active ? 'opacity-50' : '',
                          ].join(' ')}
                        >
                          <span className="truncate text-sm font-medium text-slate-900">{p.name}</span>
                          <span className="text-xs text-slate-500">
                            {p.type.toUpperCase()} {p.type === 'tcp' && p.address ? `· ${p.address}:${p.port ?? ''}` : ''}
                          </span>
                          <span className={['text-[10px] font-semibold uppercase tracking-wide', p.is_active ? 'text-green-600' : 'text-slate-400'].join(' ')}>
                            {p.is_active ? t('printers.activeBadge') : t('printers.inactiveBadge')}
                          </span>
                        </button>
                      </li>
                    );
                  })}
                </ul>
              )}
            </div>
          </div>

          {/* Редактор */}
          <div className="min-w-0 flex-1 overflow-y-auto">
            {editorVisible ? (
              <div className="flex flex-col gap-4 p-6">
                <div className="flex items-center justify-between">
                  <h2 className="text-sm font-semibold text-slate-700">
                    {isNew ? t('printers.newPrinter') : t('printers.editorTitle')}
                  </h2>
                  {!isNew && selected && selected.is_active ? (
                    <button
                      type="button"
                      disabled={saving}
                      onClick={() => void handleDeactivate()}
                      className="rounded-lg border border-red-300 px-3 py-1.5 text-xs font-medium text-red-600 hover:bg-red-50 disabled:opacity-50"
                    >
                      {t('printers.deactivate')}
                    </button>
                  ) : null}
                </div>

                {saveError ? <SafeErrorBanner error={saveError} /> : null}
                {successMsg ? (
                  <div className="rounded-lg border border-green-200 bg-green-50 px-4 py-2 text-sm text-green-700">
                    {successMsg}
                  </div>
                ) : null}

                <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
                  {/* Название */}
                  <div className="sm:col-span-2">
                    <label className="mb-1 block text-xs font-semibold text-slate-600">
                      {t('printers.form.name')}
                    </label>
                    <input
                      type="text"
                      value={form.name}
                      onChange={(e) => updateField('name', e.target.value)}
                      className={[
                        'w-full rounded-lg border px-3 py-2 text-sm outline-none transition-colors focus:border-blue-500',
                        fieldErrors.includes('name') ? 'border-red-400' : 'border-slate-300',
                      ].join(' ')}
                    />
                  </div>

                  {/* Тип */}
                  <div>
                    <label className="mb-1 block text-xs font-semibold text-slate-600">
                      {t('printers.form.type')}
                    </label>
                    <select
                      value={form.type}
                      onChange={(e) => updateField('type', e.target.value as 'tcp' | 'usb')}
                      className="w-full rounded-lg border border-slate-300 px-3 py-2 text-sm outline-none focus:border-blue-500"
                    >
                      <option value="tcp">{t('printers.types.tcp')}</option>
                      <option value="usb">{t('printers.types.usb')}</option>
                    </select>
                  </div>

                  {/* CPL */}
                  <div>
                    <label className="mb-1 block text-xs font-semibold text-slate-600">
                      {t('printers.form.cpl')}
                    </label>
                    <select
                      value={form.cpl}
                      onChange={(e) => updateField('cpl', Number(e.target.value))}
                      className={[
                        'w-full rounded-lg border px-3 py-2 text-sm outline-none focus:border-blue-500',
                        fieldErrors.includes('cpl') ? 'border-red-400' : 'border-slate-300',
                      ].join(' ')}
                    >
                      {PRINTER_CPL_OPTIONS.map((v) => (
                        <option key={v} value={v}>{v}</option>
                      ))}
                    </select>
                  </div>

                  {/* Адрес (только TCP) */}
                  {form.type === 'tcp' ? (
                    <>
                      <div>
                        <label className="mb-1 block text-xs font-semibold text-slate-600">
                          {t('printers.form.address')}
                        </label>
                        <input
                          type="text"
                          value={form.address}
                          onChange={(e) => updateField('address', e.target.value)}
                          placeholder="10.25.1.201"
                          className={[
                            'w-full rounded-lg border px-3 py-2 text-sm outline-none focus:border-blue-500',
                            fieldErrors.includes('address') ? 'border-red-400' : 'border-slate-300',
                          ].join(' ')}
                        />
                      </div>
                      <div>
                        <label className="mb-1 block text-xs font-semibold text-slate-600">
                          {t('printers.form.port')}
                        </label>
                        <input
                          type="number"
                          value={form.port}
                          onChange={(e) => updateField('port', e.target.value)}
                          placeholder="9100"
                          className="w-full rounded-lg border border-slate-300 px-3 py-2 text-sm outline-none focus:border-blue-500"
                        />
                      </div>
                    </>
                  ) : null}

                  {/* Кодировка */}
                  <div>
                    <label className="mb-1 block text-xs font-semibold text-slate-600">
                      {t('printers.form.codepage')}
                    </label>
                    <select
                      value={form.codepage}
                      onChange={(e) => updateField('codepage', e.target.value as PrinterFormValues['codepage'])}
                      className="w-full rounded-lg border border-slate-300 px-3 py-2 text-sm outline-none focus:border-blue-500"
                    >
                      <option value="">{t('printers.codepages.default')}</option>
                      <option value="cp437">{t('printers.codepages.cp437')}</option>
                      <option value="cp866">{t('printers.codepages.cp866')}</option>
                    </select>
                  </div>

                  {/* Тип отреза */}
                  <div>
                    <label className="mb-1 block text-xs font-semibold text-slate-600">
                      {t('printers.form.paperCutType')}
                    </label>
                    <select
                      value={form.paper_cut_type}
                      onChange={(e) => updateField('paper_cut_type', e.target.value as 'partial' | 'full')}
                      className="w-full rounded-lg border border-slate-300 px-3 py-2 text-sm outline-none focus:border-blue-500"
                    >
                      <option value="partial">{t('printers.paperCuts.partial')}</option>
                      <option value="full">{t('printers.paperCuts.full')}</option>
                    </select>
                  </div>

                  {/* Типы документов */}
                  <div className="sm:col-span-2">
                    <label className={[
                      'mb-2 block text-xs font-semibold',
                      fieldErrors.includes('document_types') ? 'text-red-600' : 'text-slate-600',
                    ].join(' ')}>
                      {t('printers.form.documentTypes')}
                    </label>
                    <div className="flex flex-wrap gap-2">
                      {PRINTER_DOCUMENT_TYPES.map((dt) => {
                        const checked = form.document_types.includes(dt);
                        return (
                          <button
                            key={dt}
                            type="button"
                            onClick={() => toggleDocType(dt)}
                            className={[
                              'rounded-lg border px-3 py-1.5 text-xs font-medium transition-colors',
                              checked
                                ? 'border-blue-500 bg-blue-50 text-blue-700'
                                : 'border-slate-300 text-slate-600 hover:border-slate-400',
                            ].join(' ')}
                          >
                            {t(`printers.documentTypes.${dt}`)}
                          </button>
                        );
                      })}
                    </div>
                  </div>
                </div>

                <div className="flex gap-2 pt-2">
                  <button
                    type="button"
                    disabled={saving}
                    onClick={() => void handleSave()}
                    className="rounded-lg bg-slate-900 px-4 py-2 text-sm font-semibold text-white hover:bg-slate-700 disabled:opacity-50"
                  >
                    {saving ? t('ui.loading') : t('printers.savePrinter')}
                  </button>
                  <button
                    type="button"
                    onClick={() => {
                      setIsNew(false);
                      setSelected(null);
                      setSaveError(null);
                      setFieldErrors([]);
                      setSuccessMsg('');
                    }}
                    className="rounded-lg border border-slate-300 px-4 py-2 text-sm font-medium text-slate-600 hover:bg-slate-50"
                  >
                    {t('catalog.shared.cancel')}
                  </button>
                </div>
              </div>
            ) : (
              <div className="flex h-full items-center justify-center">
                <p className="text-sm text-slate-400">{t('printers.selectPrompt')}</p>
              </div>
            )}
          </div>
        </div>
      )}
    </section>
  );
}

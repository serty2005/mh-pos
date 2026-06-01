import { CheckCircle2, Send } from 'lucide-react';

import { t } from '../../shared/i18n';
import { PosButton, PosFormRow } from '../../shared/ui';
import {
  catalogKindLabel,
  catalogKinds,
  type CatalogSuggestionState,
} from './kitchenHelpers';

export function KitchenCatalogSuggestionForm({
  form,
  busy,
  onChange,
  onSubmit,
}: {
  form: CatalogSuggestionState;
  busy: boolean;
  onChange: (patch: Partial<CatalogSuggestionState>) => void;
  onSubmit: () => void;
}) {
  return (
    <form className="border border-[var(--pos-border)] bg-[var(--pos-surface)] p-4 grid gap-1 md:grid-cols-2 md:gap-x-4" onSubmit={(event) => { event.preventDefault(); onSubmit(); }}>
      <div className="md:col-span-2 flex items-center gap-2 mb-2">
        <CheckCircle2 className="w-5 h-5 text-[var(--pos-status-success)]" />
        <h3 className="font-mono text-sm font-black uppercase tracking-widest">{t.kitchen.suggestCatalogItem}</h3>
      </div>
      <PosFormRow id="catalog-kind" label={t.kitchen.suggestionKind}>
        <select id="catalog-kind" className={inputClassName} value={form.kind} onChange={(event) => onChange({ kind: event.target.value as CatalogSuggestionState['kind'] })}>
          {catalogKinds.filter((kind) => kind !== 'all').map((kind) => <option key={kind} value={kind}>{catalogKindLabel(kind)}</option>)}
        </select>
      </PosFormRow>
      <PosFormRow id="catalog-name" label={t.kitchen.name}>
        <input id="catalog-name" className={inputClassName} value={form.name} onChange={(event) => onChange({ name: event.target.value })} />
      </PosFormRow>
      <PosFormRow id="catalog-sku" label={t.kitchen.sku}>
        <input id="catalog-sku" className={inputClassName} value={form.sku} onChange={(event) => onChange({ sku: event.target.value })} />
      </PosFormRow>
      <PosFormRow id="catalog-base-unit" label={t.kitchen.baseUnit}>
        <input id="catalog-base-unit" className={inputClassName} value={form.baseUnit} onChange={(event) => onChange({ baseUnit: event.target.value })} />
      </PosFormRow>
      <PosFormRow id="catalog-kitchen-type" label={t.kitchen.kitchenType}>
        <input id="catalog-kitchen-type" className={inputClassName} value={form.kitchenType} onChange={(event) => onChange({ kitchenType: event.target.value })} />
      </PosFormRow>
      <PosFormRow id="catalog-accounting-category" label={t.kitchen.accountingCategory}>
        <input id="catalog-accounting-category" className={inputClassName} value={form.accountingCategory} onChange={(event) => onChange({ accountingCategory: event.target.value })} />
      </PosFormRow>
      <div className="md:col-span-2">
        <PosFormRow id="catalog-reason" label={t.kitchen.reason}>
          <textarea id="catalog-reason" rows={4} className={textareaClassName} value={form.reason} onChange={(event) => onChange({ reason: event.target.value })} />
        </PosFormRow>
      </div>
      <div className="md:col-span-2">
        <PosButton fullWidth variant="primary" disabled={busy} icon={<Send className="w-4 h-4" />}>{t.kitchen.submit}</PosButton>
      </div>
    </form>
  );
}

const inputClassName = 'h-12 w-full border border-[var(--pos-border)] bg-[var(--pos-surface-raised)] px-3 text-sm text-[var(--pos-text-primary)] outline-none focus:ring-2 focus:ring-[var(--pos-focus-ring)]';
const textareaClassName = 'w-full border border-[var(--pos-border)] bg-[var(--pos-surface-raised)] px-3 py-2 text-sm text-[var(--pos-text-primary)] outline-none focus:ring-2 focus:ring-[var(--pos-focus-ring)] resize-none';

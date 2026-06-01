import { PackageCheck, Search } from 'lucide-react';

import { t } from '../../shared/i18n';
import { PosButton, PosEmptyState, PosFormRow, PosSearchInput, PosSelectableTile, PosTabs } from '../../shared/ui';
import type { BackendCatalogItem } from '../../shared/schemas';
import {
  catalogKind,
  catalogKindLabel,
  catalogKinds,
  type CatalogKindFilter,
  type StockFormState,
} from './kitchenHelpers';

export type StockTab = 'receipt' | 'count' | 'writeoff' | 'production';

export function KitchenStockTab({
  stockTab,
  filteredCatalog,
  selectedItem,
  stockForm,
  catalogSearch,
  catalogFilter,
  busy,
  onTabChange,
  onCatalogSearch,
  onCatalogFilter,
  onSelectItem,
  onFormChange,
  onSubmit,
}: {
  stockTab: StockTab;
  filteredCatalog: BackendCatalogItem[];
  selectedItem?: BackendCatalogItem;
  stockForm: StockFormState;
  catalogSearch: string;
  catalogFilter: CatalogKindFilter;
  busy: boolean;
  onTabChange: (tab: StockTab) => void;
  onCatalogSearch: (value: string) => void;
  onCatalogFilter: (value: CatalogKindFilter) => void;
  onSelectItem: (value: string) => void;
  onFormChange: (patch: Partial<StockFormState>) => void;
  onSubmit: () => void;
}) {
  const tabCatalog = filteredCatalog.filter((item) => (
    stockTab === 'production' ? catalogKind(item) === 'semi_finished' : catalogKind(item) !== 'service'
  ));

  return (
    <div className="flex-1 min-h-0 overflow-hidden flex flex-col">
      <PosTabs
        id="stock-tabs"
        activeId={stockTab}
        onChange={(id) => onTabChange(id as StockTab)}
        items={[
          { id: 'receipt', label: t.kitchen.tabReceipt },
          { id: 'count', label: t.kitchen.tabCount },
          { id: 'writeoff', label: t.kitchen.tabWriteOff },
          { id: 'production', label: t.kitchen.tabProduction },
        ]}
      />
      <div className="flex-1 min-h-0 grid gap-4 p-4 overflow-auto pos-scrollbar-thin xl:grid-cols-[minmax(0,1fr)_420px]">
        <CatalogPicker
          filteredItems={tabCatalog}
          selectedId={stockForm.itemId}
          search={catalogSearch}
          filter={catalogFilter}
          onSearch={onCatalogSearch}
          onFilter={onCatalogFilter}
          onSelect={onSelectItem}
        />
        <StockForm
          tab={stockTab}
          form={stockForm}
          selectedItem={selectedItem}
          busy={busy}
          onChange={onFormChange}
          onSubmit={onSubmit}
        />
      </div>
    </div>
  );
}

function CatalogPicker({
  filteredItems,
  selectedId,
  search,
  filter,
  onSearch,
  onFilter,
  onSelect,
}: {
  filteredItems: BackendCatalogItem[];
  selectedId: string;
  search: string;
  filter: CatalogKindFilter;
  onSearch: (value: string) => void;
  onFilter: (value: CatalogKindFilter) => void;
  onSelect: (value: string) => void;
}) {
  return (
    <div className="border border-[var(--pos-border)] bg-[var(--pos-surface)] min-h-[420px] flex flex-col">
      <div className="p-4 border-b border-[var(--pos-border)] grid gap-3 md:grid-cols-[minmax(0,1fr)_180px]">
        <PosSearchInput
          id="kitchen-catalog-search-input"
          value={search}
          onChange={onSearch}
          placeholder={t.kitchen.searchCatalog}
          clearLabel={t.common.clearSearch}
        />
        <select
          className="h-12 border border-[var(--pos-border)] bg-[var(--pos-surface-raised)] px-3 text-sm outline-none focus:ring-2 focus:ring-[var(--pos-focus-ring)]"
          value={filter}
          onChange={(event) => onFilter(event.target.value as CatalogKindFilter)}
        >
          {catalogKinds.map((kind) => (
            <option key={kind} value={kind}>{kind === 'all' ? t.kitchen.allKinds : catalogKindLabel(kind)}</option>
          ))}
        </select>
      </div>
      <div className="flex-1 min-h-0 overflow-auto pos-scrollbar-thin divide-y divide-[var(--pos-border)]">
        {filteredItems.length === 0 ? (
          <PosEmptyState title={t.kitchen.noCatalogItems} description={t.kitchen.searchCatalog} icon={<Search className="w-9 h-9" />} />
        ) : filteredItems.map((item) => {
          const active = selectedId === item.id;
          return (
            <PosSelectableTile
              key={item.id}
              active={active}
              className={`w-full p-3 text-left grid gap-1 cursor-pointer transition-colors ${
                active ? 'bg-[var(--pos-action-secondary)] text-[var(--pos-text-primary)]' : 'hover:bg-[var(--pos-surface-raised)]'
              }`}
              onClick={() => onSelect(item.id)}
            >
              <span className="font-sans text-sm font-semibold">{item.name}</span>
              <span className="font-mono text-[10px] uppercase tracking-wider text-[var(--pos-text-muted)]">
                {catalogKindLabel(catalogKind(item))} · {item.base_unit || t.common.none}{item.sku ? ` · ${item.sku}` : ''}
              </span>
            </PosSelectableTile>
          );
        })}
      </div>
    </div>
  );
}

function StockForm({
  tab,
  form,
  selectedItem,
  busy,
  onChange,
  onSubmit,
}: {
  tab: StockTab;
  form: StockFormState;
  selectedItem?: BackendCatalogItem;
  busy: boolean;
  onChange: (patch: Partial<StockFormState>) => void;
  onSubmit: () => void;
}) {
  const submitLabel = {
    receipt: t.kitchen.captureReceipt,
    count: t.kitchen.captureCount,
    writeoff: t.kitchen.captureWriteOff,
    production: t.kitchen.completeProduction,
  }[tab];

  return (
    <form className="border border-[var(--pos-border)] bg-[var(--pos-surface)] p-4" onSubmit={(event) => { event.preventDefault(); onSubmit(); }}>
      <div className="mb-4 border border-[var(--pos-border)] bg-[var(--pos-surface-raised)] p-3">
        <div className="font-mono text-[10px] uppercase tracking-widest text-[var(--pos-text-muted)]">{t.kitchen.selectedItem}</div>
        <div className="mt-1 font-sans text-sm font-semibold text-[var(--pos-text-primary)]">{selectedItem?.name || t.kitchen.selectCatalogItem}</div>
      </div>

      <PosFormRow id="stock-quantity" label={tab === 'count' ? t.kitchen.countedQuantity : t.kitchen.quantity}>
        <input id="stock-quantity" className={inputClassName} value={form.quantity} onChange={(event) => onChange({ quantity: event.target.value })} />
      </PosFormRow>
      <PosFormRow id="stock-unit" label={t.kitchen.unit}>
        <input id="stock-unit" className={inputClassName} value={form.unitCode} onChange={(event) => onChange({ unitCode: event.target.value })} />
      </PosFormRow>
      <PosFormRow id="stock-business-date" label={t.kitchen.businessDate}>
        <input id="stock-business-date" type="date" className={inputClassName} value={form.businessDate} onChange={(event) => onChange({ businessDate: event.target.value })} />
      </PosFormRow>

      {tab === 'receipt' && (
        <>
          <PosFormRow id="stock-supplier" label={t.kitchen.supplierName}>
            <input id="stock-supplier" className={inputClassName} value={form.supplierName} onChange={(event) => onChange({ supplierName: event.target.value })} />
          </PosFormRow>
          <PosFormRow id="stock-document-number" label={t.kitchen.documentNumber}>
            <input id="stock-document-number" className={inputClassName} value={form.documentNumber} onChange={(event) => onChange({ documentNumber: event.target.value })} />
          </PosFormRow>
          <PosFormRow id="stock-document-date" label={t.kitchen.documentDate}>
            <input id="stock-document-date" type="date" className={inputClassName} value={form.documentDate} onChange={(event) => onChange({ documentDate: event.target.value })} />
          </PosFormRow>
          <PosFormRow id="stock-unit-cost" label={t.kitchen.unitCostMinor}>
            <input id="stock-unit-cost" inputMode="numeric" className={inputClassName} value={form.unitCostMinor} onChange={(event) => onChange({ unitCostMinor: event.target.value })} />
          </PosFormRow>
          <PosFormRow id="stock-line-total" label={t.kitchen.lineTotalMinor}>
            <input id="stock-line-total" inputMode="numeric" className={inputClassName} value={form.lineTotalMinor} onChange={(event) => onChange({ lineTotalMinor: event.target.value })} />
          </PosFormRow>
        </>
      )}

      {tab === 'writeoff' && (
        <>
          <PosFormRow id="stock-reason-code" label={t.kitchen.writeOffReasonCode}>
            <input id="stock-reason-code" className={inputClassName} value={form.reasonCode} onChange={(event) => onChange({ reasonCode: event.target.value })} />
          </PosFormRow>
          <PosFormRow id="stock-reason" label={t.kitchen.writeOffReason}>
            <textarea id="stock-reason" rows={3} className={textareaClassName} value={form.reason} onChange={(event) => onChange({ reason: event.target.value })} />
          </PosFormRow>
        </>
      )}

      <PosButton fullWidth variant="primary" disabled={busy || !form.itemId} icon={<PackageCheck className="w-4 h-4" />}>
        {submitLabel}
      </PosButton>
    </form>
  );
}

const inputClassName = 'h-12 w-full border border-[var(--pos-border)] bg-[var(--pos-surface-raised)] px-3 text-sm text-[var(--pos-text-primary)] outline-none focus:ring-2 focus:ring-[var(--pos-focus-ring)]';
const textareaClassName = 'w-full border border-[var(--pos-border)] bg-[var(--pos-surface-raised)] px-3 py-2 text-sm text-[var(--pos-text-primary)] outline-none focus:ring-2 focus:ring-[var(--pos-focus-ring)] resize-none';

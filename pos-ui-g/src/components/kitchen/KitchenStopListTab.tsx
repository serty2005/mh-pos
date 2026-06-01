import { AlertTriangle, Send } from 'lucide-react';

import { t } from '../../shared/i18n';
import { PosButton, PosEmptyState, PosFormRow } from '../../shared/ui';
import type { BackendCatalogItem, BackendKitchenStopListState } from '../../shared/schemas';
import {
  stopListActionLabel,
  stopListActions,
  stopListSyncLabel,
  type StopListAction,
  type StopListFormState,
} from './kitchenHelpers';

export function KitchenStopListTab({
  catalog,
  stopList,
  selectedItem,
  form,
  busy,
  onChange,
  onSubmit,
}: {
  catalog: BackendCatalogItem[];
  stopList: BackendKitchenStopListState[];
  selectedItem?: BackendCatalogItem;
  form: StopListFormState;
  busy: boolean;
  onChange: (patch: Partial<StopListFormState>) => void;
  onSubmit: () => void;
}) {
  return (
    <div className="grid gap-4 xl:grid-cols-[minmax(0,1fr)_420px]">
      <div className="border border-[var(--pos-border)] bg-[var(--pos-surface)] min-h-[420px] flex flex-col">
        <div className="p-4 border-b border-[var(--pos-border)]">
          <h3 className="font-mono text-sm font-black uppercase tracking-widest text-[var(--pos-text-primary)]">{t.kitchen.stopListCurrent}</h3>
          <p className="mt-1 text-xs text-[var(--pos-text-secondary)]">{t.kitchen.stopListSyncCopy}</p>
        </div>
        <div className="flex-1 min-h-0 overflow-auto pos-scrollbar-thin divide-y divide-[var(--pos-border)]">
          {stopList.length === 0 ? (
            <PosEmptyState title={t.kitchen.stopListEmpty} description={t.kitchen.stopListSyncCopy} icon={<AlertTriangle className="w-10 h-10" />} />
          ) : stopList.map((entry) => {
            const catalogItem = catalog.find((item) => item.id === entry.catalog_item_id);
            return (
              <article key={entry.id} className="p-4 grid gap-3 md:grid-cols-[minmax(0,1fr)_180px]">
                <div className="min-w-0">
                  <div className="font-sans text-sm font-semibold text-[var(--pos-text-primary)] break-words">
                    {catalogItem?.name || entry.catalog_item_id}
                  </div>
                  <div className="mt-1 font-mono text-[10px] uppercase tracking-wider text-[var(--pos-text-muted)]">
                    {entry.active ? t.kitchen.stopListActive : t.kitchen.stopListInactive}
                    {typeof entry.available_quantity === 'number' ? ` · ${t.kitchen.stopListQty} ${entry.available_quantity}` : ''}
                  </div>
                  {entry.reason && <div className="mt-2 text-xs text-[var(--pos-text-secondary)] break-words">{entry.reason}</div>}
                </div>
                <SyncStatePill state={entry.sync_state} status={entry.outbox_status} attempts={entry.outbox_attempts} />
              </article>
            );
          })}
        </div>
      </div>
      <StopListForm
        catalog={catalog}
        selectedItem={selectedItem}
        form={form}
        busy={busy}
        onChange={onChange}
        onSubmit={onSubmit}
      />
    </div>
  );
}

function StopListForm({
  catalog,
  selectedItem,
  form,
  busy,
  onChange,
  onSubmit,
}: {
  catalog: BackendCatalogItem[];
  selectedItem?: BackendCatalogItem;
  form: StopListFormState;
  busy: boolean;
  onChange: (patch: Partial<StopListFormState>) => void;
  onSubmit: () => void;
}) {
  return (
    <form className="border border-[var(--pos-border)] bg-[var(--pos-surface)] p-4" onSubmit={(event) => { event.preventDefault(); onSubmit(); }}>
      <h3 className="font-mono text-sm font-black uppercase tracking-widest text-[var(--pos-text-primary)] mb-4">{t.kitchen.stopListEdit}</h3>
      <PosFormRow id="stop-list-item" label={t.kitchen.selectCatalogItem}>
        <select id="stop-list-item" className={inputClassName} value={form.itemId} onChange={(event) => onChange({ itemId: event.target.value })}>
          <option value="">{t.kitchen.selectCatalogItem}</option>
          {catalog.map((item) => <option key={item.id} value={item.id}>{item.name}</option>)}
        </select>
      </PosFormRow>
      <div className="mb-4 border border-[var(--pos-border)] bg-[var(--pos-surface-raised)] p-3">
        <div className="font-mono text-[10px] uppercase tracking-widest text-[var(--pos-text-muted)]">{t.kitchen.selectedItem}</div>
        <div className="mt-1 font-sans text-sm font-semibold text-[var(--pos-text-primary)]">{selectedItem?.name || t.kitchen.selectCatalogItem}</div>
      </div>
      <PosFormRow id="stop-list-action" label={t.kitchen.stopListAction}>
        <select id="stop-list-action" className={inputClassName} value={form.action} onChange={(event) => onChange({ action: event.target.value as StopListAction })}>
          {stopListActions.map((action) => <option key={action} value={action}>{stopListActionLabel(action)}</option>)}
        </select>
      </PosFormRow>
      <PosFormRow id="stop-list-reason" label={t.kitchen.reason}>
        <textarea id="stop-list-reason" rows={4} className={textareaClassName} value={form.reason} onChange={(event) => onChange({ reason: event.target.value })} />
      </PosFormRow>
      <PosButton fullWidth variant={form.action === 'stop' ? 'danger' : 'primary'} disabled={busy || !form.itemId} icon={<Send className="w-4 h-4" />}>{t.kitchen.submitStopList}</PosButton>
    </form>
  );
}

function SyncStatePill({ state, status, attempts }: { state: string; status?: string; attempts?: number }) {
  const tone = state === 'problem'
    ? 'border-[var(--pos-status-danger)] text-[var(--pos-status-danger)]'
    : state === 'acknowledged' || state === 'cloud_authority'
      ? 'border-[var(--pos-status-success)] text-[var(--pos-status-success)]'
      : 'border-[var(--pos-sync-pending)] text-[var(--pos-sync-pending)]';
  return (
    <div className={`self-start border px-2 py-1 font-mono text-[10px] uppercase tracking-wider ${tone}`}>
      <div>{stopListSyncLabel(state)}</div>
      {status && <div className="mt-0.5 text-[9px] opacity-80">{status}{attempts ? ` · ${attempts}` : ''}</div>}
    </div>
  );
}

const inputClassName = 'h-12 w-full border border-[var(--pos-border)] bg-[var(--pos-surface-raised)] px-3 text-sm text-[var(--pos-text-primary)] outline-none focus:ring-2 focus:ring-[var(--pos-focus-ring)]';
const textareaClassName = 'w-full border border-[var(--pos-border)] bg-[var(--pos-surface-raised)] px-3 py-2 text-sm text-[var(--pos-text-primary)] outline-none focus:ring-2 focus:ring-[var(--pos-focus-ring)] resize-none';

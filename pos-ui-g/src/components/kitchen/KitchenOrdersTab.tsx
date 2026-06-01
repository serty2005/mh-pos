import React, { useMemo } from 'react';
import { ClipboardList, Clock3, PackageCheck } from 'lucide-react';

import { t } from '../../shared/i18n';
import { PosButton, PosEmptyState, PosInlineStatusBadge, PosSkeleton, PosTabs } from '../../shared/ui';
import type { KitchenTicketAction } from '../../shared/api';
import type { BackendKitchenOrderQueueItem, BackendKitchenTicket } from '../../shared/schemas';
import {
  actionLabel,
  activeOrderStatuses,
  formatDateTime,
  formatMinutes,
  orderStatusLabel,
  readyOrderStatuses,
  ticketStatusLabel,
} from './kitchenHelpers';

export type OrdersTab = 'queue' | 'ready';

const orderActions: Record<string, KitchenTicketAction[]> = {
  new: ['accept', 'cancel'],
  accepted: ['start', 'hold', 'cancel'],
  in_progress: ['hold', 'ready', 'cancel'],
  hold: ['start', 'cancel'],
  ready: ['serve', 'recall'],
  served: ['recall'],
  recall: ['start', 'cancel'],
};

export function KitchenOrdersTab({
  orders,
  ordersTab,
  loading,
  busy,
  onTabChange,
  onAction,
}: {
  orders: BackendKitchenOrderQueueItem[];
  ordersTab: OrdersTab;
  loading: boolean;
  busy: boolean;
  onTabChange: (tab: OrdersTab) => void;
  onAction: (ticketId: string, action: KitchenTicketAction) => void;
}) {
  const queueCount = useMemo(
    () => orders.filter((order) => activeOrderStatuses.includes(order.kitchen_order_status)).length,
    [orders],
  );
  const readyCount = useMemo(
    () => orders.filter((order) => readyOrderStatuses.includes(order.kitchen_order_status)).length,
    [orders],
  );
  const ordersForTab = useMemo(() => {
    if (ordersTab === 'ready') {
      return orders.filter((order) => readyOrderStatuses.includes(order.kitchen_order_status));
    }
    return orders.filter((order) => activeOrderStatuses.includes(order.kitchen_order_status));
  }, [orders, ordersTab]);

  return (
    <div className="flex-1 min-h-0 overflow-hidden flex flex-col">
      <PosTabs
        id="orders-tabs"
        activeId={ordersTab}
        onChange={(id) => onTabChange(id as OrdersTab)}
        items={[
          { id: 'queue', label: t.kitchen.tabQueue, count: queueCount },
          { id: 'ready', label: t.kitchen.tabReady, count: readyCount },
        ]}
      />
      <div className="flex-1 min-h-0 p-4 overflow-auto pos-scrollbar-thin">
        {loading ? (
          <PosSkeleton type="grid" />
        ) : ordersForTab.length === 0 ? (
          <PosEmptyState
            title={ordersTab === 'ready' ? t.kitchen.emptyReady : t.kitchen.emptyQueue}
            description={t.kitchen.refresh}
            icon={ordersTab === 'ready' ? <PackageCheck className="w-10 h-10" /> : <ClipboardList className="w-10 h-10" />}
          />
        ) : (
          <div className="grid gap-4 md:grid-cols-2 xl:grid-cols-3">
            {ordersForTab.map((order) => (
              <React.Fragment key={order.order_id}>
                <OrderTile
                  order={order}
                  busy={busy}
                  onAction={onAction}
                />
              </React.Fragment>
            ))}
          </div>
        )}
      </div>
    </div>
  );
}

function OrderTile({
  order,
  busy,
  onAction,
}: {
  order: BackendKitchenOrderQueueItem;
  busy: boolean;
  onAction: (ticketId: string, action: KitchenTicketAction) => void;
}) {
  return (
    <article className="border border-[var(--pos-border)] bg-[var(--pos-surface)] min-h-[260px] flex flex-col">
      <div className="p-4 border-b border-[var(--pos-border)] flex items-start justify-between gap-3">
        <div className="min-w-0">
          <div className="font-mono text-[10px] uppercase tracking-widest text-[var(--pos-text-muted)]">
            {t.common.order}
          </div>
          <h3 className="font-mono text-base font-black text-[var(--pos-text-primary)] truncate">
            {order.table_name || order.edge_order_id || order.order_id || t.kitchen.tableFallback}
          </h3>
        </div>
        <PosInlineStatusBadge variant="neutral" className="shrink-0">
          {orderStatusLabel(order.kitchen_order_status)}
        </PosInlineStatusBadge>
      </div>

      <div className="grid grid-cols-3 border-b border-[var(--pos-border)] divide-x divide-[var(--pos-border)]">
        <TileMetric icon={<Clock3 className="w-3.5 h-3.5" />} label={t.kitchen.elapsed} value={formatMinutes(order.elapsed_seconds || 0)} />
        <TileMetric label={t.kitchen.openedAt} value={formatDateTime(order.created_at)} />
        <TileMetric label={t.kitchen.changedAt} value={formatDateTime(order.last_status_changed_at)} />
      </div>

      <div className="flex-1 divide-y divide-[var(--pos-border)]">
        {(order.tickets || []).map((ticket) => (
          <React.Fragment key={ticket.id}>
            <TicketRow ticket={ticket} busy={busy} onAction={onAction} />
          </React.Fragment>
        ))}
      </div>
    </article>
  );
}

function TileMetric({ icon, label, value }: { icon?: React.ReactNode; label: string; value: string }) {
  return (
    <div className="p-3 min-w-0">
      <div className="flex items-center gap-1.5 font-mono text-[9px] uppercase tracking-widest text-[var(--pos-text-muted)]">
        {icon}
        {label}
      </div>
      <div className="mt-1 font-mono text-xs font-bold text-[var(--pos-text-primary)] truncate">{value}</div>
    </div>
  );
}

function TicketRow({
  ticket,
  busy,
  onAction,
}: {
  ticket: BackendKitchenTicket;
  busy: boolean;
  onAction: (ticketId: string, action: KitchenTicketAction) => void;
}) {
  const actions = orderActions[ticket.status] || [];
  return (
    <div className="p-3 space-y-3">
      <div className="flex items-start justify-between gap-3">
        <div className="min-w-0">
          <div className="font-sans text-sm font-semibold text-[var(--pos-text-primary)] break-words">
            {ticket.name}
          </div>
          <div className="mt-1 font-mono text-[10px] uppercase tracking-wider text-[var(--pos-text-muted)]">
            {ticket.quantity} {ticket.unit_code}
            {ticket.course ? ` · ${t.common.course} ${ticket.course}` : ''}
          </div>
          {ticket.comment && (
            <div className="mt-1 text-xs text-[var(--pos-text-secondary)] break-words">{ticket.comment}</div>
          )}
        </div>
        <PosInlineStatusBadge variant="neutral" className="shrink-0">
          {ticketStatusLabel(ticket.status)}
        </PosInlineStatusBadge>
      </div>
      {actions.length > 0 && (
        <div className="flex flex-wrap gap-2">
          {actions.map((action) => (
            <PosButton
              key={`${ticket.id}-${action}`}
              size="sm"
              variant={action === 'cancel' ? 'danger' : action === 'ready' || action === 'serve' ? 'success' : 'secondary'}
              disabled={busy}
              onClick={() => onAction(ticket.id, action)}
            >
              {actionLabel(action)}
            </PosButton>
          ))}
        </div>
      )}
    </div>
  );
}

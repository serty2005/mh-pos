<template>
  <q-drawer
    :model-value="terminal.closedOrdersDrawer.value"
    side="right"
    overlay
    bordered
    :width="560"
    class="utility-drawer"
    @update:model-value="terminal.closedOrdersDrawer.value = $event"
  >
    <section class="drawer-body">
      <div class="section-head">
        <div>
          <p class="eyebrow">{{ terminal.t('common.status') }}</p>
          <h2>{{ terminal.t('pos.closedOrders') }}</h2>
        </div>
        <div class="drawer-actions">
          <PosButton variant="neutral" mode="flat" round dense compact icon="refresh" class="icon-touch" :aria-label="terminal.t('actions.retry')" @click="() => terminal.closedOrders.refetch()" />
          <PosButton variant="neutral" mode="flat" round dense compact icon="close" class="icon-touch" :aria-label="terminal.t('actions.close')" @click="terminal.closedOrdersDrawer.value = false" />
        </div>
      </div>

      <PosBanner v-if="terminal.closedOrders.error.value" tone="error" :label="terminal.errorLabel(terminal.closedOrders.error.value)" />
      <PosSkeleton v-if="terminal.closedOrders.isFetching.value" kind="order" class="drawer-skeleton" />

      <div class="drawer-filter-row">
        <q-input
          v-model="terminal.closedOrdersBusinessDate.value"
          dense
          outlined
          clearable
          type="date"
          :label="terminal.t('pos.businessDate')"
        />
        <PosPagination
          :value="terminal.closedOrdersOffset.value + 1"
          :previous-label="terminal.t('actions.previousPage')"
          :next-label="terminal.t('actions.nextPage')"
          :previous-disabled="!terminal.closedOrdersHasPreviousPage.value"
          :next-disabled="!terminal.closedOrdersHasNextPage.value"
          @previous="terminal.previousClosedOrdersPage"
          @next="terminal.nextClosedOrdersPage"
        />
      </div>

      <div v-if="!terminal.closedOrders.isFetching.value && terminal.closedOrders.data.value?.length" class="closed-orders-list">
        <article v-for="order in terminal.closedOrders.data.value" :key="order.id" class="closed-order-item">
          <div class="order-summary compact-summary">
            <div>
              <span>{{ terminal.t('pos.order') }}</span>
              <strong>{{ terminal.shortId(order.id) }}</strong>
            </div>
            <div>
              <span>{{ terminal.t('pos.table') }}</span>
              <strong>{{ order.table_name }}</strong>
            </div>
            <div>
              <span>{{ terminal.t('pos.total') }}</span>
              <strong>{{ terminal.money(order.total, 'RUB') }}</strong>
            </div>
            <div>
              <span>{{ terminal.t('common.status') }}</span>
              <strong>{{ terminal.statusLabel(order.status) }}</strong>
            </div>
          </div>
          <div class="closed-order-actions">
            <PosButton
              variant="danger"
              icon="cancel"
              :label="terminal.t('pos.checkCancellation')"
              :title="terminal.closedOrderCancellationUnavailableKey(order) ? terminal.t(terminal.closedOrderCancellationUnavailableKey(order)) : terminal.t('pos.checkCancellationCopy')"
              :disabled="!terminal.canCancelClosedOrder(order)"
              :loading="terminal.refundMutation.isPending.value && terminal.refundMode.value === 'check_cancellation'"
              @click="terminal.openCheckCancellationDialogForOrder(order)"
            />
            <PosButton
              variant="danger"
              mode="outline"
              icon="undo"
              :label="terminal.t('pos.checkRefund')"
              :title="terminal.closedOrderRefundUnavailableKey(order) ? terminal.t(terminal.closedOrderRefundUnavailableKey(order)) : terminal.t('pos.checkRefundCopy')"
              :disabled="!terminal.canRefundClosedOrder(order)"
              :loading="terminal.refundMutation.isPending.value && terminal.refundMode.value === 'check_refund'"
              @click="terminal.openCheckRefundDialogForOrder(order)"
            />
            <PosButton
              v-if="order.check?.payments?.some((payment) => payment.status === 'captured')"
              variant="danger"
              mode="flat"
              icon="payments"
              :label="terminal.t('pos.paymentRefundFallback')"
              :title="terminal.closedOrderPaymentRefundUnavailableKey(order) ? terminal.t(terminal.closedOrderPaymentRefundUnavailableKey(order)) : terminal.t('pos.paymentRefundFallbackCopy')"
              :disabled="!terminal.canRefundPaymentForOrder(order)"
              :loading="terminal.refundMutation.isPending.value && terminal.refundMode.value === 'payment_refund'"
              @click="terminal.openRefundDialogForOrder(order)"
            />
          </div>
        </article>
      </div>
      <PosEmptyState v-else-if="!terminal.closedOrders.isFetching.value" size="wide" :label="terminal.t('common.empty')" />
    </section>
  </q-drawer>
</template>

<script setup lang="ts">
import { PosBanner, PosButton, PosEmptyState, PosPagination, PosSkeleton } from '../../shared/ui';
import type { CashierTerminal } from './useCashierTerminal';

defineProps<{
  terminal: CashierTerminal;
}>();
</script>

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
          <q-btn flat round icon="refresh" class="icon-touch" :aria-label="terminal.t('actions.retry')" @click="() => terminal.closedOrders.refetch()" />
          <q-btn flat round icon="close" class="icon-touch" :aria-label="terminal.t('actions.close')" @click="terminal.closedOrdersDrawer.value = false" />
        </div>
      </div>

      <q-banner v-if="terminal.closedOrders.error.value" class="error-banner dense-banner" rounded>{{ terminal.t(terminal.displayErrorMessageKey(terminal.closedOrders.error.value)) }}</q-banner>
      <q-skeleton v-if="terminal.closedOrders.isFetching.value" class="order-skeleton drawer-skeleton" />

      <div v-else-if="terminal.closedOrders.data.value?.length" class="closed-orders-list">
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
            <q-btn
              color="negative"
              unelevated
              class="touch-button"
              icon="cancel"
              :label="terminal.t('pos.checkCancellation')"
              :disable="!terminal.canCancelClosedOrder(order)"
              :loading="terminal.refundMutation.isPending.value && terminal.refundMode.value === 'check_cancellation'"
              @click="terminal.openCheckCancellationDialogForOrder(order)"
            />
            <q-btn
              color="negative"
              outline
              class="touch-button"
              icon="undo"
              :label="terminal.t('pos.checkRefund')"
              :disable="!terminal.canRefundClosedOrder(order)"
              :loading="terminal.refundMutation.isPending.value && terminal.refundMode.value === 'check_refund'"
              @click="terminal.openCheckRefundDialogForOrder(order)"
            />
            <q-btn
              v-if="order.check?.payments?.some((payment) => payment.status === 'captured')"
              color="negative"
              flat
              class="touch-button"
              icon="payments"
              :label="terminal.t('pos.paymentRefund')"
              :disable="!terminal.canRefundPaymentForOrder(order)"
              :loading="terminal.refundMutation.isPending.value && terminal.refundMode.value === 'payment_refund'"
              @click="terminal.openRefundDialogForOrder(order)"
            />
          </div>
        </article>
      </div>
      <div v-else class="empty-state wide">{{ terminal.t('common.empty') }}</div>
    </section>
  </q-drawer>
</template>

<script setup lang="ts">
import type { CashierTerminal } from './useCashierTerminal';

defineProps<{
  terminal: CashierTerminal;
}>();
</script>

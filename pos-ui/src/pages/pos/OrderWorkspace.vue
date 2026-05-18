<template>
  <main class="order-pane" :aria-label="terminal.t('pos.activeOrder')">
    <div class="order-hero">
      <div>
        <p class="eyebrow">{{ terminal.selectedTable.value?.name ? `${terminal.t('pos.selectedTable')} ${terminal.selectedTable.value.name}` : terminal.t('pos.chooseTable') }}</p>
        <h2>{{ terminal.t('pos.activeOrder') }}</h2>
      </div>
      <div v-if="terminal.activeOrder.value" class="total-chip">
        <span>{{ terminal.t('pos.total') }}</span>
        <strong>{{ terminal.money(terminal.activeOrder.value.total, terminal.orderCurrency.value) }}</strong>
      </div>
      <q-btn
        color="primary"
        unelevated
        class="touch-button primary-action"
        icon="receipt_long"
        :label="terminal.t('actions.createOrder')"
        :disable="!terminal.canCreateOrder.value"
        :loading="terminal.createOrderMutation.isPending.value"
        @click="terminal.createOrderMutation.mutate()"
      />
    </div>

    <q-banner v-if="terminal.orderError.value" class="error-banner dense-banner workspace-banner" rounded>{{ terminal.orderError.value }}</q-banner>
    <q-skeleton v-if="terminal.orderLoading.value" class="order-skeleton" />

    <section v-else-if="terminal.activeOrder.value" class="order-workspace">
      <div class="order-summary">
        <div>
          <span>{{ terminal.t('pos.order') }}</span>
          <strong>{{ terminal.shortId(terminal.activeOrder.value.id) }}</strong>
        </div>
        <div>
          <span>{{ terminal.t('common.status') }}</span>
          <strong>{{ terminal.statusLabel(terminal.activeOrder.value.status) }}</strong>
        </div>
        <div>
          <span>{{ terminal.t('pos.total') }}</span>
          <strong>{{ terminal.money(terminal.activeOrder.value.total, terminal.orderCurrency.value) }}</strong>
        </div>
      </div>

      <blocking-notice
        v-if="terminal.activeOrder.value.status === 'locked'"
        :terminal="terminal"
        :title="terminal.t('pos.blocking.lockedOrder.title')"
        :reason="terminal.t('pos.blocking.lockedOrder.reason')"
        :permission="terminal.canCancelPrecheck.value ? '' : 'pos.precheck.cancel.request'"
        icon="lock"
      />
      <q-banner v-if="terminal.finalCheckData.value" class="success-banner" rounded>
        {{ terminal.t('pos.checkCreated') }}: {{ terminal.shortId(terminal.finalCheckData.value.id) }} · {{ terminal.money(terminal.finalCheckData.value.total, terminal.orderCurrency.value) }}
      </q-banner>

      <div v-if="terminal.finalCheckData.value" class="inline-action horizontal">
        <q-btn
          outline
          color="secondary"
          class="touch-button"
          icon="print"
          :label="terminal.t('actions.reprintCheck')"
          :disable="!terminal.canReprintCheck.value"
          :loading="terminal.reprintCheckMutation.isPending.value"
          @click="terminal.reprintCheckMutation.mutate(terminal.finalCheckData.value.id)"
        />
      </div>

      <div class="section-head slim">
        <h2>{{ terminal.t('pos.orderLines') }}</h2>
      </div>
      <div v-if="terminal.activeLines.value.length" class="line-table">
        <div v-for="line in terminal.activeLines.value" :key="line.id" class="line-row">
          <div class="line-title">
            <strong>{{ line.name }}</strong>
            <span>{{ terminal.money(line.unit_price, terminal.orderCurrency.value) }}</span>
            <ul v-if="line.modifiers.length" class="line-modifiers">
              <li v-for="modifier in line.modifiers" :key="modifier.id">
                <span>{{ modifier.name }} × {{ modifier.quantity }}</span>
                <strong>{{ terminal.money(modifier.total_price, terminal.orderCurrency.value) }}</strong>
              </li>
            </ul>
          </div>
          <div class="quantity-stepper" :aria-label="line.name">
            <q-btn
              flat
              round
              class="stepper-button"
              icon="remove"
              :aria-label="terminal.t('actions.remove')"
              :disable="!terminal.canChangeOrderLine.value || line.quantity <= 1"
              @click="terminal.changeQuantity(line.id, line.quantity - 1)"
            />
            <span>{{ line.quantity }}</span>
            <q-btn flat round class="stepper-button" icon="add" :aria-label="terminal.t('actions.add')" :disable="!terminal.canChangeOrderLine.value" @click="terminal.changeQuantity(line.id, line.quantity + 1)" />
          </div>
          <strong class="line-total">{{ terminal.money(line.total_price, terminal.orderCurrency.value) }}</strong>
          <q-btn flat round class="stepper-button" icon="tune" :disable="!terminal.canChangeOrderLine.value || !terminal.canEditLineModifiers(line.id)" :aria-label="terminal.t('actions.editModifiers')" @click="terminal.editLineModifiers(line.id)" />
          <q-btn flat round class="stepper-button" icon="delete" color="negative" :disable="!terminal.canVoidOrderLine.value" :aria-label="terminal.t('actions.voidLine')" @click="terminal.voidLine(line.id)" />
        </div>
      </div>
      <div v-else class="empty-state">{{ terminal.t('pos.emptyOrder') }}</div>
    </section>

    <div v-else class="empty-state wide workspace-empty">{{ terminal.selectedTableId.value ? terminal.t('pos.noActiveOrder') : terminal.t('pos.chooseTable') }}</div>
    <blocking-notice
      v-if="terminal.selectedTableId.value && !terminal.activeOrder.value && terminal.actionBlocker('pos.order.create', terminal.canCreateOrder.value)"
      class="workspace-banner"
      :terminal="terminal"
      :title="terminal.t(terminal.actionBlocker('pos.order.create', terminal.canCreateOrder.value)?.titleKey ?? '')"
      :reason="terminal.t(terminal.actionBlocker('pos.order.create', terminal.canCreateOrder.value)?.reasonKey ?? '')"
      :permission="terminal.actionBlocker('pos.order.create', terminal.canCreateOrder.value)?.permission"
      icon="lock"
    />
  </main>
</template>

<script setup lang="ts">
import BlockingNotice from './BlockingNotice.vue';
import type { CashierTerminal } from './useCashierTerminal';

defineProps<{
  terminal: CashierTerminal;
}>();
</script>

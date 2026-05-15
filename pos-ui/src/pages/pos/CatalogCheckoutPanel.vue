<template>
  <aside class="action-pane" :aria-label="terminal.t('pos.menu')">
    <div class="pane-scroll action-scroll">
      <section class="terminal-actions">
        <div class="section-head slim">
          <div>
            <p class="eyebrow">{{ terminal.t('common.status') }}</p>
            <h2>{{ terminal.t('pos.terminalActions') }}</h2>
          </div>
          <q-btn flat round icon="refresh" class="icon-touch" :aria-label="terminal.t('actions.retry')" @click="terminal.refreshOps" />
        </div>

        <q-banner v-if="terminal.statusError.value" class="error-banner dense-banner" rounded>{{ terminal.statusError.value }}</q-banner>

        <div class="ops-grid">
          <q-btn
            v-if="!terminal.currentShift.data.value"
            color="primary"
            unelevated
            class="touch-button"
            icon="schedule"
            :label="terminal.t('actions.openShift')"
            :disable="!terminal.canOpenShift.value"
            :loading="terminal.openShiftMutation.isPending.value"
            @click="terminal.openShiftMutation.mutate()"
          />
          <q-btn
            v-else-if="!terminal.currentCashSession.data.value"
            outline
            color="secondary"
            class="touch-button"
            icon="event_busy"
            :label="terminal.t('actions.closeShift')"
            :disable="!terminal.canCloseShift.value"
            :loading="terminal.closeShiftMutation.isPending.value"
            @click="terminal.closeShiftMutation.mutate(terminal.currentShift.data.value.id)"
          />

          <div v-if="terminal.currentShift.data.value && !terminal.currentCashSession.data.value" class="cash-open-row">
            <q-input
              v-model.number="terminal.openingCashAmount.value"
              dense
              outlined
              type="number"
              min="0"
              :step="terminal.currencyInputStep(terminal.currency.value)"
              :label="terminal.t('common.amount')"
              :suffix="terminal.currency.value"
            />
            <q-btn
              color="primary"
              outline
              class="touch-button"
              icon="point_of_sale"
              :label="terminal.t('actions.openCashSession')"
              :disable="!terminal.canOpenCashSession.value"
              :loading="terminal.openCashMutation.isPending.value"
              @click="terminal.openCashMutation.mutate(terminal.openingCashAmount.value)"
            />
          </div>

          <template v-if="terminal.currentCashSession.data.value">
            <q-btn
              outline
              color="secondary"
              class="touch-button"
              icon="inventory_2"
              :label="terminal.t('pos.cashDrawer')"
              :disable="!terminal.canRecordCashDrawerEvent.value"
              @click="terminal.cashDrawerDialog.value = true"
            />
            <div class="cash-open-row">
              <q-input
                v-model.number="terminal.closingCashAmount.value"
                dense
                outlined
                type="number"
                min="0"
                :step="terminal.currencyInputStep(terminal.currency.value)"
                :label="terminal.t('common.amount')"
                :suffix="terminal.currency.value"
              />
              <q-btn
                outline
                color="secondary"
                class="touch-button"
                icon="payments"
                :label="terminal.t('actions.closeCashSession')"
                :disable="!terminal.canCloseCashSession.value"
                :loading="terminal.closeCashMutation.isPending.value"
                @click="terminal.closeCashMutation.mutate({ cashSessionId: terminal.currentCashSession.data.value.id, amount: terminal.closingCashAmount.value })"
              />
            </div>
          </template>

          <q-btn
            v-if="terminal.canViewClosedOrders.value"
            outline
            color="secondary"
            class="touch-button"
            icon="history"
            :label="terminal.t('pos.closedOrders')"
            @click="terminal.closedOrdersDrawer.value = true"
          />
          <q-btn
            v-if="terminal.canViewSync.value"
            outline
            color="secondary"
            class="touch-button"
            icon="sync"
            :label="terminal.t('pos.syncStatus')"
            @click="terminal.syncDrawer.value = true"
          />
        </div>
      </section>

      <q-separator />

      <section class="catalog-section">
        <div class="section-head">
          <h2>{{ terminal.t('pos.menu') }}</h2>
          <q-btn flat round icon="refresh" class="icon-touch" :aria-label="terminal.t('actions.retry')" @click="terminal.refetchMenu" />
        </div>
        <q-input
          v-model="terminal.menuSearch.value"
          dense
          outlined
          clearable
          debounce="120"
          class="menu-search"
          :label="terminal.t('pos.searchMenu')"
        >
          <template #prepend>
            <q-icon name="search" />
          </template>
        </q-input>

        <q-skeleton v-if="terminal.menu.isPending.value" class="skeleton-row" />
        <q-banner v-else-if="terminal.menu.isError.value" class="error-banner dense-banner" rounded>{{ terminal.t('common.error') }}</q-banner>
        <div v-else-if="terminal.visibleMenuItems.value.length" class="menu-list">
          <button
            v-for="item in terminal.visibleMenuItems.value"
            :key="item.id"
            class="menu-button"
            type="button"
            :disabled="!terminal.canAddOrderLine.value"
            @click="terminal.openMenuItem(item)"
          >
            <span>
              {{ item.name }}
              <small v-if="item.modifier_groups.length">{{ terminal.t('pos.modifiersAvailable') }}</small>
            </span>
            <strong>{{ terminal.money(item.price, item.currency) }}</strong>
          </button>
        </div>
        <div v-else class="empty-state">{{ terminal.regularMenuItems.value.length ? terminal.t('pos.noMenuMatches') : terminal.t('pos.emptyMenu') }}</div>
      </section>

      <section class="catalog-section service-section">
        <div class="section-head slim">
          <h2>{{ terminal.t('pos.services') }}</h2>
        </div>
        <div v-if="terminal.visibleServiceItems.value.length" class="menu-list service-list">
          <button
            v-for="item in terminal.visibleServiceItems.value"
            :key="item.id"
            class="menu-button service-button"
            type="button"
            :disabled="!terminal.canAddOrderLine.value"
            @click="terminal.openMenuItem(item)"
          >
            <span>{{ item.name }}</span>
            <strong>{{ terminal.money(item.price, item.currency) }}</strong>
          </button>
        </div>
        <div v-else class="empty-state small">{{ terminal.serviceMenuItems.value.length ? terminal.t('pos.noMenuMatches') : terminal.t('pos.emptyServices') }}</div>
      </section>

      <section class="checkout-dock">
        <div class="section-head slim">
          <h2>{{ terminal.t('pos.precheckActions') }}</h2>
        </div>
        <div v-if="terminal.activePrecheck.value" class="precheck-box">
          <div class="state-line">
            <span>{{ terminal.t('pos.precheck') }}</span>
            <strong>{{ terminal.t('pos.precheckIssued') }}</strong>
          </div>
          <div class="state-line">
            <span>{{ terminal.t('pos.total') }}</span>
            <strong>{{ terminal.money(terminal.activePrecheck.value.total, terminal.orderCurrency.value) }}</strong>
          </div>
          <div class="state-line">
            <span>{{ terminal.t('pos.payment') }}</span>
            <strong>{{ terminal.money(terminal.activePrecheck.value.paid_total, terminal.orderCurrency.value) }}</strong>
          </div>
          <div class="payment-actions">
            <q-btn outline color="negative" class="touch-button" icon="undo" :label="terminal.t('pos.cancelPrecheck')" :disable="terminal.activePrecheck.value.paid_total > 0 || !terminal.canCancelPrecheck.value" @click="terminal.cancelDialog.value = true" />
            <q-btn
              outline
              color="secondary"
              class="touch-button"
              icon="print"
              :label="terminal.t('actions.reprintPrecheck')"
              :disable="!terminal.canReprintPrecheck.value"
              :loading="terminal.reprintPrecheckMutation.isPending.value"
              @click="terminal.reprintPrecheckMutation.mutate(terminal.activePrecheck.value.id)"
            />
          </div>
        </div>
        <q-btn
          v-else
          color="primary"
          unelevated
          class="touch-button primary-action"
          icon="request_quote"
          :label="terminal.t('actions.issuePrecheck')"
          :disable="!terminal.canIssuePrecheck.value"
          :loading="terminal.issuePrecheckMutation.isPending.value"
          @click="terminal.issuePrecheckMutation.mutate(terminal.activeOrder.value?.id ?? '')"
        />
        <q-btn
          v-if="!terminal.activePrecheck.value && terminal.latestPrecheck.value"
          outline
          color="secondary"
          class="touch-button"
          icon="print"
          :label="terminal.t('actions.reprintPrecheck')"
          :disable="!terminal.canReprintPrecheck.value"
          :loading="terminal.reprintPrecheckMutation.isPending.value"
          @click="terminal.reprintPrecheckMutation.mutate(terminal.latestPrecheck.value.id)"
        />

        <div class="payment-box">
          <q-input
            v-model.number="terminal.paymentAmount.value"
            outlined
            dense
            type="number"
            min="0"
            :step="terminal.currencyInputStep(terminal.orderCurrency.value)"
            :label="terminal.t('pos.paymentAmount')"
            :suffix="terminal.orderCurrency.value"
            :disable="!terminal.activePrecheck.value"
          />
          <div class="payment-actions">
            <q-btn
              color="primary"
              unelevated
              class="touch-button"
              icon="payments"
              :label="terminal.t('actions.payCash')"
              :disable="!terminal.canPayCash.value"
              :loading="terminal.paymentMutation.isPending.value"
              @click="terminal.pay('cash')"
            />
            <q-btn
              color="secondary"
              unelevated
              class="touch-button"
              icon="credit_card"
              :label="terminal.t('actions.payCard')"
              :disable="!terminal.canPayCard.value"
              :loading="terminal.paymentMutation.isPending.value"
              @click="terminal.pay('card')"
            />
          </div>
          <p v-if="terminal.paymentBlockedReasonKey.value" class="payment-hint">
            {{ terminal.t(terminal.paymentBlockedReasonKey.value) }}
          </p>
        </div>
      </section>
    </div>
  </aside>
</template>

<script setup lang="ts">
import type { CashierTerminal } from './useCashierTerminal';

defineProps<{
  terminal: CashierTerminal;
}>();
</script>

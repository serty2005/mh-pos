<template>
  <section class="cashier-status-bar" :aria-label="terminal.t('pos.cashierTerminal')">
    <div class="operator-block">
      <p class="eyebrow">{{ terminal.t('pos.actor') }}</p>
      <h1>{{ terminal.actorName.value }}</h1>
      <span class="operator-role">{{ terminal.auth.actor?.role_id ?? terminal.shortId(terminal.auth.actor?.employee_id ?? '') }}</span>
    </div>

    <div class="status-cluster">
      <button class="status-box" :class="{ good: terminal.pairing.data.value?.paired }" type="button" @click="terminal.refreshOps">
        <span>{{ terminal.t('pos.pairing') }}</span>
        <strong>{{ terminal.pairing.data.value?.paired ? terminal.t('status.paired') : terminal.t('common.error') }}</strong>
      </button>
      <button class="status-box" :class="{ good: terminal.currentShift.data.value }" type="button" @click="terminal.refreshOps">
        <span>{{ terminal.t('pos.shift') }}</span>
        <strong>{{ terminal.currentShift.data.value ? terminal.t('status.open') : terminal.t('pos.noShift') }}</strong>
      </button>
      <button class="status-box" :class="{ good: terminal.currentCashSession.data.value }" type="button" @click="terminal.refreshOps">
        <span>{{ terminal.t('pos.cashSession') }}</span>
        <strong>{{ terminal.currentCashSession.data.value ? terminal.t('status.open') : terminal.t('pos.noCashSession') }}</strong>
      </button>
      <button v-if="terminal.canViewSync.value" class="status-box technical" :class="{ warning: terminal.syncProblems.value > 0 }" type="button" @click="terminal.syncDrawer.value = true">
        <span>{{ terminal.t('pos.syncStatus') }}</span>
        <strong>{{ terminal.syncProblems.value > 0 ? terminal.syncProblems.value : terminal.t('status.sent') }}</strong>
      </button>
      <div class="status-box technical">
        <span>{{ terminal.t('common.node') }}</span>
        <strong>{{ terminal.shortId(terminal.auth.nodeDeviceId) }}</strong>
      </div>
      <q-btn flat round icon="lock" class="icon-touch dark-touch" :aria-label="terminal.t('actions.lock')" @click="terminal.lockTerminal" />
    </div>
  </section>
</template>

<script setup lang="ts">
import type { CashierTerminal } from './useCashierTerminal';

defineProps<{
  terminal: CashierTerminal;
}>();
</script>

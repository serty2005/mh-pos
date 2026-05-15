<template>
  <q-dialog :model-value="terminal.modifierDialog.value" persistent @update:model-value="terminal.closeModifierDialog">
    <q-card class="modifier-dialog">
      <q-card-section class="dialog-head">
        <div>
          <p class="eyebrow">{{ terminal.t('pos.modifiers') }}</p>
          <h2>{{ terminal.modifierMenuItem.value?.name }}</h2>
        </div>
        <strong v-if="terminal.selectedModifierTotal.value > 0" class="modifier-total">
          {{ terminal.money(terminal.selectedModifierTotal.value, terminal.modifierMenuItem.value?.currency ?? terminal.orderCurrency.value) }}
        </strong>
      </q-card-section>

      <q-card-section class="modifier-groups">
        <q-banner v-if="terminal.modifierValidationKey.value" class="error-banner dense-banner" rounded>
          {{ terminal.t(terminal.modifierValidationKey.value) }}
        </q-banner>

        <section v-for="group in terminal.modifierGroupsForDialog.value" :key="group.id" class="modifier-group">
          <div class="modifier-group-head">
            <div>
              <h3>{{ group.name }}</h3>
              <span>{{ group.required ? terminal.t('pos.requiredModifierGroup') : terminal.t('pos.optionalModifierGroup') }}</span>
            </div>
            <strong>{{ terminal.modifierGroupCount(group.id) }} / {{ group.max_count > 0 ? group.max_count : terminal.t('pos.noLimit') }}</strong>
          </div>

          <div class="modifier-options">
            <article v-for="option in group.options.filter((item) => item.active)" :key="option.id" class="modifier-option">
              <div>
                <strong>{{ option.name }}</strong>
                <span>{{ option.price_minor > 0 ? terminal.money(option.price_minor, terminal.modifierMenuItem.value?.currency ?? terminal.orderCurrency.value) : terminal.t('pos.freeModifier') }}</span>
              </div>
              <div class="quantity-stepper compact-stepper" :aria-label="option.name">
                <q-btn
                  flat
                  round
                  class="stepper-button"
                  icon="remove"
                  :aria-label="terminal.t('actions.remove')"
                  :disable="(terminal.modifierQuantities.value[option.id] ?? 0) <= 0"
                  @click="terminal.changeModifierQuantity(option.id, (terminal.modifierQuantities.value[option.id] ?? 0) - 1)"
                />
                <span>{{ terminal.modifierQuantities.value[option.id] ?? 0 }}</span>
                <q-btn
                  flat
                  round
                  class="stepper-button"
                  icon="add"
                  :aria-label="terminal.t('actions.add')"
                  @click="terminal.changeModifierQuantity(option.id, (terminal.modifierQuantities.value[option.id] ?? 0) + 1)"
                />
              </div>
            </article>
          </div>
        </section>
      </q-card-section>

      <q-card-actions align="right" class="dialog-actions">
        <q-btn flat :label="terminal.t('actions.cancel')" @click="terminal.closeModifierDialog" />
        <q-btn
          color="primary"
          unelevated
          icon="add_shopping_cart"
          :label="terminal.t('actions.add')"
          :loading="terminal.addLineMutation.isPending.value"
          @click="terminal.submitModifierSelection"
        />
      </q-card-actions>
    </q-card>
  </q-dialog>
</template>

<script setup lang="ts">
import type { CashierTerminal } from './useCashierTerminal';

defineProps<{
  terminal: CashierTerminal;
}>();
</script>

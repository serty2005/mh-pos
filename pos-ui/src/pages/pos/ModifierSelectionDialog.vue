<template>
  <PosDialog
    :model-value="terminal.modifierDialog.value"
    persistent
    card-class="modifier-dialog"
    body-class="modifier-groups"
    :eyebrow="terminal.t(terminal.modifierDialogMode.value === 'edit' ? 'pos.editModifiers' : 'pos.modifiers')"
    :title="terminal.modifierMenuItem.value?.name ?? ''"
    @update:model-value="terminal.closeModifierDialog"
  >
    <template #header-side>
      <strong v-if="terminal.selectedModifierTotal.value > 0" class="modifier-total">
        {{ terminal.money(terminal.selectedModifierTotal.value, terminal.modifierMenuItem.value?.currency ?? terminal.orderCurrency.value) }}
      </strong>
    </template>

        <PosBanner v-if="terminal.modifierValidationKey.value" tone="error" :label="terminal.t(terminal.modifierValidationKey.value)" />

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
              <PosQuantityStepper
                compact
                :value="terminal.modifierQuantities.value[option.id] ?? 0"
                :label="option.name"
                :decrement-label="terminal.t('actions.remove')"
                :increment-label="terminal.t('actions.add')"
                :decrement-disabled="(terminal.modifierQuantities.value[option.id] ?? 0) <= 0"
                @decrement="terminal.changeModifierQuantity(option.id, (terminal.modifierQuantities.value[option.id] ?? 0) - 1)"
                @increment="terminal.changeModifierQuantity(option.id, (terminal.modifierQuantities.value[option.id] ?? 0) + 1)"
              />
            </article>
          </div>
        </section>

      <template #actions>
        <PosButton variant="neutral" mode="flat" :label="terminal.t('actions.cancel')" @click="terminal.closeModifierDialog" />
        <PosButton
          variant="primary"
          :icon="terminal.modifierDialogMode.value === 'edit' ? 'save' : 'add_shopping_cart'"
          :label="terminal.t(terminal.modifierDialogMode.value === 'edit' ? 'actions.save' : 'actions.add')"
          :loading="terminal.modifierDialogMode.value === 'edit' ? terminal.modifierUpdateMutation.isPending.value : terminal.addLineMutation.isPending.value"
          :disabled="!terminal.canSubmitModifierSelection.value"
          @click="terminal.submitModifierSelection"
        />
      </template>
  </PosDialog>
</template>

<script setup lang="ts">
import { PosBanner, PosButton, PosDialog, PosQuantityStepper } from '../../shared/ui';
import type { CashierTerminal } from './useCashierTerminal';

defineProps<{
  terminal: CashierTerminal;
}>();
</script>

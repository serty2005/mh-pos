<template>
  <q-banner v-if="ctx.errorKey.value" class="error-banner dense-banner">
    <div class="error-content">
      <strong>{{ t(ctx.errorKey.value) }}</strong>
      <span v-if="ctx.errorCode.value">{{ t('errors.supportCode') }}: {{ ctx.errorCode.value }}</span>
      <span v-if="ctx.errorCorrelationId.value">{{ t('errors.correlationId') }}: {{ ctx.errorCorrelationId.value }}</span>
      <ul v-if="ctx.errorDetailsList.value.length > 0">
        <li v-for="item in ctx.errorDetailsList.value" :key="item.key">{{ item.label }}: {{ item.value }}</li>
      </ul>
      <div class="error-actions">
        <q-btn flat dense icon="refresh" :label="t('actions.retry')" @click="ctx.reloadActive()" />
        <q-btn
          v-if="!ctx.selectedRestaurantId.value"
          flat
          dense
          icon="storefront"
          :label="t('cloud.recovery.selectRestaurant')"
          @click="ctx.setActive('restaurants')"
        />
        <q-btn
          v-else-if="target"
          flat
          dense
          icon="open_in_new"
          :label="t('cloud.recovery.openSection')"
          @click="ctx.setActive(target)"
        />
      </div>
    </div>
  </q-banner>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n';

const { t } = useI18n();

defineProps<{
  ctx: Record<string, any>;
  target?: string;
}>();
</script>

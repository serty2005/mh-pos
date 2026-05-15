<template>
  <section class="cloud-scenario-grid">
    <article class="cloud-panel cloud-plan-panel launch-primary-panel">
      <div class="section-head stacked">
        <p class="eyebrow">{{ t('cloud.onboarding.kicker') }}</p>
        <h2>{{ t('cloud.onboarding.title') }}</h2>
      </div>
      <cloud-safe-error-banner :ctx="ctx" target="launchPlan" />
      <div class="onboarding-checks readiness-panel">
        <div v-for="check in ctx.onboardingChecks.value" :key="check.key" class="onboarding-check" :class="{ blocked: !check.ready }">
          <q-icon :name="check.ready ? 'check_circle' : 'radio_button_unchecked'" :class="{ ready: check.ready }" />
          <div>
            <strong>{{ t(check.titleKey) }}</strong>
            <span>{{ t(check.descriptionKey, check.params) }}</span>
          </div>
          <q-btn
            :outline="check.ready"
            :color="check.ready ? 'secondary' : 'primary'"
            dense
            :icon="check.icon"
            :label="t(check.actionKey)"
            @click="ctx.setActive(check.target)"
          />
        </div>
      </div>
    </article>

    <article class="cloud-panel cloud-plan-panel">
      <div class="section-head stacked">
        <p class="eyebrow">{{ t('cloud.scenarios.operatorJourney') }}</p>
        <h2>{{ t('cloud.launchPlan.title') }}</h2>
      </div>
      <ol class="cloud-roadmap">
        <li v-for="step in ctx.launchSteps" :key="step.key" :class="step.status">
          <span>{{ t(step.badgeKey) }}</span>
          <div>
            <strong>{{ t(step.titleKey) }}</strong>
            <p>{{ t(step.descriptionKey) }}</p>
          </div>
        </li>
      </ol>
    </article>

    <article class="cloud-panel cloud-plan-panel accent">
      <div class="section-head stacked">
        <p class="eyebrow">{{ t('cloud.scenarios.firstSlice') }}</p>
        <h2>{{ t('cloud.launchPlan.firstSliceTitle') }}</h2>
      </div>
      <div class="cloud-playbook">
        <div v-for="item in ctx.playbookItems" :key="item.titleKey">
          <span>{{ t(item.kickerKey) }}</span>
          <strong>{{ t(item.titleKey) }}</strong>
          <p>{{ t(item.descriptionKey) }}</p>
        </div>
      </div>
    </article>
  </section>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n';

import CloudSafeErrorBanner from './CloudSafeErrorBanner.vue';

const { t } = useI18n();

defineProps<{
  ctx: Record<string, any>;
}>();
</script>

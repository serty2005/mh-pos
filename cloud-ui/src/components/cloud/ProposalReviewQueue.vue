<template>
  <section class="proposal-review-grid">
    <div class="cloud-panel proposal-list-panel">
      <cloud-safe-error-banner :ctx="ctx" target="proposalReview" />
      <div class="section-head">
        <div>
          <p class="eyebrow">{{ t('cloud.proposals.catalogTitle') }}</p>
          <h2>{{ t('cloud.proposals.catalogQueue') }}</h2>
        </div>
        <q-btn flat dense icon="refresh" :loading="ctx.isLoading('proposal-review')" :label="t('actions.refresh')" @click="ctx.loadProposalReview()" />
      </div>

      <q-select
        v-model="ctx.proposalStatusFilter.value"
        dense
        outlined
        emit-value
        map-options
        :label="t('cloud.proposals.statusFilter')"
        :options="statusOptions"
        @update:model-value="ctx.loadProposalReview()"
      />

      <div v-if="!ctx.selectedRestaurantId.value" class="empty-state wide">{{ t('cloud.empty.selectRestaurant') }}</div>
      <div v-else-if="ctx.isLoading('proposal-review')" class="cloud-skeleton-list">
        <q-skeleton v-for="index in 4" :key="index" class="skeleton-row" />
      </div>
      <div v-else-if="catalogRows.length === 0" class="empty-state">{{ t('cloud.proposals.emptyCatalog') }}</div>
      <div v-else class="proposal-list">
        <button
          v-for="item in catalogRows"
          :key="item.key"
          type="button"
          class="proposal-row"
          :class="{ selected: selectedKey === item.key }"
          @click="selectedKey = item.key"
        >
          <span class="cloud-status" :class="item.status">{{ suggestionStatus(item.status) }}</span>
          <strong>{{ item.title }}</strong>
          <small>{{ item.subtitle }}</small>
          <span v-if="item.groupLabel" class="proposal-group-chip">{{ item.groupLabel }}</span>
        </button>
      </div>
    </div>

    <div class="cloud-panel proposal-list-panel">
      <div class="section-head">
        <div>
          <p class="eyebrow">{{ t('cloud.proposals.recipeTitle') }}</p>
          <h2>{{ t('cloud.proposals.recipeQueue') }}</h2>
        </div>
      </div>

      <div v-if="!ctx.selectedRestaurantId.value" class="empty-state wide">{{ t('cloud.empty.selectRestaurant') }}</div>
      <div v-else-if="ctx.isLoading('proposal-review')" class="cloud-skeleton-list">
        <q-skeleton v-for="index in 4" :key="index" class="skeleton-row" />
      </div>
      <div v-else-if="recipeRows.length === 0" class="empty-state">{{ t('cloud.proposals.emptyRecipe') }}</div>
      <div v-else class="proposal-list">
        <button
          v-for="item in recipeRows"
          :key="item.key"
          type="button"
          class="proposal-row"
          :class="{ selected: selectedKey === item.key }"
          @click="selectedKey = item.key"
        >
          <span class="cloud-status" :class="item.status">{{ suggestionStatus(item.status) }}</span>
          <strong>{{ item.title }}</strong>
          <small>{{ item.subtitle }}</small>
          <span v-if="item.groupLabel" class="proposal-group-chip">{{ item.groupLabel }}</span>
        </button>
      </div>
    </div>

    <aside class="cloud-panel proposal-detail-panel">
      <div v-if="!selected" class="empty-state wide">{{ t('cloud.proposals.selectPrompt') }}</div>
      <template v-else>
        <div class="section-head stacked">
          <p class="eyebrow">{{ selected.kind === 'catalog' ? t('cloud.proposals.catalogTitle') : t('cloud.proposals.recipeTitle') }}</p>
          <h2>{{ selected.title }}</h2>
          <span class="cloud-status" :class="selected.status">{{ suggestionStatus(selected.status) }}</span>
        </div>

        <div v-if="linkedGroup.length > 1" class="proposal-linked-group">
          <strong>{{ t('cloud.proposals.linkedGroup') }}</strong>
          <span v-for="item in linkedGroup" :key="item.key">{{ item.kindLabel }}: {{ item.title }}</span>
        </div>

        <dl class="proposal-facts">
          <div v-for="fact in selected.facts" :key="fact.label">
            <dt>{{ fact.label }}</dt>
            <dd>{{ fact.value }}</dd>
          </div>
        </dl>

        <section class="proposal-diff">
          <h3>{{ t('cloud.proposals.diffTitle') }}</h3>
          <div v-if="selected.kind === 'catalog'" class="proposal-diff-grid">
            <div v-for="row in catalogDiffRows(selected.raw)" :key="row.key">
              <span>{{ row.label }}</span>
              <strong>{{ row.value }}</strong>
            </div>
          </div>
          <div v-else class="recipe-change-list">
            <div v-for="change in recipeChangeRows(selected.raw)" :key="change.key" class="recipe-change-row">
              <span class="proposal-change-action">{{ change.action }}</span>
              <strong>{{ change.target }}</strong>
              <small>{{ change.meta }}</small>
            </div>
            <div v-if="recipeChangeRows(selected.raw).length === 0" class="empty-state">{{ t('cloud.proposals.noRecipeChanges') }}</div>
          </div>
        </section>

        <form class="proposal-review-form" @submit.prevent>
          <q-input v-model="reviewedByEmployeeId" dense outlined :label="t('cloud.fields.reviewed_by_employee_id')" />
          <q-input v-model="publishedBy" dense outlined :label="t('cloud.fields.published_by')" />
          <q-input v-model="reviewComment" dense outlined type="textarea" autogrow :label="t('cloud.fields.review_comment')" />
          <p class="cloud-field-hint">{{ t('cloud.proposals.reviewHint') }}</p>
          <div class="proposal-actions">
            <q-btn
              color="primary"
              unelevated
              icon="check"
              :disable="!canReview"
              :loading="ctx.isLoading('proposal-approve')"
              :label="t('cloud.proposals.approve')"
              @click="submitReview('approve')"
            />
            <q-btn
              flat
              color="negative"
              icon="close"
              :disable="!canReview"
              :loading="ctx.isLoading('proposal-reject')"
              :label="t('cloud.proposals.reject')"
              @click="submitReview('reject')"
            />
            <q-btn
              flat
              icon="edit_note"
              :disable="!canReview"
              :loading="ctx.isLoading('proposal-request_changes')"
              :label="t('cloud.proposals.requestChanges')"
              @click="submitReview('request_changes')"
            />
          </div>
        </form>

        <div class="cloud-result-box muted publication-signal">
          <strong>{{ t('cloud.proposals.publicationSignal') }}</strong>
          <span v-if="ctx.publication.value">
            {{ t('cloud.fields.version') }}: {{ ctx.publication.value.version }} ·
            {{ t('cloud.fields.published_at') }}: {{ ctx.formatDate(ctx.publication.value.published_at) }}
          </span>
          <span v-else>{{ t('cloud.empty.noPublication') }}</span>
        </div>
      </template>
    </aside>
  </section>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue';
import { useI18n } from 'vue-i18n';

import CloudSafeErrorBanner from './CloudSafeErrorBanner.vue';
import type { CatalogSuggestion, RecipeSuggestion } from '../../shared/schemas';

type ReviewAction = 'approve' | 'reject' | 'request_changes';
type ProposalKind = 'catalog' | 'recipe';
type PayloadRecord = Record<string, unknown>;

type ProposalRow = {
  key: string;
  kind: ProposalKind;
  kindLabel: string;
  id: string;
  status: string;
  groupId: string;
  title: string;
  subtitle: string;
  groupLabel: string;
  facts: { label: string; value: string }[];
  raw: CatalogSuggestion | RecipeSuggestion;
};

const props = defineProps<{
  ctx: Record<string, any>;
}>();

const { t } = useI18n();

const selectedKey = ref('');
const reviewedByEmployeeId = ref('cloud-manager');
const publishedBy = ref('cloud-ui');
const reviewComment = ref('');

const statusOptions = ['pending', 'approved', 'rejected', 'changes_requested'].map((status) => ({
  label: suggestionStatus(status),
  value: status,
}));

const catalogRows = computed(() => props.ctx.catalogSuggestions.value.map((item: CatalogSuggestion) => toCatalogRow(item)));
const recipeRows = computed(() => props.ctx.recipeSuggestions.value.map((item: RecipeSuggestion) => toRecipeRow(item)));
const allRows = computed(() => [...catalogRows.value, ...recipeRows.value]);
const selected = computed(() => allRows.value.find((item) => item.key === selectedKey.value) ?? allRows.value[0] ?? null);
const linkedGroup = computed(() => {
  if (!selected.value?.groupId) return [];
  return allRows.value.filter((item) => item.groupId === selected.value?.groupId);
});
const canReview = computed(() => Boolean(selected.value && selected.value.status === 'pending' && reviewedByEmployeeId.value.trim()));

watch(allRows, (rows) => {
  if (rows.length === 0) {
    selectedKey.value = '';
    return;
  }
  if (!rows.some((item) => item.key === selectedKey.value)) selectedKey.value = rows[0].key;
});

function toCatalogRow(item: CatalogSuggestion): ProposalRow {
  const data = payloadData(item.payload_json);
  const groupId = item.proposal_group_id || textValue(data, 'proposal_group_id');
  const name = textValue(data, 'name') || item.catalog_item_id || item.suggestion_id;
  return {
    key: `catalog:${item.id}`,
    kind: 'catalog',
    kindLabel: t('cloud.proposals.catalogTitle'),
    id: item.id,
    status: item.status,
    groupId,
    title: name,
    subtitle: `${actionLabel(item.action)} · ${props.ctx.formatDate(item.cloud_received_at)}`,
    groupLabel: groupId ? `${t('cloud.fields.proposal_group_id')}: ${shortId(groupId)}` : '',
    facts: commonFacts(item, data, [
      ['cloud.fields.kind', textValue(data, 'kind')],
      ['cloud.fields.catalog_item_id', item.catalog_item_id || textValue(data, 'catalog_item_id')],
      ['cloud.fields.applied_catalog_item_id', item.applied_catalog_item_id],
    ]),
    raw: item,
  };
}

function toRecipeRow(item: RecipeSuggestion): ProposalRow {
  const data = payloadData(item.payload_json);
  const groupId = item.proposal_group_id || textValue(data, 'proposal_group_id');
  const owner = item.owner_catalog_item_id || textValue(data, 'owner_catalog_item_id') || item.owner_catalog_suggestion_id || textValue(data, 'owner_catalog_suggestion_id');
  return {
    key: `recipe:${item.id}`,
    kind: 'recipe',
    kindLabel: t('cloud.proposals.recipeTitle'),
    id: item.id,
    status: item.status,
    groupId,
    title: owner ? `${t('cloud.fields.recipe_owner_catalog_item_id')}: ${shortId(owner)}` : item.suggestion_id,
    subtitle: `${actionLabel(item.action)} · ${props.ctx.formatDate(item.cloud_received_at)}`,
    groupLabel: groupId ? `${t('cloud.fields.proposal_group_id')}: ${shortId(groupId)}` : '',
    facts: commonFacts(item, data, [
      ['cloud.fields.recipe_version_id', item.recipe_version_id || textValue(data, 'recipe_version_id')],
      ['cloud.fields.owner_catalog_suggestion_id', item.owner_catalog_suggestion_id || textValue(data, 'owner_catalog_suggestion_id')],
      ['cloud.fields.prep_time_delta_minutes', String(item.prep_time_delta_minutes || numberValue(data, 'prep_time_delta_minutes') || 0)],
    ]),
    raw: item,
  };
}

function commonFacts(item: CatalogSuggestion | RecipeSuggestion, data: PayloadRecord, extra: [string, string][]) {
  return [
    { label: t('cloud.fields.suggestion_id'), value: safeValue(item.suggestion_id) },
    { label: t('cloud.fields.action'), value: actionLabel(item.action || textValue(data, 'action')) },
    { label: t('cloud.fields.reason'), value: safeValue(item.reason || textValue(data, 'reason')) },
    { label: t('cloud.fields.suggested_by_employee_id'), value: safeValue(textValue(data, 'suggested_by_employee_id')) },
    { label: t('cloud.fields.suggested_at'), value: props.ctx.formatDate(item.suggested_at || textValue(data, 'suggested_at')) },
    { label: t('cloud.fields.source_event_id'), value: safeValue(item.source_event_id) },
    ...extra.map(([labelKey, value]) => ({ label: t(labelKey), value: safeValue(value) })),
  ];
}

function catalogDiffRows(item: CatalogSuggestion | RecipeSuggestion) {
  const data = payloadData(item.payload_json);
  return [
    ['kind', textValue(data, 'kind')],
    ['name', textValue(data, 'name')],
    ['sku', textValue(data, 'sku')],
    ['base_unit', textValue(data, 'base_unit')],
    ['kitchen_type', textValue(data, 'kitchen_type')],
    ['accounting_category', textValue(data, 'accounting_category')],
  ].map(([key, value]) => ({
    key,
    label: t(`cloud.fields.${key}`),
    value: safeValue(value),
  }));
}

function recipeChangeRows(item: CatalogSuggestion | RecipeSuggestion) {
  const data = payloadData(item.payload_json);
  const changes = Array.isArray(data.changes) ? data.changes : [];
  return changes
    .filter((change): change is PayloadRecord => isRecord(change))
    .map((change, index) => {
      const target = textValue(change, 'to_catalog_item_id') || textValue(change, 'from_catalog_item_id') || textValue(change, 'line_id') || `${index + 1}`;
      const quantity = textValue(change, 'quantity');
      const unit = textValue(change, 'unit_code');
      const loss = textValue(change, 'loss_percent');
      const meta = [quantity && `${t('cloud.fields.quantity')}: ${quantity}`, unit && `${t('cloud.fields.unit_code')}: ${unit}`, loss && `${t('cloud.fields.loss_percent')}: ${loss}`].filter(Boolean).join(' · ');
      return {
        key: `${index}:${target}`,
        action: actionLabel(textValue(change, 'action')),
        target: shortId(target),
        meta: meta || t('cloud.proposals.noRecipeChangeMeta'),
      };
    });
}

async function submitReview(action: ReviewAction) {
  if (!selected.value || !canReview.value) return;
  await props.ctx.reviewProposalSuggestion(selected.value.kind, selected.value.id, action, {
    reviewed_by_employee_id: reviewedByEmployeeId.value.trim(),
    review_comment: reviewComment.value.trim(),
    published_by: publishedBy.value.trim(),
  });
  if (!props.ctx.errorKey.value) reviewComment.value = '';
}

function payloadData(payload: unknown): PayloadRecord {
  if (!isRecord(payload)) return {};
  const data = payload.data;
  return isRecord(data) ? data : payload;
}

function isRecord(value: unknown): value is PayloadRecord {
  return typeof value === 'object' && value !== null && !Array.isArray(value);
}

function textValue(record: PayloadRecord, key: string) {
  const value = record[key];
  return typeof value === 'string' ? value.trim() : '';
}

function numberValue(record: PayloadRecord, key: string) {
  const value = record[key];
  return typeof value === 'number' ? value : 0;
}

function safeValue(value: string) {
  if (!value) return '-';
  if (/(^|[_\s-])(payload|token|secret|pin|password|credential|sql|stack)([_\s-]|$)/i.test(value)) return t('errors.details.redacted');
  return value.length > 48 ? `${value.slice(0, 22)}...${value.slice(-12)}` : value;
}

function shortId(value: string) {
  if (!value) return '-';
  return value.length > 16 ? `${value.slice(0, 10)}...${value.slice(-4)}` : value;
}

function actionLabel(action: string) {
  const key = `cloud.proposals.actions.${action || 'unknown'}`;
  const translated = t(key);
  return translated === key ? action || '-' : translated;
}

function suggestionStatus(status: string) {
  const key = `cloud.suggestionStatuses.${status}`;
  const translated = t(key);
  return translated === key ? status : translated;
}
</script>

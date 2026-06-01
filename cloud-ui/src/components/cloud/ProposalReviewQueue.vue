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
          <span v-if="item.assignedToEmployeeId" class="proposal-group-chip assignment-chip">
            {{ t('cloud.proposals.assignedTo') }}: {{ employeeLabel(item.assignedToEmployeeId) }}
          </span>
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
          <span v-if="item.assignedToEmployeeId" class="proposal-group-chip assignment-chip">
            {{ t('cloud.proposals.assignedTo') }}: {{ employeeLabel(item.assignedToEmployeeId) }}
          </span>
        </button>
      </div>
    </div>

    <div class="cloud-panel proposal-list-panel">
      <div class="section-head">
        <div>
          <p class="eyebrow">{{ t('cloud.proposals.stopListTitle') }}</p>
          <h2>{{ t('cloud.proposals.stopListQueue') }}</h2>
        </div>
      </div>

      <div v-if="!ctx.selectedRestaurantId.value" class="empty-state wide">{{ t('cloud.empty.selectRestaurant') }}</div>
      <div v-else-if="ctx.isLoading('proposal-review')" class="cloud-skeleton-list">
        <q-skeleton v-for="index in 4" :key="index" class="skeleton-row" />
      </div>
      <div v-else-if="stopListRows.length === 0" class="empty-state">{{ t('cloud.proposals.emptyStopList') }}</div>
      <div v-else class="proposal-list">
        <button
          v-for="item in stopListRows"
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
          <span v-if="item.assignedToEmployeeId" class="proposal-group-chip assignment-chip">
            {{ t('cloud.proposals.assignedTo') }}: {{ employeeLabel(item.assignedToEmployeeId) }}
          </span>
        </button>
      </div>
    </div>

    <aside class="cloud-panel proposal-detail-panel">
      <div v-if="!selected" class="empty-state wide">{{ t('cloud.proposals.selectPrompt') }}</div>
      <template v-else>
        <div class="section-head stacked">
          <p class="eyebrow">{{ selected.kindLabel }}</p>
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

        <section class="proposal-assignment">
          <div class="proposal-section-head">
            <div>
              <p class="eyebrow">{{ t('cloud.proposals.assignmentEyebrow') }}</p>
              <h3>{{ t('cloud.proposals.assignmentTitle') }}</h3>
            </div>
            <span class="cloud-status" :class="hasAssignment ? 'approved' : 'pending'">
              {{ hasAssignment ? t('cloud.proposals.assigned') : t('cloud.proposals.unassigned') }}
            </span>
          </div>

          <dl class="proposal-facts">
            <div v-for="fact in assignmentFacts(selected.raw)" :key="fact.label">
              <dt>{{ fact.label }}</dt>
              <dd>{{ fact.value }}</dd>
            </div>
          </dl>

          <form v-if="!isTerminalReview" class="proposal-assignment-form" @submit.prevent>
            <q-select
              v-if="employeeOptions.length > 0"
              v-model="assignedToEmployeeId"
              dense
              outlined
              emit-value
              map-options
              :label="t('cloud.fields.assigned_to_employee_id')"
              :options="employeeOptions"
            />
            <q-input v-else v-model="assignedToEmployeeId" dense outlined :label="t('cloud.fields.assigned_to_employee_id')" />

            <q-select
              v-if="employeeOptions.length > 0"
              v-model="assignmentActorEmployeeId"
              dense
              outlined
              emit-value
              map-options
              :label="t('cloud.fields.assignment_actor_employee_id')"
              :options="employeeOptions"
            />
            <q-input v-else v-model="assignmentActorEmployeeId" dense outlined :label="t('cloud.fields.assignment_actor_employee_id')" />

            <q-input v-model="assignmentReason" dense outlined type="textarea" autogrow :label="t('cloud.fields.assignment_reason')" />
            <p class="cloud-field-hint">{{ t('cloud.proposals.assignmentHint') }}</p>
            <div class="proposal-actions">
              <q-btn
                color="primary"
                unelevated
                icon="assignment_ind"
                :disable="!canAssign"
                :loading="ctx.isLoading('proposal-assign')"
                :label="t('cloud.proposals.assign')"
                @click="submitAssignment"
              />
              <q-btn
                flat
                icon="person_remove"
                :disable="!canUnassign"
                :loading="ctx.isLoading('proposal-unassign')"
                :label="t('cloud.proposals.unassign')"
                @click="submitUnassignment"
              />
            </div>
          </form>
          <p v-else class="cloud-field-hint">{{ t('cloud.proposals.terminalAssignmentHint') }}</p>
        </section>

        <section class="proposal-diff">
          <h3>{{ t('cloud.proposals.diffTitle') }}</h3>
          <div v-if="selected.kind === 'catalog'" class="proposal-diff-grid">
            <div v-for="row in catalogDiffRows(selected.raw)" :key="row.key">
              <span>{{ row.label }}</span>
              <strong>{{ row.value }}</strong>
            </div>
          </div>
          <div v-else-if="selected.kind === 'recipe'" class="recipe-change-list">
            <div v-for="change in recipeChangeRows(selected.raw)" :key="change.key" class="recipe-change-row">
              <span class="proposal-change-action">{{ change.action }}</span>
              <strong>{{ change.target }}</strong>
              <small>{{ change.meta }}</small>
            </div>
            <div v-if="recipeChangeRows(selected.raw).length === 0" class="empty-state">{{ t('cloud.proposals.noRecipeChanges') }}</div>
          </div>
          <div v-else class="proposal-diff-grid">
            <div v-for="row in stopListDiffRows(selected.raw)" :key="row.key">
              <span>{{ row.label }}</span>
              <strong>{{ row.value }}</strong>
            </div>
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
import type { CatalogSuggestion, RecipeSuggestion, StopListUpdateReview } from '../../shared/schemas';

type ReviewAction = 'approve' | 'reject' | 'request_changes';
type ProposalKind = 'catalog' | 'recipe' | 'stop_list';
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
  assignedToEmployeeId: string;
  facts: { label: string; value: string }[];
  raw: CatalogSuggestion | RecipeSuggestion | StopListUpdateReview;
};

const props = defineProps<{
  ctx: Record<string, any>;
}>();

const { t } = useI18n();

const selectedKey = ref('');
const reviewedByEmployeeId = ref('cloud-manager');
const publishedBy = ref('cloud-ui');
const reviewComment = ref('');
const assignedToEmployeeId = ref('');
const assignmentActorEmployeeId = ref('cloud-manager');
const assignmentReason = ref('');

const statusOptions = ['pending', 'approved', 'rejected', 'changes_requested'].map((status) => ({
  label: suggestionStatus(status),
  value: status,
}));

const catalogRows = computed(() => props.ctx.catalogSuggestions.value.map((item: CatalogSuggestion) => toCatalogRow(item)));
const recipeRows = computed(() => props.ctx.recipeSuggestions.value.map((item: RecipeSuggestion) => toRecipeRow(item)));
const stopListRows = computed(() => props.ctx.stopListUpdateReviews.value.map((item: StopListUpdateReview) => toStopListRow(item)));
const allRows = computed(() => [...catalogRows.value, ...recipeRows.value, ...stopListRows.value]);
const selected = computed(() => allRows.value.find((item) => item.key === selectedKey.value) ?? allRows.value[0] ?? null);
const linkedGroup = computed(() => {
  if (!selected.value?.groupId) return [];
  return allRows.value.filter((item) => item.groupId === selected.value?.groupId);
});
const employeeOptions = computed(() => {
  const rows = Array.isArray(props.ctx.scopedRows?.employees) ? props.ctx.scopedRows.employees : [];
  return rows
    .filter((item: Record<string, unknown>) => String(item.status ?? 'active') === 'active')
    .map((item: Record<string, unknown>) => ({
      label: `${String(item.name ?? item.id ?? '')} (${shortId(String(item.id ?? ''))})`,
      value: String(item.id ?? ''),
    }))
    .filter((item: { value: string }) => item.value);
});
const canReview = computed(() => Boolean(selected.value && selected.value.status === 'pending' && reviewedByEmployeeId.value.trim()));
const isTerminalReview = computed(() => selected.value?.status === 'approved' || selected.value?.status === 'rejected');
const hasAssignment = computed(() => Boolean(selected.value?.assignedToEmployeeId));
const canAssign = computed(() => Boolean(selected.value && !isTerminalReview.value && assignedToEmployeeId.value.trim() && assignmentActorEmployeeId.value.trim() && assignmentReason.value.trim()));
const canUnassign = computed(() => Boolean(selected.value && !isTerminalReview.value && hasAssignment.value && assignmentActorEmployeeId.value.trim() && assignmentReason.value.trim()));

watch(allRows, (rows) => {
  if (rows.length === 0) {
    selectedKey.value = '';
    return;
  }
  if (!rows.some((item) => item.key === selectedKey.value)) selectedKey.value = rows[0].key;
});

watch(selected, (item) => {
  assignedToEmployeeId.value = item?.assignedToEmployeeId ?? '';
  assignmentReason.value = '';
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
    assignedToEmployeeId: item.assigned_to_employee_id,
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
    assignedToEmployeeId: item.assigned_to_employee_id,
    facts: commonFacts(item, data, [
      ['cloud.fields.recipe_version_id', item.recipe_version_id || textValue(data, 'recipe_version_id')],
      ['cloud.fields.owner_catalog_suggestion_id', item.owner_catalog_suggestion_id || textValue(data, 'owner_catalog_suggestion_id')],
      ['cloud.fields.prep_time_delta_minutes', String(item.prep_time_delta_minutes || numberValue(data, 'prep_time_delta_minutes') || 0)],
    ]),
    raw: item,
  };
}

function toStopListRow(item: StopListUpdateReview): ProposalRow {
  return {
    key: `stop_list:${item.id}`,
    kind: 'stop_list',
    kindLabel: t('cloud.proposals.stopListTitle'),
    id: item.id,
    status: item.status,
    groupId: '',
    title: `${t('cloud.fields.catalog_item_id')}: ${shortId(item.catalog_item_id)}`,
    subtitle: `${item.active ? t('cloud.proposals.stopListActivate') : t('cloud.proposals.stopListDeactivate')} · ${props.ctx.formatDate(item.projected_at)}`,
    groupLabel: `${t('cloud.fields.device_id')}: ${shortId(item.device_id)}`,
    assignedToEmployeeId: item.assigned_to_employee_id,
    facts: [
      { label: t('cloud.fields.source_event_id'), value: safeValue(item.id) },
      { label: t('cloud.fields.device_id'), value: safeValue(item.device_id) },
      { label: t('cloud.fields.stop_list_id'), value: safeValue(item.stop_list_id) },
      { label: t('cloud.fields.catalog_item_id'), value: safeValue(item.catalog_item_id) },
      { label: t('cloud.fields.conflict_policy'), value: safeValue(item.conflict_policy) },
      { label: t('cloud.fields.projection_action'), value: safeValue(item.projection_action) },
      { label: t('cloud.fields.reason'), value: safeValue(item.reason) },
      { label: t('cloud.fields.projected_at'), value: props.ctx.formatDate(item.projected_at) },
    ],
    raw: item,
  };
}

function assignmentFacts(item: CatalogSuggestion | RecipeSuggestion | StopListUpdateReview) {
  return [
    { label: t('cloud.fields.assigned_to_employee_id'), value: employeeLabel(item.assigned_to_employee_id) },
    { label: t('cloud.fields.assigned_by_employee_id'), value: employeeLabel(item.assigned_by_employee_id) },
    { label: t('cloud.fields.assigned_at'), value: item.assigned_at ? props.ctx.formatDate(item.assigned_at) : '-' },
    { label: t('cloud.fields.assignment_note'), value: safeValue(item.assignment_note) },
  ];
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

function stopListDiffRows(item: CatalogSuggestion | RecipeSuggestion | StopListUpdateReview) {
  if (!('projection_action' in item)) return [];
  return [
    ['active', item.active ? t('common.yes') : t('common.no')],
    ['available_quantity', item.available_quantity === null || item.available_quantity === undefined ? '-' : String(item.available_quantity)],
    ['source', item.source],
    ['warehouse_id', item.warehouse_id],
    ['applied_stop_list_id', item.applied_stop_list_id],
    ['updated_at', props.ctx.formatDate(item.updated_at)],
  ].map(([key, value]) => ({
    key,
    label: t(`cloud.fields.${key}`),
    value: safeValue(String(value ?? '')),
  }));
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

async function submitAssignment() {
  if (!selected.value || !canAssign.value) return;
  await props.ctx.assignProposalReviewItem(selected.value.kind, selected.value.id, {
    command_id: newUuidV7(),
    assigned_to_employee_id: assignedToEmployeeId.value.trim(),
    assigned_by_employee_id: assignmentActorEmployeeId.value.trim(),
    reason: assignmentReason.value.trim(),
  });
  if (!props.ctx.errorKey.value) assignmentReason.value = '';
}

async function submitUnassignment() {
  if (!selected.value || !canUnassign.value) return;
  await props.ctx.unassignProposalReviewItem(selected.value.kind, selected.value.id, {
    command_id: newUuidV7(),
    unassigned_by_employee_id: assignmentActorEmployeeId.value.trim(),
    reason: assignmentReason.value.trim(),
  });
  if (!props.ctx.errorKey.value) assignmentReason.value = '';
}

function employeeLabel(id: string) {
  if (!id) return '-';
  const rows = Array.isArray(props.ctx.scopedRows?.employees) ? props.ctx.scopedRows.employees : [];
  const found = rows.find((item: Record<string, unknown>) => String(item.id ?? '') === id);
  if (!found) return shortId(id);
  return `${String(found.name ?? id)} (${shortId(id)})`;
}

function newUuidV7() {
  const bytes: Uint8Array<ArrayBuffer> = new Uint8Array(16);
  fillRandom(bytes);
  const timestamp = Date.now();
  bytes[0] = Math.floor(timestamp / 0x10000000000) & 0xff;
  bytes[1] = Math.floor(timestamp / 0x100000000) & 0xff;
  bytes[2] = Math.floor(timestamp / 0x1000000) & 0xff;
  bytes[3] = Math.floor(timestamp / 0x10000) & 0xff;
  bytes[4] = Math.floor(timestamp / 0x100) & 0xff;
  bytes[5] = timestamp & 0xff;
  bytes[6] = (bytes[6] & 0x0f) | 0x70;
  bytes[8] = (bytes[8] & 0x3f) | 0x80;
  const hex = Array.from(bytes, (byte) => byte.toString(16).padStart(2, '0'));
  return `${hex.slice(0, 4).join('')}-${hex.slice(4, 6).join('')}-${hex.slice(6, 8).join('')}-${hex.slice(8, 10).join('')}-${hex.slice(10).join('')}`;
}

function fillRandom(bytes: Uint8Array<ArrayBuffer>) {
  if (globalThis.crypto?.getRandomValues) {
    globalThis.crypto.getRandomValues(bytes);
    return;
  }
  for (let index = 0; index < bytes.length; index += 1) {
    bytes[index] = Math.floor(Math.random() * 256);
  }
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

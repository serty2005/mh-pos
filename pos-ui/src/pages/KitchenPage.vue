<template>
  <q-page class="kitchen-page">
    <section class="kitchen-readiness">
      <div class="kitchen-readiness-head">
        <p class="eyebrow">{{ t('pos.kitchenRuntime') }}</p>
        <h1>{{ t('pos.kitchenDisplay') }}</h1>
        <p>{{ t('pos.kitchenRuntimeCopy') }}</p>
      </div>

      <PosBanner v-if="!canViewKitchen" tone="warning" :label="t('pos.kitchenPermissionViewRequired')" />
      <PosBanner v-else-if="safeError" tone="error">
        {{ t(safeError.messageKey) }}
        <span v-if="safeError.correlationId">{{ t('errors.supportCode') }}: {{ safeError.correlationId }}</span>
      </PosBanner>

      <div class="kitchen-runtime-strip" :aria-label="t('pos.kitchenRuntimeBoundary')">
        <PosStatusStrip :value="t('pos.kitchenRuntimeActive')" tone="good" />
        <PosStatusStrip :value="t(canChangeStatus ? 'pos.kitchenCommandsActive' : 'pos.kitchenCommandsNoPermission')" :tone="canChangeStatus ? 'good' : 'warning'" />
        <PosStatusStrip :value="t('pos.kitchenBackendTruth')" tone="info" />
      </div>

      <PosPanel :eyebrow="t('pos.backendContracts')" :title="t('pos.kitchenTickets')">
        <template #default>
          <div v-if="ticketsQuery.isLoading.value" class="kds-status-grid">
            <PosSkeleton v-for="status in visibleStatuses" :key="status" />
          </div>
          <PosEmptyState v-else-if="canViewKitchen && totalTickets === 0" :label="t('pos.kitchenNoTickets')" size="wide" />
          <div v-else class="kds-board">
            <section v-for="status in visibleStatuses" :key="status" class="kds-column">
              <header>
                <strong>{{ t(`pos.kdsStatuses.${status}`) }}</strong>
                <span>{{ ticketsByStatus[status]?.length ?? 0 }}</span>
              </header>

              <article v-for="ticket in ticketsByStatus[status]" :key="ticket.id" class="kds-ticket">
                <div class="kds-ticket-head">
                  <strong>{{ ticket.name }}</strong>
                  <span>{{ ticket.quantity }} {{ ticket.unit_code }}</span>
                </div>
                <p>{{ t('pos.table') }} {{ ticket.table_name }}</p>
                <p v-if="ticket.comment">{{ ticket.comment }}</p>
                <small>{{ ticket.station_routing_key || t('pos.kitchenStationUnassigned') }}</small>

                <div class="kds-ticket-actions">
                  <PosButton
                    v-for="action in actionsFor(ticket.status)"
                    :key="action"
                    dense
                    compact
                    :icon="actionIcon(action)"
                    :label="t(`pos.kitchenActions.${action}`)"
                    :disabled="!canChangeStatus"
                    :loading="pendingTicketId === ticket.id && pendingAction === action"
                    @click="() => runAction(ticket.id, action)"
                  />
                </div>
                <small v-if="!canChangeStatus" class="kitchen-muted">{{ t('pos.kitchenPermissionActionRequired') }}</small>
                <small v-else-if="actionsFor(ticket.status).length === 0" class="kitchen-muted">{{ t('pos.kitchenNoActionsForStatus') }}</small>
              </article>
            </section>
          </div>
        </template>
        <template #footer>
          <PosButton variant="secondary" mode="outline" icon="refresh" :label="t('actions.retry')" :loading="ticketsQuery.isFetching.value" @click="() => void ticketsQuery.refetch()" />
        </template>
      </PosPanel>
    </section>
  </q-page>
</template>

<script setup lang="ts">
import { useMutation, useQuery, useQueryClient } from '@tanstack/vue-query';
import { computed, ref } from 'vue';
import { useI18n } from 'vue-i18n';

import { ApiError, changeKitchenTicketStatus, listKitchenTickets, type KitchenTicketAction } from '../shared/api';
import { hasPermission, permissionCatalog } from '../shared/rbac';
import type { KitchenTicket, KitchenTicketStatus } from '../shared/schemas';
import { PosBanner, PosButton, PosEmptyState, PosPanel, PosSkeleton, PosStatusStrip } from '../shared/ui';
import { useAuthStore } from '../stores/auth';

const { t } = useI18n();
const auth = useAuthStore();
const queryClient = useQueryClient();

const pendingTicketId = ref('');
const pendingAction = ref<KitchenTicketAction | ''>('');

const grantedPermissions = computed(() => auth.actor?.permissions ?? []);
const canViewKitchen = computed(() => hasPermission(grantedPermissions.value, permissionCatalog.kitchenView));
const canChangeStatus = computed(() => hasPermission(grantedPermissions.value, permissionCatalog.kitchenStatusChange));

const visibleStatuses: KitchenTicketStatus[] = ['new', 'accepted', 'in_progress', 'hold', 'ready', 'recall', 'served', 'cancelled'];

const ticketsQuery = useQuery({
  queryKey: ['kitchen-tickets', auth.sessionId, auth.nodeDeviceId],
  queryFn: () => listKitchenTickets({ limit: 100, offset: 0 }),
  enabled: () => Boolean(auth.sessionId && auth.nodeDeviceId && canViewKitchen.value),
  refetchInterval: 10_000,
});

const ticketsByStatus = computed(() => {
  const groups = visibleStatuses.reduce((acc, status) => {
    acc[status] = [];
    return acc;
  }, {} as Record<KitchenTicketStatus, KitchenTicket[]>);
  for (const ticket of ticketsQuery.data.value ?? []) {
    groups[ticket.status].push(ticket);
  }
  return groups;
});

const totalTickets = computed(() => ticketsQuery.data.value?.length ?? 0);
const actionMutation = useMutation({
  mutationFn: ({ ticketId, action }: { ticketId: string; action: KitchenTicketAction }) => changeKitchenTicketStatus(ticketId, action),
  onMutate({ ticketId, action }) {
    pendingTicketId.value = ticketId;
    pendingAction.value = action;
  },
  async onSettled() {
    pendingTicketId.value = '';
    pendingAction.value = '';
    await queryClient.invalidateQueries({ queryKey: ['kitchen-tickets'] });
  },
});

const safeError = computed(() => {
  if (ticketsQuery.error.value instanceof ApiError) return ticketsQuery.error.value;
  if (actionMutation.error.value instanceof ApiError) return actionMutation.error.value;
  return null;
});

function actionsFor(status: KitchenTicketStatus): KitchenTicketAction[] {
  switch (status) {
    case 'new':
      return ['accept', 'cancel'];
    case 'accepted':
      return ['start', 'hold', 'cancel'];
    case 'in_progress':
      return ['hold', 'ready', 'cancel'];
    case 'hold':
      return ['start', 'cancel'];
    case 'ready':
      return ['serve', 'recall'];
    case 'recall':
      return ['start', 'cancel'];
    default:
      return [];
  }
}

function actionIcon(action: KitchenTicketAction) {
  switch (action) {
    case 'accept':
      return 'done';
    case 'start':
      return 'play_arrow';
    case 'hold':
      return 'pause';
    case 'ready':
      return 'room_service';
    case 'serve':
      return 'task_alt';
    case 'recall':
      return 'undo';
    case 'cancel':
      return 'close';
    default:
      return 'chevron_right';
  }
}

function runAction(ticketId: string, action: KitchenTicketAction) {
  if (!canChangeStatus.value || actionMutation.isPending.value) return;
  actionMutation.mutate({ ticketId, action });
}
</script>

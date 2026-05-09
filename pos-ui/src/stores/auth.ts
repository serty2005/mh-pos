import { defineStore } from 'pinia';

import { getClientDeviceId } from '../shared/clientIdentity';
import type { ActorContext, AuthSession, PairingStatus, ProvisioningStatus } from '../shared/schemas';

type AuthState = {
  clientDeviceId: string;
  nodeDeviceId: string;
  restaurantId: string;
  sessionId: string;
  actor: ActorContext | null;
};

const nodeKey = 'mh-pos.node_device_id';
const restaurantKey = 'mh-pos.restaurant_id';
const sessionKey = 'mh-pos.session_id';

export const useAuthStore = defineStore('auth', {
  state: (): AuthState => ({
    clientDeviceId: getClientDeviceId(),
    nodeDeviceId: localStorage.getItem(nodeKey) ?? '',
    restaurantId: localStorage.getItem(restaurantKey) ?? '',
    sessionId: localStorage.getItem(sessionKey) ?? '',
    actor: null,
  }),
  actions: {
    applyPairing(status: PairingStatus) {
      if (!status.paired || !status.node_device_id || !status.restaurant_id) {
        this.nodeDeviceId = '';
        this.restaurantId = '';
        localStorage.removeItem(nodeKey);
        localStorage.removeItem(restaurantKey);
        return;
      }
      this.nodeDeviceId = status.node_device_id;
      this.restaurantId = status.restaurant_id;
      localStorage.setItem(nodeKey, status.node_device_id);
      localStorage.setItem(restaurantKey, status.restaurant_id);
    },
    applyProvisioning(status: ProvisioningStatus) {
      if (!status.paired || !status.node_device_id || !status.restaurant_id) {
        this.nodeDeviceId = status.node_device_id ?? '';
        if (this.nodeDeviceId) {
          localStorage.setItem(nodeKey, this.nodeDeviceId);
        }
        return;
      }
      this.nodeDeviceId = status.node_device_id;
      this.restaurantId = status.restaurant_id;
      localStorage.setItem(nodeKey, status.node_device_id);
      localStorage.setItem(restaurantKey, status.restaurant_id);
    },
    applySession(session: AuthSession, actor: ActorContext | null) {
      if (session.status !== 'active') {
        this.clearSession();
        return;
      }
      this.sessionId = session.id;
      this.actor = actor;
      localStorage.setItem(sessionKey, session.id);
    },
    clearSession() {
      this.sessionId = '';
      this.actor = null;
      localStorage.removeItem(sessionKey);
    },
  },
});

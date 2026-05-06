import { z } from 'zod';

import { useAuthStore } from '../stores/auth';
import {
	hallSchema,
	pairingStatusSchema,
	pinLoginResultSchema,
	tableSchema,
	type PinLoginResult,
} from './schemas';

const apiBase = (import.meta.env.VITE_POS_API_BASE ?? 'http://localhost:8080/api/v1').replace(/\/$/, '');

class ApiError extends Error {
  status: number;

  constructor(status: number, message: string) {
    super(message);
    this.status = status;
  }
}

async function request<T>(path: string, schema: z.ZodType<T>, init: RequestInit = {}) {
  const auth = useAuthStore();
  const headers = new Headers(init.headers);
  headers.set('Content-Type', 'application/json');
  headers.set('X-Client-Device-ID', auth.clientDeviceId);
  if (auth.nodeDeviceId) headers.set('X-Node-Device-ID', auth.nodeDeviceId);
  if (auth.sessionId) headers.set('X-Session-ID', auth.sessionId);
  if (auth.actor?.employee_id) headers.set('X-Actor-Employee-ID', auth.actor.employee_id);

  const response = await fetch(`${apiBase}${path}`, { ...init, headers });
  const text = await response.text();
  const data = text ? JSON.parse(text) : null;
  if (!response.ok) {
    throw new ApiError(response.status, data?.error ?? response.statusText);
  }
  return schema.parse(data);
}

export function getPairingStatus() {
  return request('/system/pairing-status', pairingStatusSchema);
}

export async function pairEdgeNodeAndRefresh(pairingCode: string) {
  await request('/system/pair', z.unknown(), {
    method: 'POST',
    body: JSON.stringify({ pairing_code: pairingCode }),
  });
  return getPairingStatus();
}

export function pinLogin(pin: string) {
  const auth = useAuthStore();
  return request<PinLoginResult>('/auth/pin-login', pinLoginResultSchema, {
    method: 'POST',
    body: JSON.stringify({
      node_device_id: auth.nodeDeviceId,
      client_device_id: auth.clientDeviceId,
      pin,
    }),
  });
}

export function getAuthSession() {
  const auth = useAuthStore();
  const query = new URLSearchParams({
    session_id: auth.sessionId,
    node_device_id: auth.nodeDeviceId,
    client_device_id: auth.clientDeviceId,
  });
  return request(`/auth/session?${query}`, pinLoginResultSchema);
}

export function logout() {
  const auth = useAuthStore();
  return request('/auth/logout', z.unknown(), {
    method: 'POST',
    body: JSON.stringify({
      node_device_id: auth.nodeDeviceId,
      client_device_id: auth.clientDeviceId,
      session_id: auth.sessionId,
    }),
  });
}

export function listHalls(restaurantId: string) {
  return request(`/halls?restaurant_id=${encodeURIComponent(restaurantId)}`, z.array(hallSchema));
}

export function listTables(restaurantId: string, hallId: string) {
  const query = new URLSearchParams({ restaurant_id: restaurantId, hall_id: hallId });
  return request(`/tables?${query}`, z.array(tableSchema));
}

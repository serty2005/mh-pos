import { z } from 'zod';

import { useAuthStore } from '../stores/auth';
import {
	cashSessionSchema,
	checkSchema,
	hallSchema,
	menuItemSchema,
	orderLineSchema,
	orderSchema,
	pairingStatusSchema,
	paymentSchema,
	pinLoginResultSchema,
	precheckSchema,
	shiftSchema,
	tableSchema,
	type PinLoginResult,
} from './schemas';

const apiBase = (import.meta.env.VITE_POS_API_BASE ?? 'http://localhost:8080/api/v1').replace(/\/$/, '');

export class ApiError extends Error {
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

async function requestOptional<T>(path: string, schema: z.ZodType<T>, init: RequestInit = {}) {
  try {
    return await request(path, schema, init);
  } catch (error) {
    if (error instanceof ApiError && error.status === 404) {
      return null;
    }
    throw error;
  }
}

function actorId() {
  const auth = useAuthStore();
  return auth.actor?.employee_id ?? '';
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

export function getCurrentShift() {
  const auth = useAuthStore();
  const query = new URLSearchParams({ device_id: auth.nodeDeviceId });
  return requestOptional(`/shifts/current?${query}`, shiftSchema);
}

export function openShift(openingCashAmount: number) {
  const auth = useAuthStore();
  return request('/shifts/open', shiftSchema, {
    method: 'POST',
    body: JSON.stringify({
      restaurant_id: auth.restaurantId,
      opened_by_employee_id: actorId(),
      opening_cash_amount: openingCashAmount,
    }),
  });
}

export function closeShift(shiftId: string, closingCashAmount: number) {
  return request(`/shifts/${encodeURIComponent(shiftId)}/close`, shiftSchema, {
    method: 'POST',
    body: JSON.stringify({
      closed_by_employee_id: actorId(),
      closing_cash_amount: closingCashAmount,
    }),
  });
}

export function getCurrentCashSession() {
  const auth = useAuthStore();
  const query = new URLSearchParams({ device_id: auth.nodeDeviceId });
  return requestOptional(`/cash-sessions/current?${query}`, cashSessionSchema);
}

export function openCashSession(openingCashAmount: number) {
  const auth = useAuthStore();
  return request('/cash-sessions/open', cashSessionSchema, {
    method: 'POST',
    body: JSON.stringify({
      restaurant_id: auth.restaurantId,
      opened_by_employee_id: actorId(),
      opening_cash_amount: openingCashAmount,
    }),
  });
}

export function closeCashSession(cashSessionId: string, closingCashAmount: number) {
  return request(`/cash-sessions/${encodeURIComponent(cashSessionId)}/close`, cashSessionSchema, {
    method: 'POST',
    body: JSON.stringify({
      closed_by_employee_id: actorId(),
      closing_cash_amount: closingCashAmount,
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

export function listMenuItems() {
  return request('/menu/items', z.array(menuItemSchema));
}

export function getCurrentOrderByTable(tableId: string) {
  const query = new URLSearchParams({ table_id: tableId });
  return requestOptional(`/orders/current?${query}`, orderSchema);
}

export function getOrder(orderId: string) {
  return request(`/orders/${encodeURIComponent(orderId)}`, orderSchema);
}

export function createOrder(tableId: string, tableName: string, guestCount: number) {
  const auth = useAuthStore();
  return request('/orders', orderSchema, {
    method: 'POST',
    body: JSON.stringify({
      restaurant_id: auth.restaurantId,
      table_id: tableId,
      table_name: tableName,
      guest_count: guestCount,
    }),
  });
}

export function addOrderLine(orderId: string, menuItemId: string, quantity = 1) {
  return request(`/orders/${encodeURIComponent(orderId)}/lines`, orderLineSchema, {
    method: 'POST',
    body: JSON.stringify({
      menu_item_id: menuItemId,
      quantity,
    }),
  });
}

export function changeOrderLineQuantity(orderId: string, lineId: string, quantity: number) {
  return request(`/orders/${encodeURIComponent(orderId)}/lines/${encodeURIComponent(lineId)}`, orderLineSchema, {
    method: 'PATCH',
    body: JSON.stringify({ quantity }),
  });
}

export function voidOrderLine(orderId: string, lineId: string, reason: string) {
  return request(`/orders/${encodeURIComponent(orderId)}/lines/${encodeURIComponent(lineId)}/void`, orderLineSchema, {
    method: 'POST',
    body: JSON.stringify({ reason }),
  });
}

export function listPrechecksByOrder(orderId: string) {
  return request(`/orders/${encodeURIComponent(orderId)}/prechecks`, z.array(precheckSchema));
}

export function issuePrecheck(orderId: string) {
  return request(`/orders/${encodeURIComponent(orderId)}/precheck`, precheckSchema, {
    method: 'POST',
    body: JSON.stringify({}),
  });
}

export function cancelPrecheck(precheckId: string, managerEmployeeId: string, managerPin: string, cancellationReason: string) {
  return request(`/prechecks/${encodeURIComponent(precheckId)}/cancel`, precheckSchema, {
    method: 'POST',
    body: JSON.stringify({
      manager_employee_id: managerEmployeeId,
      manager_pin: managerPin,
      cancellation_reason: cancellationReason,
    }),
  });
}

export function capturePrecheckPayment(precheckId: string, method: 'cash' | 'card', amount: number, currency: string) {
  return request(`/prechecks/${encodeURIComponent(precheckId)}/payments`, paymentSchema, {
    method: 'POST',
    body: JSON.stringify({
      method,
      amount,
      currency,
      provider_name: method === 'card' ? 'trusted_manual' : undefined,
    }),
  });
}

export function getCheck(checkId: string) {
  return request(`/checks/${encodeURIComponent(checkId)}`, checkSchema);
}

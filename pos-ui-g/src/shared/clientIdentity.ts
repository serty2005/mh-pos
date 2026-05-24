const clientDeviceKey = 'mh-pos-g.client_device_id';

export function getClientDeviceId(): string {
  const existing = localStorage.getItem(clientDeviceKey);
  if (existing) return existing;
  const next = typeof crypto !== 'undefined' && 'randomUUID' in crypto
    ? crypto.randomUUID()
    : `client-${Date.now()}-${Math.random().toString(16).slice(2)}`;
  localStorage.setItem(clientDeviceKey, next);
  return next;
}

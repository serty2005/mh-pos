const storageKey = 'mh-pos.client_device_id';

export function getClientDeviceId() {
  const existing = localStorage.getItem(storageKey);
  if (existing) {
    return existing;
  }
  const id = crypto.randomUUID();
  localStorage.setItem(storageKey, id);
  return id;
}

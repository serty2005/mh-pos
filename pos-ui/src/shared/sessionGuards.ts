export type SessionGuardInput = {
  nodeDeviceId: string;
  sessionId: string;
};

/**
 * Выбирает fallback для protected POS route, когда нет device pairing или auth session.
 */
export function resolveProtectedPosFallback(input: SessionGuardInput): '/pair' | '/login' | null {
  if (!input.nodeDeviceId) return '/pair';
  if (!input.sessionId) return '/login';
  return null;
}

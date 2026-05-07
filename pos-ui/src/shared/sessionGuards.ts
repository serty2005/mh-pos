export type SessionGuardInput = {
  nodeDeviceId: string;
  sessionId: string;
};

/**
 * Resolve protected POS route fallback when device pairing or auth session is missing.
 */
export function resolveProtectedPosFallback(input: SessionGuardInput): '/pair' | '/login' | null {
  if (!input.nodeDeviceId) return '/pair';
  if (!input.sessionId) return '/login';
  return null;
}

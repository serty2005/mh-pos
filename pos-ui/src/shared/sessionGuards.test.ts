import { describe, expect, it } from 'vitest';

import { resolveProtectedPosFallback } from './sessionGuards';

describe('resolveProtectedPosFallback', () => {
  it('returns /pair when node_device_id is missing', () => {
    expect(resolveProtectedPosFallback({ nodeDeviceId: '', sessionId: 'session-1' })).toBe('/pair');
  });

  it('returns /login when session_id is missing', () => {
    expect(resolveProtectedPosFallback({ nodeDeviceId: 'node-1', sessionId: '' })).toBe('/login');
  });

  it('returns null when pairing and session are present', () => {
    expect(resolveProtectedPosFallback({ nodeDeviceId: 'node-1', sessionId: 'session-1' })).toBeNull();
  });
});

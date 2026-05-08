import { afterEach, describe, expect, it, vi } from 'vitest';

import { createClientDeviceId } from './clientIdentity';

describe('createClientDeviceId', () => {
  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it('uses crypto.randomUUID when available', () => {
    const randomUUID = vi.fn(() => 'fixed-client-device-id');
    vi.stubGlobal('crypto', { randomUUID });

    expect(createClientDeviceId()).toBe('fixed-client-device-id');
    expect(randomUUID).toHaveBeenCalledOnce();
  });

  it('falls back to crypto.getRandomValues UUID v4 generation', () => {
    vi.stubGlobal('crypto', {
      getRandomValues: (bytes: Uint8Array) => {
        bytes.set(Array.from({ length: bytes.length }, (_, index) => index));
        return bytes;
      },
    });

    expect(createClientDeviceId()).toBe('00010203-0405-4607-8809-0a0b0c0d0e0f');
  });
});

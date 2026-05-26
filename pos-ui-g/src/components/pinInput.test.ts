import { describe, expect, it } from 'vitest';

import { appendPinDigit, canSubmitPin, maxPinLength, minPinLength } from './pinInput';

describe('PIN input helpers', () => {
  it('allows seed PINs longer than the minimum length', () => {
    expect(appendPinDigit('1234', '5')).toBe('12345');
  });

  it('caps entered PINs at the supported maximum length', () => {
    const fullPin = '1'.repeat(maxPinLength);

    expect(appendPinDigit(fullPin, '2')).toBe(fullPin);
  });

  it('submits only PINs that reached the minimum length', () => {
    expect(canSubmitPin('1'.repeat(minPinLength - 1))).toBe(false);
    expect(canSubmitPin('1'.repeat(minPinLength))).toBe(true);
  });
});

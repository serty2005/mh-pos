import { describe, expect, it } from 'vitest';

import { appendPinDigit, canSubmitPin, maxPinLength, minPinLength, shouldAttemptPinLogin } from './pinInput';

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

  it('auto-attempts PIN login only for a fresh complete value', () => {
    const completePin = '1'.repeat(minPinLength);

    expect(shouldAttemptPinLogin('1'.repeat(minPinLength - 1), '', false)).toBe(false);
    expect(shouldAttemptPinLogin(completePin, completePin, false)).toBe(false);
    expect(shouldAttemptPinLogin(completePin, '', true)).toBe(false);
    expect(shouldAttemptPinLogin(completePin, '', false)).toBe(true);
  });
});

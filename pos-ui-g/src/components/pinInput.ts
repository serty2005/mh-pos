export const minPinLength = 4;
export const maxPinLength = 12;

export function appendPinDigit(pin: string, digit: string) {
  if (pin.length >= maxPinLength) return pin;
  return `${pin}${digit}`;
}

export function canSubmitPin(pin: string) {
  return pin.length >= minPinLength;
}

export function shouldAttemptPinLogin(pin: string, lastAttempt: string, isSubmitting: boolean) {
  return canSubmitPin(pin) && !isSubmitting && lastAttempt !== pin;
}

export function pinIndicatorCount(pin: string) {
  return Math.max(minPinLength, Math.min(pin.length, maxPinLength));
}

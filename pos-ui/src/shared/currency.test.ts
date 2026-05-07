import { describe, expect, it } from 'vitest';

import {
  currencyInputStep,
  currencyMinorUnit,
  formatMinorCurrency,
  minorToMoney,
  moneyToMinor,
  resolveCurrencyProfile,
} from './currency';

describe('currency precision helpers', () => {
  it('uses 2-decimal precision for RUB', () => {
    expect(currencyMinorUnit('rub')).toBe(2);
    expect(moneyToMinor(12.34, 'RUB')).toBe(1234);
    expect(minorToMoney(1234, 'RUB')).toBe(12.34);
    expect(currencyInputStep('RUB')).toBe('0.01');
  });

  it('uses 3-decimal precision for KWD', () => {
    expect(currencyMinorUnit('KWD')).toBe(3);
    expect(moneyToMinor(1.234, 'KWD')).toBe(1234);
    expect(minorToMoney(1234, 'KWD')).toBe(1.234);
    expect(currencyInputStep('KWD')).toBe('0.001');
  });

  it('falls back to 2-decimal precision for unknown currencies', () => {
    const profile = resolveCurrencyProfile('zzz');
    expect(profile.alphaCode).toBe('ZZZ');
    expect(profile.minorUnit).toBe(2);
    expect(moneyToMinor(9.99, 'ZZZ')).toBe(999);
  });

  it('formats minor units using Intl currency formatter', () => {
    const formatted = formatMinorCurrency(1234, 'RUB', 'ru-RU');
    expect(formatted.length).toBeGreaterThan(0);
    expect(formatted).toContain('₽');
  });
});

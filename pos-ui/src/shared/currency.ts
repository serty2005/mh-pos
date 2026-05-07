export type CurrencyProfile = {
  alphaCode: string;
  minorUnit: number;
};

const currencyProfiles: Record<string, CurrencyProfile> = {
  RUB: { alphaCode: 'RUB', minorUnit: 2 },
  USD: { alphaCode: 'USD', minorUnit: 2 },
  EUR: { alphaCode: 'EUR', minorUnit: 2 },
  KZT: { alphaCode: 'KZT', minorUnit: 2 },
  BYN: { alphaCode: 'BYN', minorUnit: 2 },
  UAH: { alphaCode: 'UAH', minorUnit: 2 },
  BHD: { alphaCode: 'BHD', minorUnit: 3 },
  JOD: { alphaCode: 'JOD', minorUnit: 3 },
  KWD: { alphaCode: 'KWD', minorUnit: 3 },
  OMR: { alphaCode: 'OMR', minorUnit: 3 },
  TND: { alphaCode: 'TND', minorUnit: 3 },
};

/**
 * Returns canonical ISO 4217 alpha code profile.
 * Falls back to 2-decimal precision for unknown currencies to keep UI resilient.
 */
export function resolveCurrencyProfile(code: string): CurrencyProfile {
  const normalized = code.trim().toUpperCase();
  return currencyProfiles[normalized] ?? { alphaCode: normalized || 'RUB', minorUnit: 2 };
}

export function currencyMinorUnit(code: string): number {
  return resolveCurrencyProfile(code).minorUnit;
}

export function moneyToMinor(value: number, code: string): number {
  const factor = 10 ** currencyMinorUnit(code);
  const safeValue = Number.isFinite(value) ? value : 0;
  return Math.round(safeValue * factor);
}

export function minorToMoney(value: number, code: string): number {
  const factor = 10 ** currencyMinorUnit(code);
  const safeValue = Number.isFinite(value) ? value : 0;
  return Math.round(safeValue) / factor;
}

export function currencyInputStep(code: string): string {
  const minorUnit = currencyMinorUnit(code);
  if (minorUnit <= 0) {
    return '1';
  }
  return `0.${'0'.repeat(Math.max(minorUnit-1, 0))}1`;
}

export function formatMinorCurrency(value: number, code: string, locale = 'ru-RU'): string {
  const profile = resolveCurrencyProfile(code);
  return new Intl.NumberFormat(locale, { style: 'currency', currency: profile.alphaCode }).format(minorToMoney(value, profile.alphaCode));
}

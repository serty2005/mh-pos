export type CurrencyProfile = {
  alphaCode: string;
  minorUnit: number;
};

const minorUnitByAlphaCode = new Map<string, number>();

/**
 * Возвращает canonical profile для ISO 4217 alpha code.
 * Для неизвестных или неподдержанных окружением кодов оставляет fallback в 2 знака.
 */
export function resolveCurrencyProfile(code: string): CurrencyProfile {
  const alphaCode = normalizeCurrencyCode(code);
  return { alphaCode, minorUnit: detectMinorUnit(alphaCode) };
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

function normalizeCurrencyCode(code: string): string {
  const normalized = code.trim().toUpperCase();
  if (normalized.length === 3) {
    return normalized;
  }
  return 'RUB';
}

function detectMinorUnit(alphaCode: string): number {
  const cached = minorUnitByAlphaCode.get(alphaCode);
  if (cached !== undefined) {
    return cached;
  }
  let minorUnit = 2;
  try {
    const formatter = new Intl.NumberFormat('en', { style: 'currency', currency: alphaCode });
    minorUnit = formatter.resolvedOptions().maximumFractionDigits ?? 2;
  } catch {
    minorUnit = 2;
  }
  if (!Number.isInteger(minorUnit) || minorUnit < 0 || minorUnit > 4) {
    minorUnit = 2;
  }
  minorUnitByAlphaCode.set(alphaCode, minorUnit);
  return minorUnit;
}

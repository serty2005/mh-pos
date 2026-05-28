import { ru } from './ru';

const messages = {
  ru,
};

type Locale = keyof typeof messages;
const locale: Locale = 'ru';

export function t(key: string): string {
  const segments = key.split('.');
  let current: unknown = messages[locale];

  for (const segment of segments) {
    if (!current || typeof current !== 'object') {
      return key;
    }
    current = (current as Record<string, unknown>)[segment];
  }

  return typeof current === 'string' ? current : key;
}

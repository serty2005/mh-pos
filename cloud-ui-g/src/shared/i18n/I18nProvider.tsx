import { createContext, useContext, type ReactNode } from 'react';
import { ru } from './ru';

type Locale = 'ru';
type Messages = typeof ru;

const messages: Record<Locale, Messages> = { ru };

type I18nContextValue = {
  locale: Locale;
  t: (key: string) => string;
};

const I18nContext = createContext<I18nContextValue | null>(null);

function translate(locale: Locale, key: string): string {
  const segments = key.split('.');
  let current: unknown = messages[locale];

  for (const segment of segments) {
    if (!current || typeof current !== 'object') return key;
    current = (current as Record<string, unknown>)[segment];
  }

  return typeof current === 'string' ? current : key;
}

export function I18nProvider({ children }: { children: ReactNode }) {
  const locale: Locale = 'ru';

  return (
    <I18nContext.Provider
      value={{
        locale,
        t: (key) => translate(locale, key),
      }}
    >
      {children}
    </I18nContext.Provider>
  );
}

export function useI18n() {
  const context = useContext(I18nContext);
  if (!context) {
    throw new Error('I18nProvider is not mounted');
  }
  return context;
}

import { beforeEach, describe, expect, it, vi } from 'vitest';

import {
  applyPOSTheme,
  defaultThemeSettings,
  posThemeSchemes,
  readStoredThemeSettings,
  writeStoredThemeSettings,
} from './theme';

describe('POS theme settings', () => {
  beforeEach(() => {
    const classes = new Set<string>();
    const storage = new Map<string, string>();

    vi.stubGlobal('document', {
      documentElement: {
        dataset: {} as Record<string, string>,
        classList: {
          contains: (className: string) => classes.has(className),
          toggle: (className: string, force?: boolean) => {
            if (force) {
              classes.add(className);
              return true;
            }
            classes.delete(className);
            return false;
          },
        },
      },
    });
    vi.stubGlobal('localStorage', {
      getItem: (key: string) => storage.get(key) ?? null,
      setItem: (key: string, value: string) => storage.set(key, value),
      clear: () => storage.clear(),
    });
  });

  it('applies dark mode and selected color scheme to the document root', () => {
    applyPOSTheme({ mode: 'dark', scheme: 'terracotta-graphite' });

    expect(document.documentElement.classList.contains('dark')).toBe(true);
    expect(document.documentElement.dataset.themeScheme).toBe('terracotta-graphite');
  });

  it('keeps noir gold as the default scheme and dark mode as the default appearance', () => {
    const settings = readStoredThemeSettings();

    expect(settings).toEqual(defaultThemeSettings);
    expect(posThemeSchemes.map((scheme) => scheme.id)).toEqual([
      'noir-gold',
      'sage-brass',
      'terracotta-graphite',
      'baltic-blue',
    ]);
  });

  it('ignores unsupported stored values and keeps a safe default', () => {
    localStorage.setItem('mh-pos.theme_mode', 'solarized');
    localStorage.setItem('mh-pos.theme_scheme', 'unknown');

    expect(readStoredThemeSettings()).toEqual(defaultThemeSettings);
  });

  it('persists selected mode and color scheme for the next POS session', () => {
    writeStoredThemeSettings({ mode: 'light', scheme: 'sage-brass' });

    expect(localStorage.getItem('mh-pos.theme_mode')).toBe('light');
    expect(localStorage.getItem('mh-pos.theme_scheme')).toBe('sage-brass');
    expect(readStoredThemeSettings()).toEqual({ mode: 'light', scheme: 'sage-brass' });
  });
});

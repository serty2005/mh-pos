export type POSThemeMode = 'light' | 'dark';

export type POSThemeSchemeId = 'noir-gold' | 'sage-brass' | 'terracotta-graphite' | 'baltic-blue';

export interface POSThemeSettings {
  mode: POSThemeMode;
  scheme: POSThemeSchemeId;
}

export interface POSThemeScheme {
  id: POSThemeSchemeId;
  label: string;
  accent: string;
}

const themeModeStorageKey = 'mh-pos.theme_mode';
const themeSchemeStorageKey = 'mh-pos.theme_scheme';

export const defaultThemeSettings: POSThemeSettings = {
  mode: 'dark',
  scheme: 'noir-gold',
};

export const posThemeSchemes: POSThemeScheme[] = [
  { id: 'noir-gold', label: 'Noir Gold', accent: '#C5A059' },
  { id: 'sage-brass', label: 'Sage Brass', accent: '#6A8B67' },
  { id: 'terracotta-graphite', label: 'Terracotta', accent: '#C85F36' },
  { id: 'baltic-blue', label: 'Baltic Blue', accent: '#227B91' },
];

const posThemeSchemeIds = new Set<POSThemeSchemeId>(posThemeSchemes.map((scheme) => scheme.id));

function isThemeMode(value: string | null): value is POSThemeMode {
  return value === 'light' || value === 'dark';
}

function isThemeSchemeId(value: string | null): value is POSThemeSchemeId {
  return Boolean(value && posThemeSchemeIds.has(value as POSThemeSchemeId));
}

export function applyPOSTheme(settings: POSThemeSettings): void {
  document.documentElement.classList.toggle('dark', settings.mode === 'dark');
  document.documentElement.dataset.themeScheme = settings.scheme;
}

export function readStoredThemeSettings(): POSThemeSettings {
  const storedMode = localStorage.getItem(themeModeStorageKey);
  const storedScheme = localStorage.getItem(themeSchemeStorageKey);

  return {
    mode: isThemeMode(storedMode) ? storedMode : defaultThemeSettings.mode,
    scheme: isThemeSchemeId(storedScheme) ? storedScheme : defaultThemeSettings.scheme,
  };
}

export function writeStoredThemeSettings(settings: POSThemeSettings): void {
  localStorage.setItem(themeModeStorageKey, settings.mode);
  localStorage.setItem(themeSchemeStorageKey, settings.scheme);
}

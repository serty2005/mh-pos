export type PosButtonVariant = 'primary' | 'secondary' | 'danger' | 'neutral';
export type PosButtonMode = 'filled' | 'outline' | 'flat';
export type PosSize = 'compact' | 'regular' | 'large';
export type PosTone = 'neutral' | 'good' | 'warning' | 'danger' | 'info' | 'primary';
export type PosActionMode = PosButtonMode;
export type PosStatusTone = PosTone;

export interface PosButtonClassOptions {
  primary?: boolean;
  compact?: boolean;
}

export function posButtonColor(variant: PosButtonVariant): string {
  if (variant === 'primary') return 'primary';
  if (variant === 'danger') return 'negative';
  if (variant === 'secondary') return 'secondary';
  return 'grey-8';
}

export function posButtonModeProps(mode: PosButtonMode) {
  return {
    unelevated: mode === 'filled',
    outline: mode === 'outline',
    flat: mode === 'flat',
  };
}

export function posButtonClasses(options: PosButtonClassOptions = {}): string[] {
  return [
    'touch-button',
    options.primary ? 'primary-action' : '',
    options.compact ? 'touch-button-compact' : '',
  ].filter(Boolean);
}

export function posToneClasses(baseClass: string, tone: PosStatusTone): string[] {
  return [
    baseClass,
    tone === 'good' ? 'good' : '',
    tone === 'warning' ? 'warning' : '',
    tone === 'danger' ? 'danger' : '',
    tone === 'info' ? 'info' : '',
    tone === 'primary' ? 'primary' : '',
  ].filter(Boolean);
}

export function posSizeClass(size: PosSize, compactClass = 'compact', largeClass = 'large'): string {
  if (size === 'compact') return compactClass;
  if (size === 'large') return largeClass;
  return '';
}

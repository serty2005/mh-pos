import { describe, expect, it } from 'vitest';

import { posButtonClasses, posButtonColor, posButtonModeProps, posSizeClass, posToneClasses } from './uiTypes';

describe('POS UI primitive helpers', () => {
  it('maps button variants to Quasar colors', () => {
    expect(posButtonColor('primary')).toBe('primary');
    expect(posButtonColor('secondary')).toBe('secondary');
    expect(posButtonColor('danger')).toBe('negative');
    expect(posButtonColor('neutral')).toBe('grey-8');
  });

  it('maps button mode props without mixing visual modes', () => {
    expect(posButtonModeProps('filled')).toEqual({ unelevated: true, outline: false, flat: false });
    expect(posButtonModeProps('outline')).toEqual({ unelevated: false, outline: true, flat: false });
    expect(posButtonModeProps('flat')).toEqual({ unelevated: false, outline: false, flat: true });
  });

  it('keeps shared button and tone classes stable', () => {
    expect(posButtonClasses({ primary: true })).toEqual(['touch-button', 'primary-action']);
    expect(posToneClasses('status-strip', 'warning')).toEqual(['status-strip', 'warning']);
  });

  it('maps shared POS sizes to stable class names', () => {
    expect(posSizeClass('compact')).toBe('compact');
    expect(posSizeClass('regular')).toBe('');
    expect(posSizeClass('large')).toBe('large');
  });
});

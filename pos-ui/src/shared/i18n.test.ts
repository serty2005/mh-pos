import { describe, expect, it } from 'vitest';

import { i18n } from './i18n';

describe('i18n messages', () => {
  it('contains POS API error category and transport keys in ru locale', () => {
    const keys = [
      'errors.conflict',
      'errors.not_found',
      'errors.permission',
      'errors.validation',
      'errors.server',
      'errors.network.unavailable',
      'errors.network.timeout',
      'errors.response.invalid',
      'errors.conflict_active_precheck',
      'errors.conflict_duplicate_command',
      'errors.conflict_duplicate_pin',
      'errors.stopListConflict',
    ];

    for (const key of keys) {
      expect(i18n.global.te(key), key).toBe(true);
      expect(i18n.global.t(key), key).not.toBe(key);
    }
  });
});

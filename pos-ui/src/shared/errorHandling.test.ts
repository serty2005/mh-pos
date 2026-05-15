import { describe, expect, it } from 'vitest';

import { ApiError } from './api';
import { normalizeApiError } from './errorHandling';

describe('error handling helpers', () => {
  it('maps permission errors to modal metadata without raw backend text', () => {
    const normalized = normalizeApiError(new ApiError({
      status: 403,
      code: 'PERMISSION_DENIED',
      messageKey: 'errors.permission',
      category: 'permission',
      correlationId: 'req-403',
    }));

    expect(normalized).toMatchObject({
      titleKey: 'errors.dialog.permissionTitle',
      messageKey: 'errors.permission',
      recommendationKey: 'errors.recommendation.permission',
      severity: 'permission_error',
      correlationId: 'req-403',
    });
    expect(JSON.stringify(normalized)).not.toContain('permission pos.');
  });

  it('maps session and network errors to different user flows', () => {
    const session = normalizeApiError(new ApiError({
      status: 401,
      code: 'SESSION_REVOKED',
      messageKey: 'errors.session.revoked',
      category: 'auth',
    }));
    const network = normalizeApiError(new ApiError({
      status: 0,
      code: 'NETWORK_ERROR',
      messageKey: 'errors.network.unavailable',
      category: 'network',
      retryable: true,
    }));

    expect(session.severity).toBe('auth_error');
    expect(session.recommendationKey).toBe('errors.recommendation.login');
    expect(network.severity).toBe('infrastructure_error');
    expect(network.recommendationKey).toBe('errors.recommendation.network');
    expect(network.retryable).toBe(true);
  });
});

import { useAuthStore } from '../stores/auth';
import { useErrorDialogStore, type ErrorDialogSeverity } from '../stores/errorDialog';
import { ApiError, type ApiErrorCategory } from './api';

export type NormalizedAppError = {
  code: string;
  messageKey: string;
  titleKey: string;
  recommendationKey: string;
  category: ApiErrorCategory | 'unknown';
  severity: ErrorDialogSeverity;
  correlationId: string;
  retryable: boolean;
};

/** normalizeApiError приводит unknown/error к безопасной модели для i18n и modal dialog. */
export function normalizeApiError(error: unknown): NormalizedAppError {
  if (error instanceof ApiError) {
    return {
      code: error.code,
      messageKey: error.messageKey,
      titleKey: titleKeyForCategory(error.category),
      recommendationKey: recommendationKeyForCategory(error.category),
      category: error.category,
      severity: severityForCategory(error.category),
      correlationId: error.correlationId,
      retryable: error.retryable,
    };
  }
  return {
    code: 'UNEXPECTED_CLIENT_ERROR',
    messageKey: 'errors.unexpected',
    titleKey: 'errors.dialog.unexpectedTitle',
    recommendationKey: 'errors.recommendation.close',
    category: 'unknown',
    severity: 'fatal',
    correlationId: '',
    retryable: false,
  };
}

/** displayErrorMessageKey возвращает i18n key для компактного inline fallback без сырого текста. */
export function displayErrorMessageKey(error: unknown) {
  return normalizeApiError(error).messageKey;
}

/** useErrorHandling централизует UX-поведение для auth/permission/network/business ошибок. */
export function useErrorHandling() {
  const dialog = useErrorDialogStore();
  const auth = useAuthStore();

  function showBusinessError(error: unknown) {
    const normalized = normalizeApiError(error);
    if (normalized.category === 'auth') {
      auth.clearSession();
    }
    dialog.show({
      titleKey: normalized.titleKey,
      messageKey: normalized.messageKey,
      recommendationKey: normalized.recommendationKey,
      severity: normalized.severity,
      category: normalized.category,
      correlationId: normalized.correlationId,
      primaryAction: normalized.category === 'auth' ? 'login' : 'close',
    });
  }

  return { showBusinessError, normalizeApiError };
}

function titleKeyForCategory(category: ApiErrorCategory) {
  switch (category) {
    case 'auth':
      return 'errors.dialog.sessionTitle';
    case 'permission':
      return 'errors.dialog.permissionTitle';
    case 'conflict':
      return 'errors.dialog.conflictTitle';
    case 'rate_limit':
      return 'errors.dialog.rateLimitTitle';
    case 'network':
    case 'timeout':
      return 'errors.dialog.networkTitle';
    case 'server':
      return 'errors.dialog.serverTitle';
    case 'validation':
      return 'errors.dialog.validationTitle';
    default:
      return 'errors.dialog.unexpectedTitle';
  }
}

function recommendationKeyForCategory(category: ApiErrorCategory) {
  switch (category) {
    case 'auth':
      return 'errors.recommendation.login';
    case 'permission':
      return 'errors.recommendation.permission';
    case 'conflict':
      return 'errors.recommendation.refresh';
    case 'rate_limit':
      return 'errors.recommendation.rateLimit';
    case 'network':
    case 'timeout':
      return 'errors.recommendation.network';
    case 'server':
      return 'errors.recommendation.support';
    case 'validation':
      return 'errors.recommendation.validation';
    default:
      return 'errors.recommendation.close';
  }
}

function severityForCategory(category: ApiErrorCategory): ErrorDialogSeverity {
  switch (category) {
    case 'auth':
      return 'auth_error';
    case 'permission':
      return 'permission_error';
    case 'conflict':
      return 'conflict';
    case 'network':
    case 'timeout':
    case 'server':
      return 'infrastructure_error';
    case 'rate_limit':
    case 'validation':
      return 'warning';
    default:
      return 'fatal';
  }
}

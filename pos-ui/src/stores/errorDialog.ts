import { defineStore } from 'pinia';

import type { ApiErrorCategory } from '../shared/api';

/** Severity определяет визуальный приоритет modal error без связи с raw HTTP текстом. */
export type ErrorDialogSeverity =
  | 'info'
  | 'warning'
  | 'business_error'
  | 'permission_error'
  | 'auth_error'
  | 'conflict'
  | 'infrastructure_error'
  | 'fatal';

/** Действие modal error задает контролируемый переход вместо ad-hoc навигации из компонентов. */
export type ErrorDialogAction = 'close' | 'login';

/** ErrorDialogState хранит только i18n keys и безопасный support code. */
export type ErrorDialogState = {
  open: boolean;
  titleKey: string;
  messageKey: string;
  recommendationKey: string;
  severity: ErrorDialogSeverity;
  category: ApiErrorCategory | 'unknown';
  correlationId: string;
  primaryAction: ErrorDialogAction;
};

/** useErrorDialogStore предоставляет единый error dialog bus для blocking бизнес-ошибок. */
export const useErrorDialogStore = defineStore('errorDialog', {
  state: (): ErrorDialogState => ({
    open: false,
    titleKey: 'errors.dialog.title',
    messageKey: 'errors.unknown',
    recommendationKey: 'errors.recommendation.close',
    severity: 'business_error',
    category: 'unknown',
    correlationId: '',
    primaryAction: 'close',
  }),
  actions: {
    show(payload: Partial<Omit<ErrorDialogState, 'open'>>) {
      this.titleKey = payload.titleKey ?? 'errors.dialog.title';
      this.messageKey = payload.messageKey ?? 'errors.unknown';
      this.recommendationKey = payload.recommendationKey ?? 'errors.recommendation.close';
      this.severity = payload.severity ?? 'business_error';
      this.category = payload.category ?? 'unknown';
      this.correlationId = payload.correlationId ?? '';
      this.primaryAction = payload.primaryAction ?? 'close';
      this.open = true;
    },
    close() {
      this.open = false;
    },
  },
});

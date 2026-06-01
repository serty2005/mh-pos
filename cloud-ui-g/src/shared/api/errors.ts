import { z } from 'zod';
import type { ApiErrorCategory, ApiErrorOptions } from './types';

export class ApiError extends Error {
  status: number;
  code: string;
  messageKey: string;
  category: ApiErrorCategory;
  details: Record<string, string>;
  correlationId: string;
  retryable: boolean;

  constructor(options: ApiErrorOptions) {
    super(options.code);
    this.name = 'ApiError';
    this.status = options.status;
    this.code = options.code;
    this.messageKey = options.messageKey;
    this.category = options.category;
    this.details = options.details ?? {};
    this.correlationId = options.correlationId ?? '';
    this.retryable = options.retryable ?? false;
  }
}

const backendErrorSchema = z.object({
  error: z.union([
    z.string(),
    z.object({
      code: z.string().optional(),
      message_key: z.string().optional(),
      details: z.record(z.string(), z.string()).optional(),
      correlation_id: z.string().optional(),
    }),
  ]),
});

function codeForStatus(status: number) {
  if (status === 400 || status === 422) return 'VALIDATION_FAILED';
  if (status === 404) return 'NOT_FOUND';
  if (status === 409) return 'CONFLICT';
  return status >= 500 ? 'INTERNAL_ERROR' : 'UNKNOWN_ERROR';
}

function messageKeyForStatus(status: number) {
  if (status === 400 || status === 422) return 'errors.validation';
  if (status === 404) return 'errors.notFound';
  if (status === 409) return 'errors.conflict';
  return status >= 500 ? 'errors.server' : 'errors.unknown';
}

function categoryForStatus(status: number): ApiErrorCategory {
  if (status === 400 || status === 422) return 'validation';
  if (status === 404) return 'not_found';
  if (status === 409) return 'conflict';
  if (status >= 500) return 'server';
  return 'unexpected';
}

export function apiErrorFromResponse(response: Response, data: unknown) {
  const parsed = backendErrorSchema.safeParse(data);
  const errorValue = parsed.success ? parsed.data.error : null;
  const structured = typeof errorValue === 'object' && errorValue !== null ? errorValue : null;

  return new ApiError({
    status: response.status,
    code: structured?.code ?? codeForStatus(response.status),
    messageKey: structured?.message_key ?? messageKeyForStatus(response.status),
    category: categoryForStatus(response.status),
    details: structured?.details,
    correlationId: structured?.correlation_id ?? response.headers.get('X-Request-ID') ?? '',
    retryable: response.status >= 500,
  });
}

export function networkApiError(error: unknown) {
  const aborted = typeof DOMException !== 'undefined' && error instanceof DOMException && error.name === 'AbortError';

  return new ApiError({
    status: 0,
    code: aborted ? 'REQUEST_TIMEOUT' : 'NETWORK_ERROR',
    messageKey: aborted ? 'errors.network.timeout' : 'errors.network.unavailable',
    category: aborted ? 'timeout' : 'network',
    retryable: true,
  });
}

export function invalidJsonApiError(status: number, correlationId = '') {
  return new ApiError({
    status,
    code: 'INVALID_JSON',
    messageKey: 'errors.response.invalid',
    category: 'unexpected',
    correlationId,
  });
}

export function invalidResponseApiError() {
  return new ApiError({
    status: 0,
    code: 'INVALID_RESPONSE',
    messageKey: 'errors.response.invalid',
    category: 'unexpected',
  });
}

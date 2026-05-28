export type ApiErrorCategory =
  | 'validation'
  | 'not_found'
  | 'conflict'
  | 'server'
  | 'network'
  | 'timeout'
  | 'unexpected';

export type ApiErrorOptions = {
  status: number;
  code: string;
  messageKey: string;
  category: ApiErrorCategory;
  details?: Record<string, string>;
  correlationId?: string;
  retryable?: boolean;
};

export type EndpointMethod = 'GET' | 'POST' | 'PATCH' | 'PUT' | 'DELETE';

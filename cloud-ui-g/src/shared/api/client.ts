import { z } from 'zod';
import {
  apiErrorFromResponse,
  invalidJsonApiError,
  invalidResponseApiError,
  networkApiError,
  type ApiError,
} from './errors';

const DEFAULT_TIMEOUT_MS = 15_000;

export function defaultApiBase() {
  const hostname = globalThis.location?.hostname;
  if (hostname === 'host.docker.internal') {
    return 'http://host.docker.internal:8090/api/v1';
  }
  return 'http://localhost:8090/api/v1';
}

export const apiBase = (import.meta.env.VITE_CLOUD_API_BASE ?? defaultApiBase()).replace(/\/$/, '');

async function parseResponseBody(response: Response) {
  const text = await response.text();
  if (!text.trim()) return null;

  try {
    return JSON.parse(text) as unknown;
  } catch {
    throw invalidJsonApiError(response.status, response.headers.get('X-Request-ID') ?? '');
  }
}

export async function request<T>(path: string, schema: z.ZodType<T>, init: RequestInit = {}) {
  const headers = new Headers(init.headers);
  if (init.body !== undefined && init.body !== null && !headers.has('Content-Type')) {
    headers.set('Content-Type', 'application/json');
  }

  const controller = new AbortController();
  const timeout = globalThis.setTimeout(() => controller.abort(), DEFAULT_TIMEOUT_MS);

  let response: Response;
  try {
    response = await fetch(`${apiBase}${path}`, { ...init, headers, signal: controller.signal });
  } catch (error) {
    throw networkApiError(error);
  } finally {
    globalThis.clearTimeout(timeout);
  }

  const data = await parseResponseBody(response);
  if (!response.ok) {
    throw apiErrorFromResponse(response, data);
  }

  const parsed = schema.safeParse(data);
  if (!parsed.success) {
    throw invalidResponseApiError();
  }

  return parsed.data;
}

export async function requestOptional<T>(path: string, schema: z.ZodType<T>) {
  try {
    return await request(path, schema.nullable());
  } catch (error) {
    if ((error as ApiError).name === 'ApiError' && (error as ApiError).status === 404) {
      return null;
    }
    throw error;
  }
}

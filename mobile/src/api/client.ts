import axios, {
  AxiosError,
  AxiosHeaders,
  type AxiosInstance,
  type AxiosRequestConfig,
  type AxiosResponse,
  type InternalAxiosRequestConfig
} from 'axios';
import { randomUUID } from 'expo-crypto';

import { API_BASE_URL } from '../config';
import { clearSession, getTokens, saveTokens } from '../storage/secure';
import type { PaginationMeta, TokenPair } from '../types';

export class ApiError extends Error {
  status: number;
  code: string;
  details: unknown;
  fieldErrors: Record<string, string>;

  constructor(
    status: number,
    message: string,
    code = 'REQUEST_FAILED',
    details?: unknown,
    fieldErrors: Record<string, string> = {}
  ) {
    super(message);
    this.name = 'ApiError';
    this.status = status;
    this.code = code;
    this.details = details;
    this.fieldErrors = fieldErrors;
  }
}

export type ApiEnvelope<T> = {
  success?: boolean;
  message?: string;
  data?: T;
  meta?: PaginationMeta;
  errors?: unknown;
};

const RAW_HEADER = 'X-PharmaOS-Raw';

const PUBLIC_PATHS = [
  '/api/v1/auth/login',
  '/api/v1/auth/refresh',
  '/api/v1/auth/forgot-password',
  '/api/v1/auth/reset-password'
];

let authLostHandler: (() => void) | null = null;

export function setAuthLostHandler(handler: (() => void) | null): void {
  authLostHandler = handler;
}

function statusCode(status: number): string {
  if (status === 0) return 'NETWORK_ERROR';
  if (status === 400) return 'BAD_REQUEST';
  if (status === 401) return 'UNAUTHORIZED';
  if (status === 403) return 'FORBIDDEN';
  if (status === 404) return 'NOT_FOUND';
  if (status === 409) return 'CONFLICT';
  if (status === 422) return 'VALIDATION_FAILED';
  if (status === 429) return 'RATE_LIMITED';
  return 'REQUEST_FAILED';
}

function extractCode(errors: unknown): string | undefined {
  if (!errors || typeof errors !== 'object' || Array.isArray(errors)) {
    return undefined;
  }
  const record = errors as Record<string, unknown>;
  if (typeof record.code === 'string') {
    return record.code;
  }
  if (typeof record.required_permission === 'string') {
    return 'PERMISSION_DENIED';
  }
  return undefined;
}

function extractFieldErrors(errors: unknown): Record<string, string> {
  if (!errors || typeof errors !== 'object' || Array.isArray(errors)) {
    return {};
  }
  const result: Record<string, string> = {};
  for (const [key, value] of Object.entries(errors)) {
    if (typeof value === 'string' && key !== 'code' && key !== 'required_permission') {
      result[key] = value;
    }
  }
  return result;
}

function unwrapEnvelope<T>(response: AxiosResponse<ApiEnvelope<T>>): T {
  const body = response.data;
  if (!body || body.success === false) {
    throw new ApiError(
      response.status,
      body?.message || 'Request failed',
      extractCode(body?.errors) || statusCode(response.status),
      body?.errors,
      extractFieldErrors(body?.errors)
    );
  }
  return body.data as T;
}

function toApiError(error: unknown): ApiError {
  if (error instanceof ApiError) {
    return error;
  }
  const axiosError = error as AxiosError<ApiEnvelope<unknown>>;
  const status = axiosError.response?.status ?? 0;
  const body = axiosError.response?.data;
  const timedOut = axiosError.code === 'ECONNABORTED';
  const message =
    body?.message ||
    (timedOut ? 'The request timed out. Check your connection and try again.' : '') ||
    (status === 0 ? 'Unable to reach PharmaOS. Check your connection.' : '') ||
    axiosError.message ||
    'Request failed';

  return new ApiError(
    status,
    message,
    extractCode(body?.errors) || statusCode(status),
    body?.errors,
    extractFieldErrors(body?.errors)
  );
}

function isPublicPath(url?: string): boolean {
  if (!url) {
    return false;
  }
  return PUBLIC_PATHS.some((path) => url.startsWith(path));
}

function hasAuthorization(config: InternalAxiosRequestConfig): boolean {
  const headers = config.headers as Record<string, unknown> | undefined;
  const readableHeaders = config.headers as { get?: (name: string) => string | null | undefined };
  return Boolean(readableHeaders.get?.('Authorization') ?? headers?.Authorization ?? headers?.authorization);
}

function isRawResponse(config: InternalAxiosRequestConfig): boolean {
  const headers = config.headers as Record<string, unknown> | undefined;
  const readableHeaders = config.headers as { get?: (name: string) => string | null | undefined };
  return Boolean(readableHeaders.get?.(RAW_HEADER) ?? headers?.[RAW_HEADER]);
}

const client: AxiosInstance = axios.create({
  baseURL: API_BASE_URL,
  timeout: 15000,
  headers: { 'Content-Type': 'application/json' }
});

const refreshClient: AxiosInstance = axios.create({
  baseURL: API_BASE_URL,
  timeout: 15000,
  headers: { 'Content-Type': 'application/json' }
});

let refreshPromise: Promise<boolean> | null = null;

async function refreshTokens(): Promise<boolean> {
  if (refreshPromise) {
    return refreshPromise;
  }
  const pending = (async () => {
    const tokens = await getTokens();
    if (!tokens?.refresh_token) {
      return false;
    }
    try {
      const response = await refreshClient.post<ApiEnvelope<{ tokens: TokenPair }>>(
        '/api/v1/auth/refresh',
        { refresh_token: tokens.refresh_token },
        { headers: { 'X-Request-ID': randomUUID() } }
      );
      const body = response.data;
      if (!body?.success || !body.data?.tokens) {
        return false;
      }
      await saveTokens(body.data.tokens);
      return true;
    } catch {
      return false;
    }
  })();

  refreshPromise = pending.finally(() => {
    refreshPromise = null;
  });
  return refreshPromise;
}

client.interceptors.request.use(async (config) => {
  config.headers.set('X-Request-ID', randomUUID());
  if (!isPublicPath(config.url)) {
    const tokens = await getTokens();
    if (tokens?.access_token) {
      config.headers.set('Authorization', `Bearer ${tokens.access_token}`);
    }
  }
  return config;
});

client.interceptors.response.use(
  (response) => (isRawResponse(response.config) ? response.data : unwrapEnvelope(response)),
  async (error: unknown) => {
    if (error instanceof ApiError) {
      throw error;
    }
    const axiosError = error as AxiosError<ApiEnvelope<unknown>>;
    const original = axiosError.config as (InternalAxiosRequestConfig & { _retry?: boolean }) | undefined;
    const shouldRefresh =
      axiosError.response?.status === 401 &&
      Boolean(original) &&
      !original?._retry &&
      !isPublicPath(original?.url) &&
      hasAuthorization(original as InternalAxiosRequestConfig);

    if (shouldRefresh && original) {
      original._retry = true;
      if (await refreshTokens()) {
        const tokens = await getTokens();
        if (tokens?.access_token) {
          original.headers.set('Authorization', `Bearer ${tokens.access_token}`);
        }
        return client.request(original);
      }
      await clearSession();
      authLostHandler?.();
    }
    throw toApiError(error);
  }
);

export function request<T>(path: string, config: AxiosRequestConfig = {}): Promise<T> {
  return client.request<T, T>({ ...config, url: path }) as unknown as Promise<T>;
}

export function get<T>(path: string, config: AxiosRequestConfig = {}): Promise<T> {
  return request<T>(path, { ...config, method: 'GET' });
}

export async function getEnvelope<T>(
  path: string,
  config: AxiosRequestConfig = {}
): Promise<ApiEnvelope<T>> {
  const headers = new AxiosHeaders(config.headers as AxiosHeaders);
  headers.set(RAW_HEADER, 'true');
  return request<ApiEnvelope<T>>(path, { ...config, headers });
}

export async function getPage<T>(
  path: string,
  config: AxiosRequestConfig = {}
): Promise<{ data: T; meta?: PaginationMeta }> {
  const envelope = await getEnvelope<T>(path, config);
  return { data: envelope.data as T, meta: envelope.meta };
}

export function post<T>(path: string, body?: unknown, config: AxiosRequestConfig = {}): Promise<T> {
  return request<T>(path, { ...config, method: 'POST', data: body ?? {} });
}

export function put<T>(path: string, body?: unknown, config: AxiosRequestConfig = {}): Promise<T> {
  return request<T>(path, { ...config, method: 'PUT', data: body ?? {} });
}

export function patch<T>(path: string, body?: unknown, config: AxiosRequestConfig = {}): Promise<T> {
  return request<T>(path, { ...config, method: 'PATCH', data: body ?? {} });
}

export function remove<T>(path: string, config: AxiosRequestConfig = {}): Promise<T> {
  return request<T>(path, { ...config, method: 'DELETE' });
}

import type { ApiResult, AuthMe, PageMeta, TokenPair } from '../types'

export interface RequestOptions {
  authenticated?: boolean
  retry?: boolean
  idempotency?: boolean
}

interface Envelope<T> {
  success: boolean
  message: string
  data?: T
  meta?: PageMeta
  errors?: unknown
}

const API_BASE_URL = (import.meta.env.VITE_API_BASE_URL || '').replace(/\/$/, '')
const TOKEN_KEY = 'pharmaos.admin.tokens'
const USER_KEY = 'pharmaos.admin.user'

export class ApiError extends Error {
  status: number
  code: string
  details: unknown

  constructor(status: number, message: string, code = 'REQUEST_FAILED', details?: unknown) {
    super(message)
    this.name = 'ApiError'
    this.status = status
    this.code = code
    this.details = details
  }
}

export function randomId(): string {
  if (typeof crypto !== 'undefined' && typeof crypto.randomUUID === 'function') {
    return crypto.randomUUID()
  }
  return `${Date.now()}-${Math.random().toString(16).slice(2)}`
}

function errorCode(errors: unknown): string {
  if (errors && typeof errors === 'object') {
    const code = (errors as Record<string, unknown>).code
    if (typeof code === 'string' && code.length > 0) {
      return code
    }
  }
  return 'REQUEST_FAILED'
}

export function readTokens(): TokenPair | null {
  const raw = localStorage.getItem(TOKEN_KEY)
  if (!raw) return null
  try {
    return JSON.parse(raw) as TokenPair
  } catch {
    return null
  }
}

export function writeTokens(tokens: TokenPair): void {
  localStorage.setItem(TOKEN_KEY, JSON.stringify(tokens))
}

export function readStoredUser(): AuthMe | null {
  const raw = localStorage.getItem(USER_KEY)
  if (!raw) return null
  try {
    return JSON.parse(raw) as AuthMe
  } catch {
    return null
  }
}

export function writeStoredUser(user: AuthMe): void {
  localStorage.setItem(USER_KEY, JSON.stringify(user))
}

export function clearSession(): void {
  localStorage.removeItem(TOKEN_KEY)
  localStorage.removeItem(USER_KEY)
}

async function parseResponse<T>(response: Response): Promise<ApiResult<T>> {
  const text = await response.text()
  let envelope: Envelope<T> | null = null
  if (text.length > 0) {
    try {
      envelope = JSON.parse(text) as Envelope<T>
    } catch {
      envelope = null
    }
  }

  if (!response.ok || !envelope?.success) {
    throw new ApiError(
      response.status,
      envelope?.message || (response.status === 401 ? 'Session expired' : 'Request failed'),
      errorCode(envelope?.errors),
      envelope?.errors,
    )
  }

  return { data: envelope.data as T, meta: envelope.meta }
}

let refreshPromise: Promise<boolean> | null = null

async function refreshTokens(): Promise<boolean> {
  if (refreshPromise) return refreshPromise
  refreshPromise = (async () => {
    const refreshToken = readTokens()?.refresh_token
    if (!refreshToken) {
      clearSession()
      return false
    }
    try {
      const response = await fetch(`${API_BASE_URL}/api/v1/auth/refresh`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json', 'X-Request-ID': randomId() },
        body: JSON.stringify({ refresh_token: refreshToken }),
      })
      const result = await parseResponse<{ tokens: TokenPair }>(response)
      writeTokens(result.data.tokens)
      return true
    } catch {
      clearSession()
      return false
    } finally {
      refreshPromise = null
    }
  })()
  return refreshPromise
}

export async function request<T>(path: string, init: RequestInit = {}, options: RequestOptions = {}): Promise<ApiResult<T>> {
  const authenticated = options.authenticated !== false
  const shouldRetry = options.retry !== false
  const headers = new Headers(init.headers)
  if (init.body !== undefined && !headers.has('Content-Type')) {
    headers.set('Content-Type', 'application/json')
  }
  headers.set('X-Request-ID', randomId())
  if (options.idempotency) {
    headers.set('Idempotency-Key', randomId())
  }

  const tokens = authenticated ? readTokens() : null
  if (tokens?.access_token) {
    headers.set('Authorization', `Bearer ${tokens.access_token}`)
  }

  const response = await fetch(`${API_BASE_URL}${path}`, { ...init, headers })
  if (response.status === 401 && authenticated && shouldRetry && (await refreshTokens())) {
    return request<T>(path, init, { ...options, retry: false })
  }
  return parseResponse<T>(response)
}

function bodyOf(body: unknown): BodyInit | undefined {
  return body === undefined ? undefined : JSON.stringify(body)
}

export function apiGet<T>(path: string, options?: RequestOptions): Promise<ApiResult<T>> {
  return request<T>(path, { method: 'GET' }, options)
}

export function apiPost<T>(path: string, body?: unknown, options?: RequestOptions): Promise<ApiResult<T>> {
  return request<T>(path, { method: 'POST', body: bodyOf(body) }, options)
}

export function apiPatch<T>(path: string, body?: unknown, options?: RequestOptions): Promise<ApiResult<T>> {
  return request<T>(path, { method: 'PATCH', body: bodyOf(body) }, options)
}

export function apiPut<T>(path: string, body?: unknown, options?: RequestOptions): Promise<ApiResult<T>> {
  return request<T>(path, { method: 'PUT', body: bodyOf(body) }, options)
}

export function apiDelete<T>(path: string, options?: RequestOptions): Promise<ApiResult<T>> {
  return request<T>(path, { method: 'DELETE' }, options)
}

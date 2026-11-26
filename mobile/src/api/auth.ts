import { get, post } from './client';
import type { AuthMe, LoginResponse } from '../types';

export function login(tenantCode: string, email: string, password: string, deviceId: string): Promise<LoginResponse> {
  return post<LoginResponse>('/api/v1/auth/login', {
    tenant_code: tenantCode,
    email,
    password,
    device_id: deviceId,
  });
}

export function forgotPassword(email: string): Promise<{ accepted: boolean; message: string }> {
  return post<{ accepted: boolean; message: string }>('/api/v1/auth/forgot-password', { email });
}

export function resetPassword(token: string, newPassword: string): Promise<void> {
  return post<void>('/api/v1/auth/reset-password', {
    token,
    new_password: newPassword
  });
}

export async function logout(): Promise<void> {
  await post<{ logged_out: boolean }>('/api/v1/auth/logout');
}

export function me(): Promise<AuthMe> {
  return get<AuthMe>('/api/v1/auth/me');
}

export function sessions(): Promise<Record<string, unknown>[]> {
  return get<Record<string, unknown>[]>('/api/v1/auth/sessions');
}

export function changePassword(currentPassword: string, newPassword: string): Promise<void> {
  return post<void>('/api/v1/auth/change-password', {
    current_password: currentPassword,
    new_password: newPassword
  });
}

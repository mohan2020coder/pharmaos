import { z } from 'zod';

export const loginSchema = z.object({
  tenantCode: z.string().trim().min(2, 'Enter the tenant code').max(40, 'Tenant code is too long'),
  email: z.string().trim().email('Enter a valid email address'),
  password: z.string().min(6, 'Password must be at least 6 characters')
});

export type LoginFormValues = z.infer<typeof loginSchema>;

export const forgotPasswordSchema = z.object({
  email: z.string().trim().email('Enter a valid email address')
});

export type ForgotPasswordFormValues = z.infer<typeof forgotPasswordSchema>;

export const resetPasswordSchema = z
  .object({
    token: z.string().trim().min(1, 'Enter the reset token'),
    newPassword: z.string().min(10, 'Password must be at least 10 characters'),
    confirmPassword: z.string().min(1, 'Confirm the new password')
  })
  .refine((values) => values.newPassword === values.confirmPassword, {
    path: ['confirmPassword'],
    message: 'Passwords do not match'
  });

export type ResetPasswordFormValues = z.infer<typeof resetPasswordSchema>;

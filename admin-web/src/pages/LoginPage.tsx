import { useEffect, useState } from 'react'
import { Navigate, useNavigate } from 'react-router-dom'

import { useAuth } from '../auth/AuthProvider'
import { ErrorBanner } from '../components/ErrorBanner'
import { FormField } from '../components/FormField'
import { useForm } from '../hooks/useForm'
import { errorMessage } from '../lib/format'

export function LoginPage() {
  const { user, ready, login } = useAuth()
  const navigate = useNavigate()
  const form = useForm({ tenantCode: '', email: '', password: '' })
  const [error, setError] = useState<string | null>(null)
  const [busy, setBusy] = useState(false)

  useEffect(() => {
    if (ready && user) navigate('/', { replace: true })
  }, [ready, user, navigate])

  if (ready && user) {
    return <Navigate to="/" replace />
  }

  const onSubmit = async (event: React.FormEvent<HTMLFormElement>) => {
    event.preventDefault()
    setBusy(true)
    setError(null)
    try {
      await login(form.values.tenantCode.trim(), form.values.email.trim(), form.values.password)
      navigate('/', { replace: true })
    } catch (reason) {
      setError(errorMessage(reason))
    } finally {
      setBusy(false)
    }
  }

  return (
    <div className="login-page">
      <div className="login-card">
        <div className="login-brand">
          <span className="brand-mark">P</span>
          <div>
            <h1>PharmaOS</h1>
            <p>Administration console</p>
          </div>
        </div>
        <h2>Sign in</h2>
        <p className="muted">Use your tenant code and administrator credentials.</p>
        <ErrorBanner error={error} />
        <form onSubmit={onSubmit}>
          <FormField label="Tenant code">
            <input required autoComplete="organization" value={form.values.tenantCode} onChange={form.setField('tenantCode')} />
          </FormField>
          <FormField label="Email">
            <input required type="email" autoComplete="username" value={form.values.email} onChange={form.setField('email')} />
          </FormField>
          <FormField label="Password">
            <input required type="password" autoComplete="current-password" value={form.values.password} onChange={form.setField('password')} />
          </FormField>
          <button className="button primary block" type="submit" disabled={busy}>
            {busy ? 'Signing in…' : 'Sign in'}
          </button>
        </form>
      </div>
    </div>
  )
}

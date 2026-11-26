import { createContext, useContext, useEffect, useMemo, useState } from 'react'
import type { ReactNode } from 'react'

import { apiGet, apiPost, clearSession, randomId, readStoredUser, writeStoredUser, writeTokens } from '../lib/api'
import type { AuthMe, LoginResponse } from '../types'

interface AuthContextValue {
  user: AuthMe | null
  ready: boolean
  login: (tenantCode: string, email: string, password: string) => Promise<void>
  logout: () => Promise<void>
}

const AuthContext = createContext<AuthContextValue | null>(null)

export function AuthProvider({ children }: { children: ReactNode }) {
  const [user, setUser] = useState<AuthMe | null>(() => readStoredUser())
  const [ready, setReady] = useState(false)

  useEffect(() => {
    const tokens = localStorage.getItem('pharmaos.admin.tokens')
    if (!tokens) {
      setReady(true)
      return
    }
    let active = true
    apiGet<AuthMe>('/api/v1/auth/me')
      .then((result) => {
        if (!active) return
        setUser(result.data)
        writeStoredUser(result.data)
      })
      .catch(() => {
        if (!active) return
        clearSession()
        setUser(null)
      })
      .finally(() => {
        if (active) setReady(true)
      })
    return () => {
      active = false
    }
  }, [])

  const value = useMemo<AuthContextValue>(
    () => ({
      user,
      ready,
      login: async (tenantCode, email, password) => {
        const result = await apiPost<LoginResponse>('/api/v1/auth/login', {
          tenant_code: tenantCode,
          email,
          password,
          device_id: randomId(),
        }, { authenticated: false })
        writeTokens(result.data.tokens)
        const profile = await apiGet<AuthMe>('/api/v1/auth/me')
        setUser(profile.data)
        writeStoredUser(profile.data)
      },
      logout: async () => {
        try {
          await apiPost('/api/v1/auth/logout')
        } finally {
          clearSession()
          setUser(null)
        }
      },
    }),
    [ready, user],
  )

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>
}

export function useAuth(): AuthContextValue {
  const context = useContext(AuthContext)
  if (!context) throw new Error('useAuth must be used inside AuthProvider')
  return context
}

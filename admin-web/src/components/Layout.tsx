import { NavLink, useNavigate } from 'react-router-dom'
import type { ReactNode } from 'react'

import { useAuth } from '../auth/AuthProvider'
import { errorMessage } from '../lib/format'

interface NavItem {
  to: string
  label: string
  permission?: string
}

const NAV_ITEMS: NavItem[] = [
  { to: '/', label: 'Dashboard' },
  { to: '/tenants', label: 'Tenants', permission: 'tenant.view' },
  { to: '/users', label: 'Users', permission: 'user.view' },
  { to: '/roles', label: 'Roles & permissions', permission: 'role.view' },
  { to: '/organization', label: 'Organization', permission: 'employee.view' },
  { to: '/master-data', label: 'Master data', permission: 'doctor.view' },
  { to: '/distribution', label: 'Distribution', permission: 'distributor.view' },
  { to: '/reports', label: 'Reports', permission: 'reports.view' },
  { to: '/integrations', label: 'Integrations', permission: 'integration.view' },
  { to: '/sync', label: 'Sync' },
]

export function Layout({ children }: { children: ReactNode }) {
  const { user, logout } = useAuth()
  const navigate = useNavigate()
  const permissions = user?.permissions ?? []
  const canSee = (permission?: string) => {
    if (!permission) return true
    return user?.is_privileged || permissions.length === 0 || permissions.includes(permission) || permissions.includes('*')
  }

  const onLogout = async () => {
    try {
      await logout()
    } catch (error) {
      navigate('/login', { replace: true })
      return
    }
    navigate('/login', { replace: true })
  }

  return (
    <div className="app-shell">
      <aside className="sidebar">
        <div className="brand">
          <span className="brand-mark">P</span>
          <div>
            <strong>PharmaOS</strong>
            <small>Administration</small>
          </div>
        </div>
        <nav>
          {NAV_ITEMS.filter((item) => canSee(item.permission)).map((item) => (
            <NavLink
              key={item.to}
              to={item.to}
              end={item.to === '/'}
              className={({ isActive }) => (isActive ? 'nav-link active' : 'nav-link')}
            >
              {item.label}
            </NavLink>
          ))}
        </nav>
        <div className="sidebar-footer">
          <div className="user-card">
            <strong>{user?.email ?? 'Signed in'}</strong>
            <small>{user?.data_scope || 'TENANT'}</small>
          </div>
          <button type="button" className="button secondary" onClick={onLogout}>
            Sign out
          </button>
        </div>
      </aside>
      <main className="main-content">
        <div className="topbar">
          <span>Operations console</span>
          <span className="status-pill">Connected</span>
        </div>
        <div className="page-content">{children}</div>
      </main>
    </div>
  )
}

export function AppError({ error }: { error: unknown }) {
  return <div className="banner banner-error">{errorMessage(error)}</div>
}

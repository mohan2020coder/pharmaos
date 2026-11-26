import { BrowserRouter, Navigate, Outlet, Route, Routes } from 'react-router-dom'

import { useAuth } from './auth/AuthProvider'
import { Layout } from './components/Layout'
import { DashboardPage } from './pages/DashboardPage'
import { DistributionPage } from './pages/DistributionPage'
import { IntegrationsPage } from './pages/IntegrationsPage'
import { LoginPage } from './pages/LoginPage'
import { MasterDataPage } from './pages/MasterDataPage'
import { OrganizationPage } from './pages/OrganizationPage'
import { ReportsPage } from './pages/ReportsPage'
import { RolesPage } from './pages/RolesPage'
import { SyncPage } from './pages/SyncPage'
import { TenantsPage } from './pages/TenantsPage'
import { UsersPage } from './pages/UsersPage'

function ProtectedLayout() {
  const { user, ready } = useAuth()
  if (!ready) return <div className="boot-screen">Loading PharmaOS…</div>
  if (!user) return <Navigate to="/login" replace />
  return <Layout><Outlet /></Layout>
}

export default function App() {
  return (
    <BrowserRouter>
      <Routes>
        <Route path="/login" element={<LoginPage />} />
        <Route element={<ProtectedLayout />}>
          <Route path="/" element={<DashboardPage />} />
          <Route path="/tenants" element={<TenantsPage />} />
          <Route path="/users" element={<UsersPage />} />
          <Route path="/roles" element={<RolesPage />} />
          <Route path="/organization" element={<OrganizationPage />} />
          <Route path="/master-data" element={<MasterDataPage />} />
          <Route path="/distribution" element={<DistributionPage />} />
          <Route path="/reports" element={<ReportsPage />} />
          <Route path="/integrations" element={<IntegrationsPage />} />
          <Route path="/sync" element={<SyncPage />} />
        </Route>
        <Route path="*" element={<Navigate to="/" replace />} />
      </Routes>
    </BrowserRouter>
  )
}

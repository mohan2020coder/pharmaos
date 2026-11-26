import { ErrorBanner } from '../components/ErrorBanner'
import { DataTable } from '../components/DataTable'
import type { DataTableColumn } from '../components/DataTable'
import { FormField } from '../components/FormField'
import { PageHeader } from '../components/PageHeader'
import { useForm } from '../hooks/useForm'
import { useMutation } from '../hooks/useMutation'
import { useResource } from '../hooks/useResource'
import { apiGet, apiPost } from '../lib/api'
import { formatDateTime } from '../lib/format'
import type { User } from '../types'

export function UsersPage() {
  const users = useResource(() => apiGet<User[]>('/api/v1/users?page_size=200').then((result) => result.data), [])
  const mutation = useMutation()
  const form = useForm({ tenant_id: '', email: '', password: '', first_name: '', last_name: '', phone: '' })

  const onSubmit = async (event: React.FormEvent<HTMLFormElement>) => {
    event.preventDefault()
    const created = await mutation.run(() => apiPost<User>('/api/v1/users', form.values))
    if (created) {
      form.reset()
      users.reload()
    }
  }

  const deactivate = async (id: string) => {
    if (!window.confirm('Deactivate this user?')) return
    await mutation.run(() => apiPost(`/api/v1/users/${id}/deactivate`))
    users.reload()
  }

  const columns: DataTableColumn<User>[] = [
    { key: 'email', header: 'Email', render: (user) => <strong>{user.email}</strong> },
    { key: 'name', header: 'Name', render: (user) => `${user.first_name} ${user.last_name}`.trim() },
    { key: 'status', header: 'Status', render: (user) => <span className="badge">{user.status}</span> },
    { key: 'verified', header: 'Verified', render: (user) => (user.email_verified ? 'Yes' : 'No') },
    { key: 'last_login', header: 'Last login', render: (user) => formatDateTime(user.last_login_at) },
    {
      key: 'actions',
      header: 'Actions',
      render: (user) => (
        <button className="button danger small" type="button" disabled={mutation.busy || user.status !== 'ACTIVE'} onClick={() => deactivate(user.id)}>
          Deactivate
        </button>
      ),
    },
  ]

  return (
    <>
      <PageHeader title="Users" subtitle="Accounts and access lifecycle" />
      <ErrorBanner error={users.error || mutation.error} />
      <section className="panel">
        <h2>Create user</h2>
        <form className="form-grid" onSubmit={onSubmit}>
          <FormField label="Tenant ID" hint="Optional for non-platform administrators">
            <input value={form.values.tenant_id} onChange={form.setField('tenant_id')} />
          </FormField>
          <FormField label="Email"><input required type="email" value={form.values.email} onChange={form.setField('email')} /></FormField>
          <FormField label="Temporary password"><input required type="password" value={form.values.password} onChange={form.setField('password')} /></FormField>
          <FormField label="First name"><input required value={form.values.first_name} onChange={form.setField('first_name')} /></FormField>
          <FormField label="Last name"><input required value={form.values.last_name} onChange={form.setField('last_name')} /></FormField>
          <FormField label="Phone"><input value={form.values.phone} onChange={form.setField('phone')} /></FormField>
          <div className="form-actions">
            <button className="button primary" type="submit" disabled={mutation.busy}>{mutation.busy ? 'Creating…' : 'Create user'}</button>
          </div>
        </form>
      </section>
      <section className="panel">
        <div className="panel-heading"><h2>User directory</h2><span className="muted">{users.data?.length ?? 0} records</span></div>
        <DataTable columns={columns} rows={users.data ?? []} getRowKey={(user) => user.id} loading={users.loading} />
      </section>
    </>
  )
}

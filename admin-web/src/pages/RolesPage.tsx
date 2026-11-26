import { useEffect, useState } from 'react'

import { ErrorBanner } from '../components/ErrorBanner'
import { DataTable } from '../components/DataTable'
import type { DataTableColumn } from '../components/DataTable'
import { FormField } from '../components/FormField'
import { PageHeader } from '../components/PageHeader'
import { useForm } from '../hooks/useForm'
import { useMutation } from '../hooks/useMutation'
import { useResource } from '../hooks/useResource'
import { apiGet, apiPost, apiPut } from '../lib/api'
import type { Permission, Role, RoleWithPermissions } from '../types'

interface RolesData {
  roles: Role[]
  permissions: Permission[]
}

export function RolesPage() {
  const resource = useResource<RolesData>(async () => {
    const [roles, permissions] = await Promise.all([
      apiGet<Role[]>('/api/v1/roles?page_size=200'),
      apiGet<Permission[]>('/api/v1/permissions?page_size=200'),
    ])
    return { roles: roles.data, permissions: permissions.data }
  }, [])
  const mutation = useMutation()
  const [selectedId, setSelectedId] = useState<string | null>(null)
  const [selectedPermissions, setSelectedPermissions] = useState<string[]>([])
  const form = useForm({ code: '', name: '', description: '', data_scope: 'TENANT' })

  useEffect(() => {
    if (!selectedId && resource.data?.roles.length) {
      setSelectedId(resource.data.roles[0].id)
    }
  }, [resource.data, selectedId])

  useEffect(() => {
    const role = resource.data?.roles.find((item) => item.id === selectedId) as RoleWithPermissions | undefined
    setSelectedPermissions(role?.permissions ?? [])
  }, [resource.data, selectedId])

  const savePermissions = async () => {
    if (!selectedId) return
    const updated = await mutation.run(() => apiPut<RoleWithPermissions>(`/api/v1/roles/${selectedId}/permissions`, { permissions: selectedPermissions }))
    if (updated) resource.reload()
  }

  const createRole = async (event: React.FormEvent<HTMLFormElement>) => {
    event.preventDefault()
    const created = await mutation.run(() => apiPost<RoleWithPermissions>('/api/v1/roles', { ...form.values, permissions: [] }))
    if (created) {
      form.reset()
      resource.reload()
      setSelectedId(created.data.id)
    }
  }

  const toggle = (code: string) => {
    setSelectedPermissions((current) => (current.includes(code) ? current.filter((item) => item !== code) : [...current, code]))
  }

  const columns: DataTableColumn<Role>[] = [
    { key: 'code', header: 'Code', render: (role) => <strong>{role.code}</strong> },
    { key: 'name', header: 'Name', render: (role) => role.name },
    { key: 'scope', header: 'Data scope', render: (role) => role.data_scope },
    { key: 'system', header: 'System', render: (role) => (role.is_system ? 'Yes' : 'No') },
    { key: 'privileged', header: 'Privileged', render: (role) => (role.is_privileged ? 'Yes' : 'No') },
    {
      key: 'select',
      header: '',
      render: (role) => (
        <button className={role.id === selectedId ? 'button small' : 'button secondary small'} type="button" onClick={() => setSelectedId(role.id)}>
          Manage
        </button>
      ),
    },
  ]

  const selectedRole = resource.data?.roles.find((role) => role.id === selectedId) as RoleWithPermissions | undefined

  return (
    <>
      <PageHeader title="Roles & permissions" subtitle="Authorization policy management" />
      <ErrorBanner error={resource.error || mutation.error} />
      <section className="panel-grid">
        <div className="panel">
          <h2>Roles</h2>
          <DataTable columns={columns} rows={resource.data?.roles ?? []} getRowKey={(role) => role.id} loading={resource.loading} />
        </div>
        <div className="panel">
          <h2>{selectedRole ? `Permissions: ${selectedRole.name}` : 'Permissions'}</h2>
          {selectedRole?.is_system ? <p className="muted">System roles are managed by migrations.</p> : null}
          <div className="permission-list">
            {(resource.data?.permissions ?? []).map((permission) => (
              <label key={permission.id} className="permission-item">
                <input
                  type="checkbox"
                  checked={selectedPermissions.includes(permission.code)}
                  disabled={selectedRole?.is_system || mutation.busy}
                  onChange={() => toggle(permission.code)}
                />
                <span>
                  <strong>{permission.code}</strong>
                  <small>{permission.module}</small>
                </span>
              </label>
            ))}
          </div>
          <button className="button primary" type="button" disabled={!selectedId || selectedRole?.is_system || mutation.busy} onClick={savePermissions}>
            {mutation.busy ? 'Saving…' : 'Save permissions'}
          </button>
        </div>
      </section>
      <section className="panel">
        <h2>Create role</h2>
        <form className="form-grid" onSubmit={createRole}>
          <FormField label="Code"><input required value={form.values.code} onChange={form.setField('code')} /></FormField>
          <FormField label="Name"><input required value={form.values.name} onChange={form.setField('name')} /></FormField>
          <FormField label="Description"><input value={form.values.description} onChange={form.setField('description')} /></FormField>
          <FormField label="Data scope">
            <select value={form.values.data_scope} onChange={form.setField('data_scope')}>
              {['SELF', 'TEAM', 'TERRITORY', 'REGION', 'DIVISION', 'TENANT'].map((scope) => (
                <option key={scope} value={scope}>{scope}</option>
              ))}
            </select>
          </FormField>
          <div className="form-actions">
            <button className="button primary" type="submit" disabled={mutation.busy}>Create role</button>
          </div>
        </form>
      </section>
    </>
  )
}

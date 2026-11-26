import { useState } from 'react'

import { ErrorBanner } from '../components/ErrorBanner'
import { DataTable } from '../components/DataTable'
import type { DataTableColumn } from '../components/DataTable'
import { FormField } from '../components/FormField'
import { PageHeader } from '../components/PageHeader'
import { Tabs } from '../components/Tabs'
import { useForm } from '../hooks/useForm'
import { useMutation } from '../hooks/useMutation'
import { useResource } from '../hooks/useResource'
import { apiGet, apiPost } from '../lib/api'
import { formatDate } from '../lib/format'
import type { Branch, Division, Employee, Position, Region, Territory } from '../types'

interface OrganizationData {
  divisions: Division[]
  regions: Region[]
  territories: Territory[]
  employees: Employee[]
  branches: Branch[]
  positions: Position[]
}

const divisionColumns: DataTableColumn<Division>[] = [
  { key: 'code', header: 'Code', render: (row) => <strong>{row.code}</strong> },
  { key: 'name', header: 'Name', render: (row) => row.name },
  { key: 'status', header: 'Status', render: (row) => <span className="badge">{row.status}</span> },
  { key: 'created', header: 'Created', render: (row) => formatDate(row.created_at) },
]

const regionColumns: DataTableColumn<Region>[] = [
  { key: 'code', header: 'Code', render: (row) => <strong>{row.code}</strong> },
  { key: 'name', header: 'Name', render: (row) => row.name },
  { key: 'division', header: 'Division ID', render: (row) => <span className="mono">{row.division_id.slice(0, 8)}</span> },
  { key: 'status', header: 'Status', render: (row) => <span className="badge">{row.status}</span> },
]

const territoryColumns: DataTableColumn<Territory>[] = [
  { key: 'code', header: 'Code', render: (row) => <strong>{row.code}</strong> },
  { key: 'name', header: 'Name', render: (row) => row.name },
  { key: 'city', header: 'City', render: (row) => row.city },
  { key: 'status', header: 'Status', render: (row) => <span className="badge">{row.status}</span> },
]

const employeeColumns: DataTableColumn<Employee>[] = [
  { key: 'code', header: 'Employee code', render: (row) => <strong>{row.employee_code}</strong> },
  { key: 'name', header: 'Name', render: (row) => `${row.first_name} ${row.last_name}`.trim() },
  { key: 'email', header: 'Email', render: (row) => row.email ?? '—' },
  { key: 'status', header: 'Status', render: (row) => <span className="badge">{row.status}</span> },
]

const branchColumns: DataTableColumn<Branch>[] = [
  { key: 'code', header: 'Code', render: (row) => <strong>{row.code}</strong> },
  { key: 'name', header: 'Name', render: (row) => row.name },
  { key: 'city', header: 'City', render: (row) => row.city },
  { key: 'status', header: 'Status', render: (row) => <span className="badge">{row.status}</span> },
]

const positionColumns: DataTableColumn<Position>[] = [
  { key: 'code', header: 'Code', render: (row) => <strong>{row.code}</strong> },
  { key: 'title', header: 'Title', render: (row) => row.title },
  { key: 'level', header: 'Level', render: (row) => row.level },
  { key: 'management', header: 'Management', render: (row) => (row.is_management ? 'Yes' : 'No') },
]

export function OrganizationPage() {
  const [tab, setTab] = useState('divisions')
  const resource = useResource<OrganizationData>(async () => {
    const [divisions, regions, territories, employees, branches, positions] = await Promise.all([
      apiGet<Division[]>('/api/v1/divisions?page_size=200'),
      apiGet<Region[]>('/api/v1/regions?page_size=200'),
      apiGet<Territory[]>('/api/v1/territories?page_size=200'),
      apiGet<Employee[]>('/api/v1/employees?page_size=200'),
      apiGet<Branch[]>('/api/v1/organization/branches?page_size=200'),
      apiGet<Position[]>('/api/v1/positions'),
    ])
    return {
      divisions: divisions.data,
      regions: regions.data,
      territories: territories.data,
      employees: employees.data,
      branches: branches.data,
      positions: positions.data,
    }
  }, [])
  const mutation = useMutation()
  const divisionForm = useForm({ code: '', name: '', description: '', status: 'ACTIVE' })
  const regionForm = useForm({ division_id: '', code: '', name: '', status: 'ACTIVE' })
  const territoryForm = useForm({ division_id: '', region_id: '', code: '', name: '', city: '', status: 'ACTIVE' })
  const employeeForm = useForm({ employee_code: '', first_name: '', last_name: '', email: '', phone: '', joining_date: '', division_id: '', region_id: '', territory_id: '' })

  const onCreate = async (event: React.FormEvent<HTMLFormElement>, path: string, values: Record<string, string>) => {
    event.preventDefault()
    const created = await mutation.run(() => apiPost(path, values))
    if (created) {
      resource.reload()
    }
  }

  return (
    <>
      <PageHeader title="Organization" subtitle="Divisions, regions, territories and field workforce" />
      <ErrorBanner error={resource.error || mutation.error} />
      <Tabs
        active={tab}
        onChange={setTab}
        items={[
          { id: 'divisions', label: 'Divisions' },
          { id: 'regions', label: 'Regions' },
          { id: 'territories', label: 'Territories' },
          { id: 'employees', label: 'Employees' },
          { id: 'branches', label: 'Branches' },
          { id: 'positions', label: 'Positions' },
        ]}
      />

      {tab === 'divisions' ? (
        <>
          <section className="panel">
            <h2>Create division</h2>
            <form className="form-grid" onSubmit={(event) => onCreate(event, '/api/v1/divisions', divisionForm.values)}>
              <FormField label="Code"><input required value={divisionForm.values.code} onChange={divisionForm.setField('code')} /></FormField>
              <FormField label="Name"><input required value={divisionForm.values.name} onChange={divisionForm.setField('name')} /></FormField>
              <FormField label="Description"><input value={divisionForm.values.description} onChange={divisionForm.setField('description')} /></FormField>
              <FormField label="Status">
                <select value={divisionForm.values.status} onChange={divisionForm.setField('status')}>
                  <option value="ACTIVE">Active</option>
                  <option value="INACTIVE">Inactive</option>
                </select>
              </FormField>
              <div className="form-actions"><button className="button primary" type="submit" disabled={mutation.busy}>Create division</button></div>
            </form>
          </section>
          <section className="panel"><h2>Division registry</h2><DataTable columns={divisionColumns} rows={resource.data?.divisions ?? []} getRowKey={(row) => row.id} loading={resource.loading} /></section>
        </>
      ) : null}

      {tab === 'regions' ? (
        <>
          <section className="panel">
            <h2>Create region</h2>
            <form className="form-grid" onSubmit={(event) => onCreate(event, '/api/v1/regions', regionForm.values)}>
              <FormField label="Division">
                <select required value={regionForm.values.division_id} onChange={regionForm.setField('division_id')}>
                  <option value="">Select division</option>
                  {(resource.data?.divisions ?? []).map((division) => <option key={division.id} value={division.id}>{division.name}</option>)}
                </select>
              </FormField>
              <FormField label="Code"><input required value={regionForm.values.code} onChange={regionForm.setField('code')} /></FormField>
              <FormField label="Name"><input required value={regionForm.values.name} onChange={regionForm.setField('name')} /></FormField>
              <FormField label="Status">
                <select value={regionForm.values.status} onChange={regionForm.setField('status')}>
                  <option value="ACTIVE">Active</option>
                  <option value="INACTIVE">Inactive</option>
                </select>
              </FormField>
              <div className="form-actions"><button className="button primary" type="submit" disabled={mutation.busy}>Create region</button></div>
            </form>
          </section>
          <section className="panel"><h2>Region registry</h2><DataTable columns={regionColumns} rows={resource.data?.regions ?? []} getRowKey={(row) => row.id} loading={resource.loading} /></section>
        </>
      ) : null}

      {tab === 'territories' ? (
        <>
          <section className="panel">
            <h2>Create territory</h2>
            <form className="form-grid" onSubmit={(event) => onCreate(event, '/api/v1/territories', territoryForm.values)}>
              <FormField label="Division">
                <select required value={territoryForm.values.division_id} onChange={territoryForm.setField('division_id')}>
                  <option value="">Select division</option>
                  {(resource.data?.divisions ?? []).map((division) => <option key={division.id} value={division.id}>{division.name}</option>)}
                </select>
              </FormField>
              <FormField label="Region">
                <select required value={territoryForm.values.region_id} onChange={territoryForm.setField('region_id')}>
                  <option value="">Select region</option>
                  {(resource.data?.regions ?? []).map((region) => <option key={region.id} value={region.id}>{region.name}</option>)}
                </select>
              </FormField>
              <FormField label="Code"><input required value={territoryForm.values.code} onChange={territoryForm.setField('code')} /></FormField>
              <FormField label="Name"><input required value={territoryForm.values.name} onChange={territoryForm.setField('name')} /></FormField>
              <FormField label="City"><input value={territoryForm.values.city} onChange={territoryForm.setField('city')} /></FormField>
              <FormField label="Status">
                <select value={territoryForm.values.status} onChange={territoryForm.setField('status')}>
                  <option value="ACTIVE">Active</option>
                  <option value="INACTIVE">Inactive</option>
                </select>
              </FormField>
              <div className="form-actions"><button className="button primary" type="submit" disabled={mutation.busy}>Create territory</button></div>
            </form>
          </section>
          <section className="panel"><h2>Territory registry</h2><DataTable columns={territoryColumns} rows={resource.data?.territories ?? []} getRowKey={(row) => row.id} loading={resource.loading} /></section>
        </>
      ) : null}

      {tab === 'employees' ? (
        <>
          <section className="panel">
            <h2>Create employee</h2>
            <form className="form-grid" onSubmit={(event) => onCreate(event, '/api/v1/employees', employeeForm.values)}>
              <FormField label="Employee code"><input required value={employeeForm.values.employee_code} onChange={employeeForm.setField('employee_code')} /></FormField>
              <FormField label="First name"><input required value={employeeForm.values.first_name} onChange={employeeForm.setField('first_name')} /></FormField>
              <FormField label="Last name"><input required value={employeeForm.values.last_name} onChange={employeeForm.setField('last_name')} /></FormField>
              <FormField label="Email"><input type="email" value={employeeForm.values.email} onChange={employeeForm.setField('email')} /></FormField>
              <FormField label="Phone"><input value={employeeForm.values.phone} onChange={employeeForm.setField('phone')} /></FormField>
              <FormField label="Joining date"><input type="date" value={employeeForm.values.joining_date} onChange={employeeForm.setField('joining_date')} /></FormField>
              <FormField label="Division">
                <select value={employeeForm.values.division_id} onChange={employeeForm.setField('division_id')}>
                  <option value="">None</option>
                  {(resource.data?.divisions ?? []).map((division) => <option key={division.id} value={division.id}>{division.name}</option>)}
                </select>
              </FormField>
              <FormField label="Region">
                <select value={employeeForm.values.region_id} onChange={employeeForm.setField('region_id')}>
                  <option value="">None</option>
                  {(resource.data?.regions ?? []).map((region) => <option key={region.id} value={region.id}>{region.name}</option>)}
                </select>
              </FormField>
              <FormField label="Territory">
                <select value={employeeForm.values.territory_id} onChange={employeeForm.setField('territory_id')}>
                  <option value="">None</option>
                  {(resource.data?.territories ?? []).map((territory) => <option key={territory.id} value={territory.id}>{territory.name}</option>)}
                </select>
              </FormField>
              <div className="form-actions"><button className="button primary" type="submit" disabled={mutation.busy}>Create employee</button></div>
            </form>
          </section>
          <section className="panel"><h2>Employee registry</h2><DataTable columns={employeeColumns} rows={resource.data?.employees ?? []} getRowKey={(row) => row.id} loading={resource.loading} /></section>
        </>
      ) : null}

      {tab === 'branches' ? <section className="panel"><h2>Branches</h2><DataTable columns={branchColumns} rows={resource.data?.branches ?? []} getRowKey={(row) => row.id} loading={resource.loading} /></section> : null}
      {tab === 'positions' ? <section className="panel"><h2>Positions</h2><DataTable columns={positionColumns} rows={resource.data?.positions ?? []} getRowKey={(row) => row.id} loading={resource.loading} /></section> : null}
    </>
  )
}

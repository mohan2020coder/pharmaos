import { ErrorBanner } from '../components/ErrorBanner'
import { DataTable } from '../components/DataTable'
import type { DataTableColumn } from '../components/DataTable'
import { FormField } from '../components/FormField'
import { PageHeader } from '../components/PageHeader'
import { useForm } from '../hooks/useForm'
import { useMutation } from '../hooks/useMutation'
import { useResource } from '../hooks/useResource'
import { apiGet, apiPost } from '../lib/api'
import { formatDate } from '../lib/format'
import type { Tenant } from '../types'

const columns: DataTableColumn<Tenant>[] = [
  { key: 'code', header: 'Code', render: (tenant) => <strong>{tenant.code}</strong> },
  { key: 'name', header: 'Name', render: (tenant) => tenant.name },
  { key: 'type', header: 'Type', render: (tenant) => tenant.tenant_type },
  { key: 'status', header: 'Status', render: (tenant) => <span className="badge">{tenant.status}</span> },
  { key: 'currency', header: 'Currency', render: (tenant) => tenant.currency_code },
  { key: 'created', header: 'Created', render: (tenant) => formatDate(tenant.created_at) },
]

export function TenantsPage() {
  const tenants = useResource(() => apiGet<Tenant[]>('/api/v1/tenants?page_size=200').then((result) => result.data), [])
  const mutation = useMutation()
  const form = useForm({
    code: '',
    name: '',
    tenant_type: 'PHARMA_COMPANY',
    country_code: 'IN',
    timezone: 'UTC',
    currency_code: 'INR',
  })

  const onSubmit = async (event: React.FormEvent<HTMLFormElement>) => {
    event.preventDefault()
    const created = await mutation.run(() => apiPost<Tenant>('/api/v1/tenants', form.values))
    if (created) {
      form.reset()
      tenants.reload()
    }
  }

  return (
    <>
      <PageHeader title="Tenants" subtitle="Organizations hosted on the platform" />
      <ErrorBanner error={tenants.error || mutation.error} />
      <section className="panel">
        <h2>Create tenant</h2>
        <form className="form-grid" onSubmit={onSubmit}>
          <FormField label="Code"><input required value={form.values.code} onChange={form.setField('code')} /></FormField>
          <FormField label="Name"><input required value={form.values.name} onChange={form.setField('name')} /></FormField>
          <FormField label="Type">
            <select value={form.values.tenant_type} onChange={form.setField('tenant_type')}>
              <option value="PHARMA_COMPANY">Pharma company</option>
              <option value="DISTRIBUTOR">Distributor</option>
            </select>
          </FormField>
          <FormField label="Country"><input value={form.values.country_code} onChange={form.setField('country_code')} /></FormField>
          <FormField label="Timezone"><input value={form.values.timezone} onChange={form.setField('timezone')} /></FormField>
          <FormField label="Currency"><input value={form.values.currency_code} onChange={form.setField('currency_code')} /></FormField>
          <div className="form-actions">
            <button className="button primary" type="submit" disabled={mutation.busy}>{mutation.busy ? 'Creating…' : 'Create tenant'}</button>
          </div>
        </form>
      </section>
      <section className="panel">
        <div className="panel-heading">
          <h2>Tenant registry</h2>
          <span className="muted">{tenants.data?.length ?? 0} records</span>
        </div>
        <DataTable columns={columns} rows={tenants.data ?? []} getRowKey={(tenant) => tenant.id} loading={tenants.loading} />
      </section>
    </>
  )
}

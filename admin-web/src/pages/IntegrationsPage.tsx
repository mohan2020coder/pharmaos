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
import { apiDelete, apiGet, apiPatch, apiPost } from '../lib/api'
import { formatDateTime, formatNumber } from '../lib/format'
import type { IntegrationConnection, WebhookDelivery } from '../types'

interface IntegrationsData {
  connections: IntegrationConnection[]
  deliveries: WebhookDelivery[]
}

export function IntegrationsPage() {
  const [tab, setTab] = useState('connections')
  const resource = useResource<IntegrationsData>(async () => {
    const [connections, deliveries] = await Promise.all([
      apiGet<IntegrationConnection[]>('/api/v1/integrations/connections?page_size=200'),
      apiGet<WebhookDelivery[]>('/api/v1/integrations/deliveries?page_size=200'),
    ])
    return { connections: connections.data, deliveries: deliveries.data }
  }, [])
  const mutation = useMutation()
  const form = useForm({ name: '', kind: 'WEBHOOK', endpoint_url: '', secret_ref: '', headers: '{}' })

  const connectionColumns: DataTableColumn<IntegrationConnection>[] = [
    { key: 'name', header: 'Name', render: (row) => <strong>{row.name}</strong> },
    { key: 'kind', header: 'Kind', render: (row) => row.kind },
    { key: 'endpoint', header: 'Endpoint', render: (row) => row.endpoint_url ?? '—' },
    { key: 'status', header: 'Status', render: (row) => <span className="badge">{row.status}</span> },
    {
      key: 'actions',
      header: 'Actions',
      render: (row) => (
        <div className="button-row">
          <button className="button small" type="button" disabled={mutation.busy} onClick={() => setStatus(row.id, row.status === 'ACTIVE' ? 'DISABLED' : 'ACTIVE')}>
            {row.status === 'ACTIVE' ? 'Disable' : 'Enable'}
          </button>
          <button className="button danger small" type="button" disabled={mutation.busy} onClick={() => remove(row.id)}>Delete</button>
        </div>
      ),
    },
  ]

  const deliveryColumns: DataTableColumn<WebhookDelivery>[] = [
    { key: 'event', header: 'Event', render: (row) => <strong>{row.event_type}</strong> },
    { key: 'event_id', header: 'Event ID', render: (row) => <span className="mono">{row.event_id.slice(0, 8)}</span> },
    { key: 'status', header: 'Status', render: (row) => <span className="badge">{row.status}</span> },
    { key: 'attempts', header: 'Attempts', render: (row) => formatNumber(row.attempts) },
    { key: 'next', header: 'Next attempt', render: (row) => formatDateTime(row.next_attempt_at) },
    { key: 'error', header: 'Last error', render: (row) => row.last_error ?? '—' },
    {
      key: 'retry',
      header: 'Actions',
      render: (row) => (
        <button className="button small" type="button" disabled={mutation.busy || ['DELIVERED', 'IN_FLIGHT'].includes(row.status)} onClick={() => retry(row.id)}>
          Retry
        </button>
      ),
    },
  ]

  const setStatus = async (id: string, status: string) => {
    await mutation.run(() => apiPatch(`/api/v1/integrations/connections/${id}/status`, { status }))
    resource.reload()
  }

  const remove = async (id: string) => {
    if (!window.confirm('Delete this integration connection?')) return
    await mutation.run(() => apiDelete(`/api/v1/integrations/connections/${id}`))
    resource.reload()
  }

  const retry = async (id: string) => {
    await mutation.run(() => apiPost(`/api/v1/integrations/deliveries/${id}/retry`))
    resource.reload()
  }

  const createConnection = async (event: React.FormEvent<HTMLFormElement>) => {
    event.preventDefault()
    let headers: Record<string, string> = {}
    try {
      headers = JSON.parse(form.values.headers || '{}') as Record<string, string>
    } catch {
      mutation.setError('Headers must be a JSON object of strings')
      return
    }
    const created = await mutation.run(() => apiPost<IntegrationConnection>('/api/v1/integrations/connections', {
      ...form.values,
      endpoint_url: form.values.endpoint_url || undefined,
      secret_ref: form.values.secret_ref || undefined,
      headers,
    }))
    if (created) {
      form.reset()
      resource.reload()
    }
  }

  return (
    <>
      <PageHeader title="Integrations" subtitle="Outbound webhooks and delivery monitoring" />
      <ErrorBanner error={resource.error || mutation.error} />
      <Tabs active={tab} onChange={setTab} items={[
        { id: 'connections', label: 'Connections' },
        { id: 'deliveries', label: 'Deliveries' },
      ]} />

      {tab === 'connections' ? (
        <>
          <section className="panel">
            <h2>Create connection</h2>
            <form className="form-grid" onSubmit={createConnection}>
              <FormField label="Name"><input required value={form.values.name} onChange={form.setField('name')} /></FormField>
              <FormField label="Kind">
                <select value={form.values.kind} onChange={form.setField('kind')}>
                  <option value="WEBHOOK">Webhook</option>
                  <option value="ERP">ERP</option>
                  <option value="ANALYTICS">Analytics</option>
                </select>
              </FormField>
              <FormField label="Endpoint URL"><input value={form.values.endpoint_url} onChange={form.setField('endpoint_url')} /></FormField>
              <FormField label="Secret reference"><input value={form.values.secret_ref} onChange={form.setField('secret_ref')} /></FormField>
              <FormField label="Headers"><textarea rows={3} value={form.values.headers} onChange={form.setField('headers')} /></FormField>
              <div className="form-actions"><button className="button primary" type="submit" disabled={mutation.busy}>Create connection</button></div>
            </form>
          </section>
          <section className="panel"><h2>Connections</h2><DataTable columns={connectionColumns} rows={resource.data?.connections ?? []} getRowKey={(row) => row.id} loading={resource.loading} /></section>
        </>
      ) : null}

      {tab === 'deliveries' ? <section className="panel"><h2>Webhook deliveries</h2><DataTable columns={deliveryColumns} rows={resource.data?.deliveries ?? []} getRowKey={(row) => row.id} loading={resource.loading} /></section> : null}
    </>
  )
}

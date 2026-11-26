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
import { buildQuery, formatDateTime, formatNumber } from '../lib/format'
import type { AuditEntry, DistributionReport, FieldActivityReport, ReportDefinition, ReportSnapshot } from '../types'

interface ReportsData {
  field: FieldActivityReport
  distribution: DistributionReport
  audit: AuditEntry[]
  definitions: ReportDefinition[]
  snapshots: ReportSnapshot[]
}

const auditColumns: DataTableColumn<AuditEntry>[] = [
  { key: 'created', header: 'When', render: (row) => formatDateTime(row.created_at) },
  { key: 'module', header: 'Module', render: (row) => row.module },
  { key: 'action', header: 'Action', render: (row) => <strong>{row.action}</strong> },
  { key: 'entity', header: 'Entity', render: (row) => `${row.entity_type}: ${row.entity_id ?? '—'}` },
]

const definitionColumns: DataTableColumn<ReportDefinition>[] = [
  { key: 'code', header: 'Code', render: (row) => <strong>{row.code}</strong> },
  { key: 'name', header: 'Name', render: (row) => row.name },
  { key: 'type', header: 'Type', render: (row) => row.report_type },
  { key: 'created', header: 'Created', render: (row) => formatDateTime(row.created_at) },
]

const snapshotColumns: DataTableColumn<ReportSnapshot>[] = [
  { key: 'id', header: 'ID', render: (row) => <span className="mono">{row.id.slice(0, 8)}</span> },
  { key: 'definition', header: 'Definition', render: (row) => <span className="mono">{row.definition_id?.slice(0, 8) ?? 'Ad hoc'}</span> },
  { key: 'period', header: 'Period', render: (row) => `${row.period_from ?? '—'} → ${row.period_to ?? '—'}` },
  { key: 'created', header: 'Created', render: (row) => formatDateTime(row.created_at) },
]

function defaultRange() {
  const to = new Date()
  const from = new Date(to)
  from.setDate(to.getDate() - 30)
  return { from: from.toISOString().slice(0, 10), to: to.toISOString().slice(0, 10) }
}

export function ReportsPage() {
  const initial = defaultRange()
  const [from, setFrom] = useState(initial.from)
  const [to, setTo] = useState(initial.to)
  const [tab, setTab] = useState('summary')
  const resource = useResource<ReportsData>(async () => {
    const query = buildQuery({ from, to })
    const [field, distribution, audit, definitions, snapshots] = await Promise.all([
      apiGet<FieldActivityReport>(`/api/v1/reports/field-activity${query}`),
      apiGet<DistributionReport>(`/api/v1/reports/distribution${query}`),
      apiGet<AuditEntry[]>(`/api/v1/reports/audit${buildQuery({ from, to, page_size: 200 })}`),
      apiGet<ReportDefinition[]>('/api/v1/reports/definitions?page_size=200'),
      apiGet<ReportSnapshot[]>('/api/v1/reports/snapshots?page_size=200'),
    ])
    return {
      field: field.data,
      distribution: distribution.data,
      audit: audit.data,
      definitions: definitions.data,
      snapshots: snapshots.data,
    }
  }, [from, to])
  const mutation = useMutation()
  const form = useForm({ code: '', name: '', description: '', report_type: 'FIELD_ACTIVITY', config: '{}' })

  const createDefinition = async (event: React.FormEvent<HTMLFormElement>) => {
    event.preventDefault()
    let config: Record<string, unknown> = {}
    try {
      config = JSON.parse(form.values.config || '{}') as Record<string, unknown>
    } catch {
      mutation.setError('Config must be valid JSON')
      return
    }
    const created = await mutation.run(() => apiPost<ReportDefinition>('/api/v1/reports/definitions', { ...form.values, config }))
    if (created) {
      form.reset()
      resource.reload()
    }
  }

  return (
    <>
      <PageHeader title="Reports" subtitle="Operational summaries, audit trail and saved report definitions" />
      <ErrorBanner error={resource.error || mutation.error} />
      <div className="filter-bar">
        <FormField label="From"><input type="date" value={from} onChange={(event) => setFrom(event.target.value)} /></FormField>
        <FormField label="To"><input type="date" value={to} onChange={(event) => setTo(event.target.value)} /></FormField>
        <button className="button secondary" type="button" onClick={resource.reload}>Refresh</button>
      </div>
      <Tabs
        active={tab}
        onChange={setTab}
        items={[
          { id: 'summary', label: 'Summary' },
          { id: 'audit', label: 'Audit trail' },
          { id: 'definitions', label: 'Definitions' },
          { id: 'snapshots', label: 'Snapshots' },
        ]}
      />

      {tab === 'summary' && resource.data ? (
        <section className="panel-grid">
          <div className="panel">
            <h2>Field activity</h2>
            <dl className="definition-list">
              <div><dt>Total visits</dt><dd>{formatNumber(resource.data.field.visits.total)}</dd></div>
              <div><dt>Submitted visits</dt><dd>{formatNumber(resource.data.field.visits.submitted)}</dd></div>
              <div><dt>Draft visits</dt><dd>{formatNumber(resource.data.field.visits.draft)}</dd></div>
              <div><dt>Unique customers</dt><dd>{formatNumber(resource.data.field.visits.unique_customers)}</dd></div>
              <div><dt>Call reports</dt><dd>{formatNumber(resource.data.field.daily_call_reports.total)}</dd></div>
              <div><dt>Approved reports</dt><dd>{formatNumber(resource.data.field.daily_call_reports.approved)}</dd></div>
            </dl>
          </div>
          <div className="panel">
            <h2>Distribution summary</h2>
            <dl className="definition-list">
              <div><dt>Total orders</dt><dd>{formatNumber(resource.data.distribution.orders.total)}</dd></div>
              <div><dt>Approved orders</dt><dd>{formatNumber(resource.data.distribution.orders.approved)}</dd></div>
              <div><dt>Cancelled orders</dt><dd>{formatNumber(resource.data.distribution.orders.cancelled)}</dd></div>
              <div><dt>Order value</dt><dd>{formatNumber(resource.data.distribution.orders.value)}</dd></div>
              <div><dt>Inventory units</dt><dd>{formatNumber(resource.data.distribution.inventory.on_hand)}</dd></div>
              <div><dt>Secondary sales</dt><dd>{formatNumber(resource.data.distribution.secondary_sales.records)}</dd></div>
            </dl>
          </div>
        </section>
      ) : null}

      {tab === 'audit' ? <section className="panel"><h2>Audit trail</h2><DataTable columns={auditColumns} rows={resource.data?.audit ?? []} getRowKey={(row) => String(row.id)} loading={resource.loading} /></section> : null}

      {tab === 'definitions' ? (
        <>
          <section className="panel">
            <h2>Create report definition</h2>
            <form className="form-grid" onSubmit={createDefinition}>
              <FormField label="Code"><input required value={form.values.code} onChange={form.setField('code')} /></FormField>
              <FormField label="Name"><input required value={form.values.name} onChange={form.setField('name')} /></FormField>
              <FormField label="Description"><input value={form.values.description} onChange={form.setField('description')} /></FormField>
              <FormField label="Report type">
                <select value={form.values.report_type} onChange={form.setField('report_type')}>
                  <option value="FIELD_ACTIVITY">Field activity</option>
                  <option value="DISTRIBUTION_SUMMARY">Distribution summary</option>
                  <option value="AUDIT_TRAIL">Audit trail</option>
                </select>
              </FormField>
              <FormField label="Config"><textarea rows={3} value={form.values.config} onChange={form.setField('config')} /></FormField>
              <div className="form-actions"><button className="button primary" type="submit" disabled={mutation.busy}>Create definition</button></div>
            </form>
          </section>
          <section className="panel"><h2>Saved definitions</h2><DataTable columns={definitionColumns} rows={resource.data?.definitions ?? []} getRowKey={(row) => row.id} loading={resource.loading} /></section>
        </>
      ) : null}

      {tab === 'snapshots' ? <section className="panel"><h2>Snapshots</h2><DataTable columns={snapshotColumns} rows={resource.data?.snapshots ?? []} getRowKey={(row) => row.id} loading={resource.loading} /></section> : null}
    </>
  )
}

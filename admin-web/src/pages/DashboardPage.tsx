import { ErrorBanner } from '../components/ErrorBanner'
import { PageHeader } from '../components/PageHeader'
import { useResource } from '../hooks/useResource'
import { apiGet } from '../lib/api'
import { buildQuery, formatDateTime, formatNumber } from '../lib/format'
import type { DistributionReport, FieldActivityReport, SyncStatus } from '../types'

interface DashboardData {
  field: FieldActivityReport
  distribution: DistributionReport
  sync: SyncStatus
}

export function DashboardPage() {
  const range = useResource<DashboardData>(async () => {
    const to = new Date()
    const from = new Date(to)
    from.setDate(to.getDate() - 30)
    const query = buildQuery({ from: from.toISOString().slice(0, 10), to: to.toISOString().slice(0, 10) })
    const [field, distribution, sync] = await Promise.all([
      apiGet<FieldActivityReport>(`/api/v1/reports/field-activity${query}`),
      apiGet<DistributionReport>(`/api/v1/reports/distribution${query}`),
      apiGet<SyncStatus>('/api/v1/sync/status'),
    ])
    return { field: field.data, distribution: distribution.data, sync: sync.data }
  }, [])

  const field = range.data?.field
  const distribution = range.data?.distribution
  const sync = range.data?.sync

  return (
    <>
      <PageHeader title="Dashboard" subtitle="Operational snapshot for the last 30 days" />
      <ErrorBanner error={range.error} />
      {range.loading ? <div className="panel-message">Loading dashboard…</div> : null}
      {field && distribution && sync ? (
        <>
          <section className="metric-grid">
            <article className="metric-card">
              <span>Field visits</span>
              <strong>{formatNumber(field.visits.total)}</strong>
              <small>{formatNumber(field.visits.submitted)} submitted</small>
            </article>
            <article className="metric-card">
              <span>Daily call reports</span>
              <strong>{formatNumber(field.daily_call_reports.total)}</strong>
              <small>{formatNumber(field.daily_call_reports.approved)} approved</small>
            </article>
            <article className="metric-card">
              <span>Orders</span>
              <strong>{formatNumber(distribution.orders.total)}</strong>
              <small>{formatNumber(distribution.orders.approved)} approved</small>
            </article>
            <article className="metric-card">
              <span>Secondary sales</span>
              <strong>{formatNumber(distribution.secondary_sales.records)}</strong>
              <small>{formatNumber(distribution.secondary_sales.units)} units</small>
            </article>
            <article className="metric-card">
              <span>Active links</span>
              <strong>{formatNumber(distribution.active_links)}</strong>
              <small>Distributor connections</small>
            </article>
            <article className="metric-card">
              <span>Pending sync</span>
              <strong>{formatNumber(sync.pending_operations)}</strong>
              <small>{formatDateTime(sync.server_time)}</small>
            </article>
          </section>
          <section className="panel-grid">
            <div className="panel">
              <h2>Field execution</h2>
              <dl className="definition-list">
                <div><dt>Unique customers</dt><dd>{formatNumber(field.visits.unique_customers)}</dd></div>
                <div><dt>Draft visits</dt><dd>{formatNumber(field.visits.draft)}</dd></div>
                <div><dt>Submitted reports</dt><dd>{formatNumber(field.daily_call_reports.submitted)}</dd></div>
              </dl>
            </div>
            <div className="panel">
              <h2>Distribution</h2>
              <dl className="definition-list">
                <div><dt>Order value</dt><dd>{formatNumber(distribution.orders.value)}</dd></div>
                <div><dt>SKUs on hand</dt><dd>{formatNumber(distribution.inventory.sku_count)}</dd></div>
                <div><dt>Units on hand</dt><dd>{formatNumber(distribution.inventory.on_hand)}</dd></div>
                <div><dt>Sales value</dt><dd>{formatNumber(distribution.secondary_sales.value)}</dd></div>
              </dl>
            </div>
          </section>
        </>
      ) : null}
    </>
  )
}

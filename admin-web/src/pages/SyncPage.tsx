import { useState } from 'react'

import { ErrorBanner } from '../components/ErrorBanner'
import { DataTable } from '../components/DataTable'
import type { DataTableColumn } from '../components/DataTable'
import { FormField } from '../components/FormField'
import { PageHeader } from '../components/PageHeader'
import { useMutation } from '../hooks/useMutation'
import { useResource } from '../hooks/useResource'
import { apiGet } from '../lib/api'
import { buildQuery, formatDateTime, formatNumber } from '../lib/format'
import type { SyncChange, SyncStatus } from '../types'

interface SyncChanges {
  next_cursor: string
  has_more: boolean
  changes: SyncChange[]
}

const changeColumns: DataTableColumn<SyncChange>[] = [
  { key: 'entity', header: 'Entity', render: (row) => <strong>{row.entity}</strong> },
  { key: 'entity_id', header: 'Entity ID', render: (row) => <span className="mono">{row.entity_id.slice(0, 8)}</span> },
  { key: 'op', header: 'Operation', render: (row) => row.op },
  { key: 'updated', header: 'Updated', render: (row) => formatDateTime(row.updated_at) },
  { key: 'payload', header: 'Payload', render: (row) => <code>{JSON.stringify(row.payload).slice(0, 120)}</code> },
]

export function SyncPage() {
  const status = useResource(() => apiGet<SyncStatus>('/api/v1/sync/status').then((result) => result.data), [])
  const mutation = useMutation()
  const [cursor, setCursor] = useState('')
  const [changes, setChanges] = useState<SyncChange[]>([])
  const [nextCursor, setNextCursor] = useState('')
  const [hasMore, setHasMore] = useState(false)

  const loadChanges = async () => {
    const result = await mutation.run(() => apiGet<SyncChanges>(`/api/v1/sync/changes${buildQuery({ since: cursor, limit: 200 })}`))
    if (!result) return
    setChanges(result.data.changes)
    setNextCursor(result.data.next_cursor)
    setHasMore(result.data.has_more)
  }

  return (
    <>
      <PageHeader title="Sync" subtitle="Replication cursor and pending operation health" />
      <ErrorBanner error={status.error || mutation.error} />
      <section className="metric-grid">
        <article className="metric-card"><span>Pending operations</span><strong>{formatNumber(status.data?.pending_operations)}</strong></article>
        <article className="metric-card"><span>Latest cursor</span><strong className="mono small-text">{status.data?.latest_cursor || '—'}</strong></article>
        <article className="metric-card"><span>Server time</span><strong className="small-text">{formatDateTime(status.data?.server_time)}</strong></article>
      </section>
      <section className="panel">
        <h2>Supported entities</h2>
        <div className="chip-row">
          {(status.data?.supported_entities ?? []).map((entity) => <span key={entity} className="chip">{entity}</span>)}
        </div>
        <dl className="definition-list">
          {Object.entries(status.data?.entity_counts ?? {}).map(([entity, count]) => (
            <div key={entity}><dt>{entity}</dt><dd>{formatNumber(count)}</dd></div>
          ))}
        </dl>
      </section>
      <section className="panel">
        <div className="filter-bar">
          <FormField label="Since cursor"><input value={cursor} onChange={(event) => setCursor(event.target.value)} placeholder="Optional cursor" /></FormField>
          <button className="button primary" type="button" disabled={mutation.busy} onClick={loadChanges}>Load changes</button>
        </div>
        <div className="panel-heading">
          <h2>Change feed</h2>
          {nextCursor ? <button className="button secondary small" type="button" onClick={() => setCursor(nextCursor)}>Use next cursor</button> : null}
          {hasMore ? <span className="muted">More changes available</span> : null}
        </div>
        <DataTable columns={changeColumns} rows={changes} getRowKey={(row, index) => `${row.entity}-${row.entity_id}-${index}`} loading={mutation.busy && changes.length === 0} />
      </section>
    </>
  )
}

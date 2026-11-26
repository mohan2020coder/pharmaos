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
import { formatDate, formatMoney, formatNumber } from '../lib/format'
import type { Distributor, DistributionLink, Inventory, InventoryMovement, Order, Product, SecondarySale } from '../types'

interface DistributionData {
  links: DistributionLink[]
  distributors: Distributor[]
  orders: Order[]
  inventory: Inventory[]
  movements: InventoryMovement[]
  sales: SecondarySale[]
  products: Product[]
}

const linkColumns: DataTableColumn<DistributionLink>[] = [
  { key: 'distributor', header: 'Distributor tenant', render: (row) => <span className="mono">{row.distributor_tenant_id.slice(0, 8)}</span> },
  { key: 'status', header: 'Status', render: (row) => <span className="badge">{row.status}</span> },
  { key: 'from', header: 'Effective from', render: (row) => formatDate(row.effective_from) },
  { key: 'to', header: 'Effective to', render: (row) => formatDate(row.effective_to) },
  { key: 'credit', header: 'Credit limit', render: (row) => formatNumber(row.credit_limit) },
]

const distributorColumns: DataTableColumn<Distributor>[] = [
  { key: 'code', header: 'Code', render: (row) => <strong>{row.code}</strong> },
  { key: 'name', header: 'Name', render: (row) => row.name },
  { key: 'license', header: 'License', render: (row) => row.license_no ?? '—' },
  { key: 'status', header: 'Status', render: (row) => <span className="badge">{row.status}</span> },
]

const orderColumns: DataTableColumn<Order>[] = [
  { key: 'number', header: 'Order', render: (row) => <strong>{row.order_number}</strong> },
  { key: 'status', header: 'Status', render: (row) => <span className="badge">{row.status}</span> },
  { key: 'date', header: 'Order date', render: (row) => formatDate(row.order_date) },
  { key: 'total', header: 'Total', render: (row) => formatMoney(row.total_amount, row.currency_code) },
  { key: 'idempotency', header: 'Key', render: (row) => <span className="mono">{row.idempotency_key.slice(0, 8)}</span> },
]

const inventoryColumns: DataTableColumn<Inventory>[] = [
  { key: 'product', header: 'Product ID', render: (row) => <span className="mono">{row.product_id.slice(0, 8)}</span> },
  { key: 'warehouse', header: 'Warehouse', render: (row) => row.warehouse_code },
  { key: 'on_hand', header: 'On hand', render: (row) => row.quantity_on_hand },
  { key: 'reserved', header: 'Reserved', render: (row) => row.quantity_reserved },
  { key: 'version', header: 'Version', render: (row) => row.version },
]

const movementColumns: DataTableColumn<InventoryMovement>[] = [
  { key: 'product', header: 'Product ID', render: (row) => <span className="mono">{row.product_id.slice(0, 8)}</span> },
  { key: 'warehouse', header: 'Warehouse', render: (row) => row.warehouse_code },
  { key: 'type', header: 'Type', render: (row) => row.movement_type },
  { key: 'delta', header: 'Delta', render: (row) => row.quantity_delta },
  { key: 'occurred', header: 'Occurred', render: (row) => formatDate(row.occurred_at) },
]

const saleColumns: DataTableColumn<SecondarySale>[] = [
  { key: 'invoice', header: 'Invoice', render: (row) => row.invoice_no ?? '—' },
  { key: 'date', header: 'Sale date', render: (row) => formatDate(row.sale_date) },
  { key: 'product', header: 'Product ID', render: (row) => <span className="mono">{row.product_id.slice(0, 8)}</span> },
  { key: 'quantity', header: 'Quantity', render: (row) => row.quantity },
  { key: 'total', header: 'Total', render: (row) => formatNumber(row.total_amount) },
]

export function DistributionPage() {
  const [tab, setTab] = useState('links')
  const resource = useResource<DistributionData>(async () => {
    const [links, distributors, orders, inventory, movements, sales, products] = await Promise.all([
      apiGet<DistributionLink[]>('/api/v1/distribution-links?page_size=200'),
      apiGet<Distributor[]>('/api/v1/distributors?page_size=200'),
      apiGet<Order[]>('/api/v1/orders?page_size=200'),
      apiGet<Inventory[]>('/api/v1/inventory?page_size=200'),
      apiGet<InventoryMovement[]>('/api/v1/inventory/movements?page_size=200'),
      apiGet<SecondarySale[]>('/api/v1/secondary-sales?page_size=200'),
      apiGet<Product[]>('/api/v1/products?page_size=200'),
    ])
    return {
      links: links.data,
      distributors: distributors.data,
      orders: orders.data,
      inventory: inventory.data,
      movements: movements.data,
      sales: sales.data,
      products: products.data,
    }
  }, [])
  const mutation = useMutation()
  const linkForm = useForm({ distributor_tenant_id: '', status: 'ACTIVE', effective_from: '', effective_to: '', credit_limit: '0', payment_terms_days: '30' })
  const distributorForm = useForm({ code: '', name: '', license_no: '', email: '', phone: '', address: '', status: 'ACTIVE' })
  const orderForm = useForm({ distributor_tenant_id: '', order_number: '', currency_code: 'INR', order_date: '', required_by: '', remarks: '', product_id: '', quantity: '1', unit_price: '0' })
  const movementForm = useForm({ product_id: '', warehouse_code: 'MAIN', movement_type: 'RECEIPT', quantity_delta: '0', occurred_at: '' })
  const saleForm = useForm({ product_id: '', quantity: '1', unit_price: '0', sale_date: '', invoice_no: '', customer_code: '', order_id: '' })

  const onCreate = async (event: React.FormEvent<HTMLFormElement>, path: string, payload: Record<string, unknown>, idempotency = false) => {
    event.preventDefault()
    const created = await mutation.run(() => apiPost(path, payload, { idempotency }))
    if (created) resource.reload()
  }

  const orderAction = async (id: string, action: string) => {
    await mutation.run(() => apiPost(`/api/v1/orders/${id}/${action}`))
    resource.reload()
  }

  const productSelect = (value: string, onChange: (event: React.ChangeEvent<HTMLSelectElement>) => void, required = true) => (
    <select required={required} value={value} onChange={onChange}>
      <option value="">Select product</option>
      {(resource.data?.products ?? []).map((product) => <option key={product.id} value={product.id}>{product.code} — {product.name}</option>)}
    </select>
  )

  return (
    <>
      <PageHeader title="Distribution" subtitle="Partner links, orders, inventory and secondary sales" />
      <ErrorBanner error={resource.error || mutation.error} />
      <Tabs
        active={tab}
        onChange={setTab}
        items={[
          { id: 'links', label: 'Links' },
          { id: 'distributors', label: 'Distributors' },
          { id: 'orders', label: 'Orders' },
          { id: 'inventory', label: 'Inventory' },
          { id: 'sales', label: 'Secondary sales' },
        ]}
      />

      {tab === 'links' ? (
        <>
          <section className="panel">
            <h2>Create distribution link</h2>
            <form className="form-grid" onSubmit={(event) => onCreate(event, '/api/v1/distribution-links', {
              ...linkForm.values,
              credit_limit: Number(linkForm.values.credit_limit),
              payment_terms_days: Number(linkForm.values.payment_terms_days),
              effective_from: linkForm.values.effective_from || undefined,
              effective_to: linkForm.values.effective_to || undefined,
            })}>
              <FormField label="Distributor tenant ID"><input required value={linkForm.values.distributor_tenant_id} onChange={linkForm.setField('distributor_tenant_id')} /></FormField>
              <FormField label="Status">
                <select value={linkForm.values.status} onChange={linkForm.setField('status')}>
                  <option value="ACTIVE">Active</option>
                  <option value="SUSPENDED">Suspended</option>
                  <option value="ENDED">Ended</option>
                </select>
              </FormField>
              <FormField label="Effective from"><input type="date" value={linkForm.values.effective_from} onChange={linkForm.setField('effective_from')} /></FormField>
              <FormField label="Effective to"><input type="date" value={linkForm.values.effective_to} onChange={linkForm.setField('effective_to')} /></FormField>
              <FormField label="Credit limit"><input type="number" min="0" step="0.01" value={linkForm.values.credit_limit} onChange={linkForm.setField('credit_limit')} /></FormField>
              <FormField label="Payment terms (days)"><input type="number" min="0" value={linkForm.values.payment_terms_days} onChange={linkForm.setField('payment_terms_days')} /></FormField>
              <div className="form-actions"><button className="button primary" type="submit" disabled={mutation.busy}>Create link</button></div>
            </form>
          </section>
          <section className="panel"><h2>Distribution links</h2><DataTable columns={linkColumns} rows={resource.data?.links ?? []} getRowKey={(row) => row.id} loading={resource.loading} /></section>
        </>
      ) : null}

      {tab === 'distributors' ? (
        <>
          <section className="panel">
            <h2>Create distributor</h2>
            <form className="form-grid" onSubmit={(event) => onCreate(event, '/api/v1/distributors', distributorForm.values)}>
              <FormField label="Code"><input required value={distributorForm.values.code} onChange={distributorForm.setField('code')} /></FormField>
              <FormField label="Name"><input required value={distributorForm.values.name} onChange={distributorForm.setField('name')} /></FormField>
              <FormField label="License"><input value={distributorForm.values.license_no} onChange={distributorForm.setField('license_no')} /></FormField>
              <FormField label="Email"><input type="email" value={distributorForm.values.email} onChange={distributorForm.setField('email')} /></FormField>
              <FormField label="Phone"><input value={distributorForm.values.phone} onChange={distributorForm.setField('phone')} /></FormField>
              <FormField label="Address"><input value={distributorForm.values.address} onChange={distributorForm.setField('address')} /></FormField>
              <FormField label="Status">
                <select value={distributorForm.values.status} onChange={distributorForm.setField('status')}>
                  <option value="ACTIVE">Active</option>
                  <option value="INACTIVE">Inactive</option>
                </select>
              </FormField>
              <div className="form-actions"><button className="button primary" type="submit" disabled={mutation.busy}>Create distributor</button></div>
            </form>
          </section>
          <section className="panel"><h2>Distributor registry</h2><DataTable columns={distributorColumns} rows={resource.data?.distributors ?? []} getRowKey={(row) => row.id} loading={resource.loading} /></section>
        </>
      ) : null}

      {tab === 'orders' ? (
        <>
          <section className="panel">
            <h2>Create order</h2>
            <form className="form-grid" onSubmit={(event) => onCreate(event, '/api/v1/orders', {
              distributor_tenant_id: orderForm.values.distributor_tenant_id,
              order_number: orderForm.values.order_number || undefined,
              currency_code: orderForm.values.currency_code,
              order_date: orderForm.values.order_date || undefined,
              required_by: orderForm.values.required_by || undefined,
              remarks: orderForm.values.remarks || undefined,
              items: [{
                product_id: orderForm.values.product_id,
                quantity: Number(orderForm.values.quantity),
                unit_price: Number(orderForm.values.unit_price),
              }],
            }, true)}>
              <FormField label="Distributor tenant ID"><input required value={orderForm.values.distributor_tenant_id} onChange={orderForm.setField('distributor_tenant_id')} /></FormField>
              <FormField label="Order number" hint="Generated when blank"><input value={orderForm.values.order_number} onChange={orderForm.setField('order_number')} /></FormField>
              <FormField label="Currency"><input value={orderForm.values.currency_code} onChange={orderForm.setField('currency_code')} /></FormField>
              <FormField label="Order date"><input type="date" value={orderForm.values.order_date} onChange={orderForm.setField('order_date')} /></FormField>
              <FormField label="Required by"><input type="date" value={orderForm.values.required_by} onChange={orderForm.setField('required_by')} /></FormField>
              <FormField label="Product">{productSelect(orderForm.values.product_id, orderForm.setField('product_id'))}</FormField>
              <FormField label="Quantity"><input type="number" min="1" value={orderForm.values.quantity} onChange={orderForm.setField('quantity')} /></FormField>
              <FormField label="Unit price"><input type="number" min="0" step="0.01" value={orderForm.values.unit_price} onChange={orderForm.setField('unit_price')} /></FormField>
              <FormField label="Remarks"><input value={orderForm.values.remarks} onChange={orderForm.setField('remarks')} /></FormField>
              <div className="form-actions"><button className="button primary" type="submit" disabled={mutation.busy}>Create order</button></div>
            </form>
          </section>
          <section className="panel">
            <h2>Orders</h2>
            <DataTable
              columns={[
                ...orderColumns,
                {
                  key: 'actions',
                  header: 'Actions',
                  render: (order) => (
                    <div className="button-row">
                      <button className="button small" type="button" disabled={mutation.busy || order.status !== 'DRAFT'} onClick={() => orderAction(order.id, 'submit')}>Submit</button>
                      <button className="button small" type="button" disabled={mutation.busy || order.status !== 'SUBMITTED'} onClick={() => orderAction(order.id, 'approve')}>Approve</button>
                      <button className="button danger small" type="button" disabled={mutation.busy || ['APPROVED', 'CANCELLED'].includes(order.status)} onClick={() => orderAction(order.id, 'cancel')}>Cancel</button>
                    </div>
                  ),
                },
              ]}
              rows={resource.data?.orders ?? []}
              getRowKey={(row) => row.id}
              loading={resource.loading}
            />
          </section>
        </>
      ) : null}

      {tab === 'inventory' ? (
        <>
          <section className="panel">
            <h2>Inventory movement</h2>
            <form className="form-grid" onSubmit={(event) => onCreate(event, '/api/v1/inventory/movements', {
              ...movementForm.values,
              quantity_delta: Number(movementForm.values.quantity_delta),
              occurred_at: movementForm.values.occurred_at || undefined,
            }, true)}>
              <FormField label="Product">{productSelect(movementForm.values.product_id, movementForm.setField('product_id'))}</FormField>
              <FormField label="Warehouse"><input required value={movementForm.values.warehouse_code} onChange={movementForm.setField('warehouse_code')} /></FormField>
              <FormField label="Movement type">
                <select value={movementForm.values.movement_type} onChange={movementForm.setField('movement_type')}>
                  <option value="RECEIPT">Receipt</option>
                  <option value="SALE">Sale</option>
                  <option value="ADJUSTMENT">Adjustment</option>
                </select>
              </FormField>
              <FormField label="Quantity delta"><input type="number" value={movementForm.values.quantity_delta} onChange={movementForm.setField('quantity_delta')} /></FormField>
              <FormField label="Occurred at"><input type="datetime-local" value={movementForm.values.occurred_at} onChange={movementForm.setField('occurred_at')} /></FormField>
              <div className="form-actions"><button className="button primary" type="submit" disabled={mutation.busy}>Post movement</button></div>
            </form>
          </section>
          <section className="panel"><h2>Inventory on hand</h2><DataTable columns={inventoryColumns} rows={resource.data?.inventory ?? []} getRowKey={(row) => row.id} loading={resource.loading} /></section>
          <section className="panel"><h2>Movements</h2><DataTable columns={movementColumns} rows={resource.data?.movements ?? []} getRowKey={(row) => row.id} loading={resource.loading} /></section>
        </>
      ) : null}

      {tab === 'sales' ? (
        <>
          <section className="panel">
            <h2>Record secondary sale</h2>
            <form className="form-grid" onSubmit={(event) => onCreate(event, '/api/v1/secondary-sales', {
              ...saleForm.values,
              quantity: Number(saleForm.values.quantity),
              unit_price: Number(saleForm.values.unit_price),
              order_id: saleForm.values.order_id || undefined,
            }, true)}>
              <FormField label="Product">{productSelect(saleForm.values.product_id, saleForm.setField('product_id'))}</FormField>
              <FormField label="Quantity"><input type="number" min="1" value={saleForm.values.quantity} onChange={saleForm.setField('quantity')} /></FormField>
              <FormField label="Unit price"><input type="number" min="0" step="0.01" value={saleForm.values.unit_price} onChange={saleForm.setField('unit_price')} /></FormField>
              <FormField label="Sale date"><input type="date" required value={saleForm.values.sale_date} onChange={saleForm.setField('sale_date')} /></FormField>
              <FormField label="Invoice"><input value={saleForm.values.invoice_no} onChange={saleForm.setField('invoice_no')} /></FormField>
              <FormField label="Customer code"><input value={saleForm.values.customer_code} onChange={saleForm.setField('customer_code')} /></FormField>
              <FormField label="Order ID"><input value={saleForm.values.order_id} onChange={saleForm.setField('order_id')} /></FormField>
              <div className="form-actions"><button className="button primary" type="submit" disabled={mutation.busy}>Record sale</button></div>
            </form>
          </section>
          <section className="panel"><h2>Secondary sales</h2><DataTable columns={saleColumns} rows={resource.data?.sales ?? []} getRowKey={(row) => row.id} loading={resource.loading} /></section>
        </>
      ) : null}
    </>
  )
}

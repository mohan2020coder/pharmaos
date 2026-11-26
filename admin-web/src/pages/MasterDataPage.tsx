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
import { formatDate, formatNumber } from '../lib/format'
import type { Chemist, Doctor, Employee, Product, ProductCategory, Stockist, Target } from '../types'

interface MasterData {
  doctors: Doctor[]
  chemists: Chemist[]
  stockists: Stockist[]
  products: Product[]
  categories: ProductCategory[]
  targets: Target[]
  employees: Employee[]
}

const doctorColumns: DataTableColumn<Doctor>[] = [
  { key: 'name', header: 'Name', render: (row) => <strong>{row.name}</strong> },
  { key: 'specialty', header: 'Specialty', render: (row) => row.specialty ?? '—' },
  { key: 'registration', header: 'Registration', render: (row) => row.registration_no ?? '—' },
  { key: 'phone', header: 'Phone', render: (row) => row.phone ?? '—' },
  { key: 'active', header: 'Active', render: (row) => (row.is_active ? 'Yes' : 'No') },
]

const chemistColumns: DataTableColumn<Chemist>[] = [
  { key: 'name', header: 'Name', render: (row) => <strong>{row.name}</strong> },
  { key: 'owner', header: 'Owner', render: (row) => row.owner_name ?? '—' },
  { key: 'license', header: 'License', render: (row) => row.license_no ?? '—' },
  { key: 'phone', header: 'Phone', render: (row) => row.phone ?? '—' },
]

const stockistColumns: DataTableColumn<Stockist>[] = [
  { key: 'name', header: 'Name', render: (row) => <strong>{row.name}</strong> },
  { key: 'gst', header: 'GST', render: (row) => row.gst_number ?? '—' },
  { key: 'credit', header: 'Credit limit', render: (row) => formatNumber(row.credit_limit) },
  { key: 'terms', header: 'Terms (days)', render: (row) => row.payment_terms_days },
]

const productColumns: DataTableColumn<Product>[] = [
  { key: 'code', header: 'Code', render: (row) => <strong>{row.code}</strong> },
  { key: 'name', header: 'Name', render: (row) => row.name },
  { key: 'brand', header: 'Brand', render: (row) => row.brand ?? '—' },
  { key: 'pack', header: 'Pack', render: (row) => row.pack_size ?? '—' },
  { key: 'mrp', header: 'MRP', render: (row) => formatNumber(row.mrp) },
]

const categoryColumns: DataTableColumn<ProductCategory>[] = [
  { key: 'code', header: 'Code', render: (row) => <strong>{row.code}</strong> },
  { key: 'name', header: 'Name', render: (row) => row.name },
  { key: 'active', header: 'Active', render: (row) => (row.is_active ? 'Yes' : 'No') },
]

const targetColumns: DataTableColumn<Target>[] = [
  { key: 'entity', header: 'Entity', render: (row) => `${row.entity_type}: ${row.entity_id.slice(0, 8)}` },
  { key: 'period', header: 'Period', render: (row) => `${formatDate(row.period_start)} → ${formatDate(row.period_end)}` },
  { key: 'visits', header: 'Target visits', render: (row) => row.target_visits },
  { key: 'sales', header: 'Target sales', render: (row) => formatNumber(row.target_sales) },
  { key: 'status', header: 'Status', render: (row) => <span className="badge">{row.status}</span> },
]

export function MasterDataPage() {
  const [tab, setTab] = useState('doctors')
  const resource = useResource<MasterData>(async () => {
    const [doctors, chemists, stockists, products, categories, targets, employees] = await Promise.all([
      apiGet<Doctor[]>('/api/v1/doctors?page_size=200'),
      apiGet<Chemist[]>('/api/v1/chemists?page_size=200'),
      apiGet<Stockist[]>('/api/v1/stockists?page_size=200'),
      apiGet<Product[]>('/api/v1/products?page_size=200'),
      apiGet<ProductCategory[]>('/api/v1/product-categories?page_size=200'),
      apiGet<Target[]>('/api/v1/targets?page_size=200'),
      apiGet<Employee[]>('/api/v1/employees?page_size=200'),
    ])
    return {
      doctors: doctors.data,
      chemists: chemists.data,
      stockists: stockists.data,
      products: products.data,
      categories: categories.data,
      targets: targets.data,
      employees: employees.data,
    }
  }, [])
  const mutation = useMutation()
  const doctorForm = useForm({ name: '', specialty: '', qualification: '', registration_no: '', email: '', phone: '', address: '', territory_id: '' })
  const productForm = useForm({ category_id: '', code: '', name: '', brand: '', manufacturer: '', composition: '', pack_size: '', unit: '', mrp: '0' })
  const targetForm = useForm({ entity_type: 'DOCTOR', entity_id: '', employee_id: '', territory_id: '', period_start: '', period_end: '', target_visits: '0', target_sales: '0', status: 'ACTIVE' })

  const onCreate = async (event: React.FormEvent<HTMLFormElement>, path: string, payload: Record<string, unknown>) => {
    event.preventDefault()
    const created = await mutation.run(() => apiPost(path, payload))
    if (created) resource.reload()
  }

  return (
    <>
      <PageHeader title="Master data" subtitle="Customers, products and field targets" />
      <ErrorBanner error={resource.error || mutation.error} />
      <Tabs
        active={tab}
        onChange={setTab}
        items={[
          { id: 'doctors', label: 'Doctors' },
          { id: 'chemists', label: 'Chemists' },
          { id: 'stockists', label: 'Stockists' },
          { id: 'products', label: 'Products' },
          { id: 'categories', label: 'Categories' },
          { id: 'targets', label: 'Targets' },
        ]}
      />

      {tab === 'doctors' ? (
        <>
          <section className="panel">
            <h2>Create doctor</h2>
            <form className="form-grid" onSubmit={(event) => onCreate(event, '/api/v1/doctors', { ...doctorForm.values, is_active: true })}>
              <FormField label="Name"><input required value={doctorForm.values.name} onChange={doctorForm.setField('name')} /></FormField>
              <FormField label="Specialty"><input value={doctorForm.values.specialty} onChange={doctorForm.setField('specialty')} /></FormField>
              <FormField label="Qualification"><input value={doctorForm.values.qualification} onChange={doctorForm.setField('qualification')} /></FormField>
              <FormField label="Registration"><input value={doctorForm.values.registration_no} onChange={doctorForm.setField('registration_no')} /></FormField>
              <FormField label="Email"><input type="email" value={doctorForm.values.email} onChange={doctorForm.setField('email')} /></FormField>
              <FormField label="Phone"><input value={doctorForm.values.phone} onChange={doctorForm.setField('phone')} /></FormField>
              <FormField label="Territory ID"><input value={doctorForm.values.territory_id} onChange={doctorForm.setField('territory_id')} /></FormField>
              <FormField label="Address"><input value={doctorForm.values.address} onChange={doctorForm.setField('address')} /></FormField>
              <div className="form-actions"><button className="button primary" type="submit" disabled={mutation.busy}>Create doctor</button></div>
            </form>
          </section>
          <section className="panel"><h2>Doctor directory</h2><DataTable columns={doctorColumns} rows={resource.data?.doctors ?? []} getRowKey={(row) => row.id} loading={resource.loading} /></section>
        </>
      ) : null}

      {tab === 'chemists' ? <section className="panel"><h2>Chemists</h2><DataTable columns={chemistColumns} rows={resource.data?.chemists ?? []} getRowKey={(row) => row.id} loading={resource.loading} /></section> : null}
      {tab === 'stockists' ? <section className="panel"><h2>Stockists</h2><DataTable columns={stockistColumns} rows={resource.data?.stockists ?? []} getRowKey={(row) => row.id} loading={resource.loading} /></section> : null}

      {tab === 'products' ? (
        <>
          <section className="panel">
            <h2>Create product</h2>
            <form className="form-grid" onSubmit={(event) => onCreate(event, '/api/v1/products', { ...productForm.values, category_id: productForm.values.category_id || undefined, mrp: Number(productForm.values.mrp), is_active: true })}>
              <FormField label="Category">
                <select value={productForm.values.category_id} onChange={productForm.setField('category_id')}>
                  <option value="">None</option>
                  {(resource.data?.categories ?? []).map((category) => <option key={category.id} value={category.id}>{category.name}</option>)}
                </select>
              </FormField>
              <FormField label="Code"><input required value={productForm.values.code} onChange={productForm.setField('code')} /></FormField>
              <FormField label="Name"><input required value={productForm.values.name} onChange={productForm.setField('name')} /></FormField>
              <FormField label="Brand"><input value={productForm.values.brand} onChange={productForm.setField('brand')} /></FormField>
              <FormField label="Manufacturer"><input value={productForm.values.manufacturer} onChange={productForm.setField('manufacturer')} /></FormField>
              <FormField label="Composition"><input value={productForm.values.composition} onChange={productForm.setField('composition')} /></FormField>
              <FormField label="Pack size"><input value={productForm.values.pack_size} onChange={productForm.setField('pack_size')} /></FormField>
              <FormField label="Unit"><input value={productForm.values.unit} onChange={productForm.setField('unit')} /></FormField>
              <FormField label="MRP"><input type="number" min="0" step="0.01" value={productForm.values.mrp} onChange={productForm.setField('mrp')} /></FormField>
              <div className="form-actions"><button className="button primary" type="submit" disabled={mutation.busy}>Create product</button></div>
            </form>
          </section>
          <section className="panel"><h2>Product catalog</h2><DataTable columns={productColumns} rows={resource.data?.products ?? []} getRowKey={(row) => row.id} loading={resource.loading} /></section>
        </>
      ) : null}

      {tab === 'categories' ? <section className="panel"><h2>Product categories</h2><DataTable columns={categoryColumns} rows={resource.data?.categories ?? []} getRowKey={(row) => row.id} loading={resource.loading} /></section> : null}

      {tab === 'targets' ? (
        <>
          <section className="panel">
            <h2>Create target</h2>
            <form className="form-grid" onSubmit={(event) => onCreate(event, '/api/v1/targets', {
              ...targetForm.values,
              territory_id: targetForm.values.territory_id || undefined,
              target_visits: Number(targetForm.values.target_visits),
              target_sales: Number(targetForm.values.target_sales),
            })}>
              <FormField label="Entity type">
                <select value={targetForm.values.entity_type} onChange={targetForm.setField('entity_type')}>
                  <option value="DOCTOR">Doctor</option>
                  <option value="CHEMIST">Chemist</option>
                  <option value="STOCKIST">Stockist</option>
                </select>
              </FormField>
              <FormField label="Entity ID"><input required value={targetForm.values.entity_id} onChange={targetForm.setField('entity_id')} /></FormField>
              <FormField label="Employee">
                <select required value={targetForm.values.employee_id} onChange={targetForm.setField('employee_id')}>
                  <option value="">Select employee</option>
                  {(resource.data?.employees ?? []).map((employee) => <option key={employee.id} value={employee.id}>{employee.first_name} {employee.last_name}</option>)}
                </select>
              </FormField>
              <FormField label="Territory ID"><input value={targetForm.values.territory_id} onChange={targetForm.setField('territory_id')} /></FormField>
              <FormField label="Period start"><input required type="date" value={targetForm.values.period_start} onChange={targetForm.setField('period_start')} /></FormField>
              <FormField label="Period end"><input required type="date" value={targetForm.values.period_end} onChange={targetForm.setField('period_end')} /></FormField>
              <FormField label="Target visits"><input type="number" min="0" value={targetForm.values.target_visits} onChange={targetForm.setField('target_visits')} /></FormField>
              <FormField label="Target sales"><input type="number" min="0" step="0.01" value={targetForm.values.target_sales} onChange={targetForm.setField('target_sales')} /></FormField>
              <FormField label="Status">
                <select value={targetForm.values.status} onChange={targetForm.setField('status')}>
                  <option value="ACTIVE">Active</option>
                  <option value="CLOSED">Closed</option>
                  <option value="CANCELLED">Cancelled</option>
                </select>
              </FormField>
              <div className="form-actions"><button className="button primary" type="submit" disabled={mutation.busy}>Create target</button></div>
            </form>
          </section>
          <section className="panel"><h2>Targets</h2><DataTable columns={targetColumns} rows={resource.data?.targets ?? []} getRowKey={(row) => row.id} loading={resource.loading} /></section>
        </>
      ) : null}
    </>
  )
}

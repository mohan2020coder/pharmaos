CREATE TABLE dist_company_distributor_links (
    id                    uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id             uuid NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    distributor_tenant_id uuid NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    status                text NOT NULL DEFAULT 'ACTIVE' CHECK (status IN ('ACTIVE','SUSPENDED','ENDED')),
    effective_from        date NOT NULL DEFAULT CURRENT_DATE,
    effective_to          date,
    credit_limit          numeric(18,2) NOT NULL DEFAULT 0 CHECK (credit_limit >= 0),
    payment_terms_days    integer NOT NULL DEFAULT 0 CHECK (payment_terms_days >= 0),
    created_by            uuid REFERENCES users(id) ON DELETE SET NULL,
    created_at            timestamptz NOT NULL DEFAULT now(),
    updated_at            timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT uq_company_distributor UNIQUE (tenant_id, distributor_tenant_id),
    CONSTRAINT ck_distinct_distribution_tenants CHECK (tenant_id <> distributor_tenant_id),
    CONSTRAINT ck_distributor_link_dates CHECK (effective_to IS NULL OR effective_to >= effective_from)
);
CREATE INDEX idx_distributor_links_distributor ON dist_company_distributor_links (distributor_tenant_id, status);

CREATE FUNCTION pharmaos_active_distributor_link(p_company uuid, p_distributor uuid)
RETURNS boolean LANGUAGE sql STABLE AS $$
    SELECT EXISTS (
        SELECT 1 FROM dist_company_distributor_links l
        WHERE l.tenant_id = p_company
          AND l.distributor_tenant_id = p_distributor
          AND l.status = 'ACTIVE'
          AND l.effective_from <= CURRENT_DATE
          AND (l.effective_to IS NULL OR l.effective_to >= CURRENT_DATE)
    );
$$;

CREATE FUNCTION pharmaos_can_view_distribution_row(p_owner_tenant uuid, p_distributor_tenant uuid)
RETURNS boolean LANGUAGE sql STABLE AS $$
    SELECT p_owner_tenant = pharmaos_current_tenant_id()
        OR (pharmaos_current_tenant_id() = p_distributor_tenant
            AND pharmaos_active_distributor_link(p_owner_tenant, p_distributor_tenant))
        OR pharmaos_is_privileged();
$$;

CREATE FUNCTION pharmaos_can_write_distribution_row(p_owner_tenant uuid, p_distributor_tenant uuid)
RETURNS boolean LANGUAGE sql STABLE AS $$
    SELECT p_owner_tenant = pharmaos_current_tenant_id() OR pharmaos_is_privileged();
$$;

CREATE FUNCTION pharmaos_can_view_distributor_data(p_distributor_tenant uuid)
RETURNS boolean LANGUAGE sql STABLE AS $$
    SELECT p_distributor_tenant = pharmaos_current_tenant_id()
        OR pharmaos_active_distributor_link(pharmaos_current_tenant_id(), p_distributor_tenant)
        OR pharmaos_is_privileged();
$$;

CREATE TABLE dist_distributors (
    id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id   uuid NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    code        citext NOT NULL,
    name        text NOT NULL,
    license_no  text,
    email       citext,
    phone       text,
    address     text,
    status      text NOT NULL DEFAULT 'ACTIVE' CHECK (status IN ('ACTIVE','INACTIVE')),
    settings    jsonb NOT NULL DEFAULT '{}'::jsonb,
    created_at  timestamptz NOT NULL DEFAULT now(),
    updated_at  timestamptz NOT NULL DEFAULT now(),
    deleted_at  timestamptz,
    UNIQUE (tenant_id, code)
);
CREATE INDEX idx_distributors_tenant_status ON dist_distributors (tenant_id, status) WHERE deleted_at IS NULL;

CREATE TABLE dist_orders (
    id                    uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id             uuid NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    distributor_tenant_id uuid NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    link_id               uuid NOT NULL REFERENCES dist_company_distributor_links(id) ON DELETE RESTRICT,
    order_number          citext NOT NULL,
    status                text NOT NULL DEFAULT 'DRAFT' CHECK (status IN ('DRAFT','SUBMITTED','APPROVED','FULFILLED','CANCELLED')),
    currency_code         char(3) NOT NULL DEFAULT 'USD',
    total_amount          numeric(18,2) NOT NULL DEFAULT 0 CHECK (total_amount >= 0),
    idempotency_key       uuid NOT NULL,
    requested_by          uuid REFERENCES users(id) ON DELETE SET NULL,
    approved_by           uuid REFERENCES users(id) ON DELETE SET NULL,
    order_date            date NOT NULL DEFAULT CURRENT_DATE,
    required_by           date,
    submitted_at          timestamptz,
    approved_at           timestamptz,
    cancelled_at          timestamptz,
    remarks               text,
    created_at            timestamptz NOT NULL DEFAULT now(),
    updated_at            timestamptz NOT NULL DEFAULT now(),
    deleted_at            timestamptz,
    UNIQUE (tenant_id, idempotency_key),
    UNIQUE (tenant_id, order_number),
    CONSTRAINT ck_order_dates CHECK (required_by IS NULL OR required_by >= order_date)
);
CREATE INDEX idx_orders_tenant_status ON dist_orders (tenant_id, status) WHERE deleted_at IS NULL;
CREATE INDEX idx_orders_distributor_status ON dist_orders (distributor_tenant_id, status) WHERE deleted_at IS NULL;

CREATE TABLE dist_order_items (
    id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id   uuid NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    order_id    uuid NOT NULL REFERENCES dist_orders(id) ON DELETE CASCADE,
    product_id  uuid NOT NULL REFERENCES cat_products(id) ON DELETE RESTRICT,
    quantity    integer NOT NULL CHECK (quantity > 0),
    unit_price  numeric(18,2) NOT NULL DEFAULT 0 CHECK (unit_price >= 0),
    line_total  numeric(18,2) NOT NULL DEFAULT 0 CHECK (line_total >= 0),
    created_at  timestamptz NOT NULL DEFAULT now(),
    updated_at  timestamptz NOT NULL DEFAULT now(),
    UNIQUE (order_id, product_id)
);
CREATE INDEX idx_order_items_tenant_order ON dist_order_items (tenant_id, order_id);

CREATE TABLE dist_inventory (
    id               uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id        uuid NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    product_id       uuid NOT NULL REFERENCES cat_products(id) ON DELETE CASCADE,
    warehouse_code   text NOT NULL DEFAULT 'MAIN',
    quantity_on_hand integer NOT NULL DEFAULT 0 CHECK (quantity_on_hand >= 0),
    quantity_reserved integer NOT NULL DEFAULT 0 CHECK (quantity_reserved >= 0),
    version          bigint NOT NULL DEFAULT 1,
    created_at       timestamptz NOT NULL DEFAULT now(),
    updated_at       timestamptz NOT NULL DEFAULT now(),
    UNIQUE (tenant_id, product_id, warehouse_code)
);
CREATE INDEX idx_inventory_tenant_product ON dist_inventory (tenant_id, product_id);

CREATE TABLE dist_inventory_movements (
    id              uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       uuid NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    product_id      uuid NOT NULL REFERENCES cat_products(id) ON DELETE RESTRICT,
    warehouse_code  text NOT NULL DEFAULT 'MAIN',
    movement_type   text NOT NULL CHECK (movement_type IN ('RECEIPT','SALE','ADJUSTMENT','ORDER_RESERVE','ORDER_RELEASE')),
    quantity_delta  integer NOT NULL,
    reference_type  text,
    reference_id    uuid,
    idempotency_key uuid NOT NULL,
    created_by      uuid REFERENCES users(id) ON DELETE SET NULL,
    occurred_at     timestamptz NOT NULL DEFAULT now(),
    created_at      timestamptz NOT NULL DEFAULT now(),
    UNIQUE (tenant_id, idempotency_key)
);
CREATE INDEX idx_inventory_movements_tenant_product ON dist_inventory_movements (tenant_id, product_id, occurred_at DESC);

CREATE TABLE dist_secondary_sales (
    id                uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id         uuid NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    order_id          uuid REFERENCES dist_orders(id) ON DELETE SET NULL,
    company_tenant_id uuid REFERENCES tenants(id) ON DELETE SET NULL,
    product_id        uuid NOT NULL REFERENCES cat_products(id) ON DELETE RESTRICT,
    quantity          integer NOT NULL CHECK (quantity > 0),
    unit_price        numeric(18,2) NOT NULL DEFAULT 0 CHECK (unit_price >= 0),
    total_amount      numeric(18,2) NOT NULL DEFAULT 0 CHECK (total_amount >= 0),
    sale_date         date NOT NULL DEFAULT CURRENT_DATE,
    invoice_no        citext,
    customer_code     text,
    idempotency_key   uuid NOT NULL,
    created_by        uuid REFERENCES users(id) ON DELETE SET NULL,
    created_at        timestamptz NOT NULL DEFAULT now(),
    updated_at        timestamptz NOT NULL DEFAULT now(),
    UNIQUE (tenant_id, idempotency_key)
);
CREATE INDEX idx_secondary_sales_tenant_date ON dist_secondary_sales (tenant_id, sale_date DESC);
CREATE INDEX idx_secondary_sales_company ON dist_secondary_sales (company_tenant_id, sale_date DESC);

DO $$
DECLARE
    t text;
BEGIN
    FOREACH t IN ARRAY ARRAY[
        'dist_company_distributor_links','dist_distributors','dist_orders',
        'dist_order_items','dist_inventory','dist_inventory_movements','dist_secondary_sales'
    ] LOOP
        EXECUTE format('ALTER TABLE %I ENABLE ROW LEVEL SECURITY', t);
        EXECUTE format('ALTER TABLE %I FORCE ROW LEVEL SECURITY', t);
    END LOOP;
    FOREACH t IN ARRAY ARRAY[
        'dist_company_distributor_links','dist_distributors','dist_orders',
        'dist_order_items','dist_inventory','dist_secondary_sales'
    ] LOOP
        EXECUTE format('CREATE TRIGGER %I BEFORE UPDATE ON %I FOR EACH ROW EXECUTE FUNCTION pharmaos_set_updated_at()',
            'trg_' || t || '_updated_at', t);
    END LOOP;
END $$;

CREATE POLICY dist_links_isolation ON dist_company_distributor_links
    FOR SELECT USING (tenant_id = pharmaos_current_tenant_id()
        OR distributor_tenant_id = pharmaos_current_tenant_id()
        OR pharmaos_is_privileged());
CREATE POLICY dist_links_write ON dist_company_distributor_links
    FOR ALL USING (pharmaos_is_privileged() OR tenant_id = pharmaos_current_tenant_id())
    WITH CHECK (pharmaos_is_privileged() OR tenant_id = pharmaos_current_tenant_id());

CREATE POLICY dist_distributors_select ON dist_distributors
    FOR SELECT USING (tenant_id = pharmaos_current_tenant_id()
        OR pharmaos_active_distributor_link(pharmaos_current_tenant_id(), tenant_id)
        OR pharmaos_is_privileged());
CREATE POLICY dist_distributors_write ON dist_distributors
    FOR ALL USING (tenant_id = pharmaos_current_tenant_id() OR pharmaos_is_privileged())
    WITH CHECK (tenant_id = pharmaos_current_tenant_id() OR pharmaos_is_privileged());

CREATE POLICY dist_orders_select ON dist_orders
    FOR SELECT USING (pharmaos_can_view_distribution_row(tenant_id, distributor_tenant_id));
CREATE POLICY dist_orders_write ON dist_orders
    FOR ALL USING (pharmaos_can_write_distribution_row(tenant_id, distributor_tenant_id))
    WITH CHECK (pharmaos_can_write_distribution_row(tenant_id, distributor_tenant_id));

CREATE POLICY dist_order_items_select ON dist_order_items
    FOR SELECT USING (EXISTS (
        SELECT 1 FROM dist_orders o
        WHERE o.id = order_id AND pharmaos_can_view_distribution_row(o.tenant_id, o.distributor_tenant_id)
    ));
CREATE POLICY dist_order_items_write ON dist_order_items
    FOR ALL USING (EXISTS (
        SELECT 1 FROM dist_orders o
        WHERE o.id = order_id AND pharmaos_can_write_distribution_row(o.tenant_id, o.distributor_tenant_id)
    )) WITH CHECK (EXISTS (
        SELECT 1 FROM dist_orders o
        WHERE o.id = order_id AND pharmaos_can_write_distribution_row(o.tenant_id, o.distributor_tenant_id)
    ));

CREATE POLICY dist_inventory_select ON dist_inventory
    FOR SELECT USING (pharmaos_can_view_distributor_data(tenant_id));
CREATE POLICY dist_inventory_write ON dist_inventory
    FOR ALL USING (tenant_id = pharmaos_current_tenant_id() OR pharmaos_is_privileged())
    WITH CHECK (tenant_id = pharmaos_current_tenant_id() OR pharmaos_is_privileged());

CREATE POLICY dist_inventory_movements_select ON dist_inventory_movements
    FOR SELECT USING (pharmaos_can_view_distributor_data(tenant_id));
CREATE POLICY dist_inventory_movements_write ON dist_inventory_movements
    FOR ALL USING (tenant_id = pharmaos_current_tenant_id() OR pharmaos_is_privileged())
    WITH CHECK (tenant_id = pharmaos_current_tenant_id() OR pharmaos_is_privileged());

CREATE POLICY dist_secondary_sales_select ON dist_secondary_sales
    FOR SELECT USING (tenant_id = pharmaos_current_tenant_id()
        OR (company_tenant_id = pharmaos_current_tenant_id()
            AND pharmaos_active_distributor_link(company_tenant_id, tenant_id))
        OR pharmaos_is_privileged());
CREATE POLICY dist_secondary_sales_write ON dist_secondary_sales
    FOR ALL USING (tenant_id = pharmaos_current_tenant_id() OR pharmaos_is_privileged())
    WITH CHECK (tenant_id = pharmaos_current_tenant_id() OR pharmaos_is_privileged());

INSERT INTO permissions (code, module, description) VALUES
    ('link.manage','distributors','Create, suspend and end company-distributor links'),
    ('order.approve','orders','Approve distributor orders'),
    ('inventory.view','inventory','View distributor inventory'),
    ('inventory.manage','inventory','Record inventory receipts and adjustments')
ON CONFLICT (code) DO NOTHING;
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r JOIN permissions p ON p.code IN (
    'link.manage','order.approve','inventory.view','inventory.manage'
) WHERE r.code IN ('SUPER_ADMIN','COMPANY_ADMIN','DISTRIBUTOR_ADMIN','DISTRIBUTOR_MANAGER')
ON CONFLICT DO NOTHING;
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r JOIN permissions p ON p.code IN (
    'distributor.view','order.view','order.manage','inventory.view','secondary_sales.view'
) WHERE r.code IN ('MR','TBM','ASM','RSM','NSM','DIVISION_ADMIN')
ON CONFLICT DO NOTHING;

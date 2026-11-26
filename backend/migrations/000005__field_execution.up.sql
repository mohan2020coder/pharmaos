CREATE TABLE tour_plans (
    id           uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id    uuid NOT NULL DEFAULT pharmaos_current_tenant_id() REFERENCES tenants(id) ON DELETE CASCADE,
    employee_id  uuid NOT NULL REFERENCES employees(id) ON DELETE CASCADE,
    title        text NOT NULL,
    period_start date NOT NULL,
    period_end   date NOT NULL,
    status       text NOT NULL DEFAULT 'DRAFT' CHECK (status IN ('DRAFT','SUBMITTED','APPROVED','REJECTED')),
    submitted_at timestamptz,
    approved_by  uuid REFERENCES users(id) ON DELETE SET NULL,
    approved_at  timestamptz,
    comments     text,
    created_at   timestamptz NOT NULL DEFAULT now(),
    updated_at   timestamptz NOT NULL DEFAULT now(),
    deleted_at   timestamptz,
    CONSTRAINT tour_plan_dates CHECK (period_end >= period_start)
);
CREATE INDEX idx_tour_plans_tenant_employee_period ON tour_plans (tenant_id, employee_id, period_start, period_end) WHERE deleted_at IS NULL;
CREATE INDEX idx_tour_plans_tenant_status ON tour_plans (tenant_id, status) WHERE deleted_at IS NULL;

CREATE TABLE tour_plan_lines (
    id             uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id      uuid NOT NULL DEFAULT pharmaos_current_tenant_id() REFERENCES tenants(id) ON DELETE CASCADE,
    tour_plan_id   uuid NOT NULL REFERENCES tour_plans(id) ON DELETE CASCADE,
    planned_date   date NOT NULL,
    territory_id   uuid REFERENCES territories(id) ON DELETE SET NULL,
    entity_type    text NOT NULL CHECK (entity_type IN ('DOCTOR','CHEMIST','STOCKIST')),
    entity_id      uuid NOT NULL,
    purpose        text,
    sequence       integer NOT NULL DEFAULT 1 CHECK (sequence > 0),
    status         text NOT NULL DEFAULT 'PLANNED' CHECK (status IN ('PLANNED','COMPLETED','SKIPPED')),
    created_at     timestamptz NOT NULL DEFAULT now(),
    updated_at     timestamptz NOT NULL DEFAULT now(),
    UNIQUE (tour_plan_id, sequence)
);
CREATE INDEX idx_tour_plan_lines_plan_date ON tour_plan_lines (tenant_id, tour_plan_id, planned_date);

CREATE TABLE visits (
    id                  uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id           uuid NOT NULL DEFAULT pharmaos_current_tenant_id() REFERENCES tenants(id) ON DELETE CASCADE,
    employee_id         uuid NOT NULL REFERENCES employees(id) ON DELETE CASCADE,
    tour_plan_line_id   uuid REFERENCES tour_plan_lines(id) ON DELETE SET NULL,
    territory_id        uuid REFERENCES territories(id) ON DELETE SET NULL,
    entity_type         text NOT NULL CHECK (entity_type IN ('DOCTOR','CHEMIST','STOCKIST')),
    entity_id           uuid NOT NULL,
    client_reference_id uuid,
    visited_at          timestamptz NOT NULL DEFAULT now(),
    check_in_lat        double precision,
    check_in_lng        double precision,
    check_in_accuracy   double precision CHECK (check_in_accuracy IS NULL OR check_in_accuracy >= 0),
    check_out_lat       double precision,
    check_out_lng       double precision,
    status              text NOT NULL DEFAULT 'DRAFT' CHECK (status IN ('DRAFT','SUBMITTED','APPROVED','REJECTED')),
    remarks             text,
    submitted_at        timestamptz,
    created_at          timestamptz NOT NULL DEFAULT now(),
    updated_at          timestamptz NOT NULL DEFAULT now(),
    deleted_at          timestamptz,
    CONSTRAINT visits_check_in_coordinates CHECK (
        (check_in_lat IS NULL AND check_in_lng IS NULL) OR
        (check_in_lat BETWEEN -90 AND 90 AND check_in_lng BETWEEN -180 AND 180)
    ),
    CONSTRAINT visits_check_out_coordinates CHECK (
        (check_out_lat IS NULL AND check_out_lng IS NULL) OR
        (check_out_lat BETWEEN -90 AND 90 AND check_out_lng BETWEEN -180 AND 180)
    ),
    UNIQUE (tenant_id, client_reference_id)
);
CREATE INDEX idx_visits_tenant_employee_time ON visits (tenant_id, employee_id, visited_at DESC) WHERE deleted_at IS NULL;
CREATE INDEX idx_visits_tenant_entity ON visits (tenant_id, entity_type, entity_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_visits_tenant_status ON visits (tenant_id, status) WHERE deleted_at IS NULL;

CREATE TABLE daily_call_reports (
    id            uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id     uuid NOT NULL DEFAULT pharmaos_current_tenant_id() REFERENCES tenants(id) ON DELETE CASCADE,
    employee_id   uuid NOT NULL REFERENCES employees(id) ON DELETE CASCADE,
    report_date   date NOT NULL,
    total_visits  integer NOT NULL DEFAULT 0 CHECK (total_visits >= 0),
    status        text NOT NULL DEFAULT 'DRAFT' CHECK (status IN ('DRAFT','SUBMITTED','APPROVED','REJECTED')),
    submitted_at  timestamptz,
    approved_by   uuid REFERENCES users(id) ON DELETE SET NULL,
    approved_at   timestamptz,
    remarks       text,
    created_at    timestamptz NOT NULL DEFAULT now(),
    updated_at    timestamptz NOT NULL DEFAULT now(),
    deleted_at    timestamptz,
    UNIQUE (tenant_id, employee_id, report_date)
);
CREATE INDEX idx_dcr_tenant_employee_date ON daily_call_reports (tenant_id, employee_id, report_date DESC) WHERE deleted_at IS NULL;
CREATE INDEX idx_dcr_tenant_status ON daily_call_reports (tenant_id, status) WHERE deleted_at IS NULL;

CREATE TABLE dcr_product_details (
    id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id   uuid NOT NULL DEFAULT pharmaos_current_tenant_id() REFERENCES tenants(id) ON DELETE CASCADE,
    dcr_id      uuid NOT NULL REFERENCES daily_call_reports(id) ON DELETE CASCADE,
    product_id  uuid NOT NULL REFERENCES cat_products(id) ON DELETE CASCADE,
    quantity    integer NOT NULL DEFAULT 0 CHECK (quantity >= 0),
    sample_count integer NOT NULL DEFAULT 0 CHECK (sample_count >= 0),
    outcome     text,
    remarks     text,
    created_at  timestamptz NOT NULL DEFAULT now(),
    updated_at  timestamptz NOT NULL DEFAULT now(),
    UNIQUE (dcr_id, product_id)
);
CREATE INDEX idx_dcr_product_details_tenant_dcr ON dcr_product_details (tenant_id, dcr_id);

CREATE TABLE visit_attachments (
    id           uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id    uuid NOT NULL DEFAULT pharmaos_current_tenant_id() REFERENCES tenants(id) ON DELETE CASCADE,
    visit_id     uuid NOT NULL REFERENCES visits(id) ON DELETE CASCADE,
    file_name    text NOT NULL,
    content_type text,
    storage_key  text NOT NULL,
    checksum     text,
    created_at   timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX idx_visit_attachments_tenant_visit ON visit_attachments (tenant_id, visit_id);

DO $$
DECLARE
    t text;
BEGIN
    FOREACH t IN ARRAY ARRAY[
        'tour_plans','tour_plan_lines','visits','daily_call_reports','dcr_product_details','visit_attachments'
    ] LOOP
        EXECUTE format('ALTER TABLE %I ENABLE ROW LEVEL SECURITY', t);
        EXECUTE format('ALTER TABLE %I FORCE ROW LEVEL SECURITY', t);
        EXECUTE format('CREATE POLICY %I ON %I USING (tenant_id = pharmaos_current_tenant_id() OR pharmaos_is_privileged()) WITH CHECK (tenant_id = pharmaos_current_tenant_id() OR pharmaos_is_privileged())',
            t || '_tenant_isolation', t);
    END LOOP;
    FOREACH t IN ARRAY ARRAY[
        'tour_plans','tour_plan_lines','visits','daily_call_reports','dcr_product_details'
    ] LOOP
        EXECUTE format('CREATE TRIGGER %I BEFORE UPDATE ON %I FOR EACH ROW EXECUTE FUNCTION pharmaos_set_updated_at()',
            'trg_' || t || '_updated_at', t);
    END LOOP;
END $$;

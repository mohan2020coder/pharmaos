CREATE TABLE business_divisions (
    id            uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id     uuid NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    code          text NOT NULL,
    name          text NOT NULL,
    description   text,
    status        text NOT NULL DEFAULT 'ACTIVE' CHECK (status IN ('ACTIVE','INACTIVE')),
    manager_user_id uuid REFERENCES users(id) ON DELETE SET NULL,
    settings      jsonb NOT NULL DEFAULT '{}'::jsonb,
    created_at    timestamptz NOT NULL DEFAULT now(),
    updated_at    timestamptz NOT NULL DEFAULT now(),
    deleted_at    timestamptz,
    UNIQUE (tenant_id, code)
);
CREATE INDEX idx_divisions_tenant_status ON business_divisions (tenant_id, status) WHERE deleted_at IS NULL;

CREATE TABLE branches (
    id            uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id     uuid NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    division_id   uuid REFERENCES business_divisions(id) ON DELETE SET NULL,
    code          text NOT NULL,
    name          text NOT NULL,
    address       text,
    city          text,
    state         text,
    postal_code   text,
    country_code  char(2),
    status        text NOT NULL DEFAULT 'ACTIVE' CHECK (status IN ('ACTIVE','INACTIVE')),
    created_at    timestamptz NOT NULL DEFAULT now(),
    updated_at    timestamptz NOT NULL DEFAULT now(),
    deleted_at    timestamptz,
    UNIQUE (tenant_id, code)
);
CREATE INDEX idx_branches_tenant_division ON branches (tenant_id, division_id) WHERE deleted_at IS NULL;

CREATE TABLE regions (
    id            uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id     uuid NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    division_id   uuid NOT NULL REFERENCES business_divisions(id) ON DELETE CASCADE,
    code          text NOT NULL,
    name          text NOT NULL,
    status        text NOT NULL DEFAULT 'ACTIVE' CHECK (status IN ('ACTIVE','INACTIVE')),
    created_at    timestamptz NOT NULL DEFAULT now(),
    updated_at    timestamptz NOT NULL DEFAULT now(),
    deleted_at    timestamptz,
    UNIQUE (tenant_id, code)
);
CREATE INDEX idx_regions_tenant_division ON regions (tenant_id, division_id) WHERE deleted_at IS NULL;

CREATE TABLE territories (
    id            uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id     uuid NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    division_id   uuid NOT NULL REFERENCES business_divisions(id) ON DELETE CASCADE,
    region_id     uuid NOT NULL REFERENCES regions(id) ON DELETE CASCADE,
    branch_id     uuid REFERENCES branches(id) ON DELETE SET NULL,
    code          text NOT NULL,
    name          text NOT NULL,
    city          text,
    status        text NOT NULL DEFAULT 'ACTIVE' CHECK (status IN ('ACTIVE','INACTIVE')),
    created_at    timestamptz NOT NULL DEFAULT now(),
    updated_at    timestamptz NOT NULL DEFAULT now(),
    deleted_at    timestamptz,
    UNIQUE (tenant_id, code)
);
CREATE INDEX idx_territories_tenant_region ON territories (tenant_id, region_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_territories_tenant_division ON territories (tenant_id, division_id) WHERE deleted_at IS NULL;

CREATE TABLE positions (
    id                  uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id           uuid NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    code                text NOT NULL,
    title               text NOT NULL,
    level               integer NOT NULL DEFAULT 1 CHECK (level BETWEEN 1 AND 12),
    is_management       boolean NOT NULL DEFAULT false,
    parent_position_id  uuid REFERENCES positions(id) ON DELETE SET NULL,
    created_at          timestamptz NOT NULL DEFAULT now(),
    updated_at          timestamptz NOT NULL DEFAULT now(),
    UNIQUE (tenant_id, code)
);

CREATE TABLE employees (
    id                  uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id           uuid NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    employee_code       text NOT NULL,
    user_id             uuid UNIQUE REFERENCES users(id) ON DELETE SET NULL,
    first_name          text NOT NULL,
    last_name           text NOT NULL,
    email               citext,
    phone               text,
    status              text NOT NULL DEFAULT 'ACTIVE' CHECK (status IN ('ACTIVE','INACTIVE','TRANSFERRED')),
    joining_date        date,
    division_id         uuid REFERENCES business_divisions(id) ON DELETE SET NULL,
    region_id           uuid REFERENCES regions(id) ON DELETE SET NULL,
    territory_id        uuid REFERENCES territories(id) ON DELETE SET NULL,
    manager_employee_id uuid REFERENCES employees(id) ON DELETE SET NULL,
    position_id         uuid REFERENCES positions(id) ON DELETE SET NULL,
    created_at          timestamptz NOT NULL DEFAULT now(),
    updated_at          timestamptz NOT NULL DEFAULT now(),
    deleted_at          timestamptz,
    UNIQUE (tenant_id, employee_code)
);
CREATE INDEX idx_employees_tenant_territory ON employees (tenant_id, territory_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_employees_tenant_manager ON employees (tenant_id, manager_employee_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_employees_tenant_division ON employees (tenant_id, division_id) WHERE deleted_at IS NULL;

CREATE TABLE employee_positions (
    id           uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id    uuid NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    employee_id  uuid NOT NULL REFERENCES employees(id) ON DELETE CASCADE,
    position_id  uuid NOT NULL REFERENCES positions(id) ON DELETE CASCADE,
    valid_from   date NOT NULL,
    valid_to     date,
    is_primary   boolean NOT NULL DEFAULT true,
    created_at   timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT employee_positions_dates CHECK (valid_to IS NULL OR valid_to >= valid_from)
);
CREATE INDEX idx_employee_positions_employee ON employee_positions (employee_id, valid_from, valid_to);

CREATE TABLE employee_territory_assignments (
    id            uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id     uuid NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    employee_id   uuid NOT NULL REFERENCES employees(id) ON DELETE CASCADE,
    territory_id  uuid NOT NULL REFERENCES territories(id) ON DELETE CASCADE,
    assignment_type text NOT NULL DEFAULT 'PRIMARY' CHECK (assignment_type IN ('PRIMARY','ADDITIONAL','TEMPORARY')),
    valid_from    date NOT NULL,
    valid_to      date,
    created_at    timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT employee_territory_dates CHECK (valid_to IS NULL OR valid_to >= valid_from)
);
CREATE INDEX idx_employee_territory_employee ON employee_territory_assignments (employee_id, valid_from, valid_to);
CREATE INDEX idx_employee_territory_territory ON employee_territory_assignments (territory_id, valid_from, valid_to);

CREATE TABLE employee_manager_history (
    id                  uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id           uuid NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    employee_id         uuid NOT NULL REFERENCES employees(id) ON DELETE CASCADE,
    manager_employee_id uuid NOT NULL REFERENCES employees(id) ON DELETE CASCADE,
    valid_from          date NOT NULL,
    valid_to            date,
    created_at          timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT employee_manager_dates CHECK (valid_to IS NULL OR valid_to >= valid_from)
);
CREATE INDEX idx_employee_manager_history_employee ON employee_manager_history (employee_id, valid_from, valid_to);

CREATE TABLE employee_transfers (
    id                 uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id          uuid NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    employee_id        uuid NOT NULL REFERENCES employees(id) ON DELETE CASCADE,
    from_division_id   uuid REFERENCES business_divisions(id),
    to_division_id     uuid REFERENCES business_divisions(id),
    from_territory_id  uuid REFERENCES territories(id),
    to_territory_id    uuid REFERENCES territories(id),
    from_manager_id    uuid REFERENCES employees(id),
    to_manager_id      uuid REFERENCES employees(id),
    effective_from     date NOT NULL,
    reason             text,
    approved_by        uuid REFERENCES users(id),
    created_at         timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX idx_employee_transfers_employee ON employee_transfers (employee_id, effective_from DESC);

CREATE FUNCTION pharmaos_validate_territory_hierarchy() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM regions
        WHERE id = NEW.region_id AND tenant_id = NEW.tenant_id AND division_id = NEW.division_id
    ) THEN
        RAISE EXCEPTION 'territory region does not belong to division';
    END IF;
    RETURN NEW;
END $$;
CREATE TRIGGER trg_territories_validate_hierarchy
    BEFORE INSERT OR UPDATE ON territories
    FOR EACH ROW EXECUTE FUNCTION pharmaos_validate_territory_hierarchy();

DO $$
DECLARE
    t text;
BEGIN
    FOREACH t IN ARRAY ARRAY[
        'business_divisions','branches','regions','territories','positions','employees',
        'employee_positions','employee_territory_assignments','employee_manager_history','employee_transfers'
    ] LOOP
        EXECUTE format('ALTER TABLE %I ENABLE ROW LEVEL SECURITY', t);
        EXECUTE format('ALTER TABLE %I FORCE ROW LEVEL SECURITY', t);
        EXECUTE format('CREATE POLICY %I ON %I USING (tenant_id = pharmaos_current_tenant_id() OR pharmaos_is_privileged()) WITH CHECK (tenant_id = pharmaos_current_tenant_id() OR pharmaos_is_privileged())',
            t || '_tenant_isolation', t);
    END LOOP;
    FOREACH t IN ARRAY ARRAY['business_divisions','branches','regions','territories','positions','employees'] LOOP
        EXECUTE format('CREATE TRIGGER %I BEFORE UPDATE ON %I FOR EACH ROW EXECUTE FUNCTION pharmaos_set_updated_at()',
            'trg_' || t || '_updated_at', t);
    END LOOP;
END $$;

CREATE TABLE crm_doctors (
    id             uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id      uuid NOT NULL DEFAULT pharmaos_current_tenant_id() REFERENCES tenants(id) ON DELETE CASCADE,
    external_id    citext,
    name           text NOT NULL,
    specialty      text,
    qualification  text,
    registration_no text,
    email          citext,
    phone          text,
    address        text,
    territory_id   uuid REFERENCES territories(id) ON DELETE SET NULL,
    division_id    uuid REFERENCES business_divisions(id) ON DELETE SET NULL,
    is_active      boolean NOT NULL DEFAULT true,
    settings       jsonb NOT NULL DEFAULT '{}'::jsonb,
    created_at     timestamptz NOT NULL DEFAULT now(),
    updated_at     timestamptz NOT NULL DEFAULT now(),
    deleted_at     timestamptz,
    UNIQUE (tenant_id, external_id)
);
CREATE INDEX idx_crm_doctors_tenant_active_name ON crm_doctors (tenant_id, is_active, name) WHERE deleted_at IS NULL;
CREATE INDEX idx_crm_doctors_tenant_territory ON crm_doctors (tenant_id, territory_id) WHERE deleted_at IS NULL;

CREATE TABLE crm_chemists (
    id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id   uuid NOT NULL DEFAULT pharmaos_current_tenant_id() REFERENCES tenants(id) ON DELETE CASCADE,
    external_id citext,
    name        text NOT NULL,
    owner_name  text,
    license_no  text,
    email       citext,
    phone       text,
    address     text,
    territory_id uuid REFERENCES territories(id) ON DELETE SET NULL,
    division_id  uuid REFERENCES business_divisions(id) ON DELETE SET NULL,
    is_active   boolean NOT NULL DEFAULT true,
    settings    jsonb NOT NULL DEFAULT '{}'::jsonb,
    created_at  timestamptz NOT NULL DEFAULT now(),
    updated_at  timestamptz NOT NULL DEFAULT now(),
    deleted_at  timestamptz,
    UNIQUE (tenant_id, external_id)
);
CREATE INDEX idx_crm_chemists_tenant_active_name ON crm_chemists (tenant_id, is_active, name) WHERE deleted_at IS NULL;
CREATE INDEX idx_crm_chemists_tenant_territory ON crm_chemists (tenant_id, territory_id) WHERE deleted_at IS NULL;

CREATE TABLE crm_stockists (
    id                 uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id          uuid NOT NULL DEFAULT pharmaos_current_tenant_id() REFERENCES tenants(id) ON DELETE CASCADE,
    external_id        citext,
    name               text NOT NULL,
    license_no         text,
    gst_number         text,
    email              citext,
    phone              text,
    address            text,
    territory_id       uuid REFERENCES territories(id) ON DELETE SET NULL,
    division_id        uuid REFERENCES business_divisions(id) ON DELETE SET NULL,
    credit_limit       numeric(14,2) NOT NULL DEFAULT 0 CHECK (credit_limit >= 0),
    payment_terms_days integer NOT NULL DEFAULT 0 CHECK (payment_terms_days >= 0),
    is_active          boolean NOT NULL DEFAULT true,
    settings           jsonb NOT NULL DEFAULT '{}'::jsonb,
    created_at         timestamptz NOT NULL DEFAULT now(),
    updated_at         timestamptz NOT NULL DEFAULT now(),
    deleted_at         timestamptz,
    UNIQUE (tenant_id, external_id)
);
CREATE INDEX idx_crm_stockists_tenant_active_name ON crm_stockists (tenant_id, is_active, name) WHERE deleted_at IS NULL;
CREATE INDEX idx_crm_stockists_tenant_territory ON crm_stockists (tenant_id, territory_id) WHERE deleted_at IS NULL;

CREATE TABLE crm_doctor_territories (
    id           uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id    uuid NOT NULL DEFAULT pharmaos_current_tenant_id() REFERENCES tenants(id) ON DELETE CASCADE,
    doctor_id    uuid NOT NULL REFERENCES crm_doctors(id) ON DELETE CASCADE,
    territory_id uuid NOT NULL REFERENCES territories(id) ON DELETE CASCADE,
    valid_from   date NOT NULL DEFAULT CURRENT_DATE,
    valid_to     date,
    is_primary   boolean NOT NULL DEFAULT true,
    created_at   timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT crm_doctor_territory_dates CHECK (valid_to IS NULL OR valid_to >= valid_from)
);
CREATE INDEX idx_crm_doctor_territories_doctor ON crm_doctor_territories (doctor_id, valid_from, valid_to);

CREATE TABLE cat_product_categories (
    id                uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id         uuid NOT NULL DEFAULT pharmaos_current_tenant_id() REFERENCES tenants(id) ON DELETE CASCADE,
    code              citext NOT NULL,
    name              text NOT NULL,
    parent_category_id uuid REFERENCES cat_product_categories(id) ON DELETE SET NULL,
    is_active         boolean NOT NULL DEFAULT true,
    created_at        timestamptz NOT NULL DEFAULT now(),
    updated_at        timestamptz NOT NULL DEFAULT now(),
    deleted_at        timestamptz,
    UNIQUE (tenant_id, code)
);
CREATE INDEX idx_cat_product_categories_tenant_active ON cat_product_categories (tenant_id, is_active, name) WHERE deleted_at IS NULL;

CREATE TABLE cat_products (
    id            uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id     uuid NOT NULL DEFAULT pharmaos_current_tenant_id() REFERENCES tenants(id) ON DELETE CASCADE,
    category_id   uuid REFERENCES cat_product_categories(id) ON DELETE SET NULL,
    code          citext NOT NULL,
    name          text NOT NULL,
    brand         text,
    manufacturer  text,
    composition   text,
    pack_size     text,
    unit          text,
    mrp           numeric(14,2) NOT NULL DEFAULT 0 CHECK (mrp >= 0),
    is_active     boolean NOT NULL DEFAULT true,
    settings      jsonb NOT NULL DEFAULT '{}'::jsonb,
    created_at    timestamptz NOT NULL DEFAULT now(),
    updated_at    timestamptz NOT NULL DEFAULT now(),
    deleted_at    timestamptz,
    UNIQUE (tenant_id, code)
);
CREATE INDEX idx_cat_products_tenant_active_name ON cat_products (tenant_id, is_active, name) WHERE deleted_at IS NULL;
CREATE INDEX idx_cat_products_tenant_category ON cat_products (tenant_id, category_id) WHERE deleted_at IS NULL;

CREATE TABLE crm_targets (
    id           uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id    uuid NOT NULL DEFAULT pharmaos_current_tenant_id() REFERENCES tenants(id) ON DELETE CASCADE,
    entity_type  text NOT NULL CHECK (entity_type IN ('DOCTOR','CHEMIST','STOCKIST')),
    entity_id    uuid NOT NULL,
    employee_id  uuid NOT NULL REFERENCES employees(id) ON DELETE CASCADE,
    territory_id uuid REFERENCES territories(id) ON DELETE SET NULL,
    period_start date NOT NULL,
    period_end   date NOT NULL,
    target_visits integer NOT NULL DEFAULT 0 CHECK (target_visits >= 0),
    target_sales  numeric(14,2) NOT NULL DEFAULT 0 CHECK (target_sales >= 0),
    status       text NOT NULL DEFAULT 'ACTIVE' CHECK (status IN ('ACTIVE','CLOSED','CANCELLED')),
    created_at   timestamptz NOT NULL DEFAULT now(),
    updated_at   timestamptz NOT NULL DEFAULT now(),
    deleted_at   timestamptz,
    CONSTRAINT crm_target_dates CHECK (period_end >= period_start),
    UNIQUE (tenant_id, entity_type, entity_id, employee_id, period_start)
);
CREATE INDEX idx_crm_targets_tenant_employee_period ON crm_targets (tenant_id, employee_id, period_start, period_end) WHERE deleted_at IS NULL;
CREATE INDEX idx_crm_targets_tenant_entity ON crm_targets (tenant_id, entity_type, entity_id) WHERE deleted_at IS NULL;

DO $$
DECLARE
    t text;
BEGIN
    FOREACH t IN ARRAY ARRAY[
        'crm_doctors','crm_chemists','crm_stockists','crm_doctor_territories',
        'cat_product_categories','cat_products','crm_targets'
    ] LOOP
        EXECUTE format('ALTER TABLE %I ENABLE ROW LEVEL SECURITY', t);
        EXECUTE format('ALTER TABLE %I FORCE ROW LEVEL SECURITY', t);
        EXECUTE format('CREATE POLICY %I ON %I USING (tenant_id = pharmaos_current_tenant_id() OR pharmaos_is_privileged()) WITH CHECK (tenant_id = pharmaos_current_tenant_id() OR pharmaos_is_privileged())',
            t || '_tenant_isolation', t);
    END LOOP;
    FOREACH t IN ARRAY ARRAY[
        'crm_doctors','crm_chemists','crm_stockists','cat_product_categories','cat_products','crm_targets'
    ] LOOP
        EXECUTE format('CREATE TRIGGER %I BEFORE UPDATE ON %I FOR EACH ROW EXECUTE FUNCTION pharmaos_set_updated_at()',
            'trg_' || t || '_updated_at', t);
    END LOOP;
END $$;

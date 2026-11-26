CREATE TABLE tenants (
    id              uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    code            citext NOT NULL,
    name            text NOT NULL,
    tenant_type     text NOT NULL CHECK (tenant_type IN ('PHARMA_COMPANY', 'DISTRIBUTOR')),
    status          text NOT NULL DEFAULT 'ACTIVE' CHECK (status IN ('ACTIVE', 'SUSPENDED', 'INACTIVE')),
    is_platform     boolean NOT NULL DEFAULT false,
    country_code    char(2),
    timezone        text NOT NULL DEFAULT 'UTC',
    currency_code   char(3) NOT NULL DEFAULT 'USD',
    settings        jsonb NOT NULL DEFAULT '{}'::jsonb,
    created_at      timestamptz NOT NULL DEFAULT now(),
    updated_at      timestamptz NOT NULL DEFAULT now(),
    deleted_at      timestamptz,
    CONSTRAINT tenants_code_unique UNIQUE (code)
);
CREATE INDEX idx_tenants_type_status ON tenants (tenant_type, status) WHERE deleted_at IS NULL;
CREATE TRIGGER trg_tenants_updated_at BEFORE UPDATE ON tenants FOR EACH ROW EXECUTE FUNCTION pharmaos_set_updated_at();

CREATE TABLE tenant_settings (
    tenant_id   uuid NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    key         text NOT NULL,
    value       jsonb NOT NULL,
    updated_at  timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (tenant_id, key)
);

CREATE TABLE legal_entities (
    id                   uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id            uuid NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    legal_name           text NOT NULL,
    registration_number  text,
    tax_identifier       text,
    address_line1        text,
    address_line2        text,
    city                 text,
    state                text,
    postal_code          text,
    country_code         char(2),
    status               text NOT NULL DEFAULT 'ACTIVE' CHECK (status IN ('ACTIVE', 'INACTIVE')),
    created_at           timestamptz NOT NULL DEFAULT now(),
    updated_at           timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX idx_legal_entities_tenant ON legal_entities (tenant_id);
CREATE TRIGGER trg_legal_entities_updated_at BEFORE UPDATE ON legal_entities FOR EACH ROW EXECUTE FUNCTION pharmaos_set_updated_at();

CREATE TABLE permissions (
    id           uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    code         citext NOT NULL UNIQUE,
    module       text NOT NULL,
    description  text NOT NULL DEFAULT '',
    created_at   timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE roles (
    id           uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id    uuid REFERENCES tenants(id) ON DELETE CASCADE,
    code         citext NOT NULL,
    name         text NOT NULL,
    description  text NOT NULL DEFAULT '',
    is_system    boolean NOT NULL DEFAULT false,
    is_privileged boolean NOT NULL DEFAULT false,
    data_scope   text NOT NULL DEFAULT 'TENANT' CHECK (data_scope IN ('SELF','TEAM','TERRITORY','REGION','DIVISION','TENANT')),
    created_at   timestamptz NOT NULL DEFAULT now(),
    updated_at   timestamptz NOT NULL DEFAULT now(),
    deleted_at   timestamptz
);
CREATE UNIQUE INDEX roles_tenant_code_unique ON roles (tenant_id, code) NULLS NOT DISTINCT WHERE deleted_at IS NULL;
CREATE INDEX idx_roles_tenant ON roles (tenant_id) WHERE deleted_at IS NULL;
CREATE TRIGGER trg_roles_updated_at BEFORE UPDATE ON roles FOR EACH ROW EXECUTE FUNCTION pharmaos_set_updated_at();

CREATE TABLE role_permissions (
    role_id        uuid NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    permission_id  uuid NOT NULL REFERENCES permissions(id) ON DELETE CASCADE,
    created_at     timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (role_id, permission_id)
);

CREATE TABLE users (
    id                     uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id              uuid NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    email                  citext NOT NULL,
    password_hash          text NOT NULL,
    first_name             text NOT NULL,
    last_name              text NOT NULL,
    phone                  text,
    status                 text NOT NULL DEFAULT 'ACTIVE' CHECK (status IN ('ACTIVE', 'INACTIVE', 'LOCKED')),
    email_verified         boolean NOT NULL DEFAULT false,
    must_change_password   boolean NOT NULL DEFAULT false,
    failed_login_attempts  integer NOT NULL DEFAULT 0,
    locked_until           timestamptz,
    last_login_at          timestamptz,
    created_at             timestamptz NOT NULL DEFAULT now(),
    updated_at             timestamptz NOT NULL DEFAULT now(),
    deleted_at             timestamptz,
    CONSTRAINT users_tenant_email_unique UNIQUE (tenant_id, email)
);
CREATE INDEX idx_users_email ON users (email) WHERE deleted_at IS NULL;
CREATE INDEX idx_users_tenant_status ON users (tenant_id, status) WHERE deleted_at IS NULL;
CREATE TRIGGER trg_users_updated_at BEFORE UPDATE ON users FOR EACH ROW EXECUTE FUNCTION pharmaos_set_updated_at();

CREATE TABLE user_roles (
    user_id     uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role_id     uuid NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    scope_type  text NOT NULL DEFAULT 'TENANT' CHECK (scope_type IN ('SELF','TERRITORY','REGION','DIVISION','TENANT','PARTNER')),
    scope_id    uuid NOT NULL DEFAULT '00000000-0000-0000-0000-000000000000',
    valid_from  date NOT NULL DEFAULT CURRENT_DATE,
    valid_to    date,
    created_at  timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (user_id, role_id, scope_type, scope_id)
);
CREATE INDEX idx_user_roles_role ON user_roles (role_id);
CREATE INDEX idx_user_roles_user_valid ON user_roles (user_id, valid_from, valid_to);

CREATE TABLE auth_sessions (
    id                uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id           uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    tenant_id         uuid NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    family_id         uuid NOT NULL,
    refresh_token_hash text NOT NULL,
    ip_address        inet,
    user_agent        text,
    device_id         text,
    expires_at        timestamptz NOT NULL,
    revoked_at        timestamptz,
    created_at        timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX idx_auth_sessions_user ON auth_sessions (user_id, revoked_at);
CREATE UNIQUE INDEX idx_auth_sessions_refresh_hash ON auth_sessions (refresh_token_hash);
CREATE INDEX idx_auth_sessions_family ON auth_sessions (family_id);

CREATE TABLE password_reset_tokens (
    id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash  text NOT NULL UNIQUE,
    expires_at  timestamptz NOT NULL,
    used_at     timestamptz,
    created_at  timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE idempotency_keys (
    tenant_id    uuid NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    key          uuid NOT NULL,
    operation    text NOT NULL,
    entity_id    uuid,
    result       jsonb,
    status       text NOT NULL DEFAULT 'PENDING' CHECK (status IN ('PENDING','COMPLETED','FAILED')),
    error_message text,
    expires_at   timestamptz NOT NULL DEFAULT (now() + interval '7 days'),
    created_at   timestamptz NOT NULL DEFAULT now(),
    updated_at   timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (tenant_id, key)
);

CREATE TABLE audit_logs (
    id             bigserial PRIMARY KEY,
    tenant_id      uuid,
    user_id        uuid,
    action         text NOT NULL,
    module         text NOT NULL,
    entity_type    text NOT NULL,
    entity_id      text,
    previous_values jsonb,
    new_values     jsonb,
    ip_address     inet,
    user_agent     text,
    correlation_id text,
    created_at     timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX idx_audit_logs_tenant_time ON audit_logs (tenant_id, created_at DESC);
CREATE INDEX idx_audit_logs_entity ON audit_logs (entity_type, entity_id);
CREATE INDEX idx_audit_logs_action ON audit_logs (action, created_at DESC);

ALTER TABLE roles ENABLE ROW LEVEL SECURITY;
ALTER TABLE roles FORCE ROW LEVEL SECURITY;
CREATE POLICY roles_tenant_isolation ON roles
    USING (tenant_id IS NULL OR tenant_id = pharmaos_current_tenant_id() OR pharmaos_is_privileged())
    WITH CHECK (tenant_id IS NULL OR tenant_id = pharmaos_current_tenant_id() OR pharmaos_is_privileged());

ALTER TABLE users ENABLE ROW LEVEL SECURITY;
ALTER TABLE users FORCE ROW LEVEL SECURITY;
CREATE POLICY users_tenant_isolation ON users
    USING (tenant_id = pharmaos_current_tenant_id() OR pharmaos_is_privileged())
    WITH CHECK (tenant_id = pharmaos_current_tenant_id() OR pharmaos_is_privileged());

ALTER TABLE tenant_settings ENABLE ROW LEVEL SECURITY;
ALTER TABLE tenant_settings FORCE ROW LEVEL SECURITY;
CREATE POLICY tenant_settings_isolation ON tenant_settings
    USING (tenant_id = pharmaos_current_tenant_id() OR pharmaos_is_privileged())
    WITH CHECK (tenant_id = pharmaos_current_tenant_id() OR pharmaos_is_privileged());

ALTER TABLE legal_entities ENABLE ROW LEVEL SECURITY;
ALTER TABLE legal_entities FORCE ROW LEVEL SECURITY;
CREATE POLICY legal_entities_isolation ON legal_entities
    USING (tenant_id = pharmaos_current_tenant_id() OR pharmaos_is_privileged())
    WITH CHECK (tenant_id = pharmaos_current_tenant_id() OR pharmaos_is_privileged());

ALTER TABLE auth_sessions ENABLE ROW LEVEL SECURITY;
ALTER TABLE auth_sessions FORCE ROW LEVEL SECURITY;
CREATE POLICY auth_sessions_isolation ON auth_sessions
    USING (tenant_id = pharmaos_current_tenant_id() OR pharmaos_is_privileged())
    WITH CHECK (tenant_id = pharmaos_current_tenant_id() OR pharmaos_is_privileged());

ALTER TABLE audit_logs ENABLE ROW LEVEL SECURITY;
ALTER TABLE audit_logs FORCE ROW LEVEL SECURITY;
CREATE POLICY audit_logs_isolation ON audit_logs
    USING (tenant_id = pharmaos_current_tenant_id() OR pharmaos_is_privileged())
    WITH CHECK (tenant_id = pharmaos_current_tenant_id() OR pharmaos_is_privileged());

INSERT INTO permissions (code, module, description) VALUES
 ('tenant.view','tenants','View tenant records'), ('tenant.manage','tenants','Manage tenant records'),
 ('user.view','users','View users'), ('user.manage','users','Create, update and deactivate users'),
 ('role.view','roles','View roles and permissions'), ('role.manage','roles','Configure roles and permissions'),
 ('doctor.view','doctors','View healthcare professionals'), ('doctor.create','doctors','Create healthcare professionals'), ('doctor.update','doctors','Update healthcare professionals'),
 ('chemist.view','chemists','View chemists'), ('chemist.manage','chemists','Manage chemists'),
 ('stockist.view','stockists','View stockists'), ('stockist.manage','stockists','Manage stockists'),
 ('product.view','products','View products'), ('product.manage','products','Manage products'),
 ('tourplan.view','tourplans','View tour plans'), ('tourplan.create','tourplans','Create tour plans'), ('tourplan.approve','tourplans','Approve tour plans'),
 ('dcr.view','dcr','View daily call reports'), ('dcr.create','dcr','Create daily call reports'), ('dcr.review','dcr','Review daily call reports'),
 ('visit.view','visits','View field visits'), ('visit.create','visits','Record field visits'),
 ('target.view','targets','View targets'), ('target.manage','targets','Manage targets'),
 ('distributor.view','distributors','View distributor relationships'), ('distributor.manage','distributors','Manage distributor relationships'),
 ('order.view','orders','View orders'), ('order.manage','orders','Create and approve orders'),
 ('secondary_sales.import','secondary_sales','Import secondary sales data'), ('secondary_sales.view','secondary_sales','View secondary sales'),
 ('reports.view','reports','View dashboards and reports'), ('audit.view','audit','View audit logs'),
 ('integration.manage','integrations','Configure ERP integrations'),
 ('employee.view','employees','View employees'), ('employee.manage','employees','Manage employees');

INSERT INTO roles (code, name, description, is_system, is_privileged, data_scope, tenant_id) VALUES
 ('SUPER_ADMIN','Super Administrator','Platform-wide administrator', true, true, 'TENANT', NULL),
 ('COMPANY_ADMIN','Company Administrator','Pharmaceutical company administrator', true, false, 'TENANT', NULL),
 ('DIVISION_ADMIN','Division Administrator','Division-level administrator', true, false, 'DIVISION', NULL),
 ('NSM','National Sales Manager','National sales leadership', true, false, 'TENANT', NULL),
 ('RSM','Regional Sales Manager','Regional sales leadership', true, false, 'REGION', NULL),
 ('ASM','Area Sales Manager','Area sales management', true, false, 'TERRITORY', NULL),
 ('TBM','Territory Business Manager','Territory management', true, false, 'TERRITORY', NULL),
 ('MR','Medical Representative','Field force user', true, false, 'SELF', NULL),
 ('DISTRIBUTOR_ADMIN','Distributor Administrator','Distributor organization administrator', true, false, 'TENANT', NULL),
 ('DISTRIBUTOR_MANAGER','Distributor Manager','Distributor branch/warehouse manager', true, false, 'TERRITORY', NULL),
 ('DISTRIBUTOR_SALESMAN','Distributor Salesman','Distributor field salesman', true, false, 'SELF', NULL),
 ('COMPLIANCE_OFFICER','Compliance Officer','Compliance and audit reviewer', true, false, 'TENANT', NULL),
 ('REPORT_VIEWER','Report Viewer','Read-only reporting user', true, false, 'TENANT', NULL);

INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r CROSS JOIN permissions p WHERE r.code = 'SUPER_ADMIN';

INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r CROSS JOIN permissions p
WHERE r.code = 'COMPANY_ADMIN' AND p.code NOT IN ('audit.view');

INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r JOIN permissions p ON p.code IN (
 'user.view','employee.view','employee.manage','doctor.view','chemist.view','product.view',
 'tourplan.view','tourplan.create','tourplan.approve','dcr.view','dcr.review','visit.view',
 'target.view','target.manage','distributor.view','secondary_sales.view','reports.view')
WHERE r.code IN ('NSM','RSM','ASM','DIVISION_ADMIN');

INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r JOIN permissions p ON p.code IN (
 'user.view','employee.view','doctor.view','chemist.view','product.view','tourplan.view',
 'tourplan.create','dcr.view','dcr.review','visit.view','target.view','reports.view')
WHERE r.code = 'TBM';

INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r JOIN permissions p ON p.code IN (
 'doctor.view','doctor.create','doctor.update','chemist.view','chemist.manage','product.view',
 'tourplan.view','tourplan.create','dcr.view','dcr.create','visit.view','visit.create',
 'target.view','reports.view')
WHERE r.code = 'MR';

INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r JOIN permissions p ON p.code IN (
 'distributor.view','distributor.manage','order.view','order.manage','product.view',
 'secondary_sales.view','secondary_sales.import','reports.view','user.view','user.manage')
WHERE r.code IN ('DISTRIBUTOR_ADMIN','DISTRIBUTOR_MANAGER');

INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r JOIN permissions p ON p.code IN (
 'order.view','order.manage','product.view','reports.view')
WHERE r.code = 'DISTRIBUTOR_SALESMAN';

INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r JOIN permissions p ON p.code IN ('audit.view','reports.view','user.view','doctor.view')
WHERE r.code = 'COMPLIANCE_OFFICER';

INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r JOIN permissions p ON p.code = 'reports.view'
WHERE r.code = 'REPORT_VIEWER';

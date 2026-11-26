CREATE TABLE rpt_report_definitions (
    id              uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       uuid NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    code            text NOT NULL,
    name            text NOT NULL,
    description     text,
    report_type     text NOT NULL CHECK (report_type IN ('FIELD_ACTIVITY','DISTRIBUTION_SUMMARY','AUDIT_TRAIL')),
    config          jsonb NOT NULL DEFAULT '{}'::jsonb,
    created_by      uuid REFERENCES users(id) ON DELETE SET NULL,
    created_at      timestamptz NOT NULL DEFAULT now(),
    updated_at      timestamptz NOT NULL DEFAULT now(),
    UNIQUE (tenant_id, code)
);

CREATE TABLE rpt_report_snapshots (
    id              uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       uuid NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    definition_id   uuid REFERENCES rpt_report_definitions(id) ON DELETE SET NULL,
    period_from     date,
    period_to       date,
    payload         jsonb NOT NULL,
    created_by      uuid REFERENCES users(id) ON DELETE SET NULL,
    created_at      timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE int_integration_connections (
    id              uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       uuid NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    name            text NOT NULL,
    kind            text NOT NULL CHECK (kind IN ('WEBHOOK','ERP','ANALYTICS')),
    endpoint_url    text,
    secret_ref      text,
    headers         jsonb NOT NULL DEFAULT '{}'::jsonb,
    status          text NOT NULL DEFAULT 'ACTIVE' CHECK (status IN ('ACTIVE','DISABLED')),
    created_by      uuid REFERENCES users(id) ON DELETE SET NULL,
    created_at      timestamptz NOT NULL DEFAULT now(),
    updated_at      timestamptz NOT NULL DEFAULT now(),
    UNIQUE (tenant_id, name)
);

CREATE TABLE int_webhook_deliveries (
    id              uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       uuid NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    connection_id   uuid NOT NULL REFERENCES int_integration_connections(id) ON DELETE CASCADE,
    event_type      text NOT NULL,
    event_id        text NOT NULL,
    payload         jsonb NOT NULL,
    status          text NOT NULL DEFAULT 'PENDING' CHECK (status IN ('PENDING','IN_FLIGHT','DELIVERED','FAILED','DEAD')),
    attempts        int NOT NULL DEFAULT 0,
    next_attempt_at timestamptz NOT NULL DEFAULT now(),
    last_error      text,
    response_status int,
    delivered_at    timestamptz,
    created_at      timestamptz NOT NULL DEFAULT now(),
    updated_at      timestamptz NOT NULL DEFAULT now(),
    UNIQUE (tenant_id, connection_id, event_type, event_id)
);

CREATE INDEX idx_rpt_snapshots_tenant_created ON rpt_report_snapshots (tenant_id, created_at DESC);
CREATE INDEX idx_int_connections_tenant_status ON int_integration_connections (tenant_id, status);
CREATE INDEX idx_int_deliveries_due ON int_webhook_deliveries (status, next_attempt_at) WHERE status IN ('PENDING','FAILED');

CREATE TRIGGER trg_rpt_report_definitions_updated_at
    BEFORE UPDATE ON rpt_report_definitions
    FOR EACH ROW EXECUTE FUNCTION pharmaos_set_updated_at();
CREATE TRIGGER trg_int_integration_connections_updated_at
    BEFORE UPDATE ON int_integration_connections
    FOR EACH ROW EXECUTE FUNCTION pharmaos_set_updated_at();
CREATE TRIGGER trg_int_webhook_deliveries_updated_at
    BEFORE UPDATE ON int_webhook_deliveries
    FOR EACH ROW EXECUTE FUNCTION pharmaos_set_updated_at();

ALTER TABLE rpt_report_definitions ENABLE ROW LEVEL SECURITY;
ALTER TABLE rpt_report_definitions FORCE ROW LEVEL SECURITY;
CREATE POLICY rpt_report_definitions_isolation ON rpt_report_definitions
    USING (tenant_id = pharmaos_current_tenant_id() OR pharmaos_is_privileged())
    WITH CHECK (tenant_id = pharmaos_current_tenant_id() OR pharmaos_is_privileged());

ALTER TABLE rpt_report_snapshots ENABLE ROW LEVEL SECURITY;
ALTER TABLE rpt_report_snapshots FORCE ROW LEVEL SECURITY;
CREATE POLICY rpt_report_snapshots_isolation ON rpt_report_snapshots
    USING (tenant_id = pharmaos_current_tenant_id() OR pharmaos_is_privileged())
    WITH CHECK (tenant_id = pharmaos_current_tenant_id() OR pharmaos_is_privileged());

ALTER TABLE int_integration_connections ENABLE ROW LEVEL SECURITY;
ALTER TABLE int_integration_connections FORCE ROW LEVEL SECURITY;
CREATE POLICY int_integration_connections_isolation ON int_integration_connections
    USING (tenant_id = pharmaos_current_tenant_id() OR pharmaos_is_privileged())
    WITH CHECK (tenant_id = pharmaos_current_tenant_id() OR pharmaos_is_privileged());

ALTER TABLE int_webhook_deliveries ENABLE ROW LEVEL SECURITY;
ALTER TABLE int_webhook_deliveries FORCE ROW LEVEL SECURITY;
CREATE POLICY int_webhook_deliveries_isolation ON int_webhook_deliveries
    USING (tenant_id = pharmaos_current_tenant_id() OR pharmaos_is_privileged())
    WITH CHECK (tenant_id = pharmaos_current_tenant_id() OR pharmaos_is_privileged());

CREATE OR REPLACE FUNCTION pharmaos_reject_audit_mutation()
RETURNS trigger AS $$
BEGIN
    RAISE EXCEPTION 'audit_logs is append-only' USING ERRCODE = '42509';
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_audit_logs_immutable ON audit_logs;
CREATE TRIGGER trg_audit_logs_immutable
    BEFORE UPDATE OR DELETE ON audit_logs
    FOR EACH ROW EXECUTE FUNCTION pharmaos_reject_audit_mutation();

INSERT INTO permissions (code, module, description) VALUES
    ('report.manage','reports','Create saved report definitions and snapshots'),
    ('integration.view','integrations','View integration connections and delivery history'),
    ('webhook.retry','integrations','Retry failed webhook deliveries')
ON CONFLICT (code) DO NOTHING;

INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r JOIN permissions p ON p.code IN (
    'report.manage','integration.view','webhook.retry'
) WHERE r.code IN ('SUPER_ADMIN','COMPANY_ADMIN','DISTRIBUTOR_ADMIN')
ON CONFLICT DO NOTHING;

INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r JOIN permissions p ON p.code IN ('reports.view','integration.view')
WHERE r.code IN ('NSM','RSM','ASM','DIVISION_ADMIN','COMPLIANCE_OFFICER','REPORT_VIEWER')
ON CONFLICT DO NOTHING;

GRANT SELECT, INSERT, UPDATE, DELETE ON rpt_report_definitions, rpt_report_snapshots,
    int_integration_connections, int_webhook_deliveries TO pharmaos_app;

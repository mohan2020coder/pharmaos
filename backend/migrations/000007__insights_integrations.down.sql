DROP TRIGGER IF EXISTS trg_audit_logs_immutable ON audit_logs;
DROP FUNCTION IF EXISTS pharmaos_reject_audit_mutation();

DROP TABLE IF EXISTS int_webhook_deliveries;
DROP TABLE IF EXISTS int_integration_connections;
DROP TABLE IF EXISTS rpt_report_snapshots;
DROP TABLE IF EXISTS rpt_report_definitions;

DELETE FROM role_permissions WHERE permission_id IN (
    SELECT id FROM permissions WHERE code IN ('report.manage','integration.view','webhook.retry')
);
DELETE FROM permissions WHERE code IN ('report.manage','integration.view','webhook.retry');

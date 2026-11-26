CREATE EXTENSION IF NOT EXISTS pgcrypto;
CREATE EXTENSION IF NOT EXISTS citext;

CREATE OR REPLACE FUNCTION pharmaos_set_updated_at() RETURNS trigger AS $$
BEGIN
  NEW.updated_at = now();
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE FUNCTION pharmaos_current_tenant_id() RETURNS uuid AS $$
  SELECT NULLIF(current_setting('app.current_tenant_id', true), '')::uuid;
$$ LANGUAGE sql STABLE;

CREATE OR REPLACE FUNCTION pharmaos_is_privileged() RETURNS boolean AS $$
  SELECT COALESCE(NULLIF(current_setting('app.is_privileged', true), '')::boolean, false);
$$ LANGUAGE sql STABLE;

CREATE OR REPLACE FUNCTION pharmaos_touch_updated_at(table_name text) RETURNS void AS $$
BEGIN
  EXECUTE format('CREATE TRIGGER trg_%1$s_updated_at BEFORE UPDATE ON %1$s
                  FOR EACH ROW EXECUTE FUNCTION pharmaos_set_updated_at()', table_name);
END;
$$ LANGUAGE plpgsql;

-- Application queries run as a non-superuser role so RLS cannot be bypassed.
-- The migration/bootstrap role keeps DDL privileges and can SET LOCAL ROLE.
DO $$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'pharmaos_app') THEN
    CREATE ROLE pharmaos_app NOLOGIN NOSUPERUSER NOBYPASSRLS;
  END IF;
END $$;
GRANT USAGE ON SCHEMA public TO pharmaos_app;
GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA public TO pharmaos_app;
GRANT USAGE, SELECT ON ALL SEQUENCES IN SCHEMA public TO pharmaos_app;
ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT SELECT, INSERT, UPDATE, DELETE ON TABLES TO pharmaos_app;
ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT USAGE, SELECT ON SEQUENCES TO pharmaos_app;

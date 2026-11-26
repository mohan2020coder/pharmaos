CREATE TABLE ops_rate_limit_counters (
    counter_key  text PRIMARY KEY,
    window_start timestamptz NOT NULL DEFAULT now(),
    hits         integer NOT NULL DEFAULT 0,
    updated_at   timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX idx_ops_rate_limit_cleanup ON ops_rate_limit_counters (window_start);

CREATE OR REPLACE FUNCTION pharmaos_rate_limit(p_key text, p_limit integer, p_window_seconds integer)
RETURNS boolean
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = public
AS $$
DECLARE
    v_allowed boolean;
    v_cutoff timestamptz := now() - make_interval(secs => p_window_seconds);
BEGIN
    IF p_limit < 1 OR p_window_seconds < 1 THEN
        RETURN false;
    END IF;

    INSERT INTO ops_rate_limit_counters AS counter (counter_key, window_start, hits, updated_at)
    VALUES (p_key, now(), 1, now())
    ON CONFLICT (counter_key) DO UPDATE
    SET window_start = CASE WHEN counter.window_start <= v_cutoff THEN now() ELSE counter.window_start END,
        hits = CASE WHEN counter.window_start <= v_cutoff THEN 1 ELSE counter.hits + 1 END,
        updated_at = now()
    RETURNING hits <= p_limit INTO v_allowed;

    IF random() < 0.01 THEN
        DELETE FROM ops_rate_limit_counters WHERE window_start < now() - interval '1 hour';
    END IF;

    RETURN COALESCE(v_allowed, false);
END;
$$;

REVOKE ALL ON FUNCTION pharmaos_rate_limit(text, integer, integer) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION pharmaos_rate_limit(text, integer, integer) TO pharmaos_app;

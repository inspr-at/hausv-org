-- Each row is an immutable, complete calculation with its input snapshot.
CREATE TABLE annual_statement_runs (
    tenant_id TEXT NOT NULL REFERENCES tenant(tenant_id) ON UPDATE RESTRICT ON DELETE CASCADE,
    tenant_slug TEXT NOT NULL,
    id TEXT NOT NULL,
    period_year INTEGER NOT NULL,
    revision INTEGER NOT NULL,
    data TEXT NOT NULL,
    PRIMARY KEY (tenant_id, id),
    UNIQUE (tenant_id, period_year, revision)
);

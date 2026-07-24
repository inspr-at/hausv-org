-- Abstimmungen per tenant (HAUSV-168/170). Replaces /data/votes.json. The whole
-- ballot — including its votes and reminder timestamps — is one JSON document
-- keyed by (tenant, id), so casting a vote is a single-row read-modify-write
-- inside a transaction.
CREATE TABLE IF NOT EXISTS ballots (
    tenant_slug TEXT NOT NULL,
    id          TEXT NOT NULL,
    data        TEXT NOT NULL,
    PRIMARY KEY (tenant_slug, id)
);

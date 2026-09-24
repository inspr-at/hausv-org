-- HAUSV-778: immutable calculation snapshots; delivery bookkeeping is separate.
CREATE TABLE valorisation_runs (
 tenant_id TEXT NOT NULL REFERENCES tenant(tenant_id) ON DELETE CASCADE,
 tenant_slug TEXT NOT NULL, id TEXT NOT NULL, org_key TEXT NOT NULL,
 effective_on TEXT NOT NULL, revision INTEGER NOT NULL,
 status TEXT NOT NULL CHECK(status IN ('draft','approved','sent','cancelled')),
 inputs_sha256 TEXT NOT NULL, index_snapshot TEXT NOT NULL, data TEXT NOT NULL,
 created_by TEXT NOT NULL, created_at TEXT NOT NULL,
 approved_by TEXT NOT NULL DEFAULT '', approved_at TEXT NOT NULL DEFAULT '',
 cancel_reason TEXT NOT NULL DEFAULT '', cancelled_by TEXT NOT NULL DEFAULT '',
 PRIMARY KEY(tenant_id,id), UNIQUE(tenant_id,effective_on,revision)
);
CREATE TABLE valorisation_items (
 tenant_id TEXT NOT NULL REFERENCES tenant(tenant_id) ON DELETE CASCADE,
 tenant_slug TEXT NOT NULL, id TEXT NOT NULL, run_id TEXT NOT NULL,
 lease_id TEXT NOT NULL, clause_id TEXT NOT NULL, data TEXT NOT NULL,
 letter_document_id TEXT NOT NULL DEFAULT '', letter_sha256 TEXT NOT NULL DEFAULT '',
 collectable_from TEXT NOT NULL DEFAULT '',
 PRIMARY KEY(tenant_id,id), UNIQUE(tenant_id,run_id,lease_id),
 FOREIGN KEY(tenant_id,run_id) REFERENCES valorisation_runs(tenant_id,id) ON DELETE CASCADE
);
CREATE TABLE valorisation_deliveries (
 tenant_id TEXT NOT NULL REFERENCES tenant(tenant_id) ON DELETE CASCADE,
 tenant_slug TEXT NOT NULL, id TEXT NOT NULL, run_id TEXT NOT NULL,
 revision INTEGER NOT NULL, party_id TEXT NOT NULL, unit_id TEXT NOT NULL,
 document_id TEXT NOT NULL, sha256 TEXT NOT NULL, recipient TEXT NOT NULL,
 sent_at TEXT NOT NULL, status TEXT NOT NULL CHECK(status IN ('pending','sent','failed')),
 error TEXT NOT NULL, actor TEXT NOT NULL, attempt INTEGER NOT NULL,
 PRIMARY KEY(tenant_id,id), UNIQUE(tenant_id,run_id,revision,party_id,unit_id,attempt),
 FOREIGN KEY(tenant_id,run_id) REFERENCES valorisation_runs(tenant_id,id)
);
CREATE UNIQUE INDEX valorisation_one_sent ON valorisation_deliveries(tenant_id,run_id,unit_id,recipient) WHERE status='sent';
CREATE TABLE valorisation_events (
 tenant_id TEXT NOT NULL REFERENCES tenant(tenant_id) ON DELETE CASCADE,
 tenant_slug TEXT NOT NULL, id TEXT NOT NULL, run_id TEXT NOT NULL,
 action TEXT NOT NULL, actor TEXT NOT NULL, created_at TEXT NOT NULL, data TEXT NOT NULL,
 PRIMARY KEY(tenant_id,id), FOREIGN KEY(tenant_id,run_id) REFERENCES valorisation_runs(tenant_id,id)
);

CREATE TRIGGER valorisation_run_frozen BEFORE UPDATE ON valorisation_runs
WHEN OLD.status <> 'draft' AND (NEW.id <> OLD.id OR NEW.tenant_id <> OLD.tenant_id OR NEW.tenant_slug <> OLD.tenant_slug OR NEW.data <> OLD.data OR NEW.inputs_sha256 <> OLD.inputs_sha256 OR NEW.index_snapshot <> OLD.index_snapshot OR NEW.effective_on <> OLD.effective_on OR NEW.org_key <> OLD.org_key OR NEW.revision <> OLD.revision OR NEW.created_by <> OLD.created_by OR NEW.created_at <> OLD.created_at OR NEW.approved_by <> OLD.approved_by OR NEW.approved_at <> OLD.approved_at OR NEW.status = 'draft')
BEGIN SELECT RAISE(ABORT, 'approved valorisation run is immutable'); END;
CREATE TRIGGER valorisation_item_frozen BEFORE UPDATE ON valorisation_items
WHEN (SELECT status FROM valorisation_runs WHERE tenant_id=OLD.tenant_id AND id=OLD.run_id) <> 'draft'
AND (NEW.tenant_id <> OLD.tenant_id OR NEW.tenant_slug <> OLD.tenant_slug OR NEW.data <> OLD.data OR NEW.run_id <> OLD.run_id OR NEW.lease_id <> OLD.lease_id OR NEW.clause_id <> OLD.clause_id OR NEW.id <> OLD.id)
BEGIN SELECT RAISE(ABORT, 'approved valorisation item is immutable'); END;
CREATE TRIGGER valorisation_run_no_delete BEFORE DELETE ON valorisation_runs WHEN OLD.status <> 'draft'
BEGIN SELECT RAISE(ABORT, 'approved valorisation run is immutable'); END;
CREATE TRIGGER valorisation_item_no_delete BEFORE DELETE ON valorisation_items
WHEN (SELECT status FROM valorisation_runs WHERE tenant_id=OLD.tenant_id AND id=OLD.run_id) <> 'draft'
BEGIN SELECT RAISE(ABORT, 'approved valorisation item is immutable'); END;

CREATE TRIGGER valorisation_item_no_insert BEFORE INSERT ON valorisation_items
WHEN (SELECT status FROM valorisation_runs WHERE tenant_id=NEW.tenant_id AND id=NEW.run_id) <> 'draft'
BEGIN SELECT RAISE(ABORT, 'approved valorisation item is immutable'); END;

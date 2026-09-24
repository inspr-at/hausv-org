-- HAUSV-778: immutable calculation snapshots; delivery bookkeeping is separate.
CREATE TABLE valorisation_runs (
 tenant_id varchar(26) NOT NULL REFERENCES tenant(tenant_id) ON DELETE CASCADE,
 tenant_slug TEXT NOT NULL, id TEXT NOT NULL, org_key TEXT NOT NULL,
 effective_on TEXT NOT NULL, revision bigint NOT NULL,
 status TEXT NOT NULL CHECK(status IN ('draft','approved','sent','cancelled')),
 inputs_sha256 TEXT NOT NULL, index_snapshot TEXT NOT NULL, data TEXT NOT NULL,
 created_by TEXT NOT NULL, created_at TEXT NOT NULL,
 approved_by TEXT NOT NULL DEFAULT '', approved_at TEXT NOT NULL DEFAULT '',
 cancel_reason TEXT NOT NULL DEFAULT '', cancelled_by TEXT NOT NULL DEFAULT '',
 PRIMARY KEY(tenant_id,id), UNIQUE(tenant_id,effective_on,revision)
);
CREATE TABLE valorisation_items (
 tenant_id varchar(26) NOT NULL REFERENCES tenant(tenant_id) ON DELETE CASCADE,
 tenant_slug TEXT NOT NULL, id TEXT NOT NULL, run_id TEXT NOT NULL,
 lease_id TEXT NOT NULL, clause_id TEXT NOT NULL, data TEXT NOT NULL,
 letter_document_id TEXT NOT NULL DEFAULT '', letter_sha256 TEXT NOT NULL DEFAULT '',
 collectable_from TEXT NOT NULL DEFAULT '',
 PRIMARY KEY(tenant_id,id), UNIQUE(tenant_id,run_id,lease_id),
 FOREIGN KEY(tenant_id,run_id) REFERENCES valorisation_runs(tenant_id,id) ON DELETE CASCADE
);
CREATE TABLE valorisation_deliveries (
 tenant_id varchar(26) NOT NULL REFERENCES tenant(tenant_id) ON DELETE CASCADE,
 tenant_slug TEXT NOT NULL, id TEXT NOT NULL, run_id TEXT NOT NULL,
 revision bigint NOT NULL, party_id TEXT NOT NULL, unit_id TEXT NOT NULL,
 document_id TEXT NOT NULL, sha256 TEXT NOT NULL, recipient TEXT NOT NULL,
 sent_at TEXT NOT NULL, status TEXT NOT NULL CHECK(status IN ('pending','sent','failed')),
 error TEXT NOT NULL, actor TEXT NOT NULL, attempt bigint NOT NULL,
 PRIMARY KEY(tenant_id,id), UNIQUE(tenant_id,run_id,revision,party_id,unit_id,attempt),
 FOREIGN KEY(tenant_id,run_id) REFERENCES valorisation_runs(tenant_id,id)
);
CREATE UNIQUE INDEX valorisation_one_sent ON valorisation_deliveries(tenant_id,run_id,unit_id,recipient) WHERE status='sent';
CREATE TABLE valorisation_events (
 tenant_id varchar(26) NOT NULL REFERENCES tenant(tenant_id) ON DELETE CASCADE,
 tenant_slug TEXT NOT NULL, id TEXT NOT NULL, run_id TEXT NOT NULL,
 action TEXT NOT NULL, actor TEXT NOT NULL, created_at TEXT NOT NULL, data TEXT NOT NULL,
 PRIMARY KEY(tenant_id,id), FOREIGN KEY(tenant_id,run_id) REFERENCES valorisation_runs(tenant_id,id)
);

DO $$
DECLARE table_name text;
BEGIN
 FOREACH table_name IN ARRAY ARRAY['valorisation_runs','valorisation_items','valorisation_deliveries','valorisation_events'] LOOP
  EXECUTE format('ALTER TABLE %I ENABLE ROW LEVEL SECURITY', table_name);
  EXECUTE format('ALTER TABLE %I FORCE ROW LEVEL SECURITY', table_name);
  EXECUTE format($policy$CREATE POLICY tenant_isolation ON %I USING (coalesce(current_setting('hausv.cross_tenant',true),'')='on' OR tenant_id=coalesce(current_setting('hausv.tenant_id',true),'')) WITH CHECK (coalesce(current_setting('hausv.cross_tenant',true),'')='on' OR tenant_id=coalesce(current_setting('hausv.tenant_id',true),''))$policy$,table_name);
  EXECUTE format('CREATE TRIGGER tenant_id_immutable BEFORE UPDATE OF tenant_id ON %I FOR EACH ROW EXECUTE FUNCTION hausv_reject_tenant_id_change()',table_name);
 END LOOP;
END;
$$;
CREATE FUNCTION hausv_valorisation_frozen() RETURNS trigger LANGUAGE plpgsql AS $$
DECLARE frozen boolean;
BEGIN
 IF TG_TABLE_NAME='valorisation_runs' THEN
  frozen := OLD.status <> 'draft';
  IF frozen AND TG_OP='UPDATE' THEN
   IF (to_jsonb(NEW) - ARRAY['status','cancel_reason','cancelled_by']) = (to_jsonb(OLD) - ARRAY['status','cancel_reason','cancelled_by']) AND NEW.status <> 'draft' THEN RETURN NEW; END IF;
  END IF;
 ELSE
  SELECT status <> 'draft' INTO frozen FROM valorisation_runs WHERE tenant_id=OLD.tenant_id AND id=OLD.run_id;
  IF frozen AND TG_OP='UPDATE' THEN
   IF (to_jsonb(NEW) - ARRAY['letter_document_id','letter_sha256','collectable_from']) = (to_jsonb(OLD) - ARRAY['letter_document_id','letter_sha256','collectable_from']) THEN RETURN NEW; END IF;
  END IF;
 END IF;
 IF frozen THEN RAISE EXCEPTION 'approved valorisation calculation is immutable' USING ERRCODE='23514'; END IF;
 IF TG_OP='DELETE' THEN RETURN OLD; END IF;
 RETURN NEW;
END;
$$;
CREATE TRIGGER valorisation_run_frozen BEFORE UPDATE OR DELETE ON valorisation_runs FOR EACH ROW EXECUTE FUNCTION hausv_valorisation_frozen();
CREATE TRIGGER valorisation_item_frozen BEFORE UPDATE OR DELETE ON valorisation_items FOR EACH ROW EXECUTE FUNCTION hausv_valorisation_frozen();

CREATE FUNCTION hausv_valorisation_item_no_insert() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
 IF EXISTS(SELECT 1 FROM valorisation_runs WHERE tenant_id=NEW.tenant_id AND id=NEW.run_id AND status <> 'draft') THEN
  RAISE EXCEPTION 'approved valorisation item is immutable' USING ERRCODE='23514';
 END IF;
 RETURN NEW;
END;
$$;
CREATE TRIGGER valorisation_item_no_insert BEFORE INSERT ON valorisation_items FOR EACH ROW EXECUTE FUNCTION hausv_valorisation_item_no_insert();

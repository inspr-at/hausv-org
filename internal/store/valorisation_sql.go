package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/inspr-at/hausv-org/internal/indexation"
)

var ErrValorisationConflict = errors.New("Lauf geändert, bereits freigegeben oder Eingaben veraltet; bitte neu berechnen")
var ErrValorisationDenied = errors.New("Keine Berechtigung zur Wertsicherung")

type ValorisationActor struct {
	Email           string
	Manage, Approve bool
}
type ValorisationRenderer func(ValorisationRun, ValorisationItem, time.Time) ([]byte, error)

type ValorisationRepository struct {
	db        *TenantDB
	tenant    TenantRef
	documents *SQLDocumentStore
}

func BindValorisationRepository(db *TenantDB, documents *SQLDocumentStore, tenant TenantRef) (*ValorisationRepository, bool) {
	tenant, ok := validTenantRef(tenant)
	if db == nil || !ok {
		return nil, false
	}
	return &ValorisationRepository{db, tenant, documents}, true
}
func (r *ValorisationRepository) Preview(input ValorisationInput, now time.Time) (ValorisationRun, error) {
	tx, err := r.db.For(r.tenant).Begin()
	if err != nil {
		return ValorisationRun{}, err
	}
	defer tx.Rollback()
	return r.previewTx(tx, input, now)
}
func (r *ValorisationRepository) previewTx(tx *sql.Tx, input ValorisationInput, now time.Time) (ValorisationRun, error) {
	leases, err := loadLeasesTx(tx, r.tenant, "")
	if err != nil {
		return ValorisationRun{}, err
	}
	input.Leases = leases
	units, err := readStoredUnits(tx, r.tenant)
	if err != nil {
		return ValorisationRun{}, err
	}
	input.UnitLabels = map[string]string{}
	for _, unit := range units {
		input.UnitLabels[unit.id] = unit.unit.Label
	}
	input.Prior = map[string]ValorisationItem{}
	for _, lease := range leases {
		for _, clause := range lease.Clauses {
			if clause.State != nil && clause.State.LastRunItemID != "" {
				var raw string
				if err := tx.QueryRow(`SELECT data FROM valorisation_items WHERE tenant_id=$1 AND id=$2`, r.tenant.ID, clause.State.LastRunItemID).Scan(&raw); err != nil {
					return ValorisationRun{}, err
				}
				var prior ValorisationItem
				if err := json.Unmarshal([]byte(raw), &prior); err != nil {
					return ValorisationRun{}, err
				}
				input.Prior[clause.ID] = prior
			}
		}
	}
	snapshot, err := runtimeIndexSnapshot(tx)
	if err != nil {
		return ValorisationRun{}, err
	}
	run, err := PreviewValorisation(input, snapshot, now)
	run.TenantSlug = r.tenant.Slug
	if err == nil {
		run.IndexDisputed, err = indexRunDisputed(tx, run)
	}
	return run, err
}
func (r *ValorisationRepository) Create(input ValorisationInput, org string, actor ValorisationActor, now time.Time) (ValorisationRun, error) {
	if !actor.Manage || strings.TrimSpace(actor.Email) == "" {
		return ValorisationRun{}, ErrValorisationDenied
	}
	tx, err := r.db.For(r.tenant).Begin()
	if err != nil {
		return ValorisationRun{}, err
	}
	defer tx.Rollback()
	run, err := r.createTx(tx, input, org, actor, now)
	if err != nil {
		return run, err
	}
	return run, tx.Commit()
}

// SeedValorisationDraft creates the demo's reviewable draft in the lease seed transaction.
func SeedValorisationDraft(tx *sql.Tx, tenant TenantRef, input ValorisationInput, org string, now time.Time) error {
	var n int
	if err := tx.QueryRow(`SELECT COUNT(*) FROM valorisation_runs WHERE tenant_id=$1`, tenant.ID).Scan(&n); err != nil {
		return err
	}
	if n > 0 {
		return nil
	}
	r := &ValorisationRepository{tenant: tenant}
	_, err := r.createTx(tx, input, org, ValorisationActor{Email: "vera.verwalter@musterstadt.example", Manage: true}, now)
	return err
}
func (r *ValorisationRepository) createTx(tx *sql.Tx, input ValorisationInput, org string, actor ValorisationActor, now time.Time) (ValorisationRun, error) {
	run, err := r.previewTx(tx, input, now)
	if err != nil {
		return run, err
	}
	run.ID, err = newLeaseID()
	if err != nil {
		return run, err
	}
	run.OrgKey, run.CreatedBy, run.CreatedAt = org, strings.ToLower(actor.Email), now.UTC()
	if err = tx.QueryRow(`SELECT COALESCE(MAX(revision),0)+1 FROM valorisation_runs WHERE tenant_id=$1 AND effective_on=$2`, r.tenant.ID, run.EffectiveOn).Scan(&run.Revision); err != nil {
		return run, err
	}
	for i := range run.Items {
		run.Items[i].ID, err = newLeaseID()
		if err != nil {
			return run, err
		}
	}
	raw, _ := json.Marshal(run.Input)
	snapshot, _ := json.Marshal(struct {
		Version, SHA256 string
		Values          []ValorisationIndex
	}{run.IndexVersion, run.IndexSHA256, run.IndexSnapshot})
	_, err = tx.Exec(`INSERT INTO valorisation_runs(tenant_id,tenant_slug,id,org_key,effective_on,revision,status,inputs_sha256,index_snapshot,data,created_by,created_at) VALUES($1,$2,$3,$4,$5,$6,'draft',$7,$8,$9,$10,$11)`, r.tenant.ID, r.tenant.Slug, run.ID, org, run.EffectiveOn, run.Revision, run.InputsSHA256, string(snapshot), string(raw), run.CreatedBy, stamp(now))
	if err != nil {
		return run, err
	}
	for _, item := range run.Items {
		raw, _ := json.Marshal(item)
		_, err = tx.Exec(`INSERT INTO valorisation_items(tenant_id,tenant_slug,id,run_id,lease_id,clause_id,data) VALUES($1,$2,$3,$4,$5,$6,$7)`, r.tenant.ID, r.tenant.Slug, item.ID, run.ID, item.LeaseID, item.ClauseID, string(raw))
		if err != nil {
			return run, err
		}
	}
	if err = r.event(tx, run, "valorisation_run.create", actor.Email, now, map[string]string{"inputs_sha256": run.InputsSHA256}); err != nil {
		return run, err
	}
	return run, nil
}
func (r *ValorisationRepository) Get(id string) (ValorisationRun, bool, error) {
	tx, err := r.db.For(r.tenant).Begin()
	if err != nil {
		return ValorisationRun{}, false, err
	}
	defer tx.Rollback()
	run, err := r.getTx(tx, id)
	if errors.Is(err, sql.ErrNoRows) {
		return run, false, nil
	}
	return run, err == nil, err
}
func (r *ValorisationRepository) getTx(tx *sql.Tx, id string) (ValorisationRun, error) {
	var run ValorisationRun
	var raw, snapshot, created, approved string
	err := tx.QueryRow(`SELECT id,org_key,effective_on,revision,status,inputs_sha256,index_snapshot,data,created_by,created_at,approved_by,approved_at,cancel_reason,cancelled_by FROM valorisation_runs WHERE tenant_id=$1 AND id=$2`, r.tenant.ID, id).Scan(&run.ID, &run.OrgKey, &run.EffectiveOn, &run.Revision, &run.Status, &run.InputsSHA256, &snapshot, &raw, &run.CreatedBy, &created, &run.ApprovedBy, &approved, &run.CancelReason, &run.CancelledBy)
	if err != nil {
		return run, err
	}
	run.TenantSlug = r.tenant.Slug
	run.CreatedAt, _ = time.Parse(time.RFC3339Nano, created)
	run.ApprovedAt, _ = time.Parse(time.RFC3339Nano, approved)
	if err = json.Unmarshal([]byte(raw), &run.Input); err != nil {
		return run, err
	}
	var snap struct {
		Version, SHA256 string
		Values          []ValorisationIndex
	}
	if err = json.Unmarshal([]byte(snapshot), &snap); err != nil {
		return run, err
	}
	run.IndexVersion, run.IndexSHA256, run.IndexSnapshot = snap.Version, snap.SHA256, snap.Values
	rows, err := tx.Query(`SELECT data,letter_document_id,letter_sha256,collectable_from FROM valorisation_items WHERE tenant_id=$1 AND run_id=$2 ORDER BY lease_id`, r.tenant.ID, id)
	if err != nil {
		return run, err
	}
	defer rows.Close()
	for rows.Next() {
		var raw, doc, sha, due string
		if err = rows.Scan(&raw, &doc, &sha, &due); err != nil {
			return run, err
		}
		var item ValorisationItem
		if err = json.Unmarshal([]byte(raw), &item); err != nil {
			return run, err
		}
		item.LetterDocumentID, item.LetterSHA256 = doc, sha
		if due != "" {
			item.CollectableFrom = due
		}
		run.Items = append(run.Items, item)
	}
	if err = rows.Err(); err != nil {
		return run, err
	}
	if err = rows.Close(); err != nil {
		return run, err
	}
	run.IndexRevised, err = indexRunRevised(tx, run)
	if err != nil {
		return run, err
	}
	run.IndexDisputed, err = indexRunDisputed(tx, run)
	return run, err
}
func (r *ValorisationRepository) List() ([]ValorisationRun, error) {
	rows, err := r.db.For(r.tenant).Query(`SELECT id FROM valorisation_runs WHERE tenant_id=$1 ORDER BY effective_on DESC,revision DESC`, r.tenant.ID)
	if err != nil {
		return nil, err
	}
	var ids []string
	for rows.Next() {
		var id string
		if err = rows.Scan(&id); err != nil {
			rows.Close()
			return nil, err
		}
		ids = append(ids, id)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return nil, err
	}
	out := []ValorisationRun{}
	for _, id := range ids {
		run, _, err := r.Get(id)
		if err != nil {
			return nil, err
		}
		out = append(out, run)
	}
	return out, nil
}
func (r *ValorisationRepository) lockDraft(tx *sql.Tx, id string) (ValorisationRun, error) {
	result, err := tx.Exec(`UPDATE valorisation_runs SET status=status WHERE tenant_id=$1 AND id=$2 AND status='draft'`, r.tenant.ID, id)
	if err != nil {
		return ValorisationRun{}, err
	}
	n, _ := result.RowsAffected()
	if n != 1 {
		return ValorisationRun{}, ErrValorisationConflict
	}
	return r.getTx(tx, id)
}
func (r *ValorisationRepository) Approve(id string, actor ValorisationActor, settings ValorisationSettings, now time.Time, render ValorisationRenderer) (ValorisationRun, error) {
	if !actor.Manage || !actor.Approve || actor.Email == "" {
		return ValorisationRun{}, ErrValorisationDenied
	}
	if render == nil || r.documents == nil {
		return ValorisationRun{}, fmt.Errorf("Dokumentenarchiv nicht verfügbar")
	}
	tx, err := r.db.For(r.tenant).Begin()
	if err != nil {
		return ValorisationRun{}, err
	}
	defer tx.Rollback()
	if err = lockIndexReference(context.Background(), tx); err != nil {
		return ValorisationRun{}, err
	}
	run, err := r.lockDraft(tx, id)
	if err != nil {
		return run, err
	}
	if settings.Normalized() != run.Input.Settings.Normalized() {
		return run, ErrValorisationConflict
	}
	if settings.FourEyes && strings.EqualFold(run.CreatedBy, actor.Email) {
		return run, fmt.Errorf("Vier-Augen-Regel: eine andere Person muss freigeben")
	}
	// Different draft revisions have different run locks but may prescribe the
	// same leases. Serialize their approvals before checking the frozen graph.
	if _, err = tx.Exec(`UPDATE leases SET status=status WHERE tenant_id=$1`, r.tenant.ID); err != nil {
		return run, err
	}
	leases, err := loadLeasesTx(tx, r.tenant, "")
	if err != nil {
		return run, err
	}
	// Compare each frozen lease, including parties and exact curve state. Adding
	// a lease also invalidates a draft, so approval cannot silently omit it.
	current := map[string]Lease{}
	for _, lease := range leases {
		current[lease.ID] = lease
	}
	if len(current) != len(run.Input.Leases) {
		return run, ErrValorisationConflict
	}
	for _, lease := range run.Input.Leases {
		if !reflect.DeepEqual(current[lease.ID], lease) {
			return run, ErrValorisationConflict
		}
	}
	if run.IndexRevised {
		return run, fmt.Errorf("Indexwerte wurden berichtigt. Bitte einen neuen Lauf berechnen.")
	}
	included := 0
	run.ApprovedBy, run.ApprovedAt = actor.Email, now.UTC()
	for idx, item := range run.Items {
		if item.Excluded || item.Outcome == "unchanged" && item.Group != "exception" {
			continue
		}
		if issue := item.ApprovalException(now); issue != "" {
			return run, fmt.Errorf("Ausnahmen zuerst bearbeiten oder mit Begründung ausschließen: %s", issue)
		}
		item, err = ValorisationLetterTiming(item, now)
		if err != nil {
			return run, err
		}
		item.Exceptions = nil
		item.Group = "ready"
		run.Items[idx] = item
		doc, err := r.archiveTx(tx, run, item, actor, now, render)
		if err != nil {
			return run, err
		}
		item.LetterDocumentID, item.LetterSHA256 = doc.ID, doc.ValorisationArchive.SHA256
		if err = r.updateItemTx(tx, run.ID, item); err != nil {
			return run, err
		}
		component, err := normalizeComponent(item.LeaseID, RentComponent{Kind: ComponentHMZ, NetCents: item.NewCents, VATRateBP: item.VATRateBP, ValidFrom: item.WirksamOn, Origin: "valorisation_item:" + item.ID, CreatedAt: now})
		if err != nil {
			return run, err
		}
		if err = insertComponent(tx, r.tenant, component); err != nil {
			return run, err
		}
		state := ValorisationState{ClauseID: item.ClauseID, ContractValue: euroDecimal(item.ContractCents), ContractBasePeriod: string(item.NextClause.BaseMonth), ContractBaseValue: item.NextClause.BaseValue.String(), CapValue: euroDecimal(item.CapCents), CapAnchorPeriod: string(item.CapAnchor), CapLastYear: mustMonthDate(item.CapAnchor).Year(), LastEffectiveOn: item.WirksamOn, LastRunItemID: item.ID}
		if item.MieWeG {
			state.CapLastYear = mustDate(run.EffectiveOn).Year() - 1
		}
		if err = upsertValorisation(tx, r.tenant, state); err != nil {
			return run, err
		}
		run.Items[idx] = item
		included++
	}
	if included == 0 {
		return run, fmt.Errorf("Keine freigabefähige Anpassung")
	}
	_, err = tx.Exec(`UPDATE valorisation_runs SET status='approved',approved_by=$1,approved_at=$2 WHERE tenant_id=$3 AND id=$4 AND status='draft'`, actor.Email, stamp(now), r.tenant.ID, id)
	if err != nil {
		return run, err
	}
	run.Status = "approved"
	if err = r.event(tx, run, "valorisation_run.approve", actor.Email, now, map[string]string{"inputs_sha256": run.InputsSHA256}); err != nil {
		return run, err
	}
	return run, tx.Commit()
}
func (r *ValorisationRepository) updateItemTx(tx *sql.Tx, runID string, item ValorisationItem) error {
	raw, err := json.Marshal(item)
	if err != nil {
		return err
	}
	_, err = tx.Exec(`UPDATE valorisation_items SET data=$1,letter_document_id=$2,letter_sha256=$3,collectable_from=$4 WHERE tenant_id=$5 AND run_id=$6 AND id=$7`, string(raw), item.LetterDocumentID, item.LetterSHA256, item.CollectableFrom, r.tenant.ID, runID, item.ID)
	return err
}
func (r *ValorisationRepository) archiveTx(tx *sql.Tx, run ValorisationRun, item ValorisationItem, actor ValorisationActor, now time.Time, render ValorisationRenderer) (DocumentRecord, error) {
	final := run
	final.Status = "approved"
	data, err := render(final, item, now)
	if err != nil {
		return DocumentRecord{}, err
	}
	doc := DocumentRecord{TenantSlug: r.tenant.Slug, Title: "Wertsicherung · " + item.Label() + " · " + run.EffectiveOn, UnitID: item.UnitID, UploadedBy: actor.Email, ValorisationArchive: &ValorisationArchiveMetadata{RunID: run.ID, ItemID: item.ID, LetterDate: now.Format(time.DateOnly), CollectableFrom: item.CollectableFrom}}
	doc, err = prepareArchiveRecord(doc, "wertsicherung-"+item.ID+".pdf", "application/pdf", data, now)
	if err != nil {
		return doc, err
	}
	if err = writeArchiveFile(r.documents.fileDir, doc, data); err != nil {
		return doc, err
	}
	raw, _ := json.Marshal(doc)
	_, err = tx.Exec(`INSERT INTO documents(tenant_id,tenant_slug,id,data) VALUES($1,$2,$3,$4) ON CONFLICT(tenant_slug,id) DO NOTHING`, r.tenant.ID, r.tenant.Slug, doc.ID, string(raw))
	if err != nil {
		return doc, err
	}
	err = r.event(tx, run, "letter.archive", actor.Email, now, map[string]string{"item_id": item.ID, "document_id": doc.ID, "sha256": doc.ValorisationArchive.SHA256})
	return doc, err
}
func (r *ValorisationRepository) ItemAction(runID, itemID, action, reason string, amount *int64, actor ValorisationActor, now time.Time) error {
	if !actor.Manage || actor.Email == "" {
		return ErrValorisationDenied
	}
	reason = strings.TrimSpace(reason)
	if reason == "" || len(reason) > 4000 {
		return fmt.Errorf("Begründung erforderlich (höchstens 4000 Zeichen)")
	}
	tx, err := r.db.For(r.tenant).Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	run, err := r.lockDraft(tx, runID)
	if err != nil {
		return err
	}
	var selected *ValorisationItem
	for i := range run.Items {
		if run.Items[i].ID == itemID {
			selected = &run.Items[i]
		}
	}
	if selected == nil {
		return sql.ErrNoRows
	}
	item := *selected
	switch action {
	case "exclude":
		item.Excluded = true
		item.ExcludeReason = reason
	case "override":
		if !actor.Approve {
			return ErrValorisationDenied
		}
		if amount == nil || *amount <= 0 || *amount > item.NewCents || (item.Group != "ready" && !(len(item.Exceptions) == 1 && item.Exceptions[0] == "letter_too_early")) {
			return fmt.Errorf("Manuelle Entscheidung nur für berechnete Beträge innerhalb der zulässigen Obergrenze")
		}
		item.OverrideCents, item.OverrideReason, item.OverrideBy = amount, reason, actor.Email
		item.NewCents = *amount
		item.Outcome = "increase"
		if item.NewCents < item.OldCents {
			item.Outcome = "decrease"
		}
		if item.NewCents == item.OldCents {
			item.Outcome = "unchanged"
		}
		if item.NewCents <= item.OldCents {
			item.RequiresMRGNotice = false
			item.TimingInput.RequiresMRGNotice = false
			item.TimingInput.Mode = indexation.ContractualTiming
			item.TimingInput.ContractualEffectiveOn = mustDate(item.WirksamOn)
			item.Timing, err = indexation.Timing(item.TimingInput)
			if err != nil {
				return err
			}
		}
		item = finishValorisationItem(item)
	default:
		return fmt.Errorf("Unbekannte Aktion")
	}
	if err = r.updateItemTx(tx, run.ID, item); err != nil {
		return err
	}
	if err = r.event(tx, run, "valorisation_item."+action, actor.Email, now, item); err != nil {
		return err
	}
	return tx.Commit()
}
func (r *ValorisationRepository) Cancel(id, reason string, actor ValorisationActor, now time.Time) error {
	if !actor.Manage || actor.Email == "" {
		return ErrValorisationDenied
	}
	reason = strings.TrimSpace(reason)
	if reason == "" || len(reason) > 4000 {
		return fmt.Errorf("Begründung erforderlich")
	}
	tx, err := r.db.For(r.tenant).Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	result, err := tx.Exec(`UPDATE valorisation_runs SET status='cancelled',cancel_reason=$1,cancelled_by=$2 WHERE tenant_id=$3 AND id=$4 AND status<>'cancelled'`, reason, actor.Email, r.tenant.ID, id)
	if err != nil {
		return err
	}
	n, _ := result.RowsAffected()
	if n != 1 {
		return ErrValorisationConflict
	}
	var pending int
	if err = tx.QueryRow(`SELECT COUNT(*) FROM valorisation_deliveries WHERE tenant_id=$1 AND run_id=$2 AND status='pending'`, r.tenant.ID, id).Scan(&pending); err != nil {
		return err
	}
	if pending > 0 {
		return ErrDeliveryInProgress
	}
	run, err := r.getTx(tx, id)
	if err != nil {
		return err
	}
	if err = r.event(tx, run, "valorisation_run.cancel", actor.Email, now, map[string]string{"reason": reason, "correction_required": "Mietzins und versandte Schreiben bleiben bestehen; Korrektur gesondert prüfen."}); err != nil {
		return err
	}
	return tx.Commit()
}
func (r *ValorisationRepository) PrepareLetter(id, itemID string, actor ValorisationActor, now time.Time, render ValorisationRenderer) (DocumentRecord, error) {
	if !actor.Manage || actor.Email == "" {
		return DocumentRecord{}, ErrValorisationDenied
	}
	if r.documents == nil || render == nil {
		return DocumentRecord{}, fmt.Errorf("Archiv nicht verfügbar")
	}
	tx, err := r.db.For(r.tenant).Begin()
	if err != nil {
		return DocumentRecord{}, err
	}
	defer tx.Rollback()
	res, err := tx.Exec(`UPDATE valorisation_runs SET status=status WHERE tenant_id=$1 AND id=$2 AND status='approved'`, r.tenant.ID, id)
	if err != nil {
		return DocumentRecord{}, err
	}
	n, _ := res.RowsAffected()
	if n != 1 {
		return DocumentRecord{}, ErrValorisationConflict
	}
	run, err := r.getTx(tx, id)
	if err != nil {
		return DocumentRecord{}, err
	}
	for _, item := range run.Items {
		if item.ID == itemID && !item.Excluded && item.Group == "ready" && item.Outcome != "unchanged" {
			item, err = ValorisationLetterTiming(item, now)
			if err != nil {
				return DocumentRecord{}, err
			}
			doc, err := r.archiveTx(tx, run, item, actor, now, render)
			if err != nil {
				return doc, err
			}
			_, err = tx.Exec(`UPDATE valorisation_items SET letter_document_id=$1,letter_sha256=$2,collectable_from=$3 WHERE tenant_id=$4 AND id=$5`, doc.ID, doc.ValorisationArchive.SHA256, item.CollectableFrom, r.tenant.ID, itemID)
			if err != nil {
				return doc, err
			}
			return doc, tx.Commit()
		}
	}
	return DocumentRecord{}, sql.ErrNoRows
}
func (r *ValorisationRepository) FinishSend(id string, actor ValorisationActor, now time.Time) error {
	if !actor.Manage {
		return ErrValorisationDenied
	}
	tx, err := r.db.For(r.tenant).Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	run, err := r.getTx(tx, id)
	if err != nil {
		return err
	}
	if run.Status != "approved" {
		return ErrValorisationConflict
	}
	complete := true
	for _, item := range run.Items {
		if item.Excluded || item.Group != "ready" || item.Outcome == "unchanged" {
			continue
		}
		for _, party := range item.Recipients {
			var n int
			if err = tx.QueryRow(`SELECT COUNT(*) FROM valorisation_deliveries WHERE tenant_id=$1 AND run_id=$2 AND unit_id=$3 AND recipient=$4 AND status='sent'`, r.tenant.ID, id, item.UnitID, party.Email).Scan(&n); err != nil {
				return err
			}
			if n == 0 {
				complete = false
			}
		}
	}
	if complete {
		_, err = tx.Exec(`UPDATE valorisation_runs SET status='sent' WHERE tenant_id=$1 AND id=$2 AND status='approved'`, r.tenant.ID, id)
		if err != nil {
			return err
		}
	}
	if err = r.event(tx, run, "letter.send", actor.Email, now, map[string]bool{"complete": complete}); err != nil {
		return err
	}
	return tx.Commit()
}
func (r *ValorisationRepository) event(tx *sql.Tx, run ValorisationRun, action, actor string, now time.Time, details any) error {
	id, err := newLeaseID()
	if err != nil {
		return err
	}
	raw, err := json.Marshal(struct {
		Hash    string
		Details any
	}{run.InputsSHA256, details})
	if err != nil {
		return err
	}
	_, err = tx.Exec(`INSERT INTO valorisation_events(tenant_id,tenant_slug,id,run_id,action,actor,created_at,data) VALUES($1,$2,$3,$4,$5,$6,$7,$8)`, r.tenant.ID, r.tenant.Slug, id, run.ID, action, actor, stamp(now), string(raw))
	return err
}
func euroDecimal(cents int64) string { return fmt.Sprintf("%d.%02d", cents/100, cents%100) }
func mustDate(raw string) time.Time  { d, _ := time.Parse(time.DateOnly, raw); return d }

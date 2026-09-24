package store

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"
)

// AnnualStatementReceipt links trusted invoice metadata to one original file
// in Dokumente. Deleting the link deliberately never deletes the document.
type AnnualStatementReceipt struct {
	HeatingCategory string    `json:"heating_category,omitempty"`
	Supplier        string    `json:"supplier"`
	ID              string    `json:"id"`
	DocumentID      string    `json:"document_id"`
	PeriodYear      int       `json:"period_year"`
	CostTypeKey     string    `json:"cost_type_key"`
	AmountCents     int64     `json:"amount_cents"`
	InvoiceDate     string    `json:"invoice_date"`
	CreatedAt       time.Time `json:"created_at"`
	CreatedBy       string    `json:"created_by"`
	UpdatedAt       time.Time `json:"updated_at"`
	UpdatedBy       string    `json:"updated_by"`
}

type AnnualStatementReceiptRepository interface {
	UpdateMetadata(id, supplier, category, actor string) (AnnualStatementReceipt, error)
	Create(receipt AnnualStatementReceipt) (AnnualStatementReceipt, error)
	UpdateAmount(id string, amountCents int64, actor string) (AnnualStatementReceipt, error)
	Get(id string) (AnnualStatementReceipt, bool)
	List() []AnnualStatementReceipt
	ListByPeriod(year int) []AnnualStatementReceipt
	Delete(id string) (bool, error)
}

type AnnualStatementReceiptStorage interface {
	annualStatementReceiptStorage()
}

type annualStatementReceiptBackend interface {
	updateAnnualStatementReceiptMetadata(TenantRef, string, string, string, string) (AnnualStatementReceipt, error)
	createAnnualStatementReceipt(tenant TenantRef, receipt AnnualStatementReceipt) (AnnualStatementReceipt, error)
	updateAnnualStatementReceiptAmount(tenant TenantRef, id string, amountCents int64, actor string) (AnnualStatementReceipt, error)
	getAnnualStatementReceipt(tenant TenantRef, id string) (AnnualStatementReceipt, bool)
	listAnnualStatementReceipts(tenant TenantRef, periodYear int) []AnnualStatementReceipt
	deleteAnnualStatementReceipt(tenant TenantRef, id string) (bool, error)
}

type boundAnnualStatementReceiptRepository struct {
	storage annualStatementReceiptBackend
	tenant  TenantRef
}

func BindAnnualStatementReceiptRepository(storage AnnualStatementReceiptStorage, tenant TenantRef) (AnnualStatementReceiptRepository, bool) {
	backend, ok := storage.(annualStatementReceiptBackend)
	resolved, tenantOK := validTenantRef(tenant)
	if !ok || !tenantOK {
		return nil, false
	}
	return &boundAnnualStatementReceiptRepository{storage: backend, tenant: resolved}, true
}

func (r *boundAnnualStatementReceiptRepository) Create(receipt AnnualStatementReceipt) (AnnualStatementReceipt, error) {
	return r.storage.createAnnualStatementReceipt(r.tenant, receipt)
}

func (r *boundAnnualStatementReceiptRepository) UpdateAmount(id string, amountCents int64, actor string) (AnnualStatementReceipt, error) {
	return r.storage.updateAnnualStatementReceiptAmount(r.tenant, id, amountCents, actor)
}

func (r *boundAnnualStatementReceiptRepository) Get(id string) (AnnualStatementReceipt, bool) {
	return r.storage.getAnnualStatementReceipt(r.tenant, id)
}

func (r *boundAnnualStatementReceiptRepository) List() []AnnualStatementReceipt {
	return r.storage.listAnnualStatementReceipts(r.tenant, 0)
}

func (r *boundAnnualStatementReceiptRepository) ListByPeriod(year int) []AnnualStatementReceipt {
	return r.storage.listAnnualStatementReceipts(r.tenant, year)
}

func (r *boundAnnualStatementReceiptRepository) Delete(id string) (bool, error) {
	return r.storage.deleteAnnualStatementReceipt(r.tenant, id)
}

// MemoryAnnualStatementReceiptStore uses the same tenant-bound repositories as
// production to validate every referenced period, cost type and document.
type MemoryAnnualStatementReceiptStore struct {
	mu        sync.Mutex
	byHome    map[string]map[string]AnnualStatementReceipt
	periods   AnnualStatementPeriodStorage
	costTypes AnnualStatementCostTypeStorage
	documents DocumentStorage
}

func NewMemoryAnnualStatementReceiptStore(periods AnnualStatementPeriodStorage, costTypes AnnualStatementCostTypeStorage, documents DocumentStorage) *MemoryAnnualStatementReceiptStore {
	return &MemoryAnnualStatementReceiptStore{
		byHome: map[string]map[string]AnnualStatementReceipt{}, periods: periods, costTypes: costTypes, documents: documents,
	}
}

func (*MemoryAnnualStatementReceiptStore) annualStatementReceiptStorage() {}

func (s *MemoryAnnualStatementReceiptStore) createAnnualStatementReceipt(tenant TenantRef, receipt AnnualStatementReceipt) (AnnualStatementReceipt, error) {
	receipt, err := normalizeAnnualStatementReceipt(receipt)
	if err != nil {
		return AnnualStatementReceipt{}, err
	}
	periods, periodsOK := BindAnnualStatementPeriodRepository(s.periods, tenant)
	costTypes, costTypesOK := BindAnnualStatementCostTypeRepository(s.costTypes, tenant)
	documents, documentsOK := BindDocumentRepository(s.documents, tenant)
	if !periodsOK || !costTypesOK || !documentsOK || !annualStatementReceiptReferencesValid(receipt, periods, costTypes, documents) {
		return AnnualStatementReceipt{}, fmt.Errorf("invalid annual statement receipt reference")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.byHome == nil {
		s.byHome = map[string]map[string]AnnualStatementReceipt{}
	}
	if s.byHome[tenant.ID] == nil {
		s.byHome[tenant.ID] = map[string]AnnualStatementReceipt{}
	}
	for _, existing := range s.byHome[tenant.ID] {
		if existing.DocumentID == receipt.DocumentID {
			return AnnualStatementReceipt{}, fmt.Errorf("annual statement receipt already exists for document")
		}
	}
	if _, exists := s.byHome[tenant.ID][receipt.ID]; exists {
		return AnnualStatementReceipt{}, fmt.Errorf("annual statement receipt id already exists")
	}
	s.byHome[tenant.ID][receipt.ID] = receipt
	return receipt, nil
}

func (s *MemoryAnnualStatementReceiptStore) updateAnnualStatementReceiptAmount(tenant TenantRef, id string, amountCents int64, actor string) (AnnualStatementReceipt, error) {
	id, actor = strings.TrimSpace(id), strings.ToLower(strings.TrimSpace(actor))
	if id == "" || amountCents <= 0 || actor == "" {
		return AnnualStatementReceipt{}, fmt.Errorf("invalid annual statement receipt amount update")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	receipt, ok := s.byHome[tenant.ID][id]
	if !ok {
		return AnnualStatementReceipt{}, fmt.Errorf("annual statement receipt not found")
	}
	receipt.AmountCents = amountCents
	receipt.UpdatedAt = time.Now().UTC()
	receipt.UpdatedBy = actor
	s.byHome[tenant.ID][id] = receipt
	return receipt, nil
}

func (s *MemoryAnnualStatementReceiptStore) getAnnualStatementReceipt(tenant TenantRef, id string) (AnnualStatementReceipt, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	receipt, ok := s.byHome[tenant.ID][strings.TrimSpace(id)]
	return receipt, ok
}

func (s *MemoryAnnualStatementReceiptStore) listAnnualStatementReceipts(tenant TenantRef, periodYear int) []AnnualStatementReceipt {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := []AnnualStatementReceipt{}
	for _, receipt := range s.byHome[tenant.ID] {
		if periodYear == 0 || receipt.PeriodYear == periodYear {
			out = append(out, receipt)
		}
	}
	sortAnnualStatementReceipts(out)
	return out
}

func (s *MemoryAnnualStatementReceiptStore) deleteAnnualStatementReceipt(tenant TenantRef, id string) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	id = strings.TrimSpace(id)
	if _, ok := s.byHome[tenant.ID][id]; !ok {
		return false, nil
	}
	delete(s.byHome[tenant.ID], id)
	return true, nil
}

// SQLAnnualStatementReceiptStore persists receipt links while Dokumente keeps
// ownership of the original file bytes.
type SQLAnnualStatementReceiptStore struct {
	db *TenantDB
}

func NewSQLAnnualStatementReceiptStore(db *TenantDB) *SQLAnnualStatementReceiptStore {
	return &SQLAnnualStatementReceiptStore{db: db}
}

func (*SQLAnnualStatementReceiptStore) annualStatementReceiptStorage() {}

func (s *SQLAnnualStatementReceiptStore) createAnnualStatementReceipt(tenant TenantRef, receipt AnnualStatementReceipt) (AnnualStatementReceipt, error) {
	receipt, err := normalizeAnnualStatementReceipt(receipt)
	if err != nil {
		return AnnualStatementReceipt{}, err
	}
	tx, err := s.db.For(tenant).Begin()
	if err != nil {
		return AnnualStatementReceipt{}, err
	}
	defer tx.Rollback()
	if ok, err := sqlAnnualStatementReceiptReferencesValid(tx, tenant, receipt); err != nil || !ok {
		if err != nil {
			return AnnualStatementReceipt{}, err
		}
		return AnnualStatementReceipt{}, fmt.Errorf("invalid annual statement receipt reference")
	}
	_, err = tx.Exec(
		`INSERT INTO annual_statement_receipts(
		 tenant_id, tenant_slug, id, document_id, period_year, cost_type_key, amount_cents, invoice_date, created_at, created_by, updated_at, updated_by, supplier, heating_category)
		 VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14)`,
		tenant.ID, tenant.Slug, receipt.ID, receipt.DocumentID, receipt.PeriodYear, receipt.CostTypeKey,
		receipt.AmountCents, receipt.InvoiceDate, receipt.CreatedAt.Format(time.RFC3339Nano), receipt.CreatedBy,
		receipt.UpdatedAt.Format(time.RFC3339Nano), receipt.UpdatedBy, receipt.Supplier, receipt.HeatingCategory,
	)
	if err != nil {
		return AnnualStatementReceipt{}, err
	}
	return receipt, tx.Commit()
}

func (s *SQLAnnualStatementReceiptStore) updateAnnualStatementReceiptAmount(tenant TenantRef, id string, amountCents int64, actor string) (AnnualStatementReceipt, error) {
	id, actor = strings.TrimSpace(id), strings.ToLower(strings.TrimSpace(actor))
	if id == "" || amountCents <= 0 || actor == "" {
		return AnnualStatementReceipt{}, fmt.Errorf("invalid annual statement receipt amount update")
	}
	now := time.Now().UTC()
	result, err := s.db.For(tenant).Exec(
		`UPDATE annual_statement_receipts SET amount_cents=$1, updated_at=$2, updated_by=$3 WHERE tenant_id=$4 AND id=$5`,
		amountCents, now.Format(time.RFC3339Nano), actor, tenant.ID, id,
	)
	if err != nil {
		return AnnualStatementReceipt{}, err
	}
	affected, err := result.RowsAffected()
	if err != nil || affected == 0 {
		if err != nil {
			return AnnualStatementReceipt{}, err
		}
		return AnnualStatementReceipt{}, fmt.Errorf("annual statement receipt not found")
	}
	receipt, ok := s.getAnnualStatementReceipt(tenant, id)
	if !ok {
		return AnnualStatementReceipt{}, fmt.Errorf("annual statement receipt not found")
	}
	return receipt, nil
}

func (s *SQLAnnualStatementReceiptStore) getAnnualStatementReceipt(tenant TenantRef, id string) (AnnualStatementReceipt, bool) {
	row := s.db.For(tenant).QueryRow(
		`SELECT id, document_id, period_year, cost_type_key, amount_cents, invoice_date, created_at, created_by, updated_at, updated_by, supplier, heating_category
		 FROM annual_statement_receipts WHERE tenant_id=$1 AND id=$2`, tenant.ID, strings.TrimSpace(id))
	receipt, err := scanAnnualStatementReceipt(row)
	return receipt, err == nil
}

func (s *SQLAnnualStatementReceiptStore) listAnnualStatementReceipts(tenant TenantRef, periodYear int) []AnnualStatementReceipt {
	query := `SELECT id, document_id, period_year, cost_type_key, amount_cents, invoice_date, created_at, created_by, updated_at, updated_by, supplier, heating_category
	 FROM annual_statement_receipts WHERE tenant_id=$1`
	args := []any{tenant.ID}
	if periodYear != 0 {
		query += ` AND period_year=$2`
		args = append(args, periodYear)
	}
	query += ` ORDER BY invoice_date DESC, id`
	rows, err := s.db.For(tenant).Query(query, args...)
	if err != nil {
		return nil
	}
	defer rows.Close()
	out := []AnnualStatementReceipt{}
	for rows.Next() {
		receipt, err := scanAnnualStatementReceipt(rows)
		if err == nil {
			out = append(out, receipt)
		}
	}
	return out
}

func (s *SQLAnnualStatementReceiptStore) deleteAnnualStatementReceipt(tenant TenantRef, id string) (bool, error) {
	result, err := s.db.For(tenant).Exec(`DELETE FROM annual_statement_receipts WHERE tenant_id=$1 AND id=$2`, tenant.ID, strings.TrimSpace(id))
	if err != nil {
		return false, err
	}
	affected, err := result.RowsAffected()
	return affected > 0, err
}

type annualStatementReceiptScanner interface {
	Scan(dest ...any) error
}

func scanAnnualStatementReceipt(scanner annualStatementReceiptScanner) (AnnualStatementReceipt, error) {
	var receipt AnnualStatementReceipt
	var createdAt, updatedAt string
	if err := scanner.Scan(
		&receipt.ID, &receipt.DocumentID, &receipt.PeriodYear, &receipt.CostTypeKey, &receipt.AmountCents,
		&receipt.InvoiceDate, &createdAt, &receipt.CreatedBy, &updatedAt, &receipt.UpdatedBy, &receipt.Supplier, &receipt.HeatingCategory,
	); err != nil {
		return AnnualStatementReceipt{}, err
	}
	var err error
	if receipt.CreatedAt, err = time.Parse(time.RFC3339Nano, createdAt); err != nil {
		return AnnualStatementReceipt{}, err
	}
	if receipt.UpdatedAt, err = time.Parse(time.RFC3339Nano, updatedAt); err != nil {
		return AnnualStatementReceipt{}, err
	}
	return receipt, nil
}

func normalizeAnnualStatementReceipt(receipt AnnualStatementReceipt) (AnnualStatementReceipt, error) {
	if receipt.HeatingCategory != "" && receipt.HeatingCategory != "energie" && receipt.HeatingCategory != "sonstige_betriebskosten" {
		return AnnualStatementReceipt{}, fmt.Errorf("invalid heating category")
	}
	receipt.Supplier = strings.TrimSpace(receipt.Supplier)
	if len(receipt.Supplier) > 300 {
		return AnnualStatementReceipt{}, fmt.Errorf("supplier too long")
	}
	receipt.ID = strings.TrimSpace(receipt.ID)
	receipt.DocumentID = strings.TrimSpace(receipt.DocumentID)
	receipt.CostTypeKey = strings.ToLower(strings.TrimSpace(receipt.CostTypeKey))
	receipt.InvoiceDate = strings.TrimSpace(receipt.InvoiceDate)
	receipt.CreatedBy = strings.ToLower(strings.TrimSpace(receipt.CreatedBy))
	receipt.UpdatedBy = strings.ToLower(strings.TrimSpace(receipt.UpdatedBy))
	if receipt.ID == "" {
		var err error
		receipt.ID, err = randomToken(12)
		if err != nil {
			return AnnualStatementReceipt{}, err
		}
	}
	now := time.Now().UTC()
	if receipt.CreatedAt.IsZero() {
		receipt.CreatedAt = now
	} else {
		receipt.CreatedAt = receipt.CreatedAt.UTC()
	}
	if receipt.UpdatedAt.IsZero() {
		receipt.UpdatedAt = receipt.CreatedAt
	} else {
		receipt.UpdatedAt = receipt.UpdatedAt.UTC()
	}
	if receipt.UpdatedBy == "" {
		receipt.UpdatedBy = receipt.CreatedBy
	}
	date, dateErr := time.Parse("2006-01-02", receipt.InvoiceDate)
	if receipt.DocumentID == "" || receipt.PeriodYear < 1 || receipt.PeriodYear > 9999 || receipt.CostTypeKey == "" ||
		receipt.AmountCents <= 0 || dateErr != nil || date.Format("2006-01-02") != receipt.InvoiceDate || receipt.CreatedBy == "" || receipt.UpdatedBy == "" {
		return AnnualStatementReceipt{}, fmt.Errorf("invalid annual statement receipt")
	}
	return receipt, nil
}

func annualStatementReceiptReferencesValid(receipt AnnualStatementReceipt, periods AnnualStatementPeriodRepository, costTypes AnnualStatementCostTypeRepository, documents DocumentRepository) bool {
	periodFound := false
	for _, period := range periods.List() {
		if period.Year == receipt.PeriodYear {
			periodFound = true
			break
		}
	}
	if !periodFound {
		return false
	}
	costTypeFound := false
	periodCostTypes := costTypes.List()
	if structure, found := periods.Structure(receipt.PeriodYear); found {
		periodCostTypes = structure.CostTypes
	}
	for _, costType := range periodCostTypes {
		if costType.Key == receipt.CostTypeKey {
			costTypeFound = true
			break
		}
	}
	if !costTypeFound {
		return false
	}
	document, ok := documents.Get(receipt.DocumentID)
	return ok && document.Current && annualStatementReceiptDocumentTypeSupported(document.ContentType)
}

func sqlAnnualStatementReceiptReferencesValid(tx *sql.Tx, tenant TenantRef, receipt AnnualStatementReceipt) (bool, error) {
	var periodExists, costTypeExists bool
	if err := tx.QueryRow(`SELECT EXISTS(SELECT 1 FROM annual_statement_periods WHERE tenant_id=$1 AND year=$2)`, tenant.ID, receipt.PeriodYear).Scan(&periodExists); err != nil {
		return false, err
	}
	if err := tx.QueryRow(`SELECT CASE
		WHEN EXISTS(SELECT 1 FROM annual_statement_period_cost_types WHERE tenant_id=$1 AND period_year=$2)
		THEN EXISTS(SELECT 1 FROM annual_statement_period_cost_types WHERE tenant_id=$1 AND period_year=$2 AND key=$3)
		ELSE EXISTS(SELECT 1 FROM annual_statement_cost_types WHERE tenant_id=$1 AND key=$3)
		END`, tenant.ID, receipt.PeriodYear, receipt.CostTypeKey).Scan(&costTypeExists); err != nil {
		return false, err
	}
	var raw string
	if err := tx.QueryRow(`SELECT data FROM documents WHERE tenant_id=$1 AND id=$2`, tenant.ID, receipt.DocumentID).Scan(&raw); err != nil {
		if err == sql.ErrNoRows {
			return false, nil
		}
		return false, err
	}
	var document DocumentRecord
	if err := json.Unmarshal([]byte(raw), &document); err != nil {
		return false, err
	}
	return periodExists && costTypeExists && document.Current && annualStatementReceiptDocumentTypeSupported(document.ContentType), nil
}

func annualStatementReceiptDocumentTypeSupported(contentType string) bool {
	switch stripContentTypeParams(contentType) {
	case "application/pdf", "image/jpeg", "image/png", "image/webp":
		return true
	default:
		return false
	}
}

func sortAnnualStatementReceipts(receipts []AnnualStatementReceipt) {
	sort.Slice(receipts, func(i, j int) bool {
		if receipts[i].InvoiceDate == receipts[j].InvoiceDate {
			return receipts[i].ID < receipts[j].ID
		}
		return receipts[i].InvoiceDate > receipts[j].InvoiceDate
	})
}

var _ AnnualStatementReceiptStorage = (*MemoryAnnualStatementReceiptStore)(nil)
var _ AnnualStatementReceiptStorage = (*SQLAnnualStatementReceiptStore)(nil)

func (r *boundAnnualStatementReceiptRepository) UpdateMetadata(id, supplier, category, actor string) (AnnualStatementReceipt, error) {
	return r.storage.updateAnnualStatementReceiptMetadata(r.tenant, id, supplier, category, actor)
}
func receiptMetadata(receipt AnnualStatementReceipt, supplier, category, actor string) (AnnualStatementReceipt, error) {
	receipt.Supplier = supplier
	receipt.HeatingCategory = category
	receipt.UpdatedBy = actor
	receipt.UpdatedAt = time.Now().UTC()
	if strings.TrimSpace(actor) == "" {
		return AnnualStatementReceipt{}, fmt.Errorf("missing actor")
	}
	return normalizeAnnualStatementReceipt(receipt)
}
func (s *MemoryAnnualStatementReceiptStore) updateAnnualStatementReceiptMetadata(tenant TenantRef, id, supplier, category, actor string) (AnnualStatementReceipt, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	receipt, ok := s.byHome[tenant.ID][id]
	if !ok {
		return AnnualStatementReceipt{}, fmt.Errorf("receipt not found")
	}
	receipt, err := receiptMetadata(receipt, supplier, category, actor)
	if err != nil {
		return AnnualStatementReceipt{}, err
	}
	s.byHome[tenant.ID][id] = receipt
	return receipt, nil
}
func (s *SQLAnnualStatementReceiptStore) updateAnnualStatementReceiptMetadata(tenant TenantRef, id, supplier, category, actor string) (AnnualStatementReceipt, error) {
	receipt, ok := s.getAnnualStatementReceipt(tenant, id)
	if !ok {
		return AnnualStatementReceipt{}, fmt.Errorf("receipt not found")
	}
	receipt, err := receiptMetadata(receipt, supplier, category, actor)
	if err != nil {
		return AnnualStatementReceipt{}, err
	}
	result, err := s.db.For(tenant).Exec(`UPDATE annual_statement_receipts SET supplier=$1,heating_category=$2,updated_by=$3,updated_at=$4 WHERE tenant_id=$5 AND id=$6`, receipt.Supplier, receipt.HeatingCategory, receipt.UpdatedBy, receipt.UpdatedAt.Format(time.RFC3339Nano), tenant.ID, id)
	if err != nil {
		return AnnualStatementReceipt{}, err
	}
	n, err := result.RowsAffected()
	if err != nil || n != 1 {
		return AnnualStatementReceipt{}, fmt.Errorf("receipt update failed")
	}
	saved, ok := s.getAnnualStatementReceipt(tenant, id)
	if !ok {
		return AnnualStatementReceipt{}, fmt.Errorf("receipt not found")
	}
	return saved, nil
}

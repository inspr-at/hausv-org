package integrations

import (
	"context"
	"os"
	"strings"
	"testing"
)

func TestCAMT053AdapterParses2009And2019GoldenFiles(t *testing.T) {
	for _, tc := range []struct {
		name      string
		fixture   string
		version   string
		reference string
		cents     int64
	}{
		{"2009", "testdata/camt053-2009.xml", "2009/camt.053.001.02", "HV-JHW22-202606-ABC123", 13304},
		{"2019", "testdata/camt053-2019.xml", "2019/camt.053.001.08", "HV-JHW22-202607-DEF456", 42},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f, err := os.Open(tc.fixture)
			if err != nil {
				t.Fatalf("open fixture: %v", err)
			}
			defer f.Close()
			result, err := CAMT053Adapter{}.ParsePayments(context.Background(), Source{Format: FormatCAMT053, Filename: tc.fixture}, f)
			if err != nil {
				t.Fatalf("ParsePayments: %v", err)
			}
			if result.Report.Accepted != 1 || result.Report.Rejected != 0 || len(result.Payments) != 1 {
				t.Fatalf("result = %+v", result)
			}
			payment := result.Payments[0]
			if payment.Source.Version != tc.version || payment.Reference != tc.reference || payment.Amount.Cents != tc.cents || payment.Amount.Currency != "EUR" {
				t.Fatalf("payment = %+v", payment)
			}
			if payment.BookingDate.IsZero() || payment.RawDigest == "" || payment.DebtorName == "" || len(payment.RemittanceLines) == 0 {
				t.Fatalf("payment missing expected fields: %+v", payment)
			}
		})
	}
}

func TestCAMT053AdapterReportsBadRecordsWithoutAborting(t *testing.T) {
	xml := `<?xml version="1.0" encoding="UTF-8"?>
<Document xmlns="urn:iso:std:iso:20022:tech:xsd:camt.053.001.08">
  <BkToCstmrStmt>
    <Stmt>
      <Id>JHW22</Id>
      <Acct><Ownr><Nm>jhw22</Nm></Ownr></Acct>
      <Ntry>
        <NtryRef>ok-1</NtryRef><Amt Ccy="EUR">1.00</Amt><CdtDbtInd>CRDT</CdtDbtInd><BookgDt><Dt>2026-07-08</Dt></BookgDt>
        <NtryDtls><TxDtls><Refs><EndToEndId>HV-JHW22-202607-OK123</EndToEndId></Refs><RmtInf><Ustrd>ok</Ustrd></RmtInf></TxDtls></NtryDtls>
      </Ntry>
      <Ntry>
        <NtryRef>debit-1</NtryRef><Amt Ccy="EUR">2.00</Amt><CdtDbtInd>DBIT</CdtDbtInd><BookgDt><Dt>2026-07-08</Dt></BookgDt>
      </Ntry>
      <Ntry>
        <NtryRef>bad-amount</NtryRef><Amt Ccy="EUR">2.123</Amt><CdtDbtInd>CRDT</CdtDbtInd><BookgDt><Dt>2026-07-08</Dt></BookgDt>
      </Ntry>
    </Stmt>
  </BkToCstmrStmt>
</Document>`
	result, err := CAMT053Adapter{}.ParsePayments(context.Background(), Source{}, strings.NewReader(xml))
	if err != nil {
		t.Fatalf("ParsePayments: %v", err)
	}
	if len(result.Payments) != 1 || result.Report.Accepted != 1 || result.Report.Rejected != 2 {
		t.Fatalf("result = %+v", result)
	}
	if result.Report.Errors[0].RecordID != "debit-1" || result.Report.Errors[1].RecordID != "bad-amount" {
		t.Fatalf("record errors = %+v", result.Report.Errors)
	}
}

func TestCAMT053AdapterRejectsUnsupportedNamespaceAsReportError(t *testing.T) {
	result, err := CAMT053Adapter{}.ParsePayments(context.Background(), Source{}, strings.NewReader(`<Document xmlns="urn:example"><BkToCstmrStmt /></Document>`))
	if err != nil {
		t.Fatalf("ParsePayments: %v", err)
	}
	if len(result.Payments) != 0 || result.Report.Rejected != 1 || result.Report.Errors[0].Field != "namespace" {
		t.Fatalf("result = %+v", result)
	}
}

func TestCAMT053AdapterUsesTenantBoundSourceInsteadOfBankOwnerName(t *testing.T) {
	xml := `<?xml version="1.0" encoding="UTF-8"?>
<Document xmlns="urn:iso:std:iso:20022:tech:xsd:camt.053.001.08">
  <BkToCstmrStmt><Stmt><Id>statement-1</Id>
    <Acct><Ownr><Nm>Unrelated account owner</Nm></Ownr></Acct>
    <Ntry><NtryRef>entry-1</NtryRef><Amt Ccy="EUR">1.00</Amt><CdtDbtInd>CRDT</CdtDbtInd><BookgDt><Dt>2026-07-08</Dt></BookgDt>
      <NtryDtls><TxDtls><Refs><EndToEndId>HV-JHW22-202607-OK123</EndToEndId></Refs></TxDtls></NtryDtls>
    </Ntry>
  </Stmt></BkToCstmrStmt>
</Document>`
	result, err := CAMT053Adapter{}.ParsePayments(context.Background(), Source{
		TenantSlug: "jhw22",
		Format:     FormatCAMT053,
	}, strings.NewReader(xml))
	if err != nil {
		t.Fatalf("ParsePayments: %v", err)
	}
	if len(result.Payments) != 1 || result.Payments[0].TenantSlug != "jhw22" {
		t.Fatalf("tenant-bound payment = %+v", result.Payments)
	}
}

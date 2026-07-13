package main

import (
	"context"
	"os"
	"strings"
	"testing"
)

func TestCAMT054AdapterParsesPaymentAdviceGoldenFile(t *testing.T) {
	f, err := os.Open("testdata/camt054-2019.xml")
	if err != nil {
		t.Fatalf("open fixture: %v", err)
	}
	defer f.Close()
	result, err := camt054Adapter{}.ParsePayments(context.Background(), integrationSource{Filename: "testdata/camt054-2019.xml"}, f)
	if err != nil {
		t.Fatalf("ParsePayments: %v", err)
	}
	if result.Report.Accepted != 1 || result.Report.Rejected != 0 || len(result.Payments) != 1 {
		t.Fatalf("result = %+v", result)
	}
	payment := result.Payments[0]
	if payment.Source.Format != integrationFormatCAMT054 || payment.Source.Version != "2019/camt.054.001.08" {
		t.Fatalf("source = %+v", payment.Source)
	}
	if payment.Reference != "HV-JHW22-202607-GHI789" || payment.Amount.Cents != 13304 || payment.Amount.Currency != "EUR" {
		t.Fatalf("payment = %+v", payment)
	}
	if payment.BookingDate.IsZero() || payment.RawDigest == "" || len(payment.RemittanceLines) != 1 {
		t.Fatalf("payment missing expected fields: %+v", payment)
	}
}

func TestCAMT054AdapterRejectsUnsupportedNamespaceAsReportError(t *testing.T) {
	result, err := camt054Adapter{}.ParsePayments(context.Background(), integrationSource{}, strings.NewReader(`<Document xmlns="urn:example"><BkToCstmrDbtCdtNtfctn /></Document>`))
	if err != nil {
		t.Fatalf("ParsePayments: %v", err)
	}
	if len(result.Payments) != 0 || result.Report.Rejected != 1 || result.Report.Errors[0].Field != "namespace" {
		t.Fatalf("result = %+v", result)
	}
}

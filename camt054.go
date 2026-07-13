package main

import (
	"context"
	"encoding/xml"
	"io"
	"strings"
)

type camt054Adapter struct{}

func (camt054Adapter) ParsePayments(ctx context.Context, source integrationSource, r io.Reader) (paymentImportResult, error) {
	select {
	case <-ctx.Done():
		return paymentImportResult{}, ctx.Err()
	default:
	}
	data, err := io.ReadAll(r)
	if err != nil {
		return paymentImportResult{}, err
	}
	var document camt054Document
	if err := xml.Unmarshal(data, &document); err != nil {
		return paymentImportResult{}, err
	}
	version, ok := camt054VersionFromNamespace(document.XMLName.Space)
	if !ok {
		report := buildIntegrationReport(source, 0, []integrationRecordError{{
			RecordType: integrationRecordPayment,
			Field:      "namespace",
			Message:    "unsupported camt.054 namespace",
		}})
		return paymentImportResult{Report: report}, nil
	}
	if source.Format == "" {
		source.Format = integrationFormatCAMT054
	}
	if source.Version == "" {
		source.Version = version
	}

	payments := []canonicalPayment{}
	errors := []integrationRecordError{}
	for notificationIndex, notification := range document.CustomerNotification.Notifications {
		for entryIndex, entry := range notification.Entries {
			payment, recordErrors := camtPaymentFromEntry(source, notification, entry, notificationIndex, entryIndex)
			if len(recordErrors) > 0 {
				errors = append(errors, recordErrors...)
				continue
			}
			payments = append(payments, payment)
		}
	}
	return paymentImportResult{
		Payments: payments,
		Report:   buildIntegrationReport(source, len(payments), errors),
	}, nil
}

func camt054VersionFromNamespace(namespace string) (string, bool) {
	switch strings.TrimSpace(namespace) {
	case "urn:iso:std:iso:20022:tech:xsd:camt.054.001.02":
		return "2009/camt.054.001.02", true
	case "urn:iso:std:iso:20022:tech:xsd:camt.054.001.08":
		return "2019/camt.054.001.08", true
	default:
		return "", false
	}
}

type camt054Document struct {
	XMLName              xml.Name             `xml:"Document"`
	CustomerNotification camtCustomerNotified `xml:"BkToCstmrDbtCdtNtfctn"`
}

type camtCustomerNotified struct {
	Notifications []camtStatement `xml:"Ntfctn"`
}

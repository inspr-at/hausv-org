package integrations

import (
	"context"
	"encoding/xml"
	"io"
	"strings"
)

type CAMT054Adapter struct{}

func (CAMT054Adapter) ParsePayments(ctx context.Context, source Source, r io.Reader) (PaymentImportResult, error) {
	select {
	case <-ctx.Done():
		return PaymentImportResult{}, ctx.Err()
	default:
	}
	data, err := io.ReadAll(r)
	if err != nil {
		return PaymentImportResult{}, err
	}
	var document camt054Document
	if err := xml.Unmarshal(data, &document); err != nil {
		return PaymentImportResult{}, err
	}
	version, ok := camt054VersionFromNamespace(document.XMLName.Space)
	if !ok {
		report := buildReport(source, 0, []RecordError{{
			RecordType: RecordPayment,
			Field:      "namespace",
			Message:    "unsupported camt.054 namespace",
		}})
		return PaymentImportResult{Report: report}, nil
	}
	if source.Format == "" {
		source.Format = FormatCAMT054
	}
	if source.Version == "" {
		source.Version = version
	}

	payments := []Payment{}
	errors := []RecordError{}
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
	return PaymentImportResult{
		Payments: payments,
		Report:   buildReport(source, len(payments), errors),
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

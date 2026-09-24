package server

import (
	"errors"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/inspr-at/hausv-org/internal/store"
)

func TestStaffelFormParsing(t *testing.T) {
	form := url.Values{"starts_on": {"2025-02-01"}, "clause_type": {"staffel"}, "staffel_date": {"2026-04-01", "2027-04-01"}, "staffel_kind": {"amount", "percent"}, "staffel_value": {"1.050,00", "2,5"}}
	parse := func(v url.Values) error {
		r := httptest.NewRequest("POST", "/", strings.NewReader(v.Encode()))
		r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		_, _, _, clause, _, err := leaseFromForm(r, "top-1", "manager@example.com")
		if err == nil && (len(clause.StaffelSteps) != 2 || *clause.StaffelSteps[0].NetCents != 105000 || clause.StaffelSteps[1].Percent != "2.5") {
			t.Fatal(clause)
		}
		return err
	}
	if err := parse(form); err != nil {
		t.Fatal(err)
	}
	form["staffel_value"][1] = "2,5%"
	if err := parse(form); err != nil {
		t.Fatal("explicit percent sign", err)
	}
	form["staffel_value"][0] = "2%"
	if err := parse(form); !errors.Is(err, store.ErrLeaseInvalid) {
		t.Fatal("percent in amount field accepted", err)
	}
	form["staffel_value"][0] = "1.050,00"
	form["staffel_date"][1] = "2026-04-01"
	if err := parse(form); !errors.Is(err, store.ErrLeaseInvalid) {
		t.Fatal("duplicate date not classified as invalid lease", err)
	}
	form["staffel_date"][1] = "2027-04-01"
	form["staffel_value"] = []string{"1000"}
	if err := parse(form); err == nil {
		t.Fatal("incomplete row accepted")
	}
}

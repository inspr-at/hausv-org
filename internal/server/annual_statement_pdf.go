package server

import (
	"errors"
	"mime"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/inspr-at/hausv-org/internal/statementpdf"
	"github.com/inspr-at/hausv-org/internal/store"
	"github.com/inspr-at/hausv-org/internal/textutil"
)

func (a *app) downloadAnnualStatementPDF(w http.ResponseWriter, r *http.Request, ac authCtx) {
	if _, _, _, _, ok := a.buildingSettingsContext(w, ac); !ok {
		return
	}
	if ac.repositories.annualStatementRuns == nil {
		http.Error(w, "Abrechnungslauf derzeit nicht verfügbar.", http.StatusServiceUnavailable)
		return
	}
	run, found, err := ac.repositories.annualStatementRuns.Get(r.PathValue("runID"))
	if err != nil {
		http.Error(w, "Abrechnungslauf konnte nicht gelesen werden.", http.StatusInternalServerError)
		return
	}
	if !found {
		http.NotFound(w, r)
		return
	}
	query := r.URL.Query()
	unit, party := query.Get("unit"), query.Get("party")
	if query.Has("unit") != query.Has("party") || (query.Has("unit") && (unit == "" || party == "")) || len(query["unit"]) > 1 || len(query["party"]) > 1 {
		http.NotFound(w, r)
		return
	}
	data, err := statementpdf.Render(run, unit, party)
	if query.Get("aushang") == "1" {
		if unit != "" || party != "" {
			http.NotFound(w, r)
			return
		}
		data, err = statementpdf.RenderAushang(run)
	}
	if errors.Is(err, statementpdf.ErrNotFound) {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		http.Error(w, "PDF konnte nicht erstellt werden.", http.StatusInternalServerError)
		return
	}
	filename := annualStatementPDFFilename(run, unit)
	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", mime.FormatMediaType("attachment", map[string]string{"filename": filename}))
	w.Header().Set("Cache-Control", "private, no-store")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Content-Length", strconv.Itoa(len(data)))
	_, _ = w.Write(data)
}

func annualStatementPDFURL(runID, unit, party string) string {
	target := "/app/settings/annual-statement/runs/" + url.PathEscape(runID) + "/pdf"
	if unit != "" {
		target += "?" + (url.Values{"unit": {unit}, "party": {party}}).Encode()
	}
	return target
}

func annualStatementPDFFilename(run store.AnnualStatementRun, unitID string) string {
	label := "alle-dokumente"
	for _, unit := range run.Result.Units {
		if unit.UnitID == unitID {
			label = unit.Label
			break
		}
	}
	// Slug normalization can retain Unicode: the final allowlist is explicitly ASCII.
	safe := func(value string) string {
		value = textutil.Slug(value)
		var b strings.Builder
		for _, c := range value {
			if c >= 'a' && c <= 'z' || c >= '0' && c <= '9' || c == '-' {
				b.WriteRune(c)
			}
		}
		result := strings.Trim(b.String(), "-")
		if result == "" {
			return "abrechnung"
		}
		if len(result) > 60 {
			result = result[:60]
		}
		return result
	}
	return safe(run.Input.Presentation.EstateSlug) + "-" + strconv.Itoa(run.PeriodYear) + "-" + safe(label) + "-lauf-" + strconv.Itoa(run.Revision) + ".pdf"
}

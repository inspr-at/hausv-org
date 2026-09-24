package store

import (
	"fmt"
	"strings"

	"github.com/inspr-at/hausv-org/internal/indexation"
)

// ParseStaffel accepts dates and literal euro amounts or percentages only.
// CSV callers must quote the cell when its semicolons match the CSV delimiter.
func ParseStaffel(raw string) ([]indexation.StaffelStep, error) {
	if strings.TrimSpace(raw) == "" {
		return nil, fmt.Errorf("Staffelstufen fehlen")
	}
	var steps []indexation.StaffelStep
	for _, entry := range strings.Split(raw, ";") {
		date, value, ok := strings.Cut(strings.TrimSpace(entry), ":")
		if !ok {
			return nil, fmt.Errorf("Staffel: Datum:Betrag oder Datum:Prozent%% erwartet")
		}
		step, err := ParseStaffelStep(strings.TrimSpace(date), strings.TrimSpace(value))
		if err != nil {
			return nil, err
		}
		steps = append(steps, step)
	}
	return steps, indexation.ValidateStaffelSteps(steps)
}

// ParseStaffelStep is shared by CSV import and the structured form rows.
func ParseStaffelStep(date, value string) (indexation.StaffelStep, error) {
	step := indexation.StaffelStep{EffectiveOn: date}
	if strings.HasSuffix(value, "%") {
		p, err := indexation.ParseDecimal(strings.TrimSpace(strings.TrimSuffix(value, "%")))
		if err != nil {
			return step, fmt.Errorf("ungültiger Staffelprozentsatz")
		}
		step.Percent = p.String()
	} else {
		// Unlike generic money import, reject ambiguous sub-cent amounts and
		// overflow before conversion. Austrian grouping is allowed with comma.
		value = strings.TrimSpace(strings.TrimSuffix(value, "€"))
		if strings.Contains(value, ",") {
			value = strings.ReplaceAll(value, ".", "")
		}
		d, err := indexation.ParseDecimal(value)
		if err != nil || d <= 0 || d%(indexation.Unit/100) != 0 {
			return step, fmt.Errorf("Staffel-HMZ muss ein positiver Centbetrag sein")
		}
		cents := int64(d / (indexation.Unit / 100))
		step.NetCents = &cents
	}
	if err := indexation.ValidateStaffelSteps([]indexation.StaffelStep{step}); err != nil {
		return step, err
	}
	return step, nil
}

func staffelStartCents(lease Lease, clause IndexClause) (int64, error) {
	var cents int64
	last := ""
	for _, c := range lease.Components {
		if c.Kind == ComponentHMZ && c.ValidFrom <= clause.ValidFrom && c.ValidFrom >= last {
			last, cents = c.ValidFrom, c.NetCents
		}
	}
	if cents <= 0 {
		return 0, fmt.Errorf("HMZ zum Beginn der Staffelklausel fehlt")
	}
	return cents, nil
}

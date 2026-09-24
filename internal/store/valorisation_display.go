package store

import (
	"fmt"
	"math/big"
	"strings"

	"github.com/inspr-at/hausv-org/internal/indexation"
)

// Display helpers never feed rounded values back into either calculation curve.
func ValorisationNumber(raw string, precision int) string {
	r, ok := new(big.Rat).SetString(raw)
	if !ok {
		return "–"
	}
	s := r.FloatString(precision)
	if precision > 0 {
		s = strings.TrimRight(strings.TrimRight(s, "0"), ".")
	}
	return strings.ReplaceAll(s, ".", ",")
}

func ValorisationPercent(raw string, precision int) string {
	n := ValorisationNumber(raw, precision)
	if r, ok := new(big.Rat).SetString(raw); ok && r.Sign() > 0 {
		n = "+" + n
	}
	return n + " %"
}

func ValorisationThresholdUnit(kind string) string {
	if kind == "points" {
		return "Punkte"
	}
	return "%"
}

func ValorisationExactMoney(raw string) string {
	r, ok := new(big.Rat).SetString(raw)
	if !ok {
		return "–"
	}
	r.Quo(r, big.NewRat(100, 1))
	parts := strings.Split(r.FloatString(2), ".")
	digits := parts[0]
	for j := len(digits) - 3; j > 0 && digits[j-1] != '-'; j -= 3 {
		digits = digits[:j] + "." + digits[j:]
	}
	return digits + "," + parts[1] + " €"
}

func (i ValorisationItem) Label() string {
	label := i.UnitLabel
	if label == "" {
		// Compatibility for drafts created before unit labels were frozen.
		label = strings.ReplaceAll(i.UnitID, "-", " ")
		if strings.HasPrefix(label, "top ") {
			label = "Top " + strings.TrimPrefix(label, "top ")
		}
	}
	for _, party := range i.Recipients {
		if party.Role == PartyHauptmieter && party.Name != "" {
			return label + " · " + party.Name
		}
	}
	return label
}

func (i ValorisationItem) UsedIndices() []ValorisationIndex {
	var values []ValorisationIndex
	for _, value := range i.Indices {
		if value.Used {
			values = append(values, value)
		}
	}
	return values
}

func (v ValorisationIndex) StatusLabel() string {
	switch v.Status {
	case "final":
		return "Endgültig"
	case "preliminary":
		return "Vorläufig"
	case "derived":
		return "Abgeleitet"
	}
	return "Unbekannt"
}

func (v ValorisationIndex) SeriesLabel() string {
	if strings.HasPrefix(v.Series, "VPI") {
		return "VPI " + strings.TrimPrefix(v.Series, "VPI")
	}
	return v.Series
}

func (r ValorisationRun) DeliveryLabel(unitID string) string {
	for _, item := range r.Items {
		if item.UnitID == unitID {
			return item.Label()
		}
	}
	return "Einheit"
}

// NoticeDeadlines shows each distinct deadline once, in item order.
func (r ValorisationRun) NoticeDeadlines() []ValorisationItem {
	seen := map[string]bool{}
	var deadlines []ValorisationItem
	for _, item := range r.Items {
		key := fmt.Sprintf("%s/%s", item.NoticeDeadline, item.CollectableFrom)
		if item.Group == "ready" && !item.Excluded && item.NoticeDeadline != "" && !seen[key] {
			deadlines = append(deadlines, item)
			seen[key] = true
		}
	}
	return deadlines
}

func valorisationContractExplanation(step indexation.ExplanationStep, clause IndexClause) string {
	change := ValorisationPercent(step.ExactChangePercent, 2)
	if clause.ThresholdKind == "points" {
		points := ValorisationNumber((step.Index - step.Base).String(), 1)
		if step.Index > step.Base {
			points = "+" + points
		}
		change = points + " Punkte (" + change + ")"
	}
	trigger := fmt.Sprintf("Schwelle %s %s überschritten: %s", ValorisationNumber(step.Threshold.String(), 2), ValorisationThresholdUnit(clause.ThresholdKind), change)
	if clause.ClauseType == ClauseVPIPeriodic {
		trigger = "Jährliche Anpassung: " + change
	}
	return fmt.Sprintf("%s: Index %s, Basis %s; %s. Vertragskurve: %s (gerundet angezeigt).", step.Month, ValorisationNumber(step.Index.String(), 1), ValorisationNumber(step.Base.String(), 1), trigger, ValorisationExactMoney(step.ExactAmountCents))
}

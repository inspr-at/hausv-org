package web

import (
	"strings"
	"testing"

	"github.com/inspr-at/hausv-org/internal/view"
)

func TestBallotCardsKeepResultsAndVoteControlsBehindViewGates(t *testing.T) {
	ballot := view.BallotView{ID: "test", Title: "Entscheidung", IsOpen: true, Participation: "42,0 %", Options: []view.BallotOptionView{{Value: "Ja", Label: "Ja", PercentStyle: "0"}}}
	body := renderComponent(t, BallotCard(ballot))
	if strings.Contains(body, `name="option"`) || strings.Contains(body, "42,0 %") || strings.Contains(body, "vote-result-rows") {
		t.Fatal("read-only open ballot must not expose voting or unpublished tally")
	}
	ballot.CanVote = true
	body = renderComponent(t, BallotCard(ballot))
	for _, want := range []string{`<fieldset class="vote-options">`, `name="option"`, `name="ballot_id" value="test"`, "Stimme speichern"} {
		if !strings.Contains(body, want) {
			t.Fatalf("voting flow lost %q", want)
		}
	}
	ballot.CanVote = false
	ballot.IsOpen = false
	ballot.IsClosed = true
	ballot.HasResults = true
	ballot.HasProtocol = true
	ballot.ProtocolURL = "/app/abstimmungen/test/protokoll"
	ballot.QuorumStatus = "Quorum offen"
	body = renderComponent(t, BallotCard(ballot))
	for _, want := range []string{"Abstimmungsergebnis", "42,0 %", "Quorum offen", "Noch keine Stimmen", `href="/app/abstimmungen/test/protokoll"`} {
		if !strings.Contains(body, want) {
			t.Fatalf("closed ballot lost %q", want)
		}
	}
	if strings.Contains(body, `name="option"`) {
		t.Fatal("closed ballot still offers voting")
	}
}

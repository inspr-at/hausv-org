package main

import (
	"encoding/json"
	"path/filepath"
	"testing"

	"github.com/inspr-at/hausv-org/internal/store"
)

func TestDemoClosedBallotUsesOwnersRealShares(t *testing.T) {
	generated, _ := buildHousesAndPersons()
	var committed []house
	readJSON(t, filepath.Join("..", "seed", "houses.json"), &committed)
	for _, tc := range []struct {
		name string
		home house
	}{{"generated", generated[0]}, {"committed", committed[0]}} {
		t.Run(tc.name, func(t *testing.T) {
			home := tc.home
			raw, err := json.Marshal(home.Ballots[0])
			if err != nil {
				t.Fatal(err)
			}
			var ballot store.Ballot
			if err := json.Unmarshal(raw, &ballot); err != nil {
				t.Fatal(err)
			}
			expected := map[string]int{}
			for i, u := range home.Units {
				expected[u.OwnerEmail] += demoUnitSharePPM(i, len(home.Units))
			}
			participation, yes := 0, 0
			for email, vote := range ballot.Votes {
				if vote.Weight <= 0 || vote.Weight != expected[email] {
					t.Errorf("%s weight=%d actual share=%d", email, vote.Weight, expected[email])
				}
				participation += vote.Weight
				if vote.Option == "Ja" {
					yes += vote.Weight
				}
			}
			if len(ballot.Votes) < 5 || participation < 600000 || participation > 750000 || yes*100 < participation*60 {
				t.Errorf("unrealistic votes=%d participation=%d yes=%d", len(ballot.Votes), participation, yes)
			}
		})
	}
}

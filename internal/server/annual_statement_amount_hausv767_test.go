package server

import (
	"math"
	"testing"
)

func TestAnnualStatementAmountSyntax(t *testing.T) {
	for _, parser := range []struct {
		name  string
		parse func(string) (int64, bool)
		zero  bool
	}{{"prepayment", parseAnnualStatementPrepaymentAmount, true}, {"receipt", parseAnnualStatementReceiptAmount, false}} {
		t.Run(parser.name, func(t *testing.T) {
			for _, tc := range []struct {
				raw   string
				want  int64
				valid bool
			}{
				{"-0,50", 0, false}, {"-0.00", 0, false}, {"1,+1", 0, false}, {"1,-0", 0, false}, {"+1,00", 0, false},
				{"0,00", 0, parser.zero}, {" 001,05 ", 105, true}, {"1234.56", 123456, true},
				{"92233720368547758,07", math.MaxInt64, true}, {"92233720368547758,08", 0, false},
			} {
				t.Run(tc.raw, func(t *testing.T) {
					got, valid := parser.parse(tc.raw)
					if valid != tc.valid || (valid && got != tc.want) {
						t.Fatalf("parse(%q) = %d, %t; want %d, %t", tc.raw, got, valid, tc.want, tc.valid)
					}
				})
			}
		})
	}
}

// Package indexation implements offline Austrian rent indexation calculations.
// Contractual results are distinct from statutory ceilings and notice dates.
package indexation

import (
	"fmt"
	"math/big"
	"strconv"
	"strings"
)

// Decimal is a fixed-point number with six decimal places. It represents index
// points, percentages (5*Unit means 5%, not 0.05), and chaining factors.
type Decimal int64

const Unit Decimal = 1_000_000

// ParseDecimal accepts dot or comma decimal separators, without exponents or
// grouping separators. No binary floating-point arithmetic is used.
func ParseDecimal(s string) (Decimal, error) {
	s = strings.ReplaceAll(strings.TrimSpace(s), ",", ".")
	negative := strings.HasPrefix(s, "-")
	if negative {
		s = s[1:]
	}
	parts := strings.Split(s, ".")
	if len(parts) > 2 || parts[0] == "" {
		return 0, fmt.Errorf("invalid decimal %q", s)
	}
	fraction := ""
	if len(parts) == 2 {
		fraction = parts[1]
		if fraction == "" || len(fraction) > 6 {
			return 0, fmt.Errorf("invalid decimal precision %q", s)
		}
	}
	digits := parts[0] + fraction + strings.Repeat("0", 6-len(fraction))
	for _, c := range digits {
		if c < '0' || c > '9' {
			return 0, fmt.Errorf("invalid decimal %q", s)
		}
	}
	if negative {
		digits = "-" + digits
	}
	n, err := strconv.ParseInt(digits, 10, 64)
	return Decimal(n), err
}

func (d Decimal) String() string {
	n := big.NewInt(int64(d))
	sign := ""
	if n.Sign() < 0 {
		sign = "-"
		n.Abs(n)
	}
	q, r := new(big.Int), new(big.Int)
	q.QuoRem(n, big.NewInt(int64(Unit)), r)
	fraction := strings.TrimRight(fmt.Sprintf("%06d", r.Int64()), "0")
	if fraction == "" {
		fraction = "0"
	}
	return sign + q.String() + "." + fraction
}

// roundedRatio uses arbitrary-precision intermediates and rejects int64 overflow.
// For statutory amounts an exact half cent rounds down (§ 1 Abs 2 Z 3 MieWeG).
func roundedRatio(a, b, denominator int64, halfUp bool) (int64, error) {
	if denominator <= 0 {
		return 0, fmt.Errorf("denominator must be positive")
	}
	n := new(big.Int).Mul(big.NewInt(a), big.NewInt(b))
	return roundRat(new(big.Rat).SetFrac(n, big.NewInt(denominator)), halfUp)
}

func roundRat(value *big.Rat, halfUp bool) (int64, error) {
	n := new(big.Int).Set(value.Num())
	d := value.Denom()
	negative := n.Sign() < 0
	n.Abs(n)
	q, r := new(big.Int), new(big.Int)
	q.QuoRem(n, d, r)
	cmp := r.Lsh(r, 1).Cmp(d)
	if cmp > 0 || (cmp == 0 && halfUp) {
		q.Add(q, big.NewInt(1))
	}
	if negative {
		q.Neg(q)
	}
	if !q.IsInt64() {
		return 0, fmt.Errorf("calculation exceeds int64")
	}
	return q.Int64(), nil
}

func decimalRat(d Decimal) *big.Rat {
	return new(big.Rat).SetFrac64(int64(d), int64(Unit))
}

// exactCents preserves a curve's integer-ratio state across calls. Public money
// amounts remain integer cents. Only intermediate calculations carry fractions.
func exactCents(s string, cents int64) (*big.Rat, error) {
	if cents < 0 {
		return nil, fmt.Errorf("negative monetary amount")
	}
	if s == "" {
		return new(big.Rat).SetInt64(cents), nil
	}
	n, err := parseExactRatio(s)
	if err != nil {
		return nil, err
	}
	rounded, err := roundRat(n, true)
	if err != nil || rounded != cents {
		return nil, fmt.Errorf("exact amount does not match contractual cents")
	}
	return n, nil
}

func parseExactRatio(s string) (*big.Rat, error) {
	// Accept only integer fractions emitted by RatString. In particular, reject
	// exponent notation before math/big can allocate an enormous power of ten.
	if len(s) == 0 || len(s) > 2048 {
		return nil, fmt.Errorf("invalid exact amount length")
	}
	for _, part := range strings.Split(s, "/") {
		if part == "" {
			return nil, fmt.Errorf("invalid exact amount")
		}
		for _, c := range part {
			if c < '0' || c > '9' {
				return nil, fmt.Errorf("invalid exact amount")
			}
		}
	}
	n, ok := new(big.Rat).SetString(s)
	if !ok || n.Sign() < 0 {
		return nil, fmt.Errorf("invalid exact amount")
	}
	return n, nil
}

func displayDecimal(r *big.Rat) (Decimal, error) {
	n, err := roundRat(new(big.Rat).Mul(r, new(big.Rat).SetInt64(int64(Unit))), true)
	return Decimal(n), err
}

func roundTenth(d Decimal) (Decimal, error) {
	n, err := roundedRatio(int64(d), 1, int64(Unit/10), true)
	if err != nil || n > int64(^uint64(0)>>1)/int64(Unit/10) || n < -int64(^uint64(0)>>1)/int64(Unit/10) {
		return 0, fmt.Errorf("decimal rounding overflow")
	}
	return Decimal(n) * (Unit / 10), nil
}

func percentChange(base, current Decimal) (Decimal, error) {
	if base <= 0 || current <= 0 {
		return 0, fmt.Errorf("index values must be positive")
	}
	// Round directly to tenths of a percent, avoiding double rounding.
	n, err := roundedRatio(int64(current-base), 1000, int64(base), true)
	if err != nil || n > int64(^uint64(0)>>1)/int64(Unit/10) {
		return 0, fmt.Errorf("percentage overflow")
	}
	return Decimal(n) * (Unit / 10), nil
}

func applyPercent(cents int64, rate Decimal, halfUp bool) (int64, error) {
	if cents < 0 || rate < -100*Unit || rate > Decimal(^uint64(0)>>1)-100*Unit {
		return 0, fmt.Errorf("invalid amount or percentage")
	}
	return roundedRatio(cents, int64(100*Unit+rate), int64(100*Unit), halfUp)
}

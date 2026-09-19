package api

import (
	"math/big"
	"strings"
)

// Money is handled as exact rationals, never float64.
//
// PostgreSQL stores these amounts as numeric(14,2) and does the authoritative
// arithmetic during checkout. Anything the API has to work out itself without
// writing - a quoted fee, a preview - uses big.Rat so it agrees with the
// database to the tiyn.

// parseMoney reads a decimal amount into an exact rational.
func parseMoney(amount string) (*big.Rat, bool) {
	value, ok := new(big.Rat).SetString(strings.TrimSpace(amount))
	return value, ok
}

// hundred and half are reused by the rounding below.
var (
	hundred = big.NewRat(100, 1)
	half    = big.NewRat(1, 2)
)

// roundMoney rounds to two decimal places, half away from zero - which is what
// PostgreSQL's round(numeric, 2) does, so the preview and the stored order
// agree to the tiyn.
func roundMoney(value *big.Rat) *big.Rat {
	scaled := new(big.Rat).Mul(value, hundred)

	if scaled.Sign() < 0 {
		scaled.Sub(scaled, half)
	} else {
		scaled.Add(scaled, half)
	}

	// Truncate towards zero after the half-adjustment.
	whole := new(big.Int).Quo(scaled.Num(), scaled.Denom())
	return new(big.Rat).SetFrac(whole, big.NewInt(100))
}

// formatMoney renders an amount the way the API returns money: a plain decimal
// string with exactly two places.
func formatMoney(value *big.Rat) string {
	return roundMoney(value).FloatString(2)
}

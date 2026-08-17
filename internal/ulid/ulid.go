// Package ulid mints and validates the 26-character identifiers the 1.x schema
// uses for tenant identity.
//
// Written rather than vendored: the whole surface is New and Valid, the encoding
// is fixed by Crockford base32, and the target schema already pins the shape
// with a CHECK constraint. A module dependency for thirty lines would be a
// supply-chain entry to maintain for no gain.
//
// Layout is the ULID spec: 48 bits of millisecond timestamp then 80 bits of
// randomness, both big-endian, encoded as 26 base32 characters. Lexical order
// therefore follows creation order, which keeps index locality sane.
package ulid

import (
	"crypto/rand"
	"errors"
	"strings"
	"time"
)

// Crockford base32: no I, L, O or U, so the alphabet cannot produce a string
// that reads as a different one.
const alphabet = "0123456789ABCDEFGHJKMNPQRSTVWXYZ"

// Length is the encoded character count.
const Length = 26

// New returns a fresh identifier. It fails only if the system entropy source
// does; callers should treat that as fatal rather than fall back to a weaker
// source, because these values become primary keys.
func New() (string, error) {
	return newAt(time.Now())
}

func newAt(now time.Time) (string, error) {
	var raw [16]byte
	ms := uint64(now.UTC().UnixMilli())
	raw[0] = byte(ms >> 40)
	raw[1] = byte(ms >> 32)
	raw[2] = byte(ms >> 24)
	raw[3] = byte(ms >> 16)
	raw[4] = byte(ms >> 8)
	raw[5] = byte(ms)
	if _, err := rand.Read(raw[6:]); err != nil {
		return "", errors.New("ulid: system entropy unavailable")
	}
	return encode(raw), nil
}

// encode writes the 128 bits as 26 base32 characters, five bits at a time from
// the most significant end. 26*5 is 130, so the first character carries only the
// top three bits — which is why a valid ULID never starts above '7'.
func encode(raw [16]byte) string {
	var out [Length]byte
	var bitBuffer uint16
	var bitCount uint
	pos := Length
	for i := len(raw) - 1; i >= 0; i-- {
		bitBuffer |= uint16(raw[i]) << bitCount
		bitCount += 8
		for bitCount >= 5 {
			pos--
			out[pos] = alphabet[bitBuffer&0x1f]
			bitBuffer >>= 5
			bitCount -= 5
		}
	}
	if bitCount > 0 && pos > 0 {
		pos--
		out[pos] = alphabet[bitBuffer&0x1f]
	}
	for pos > 0 {
		pos--
		out[pos] = '0'
	}
	return string(out[:])
}

// Valid reports whether value is a well-formed identifier. It matches the CHECK
// constraint in the PostgreSQL target schema exactly; if the two ever disagree,
// rows will be rejected at insert time with no useful message.
func Valid(value string) bool {
	if len(value) != Length || value[0] > '7' {
		return false
	}
	for _, char := range value {
		if !strings.ContainsRune(alphabet, char) {
			return false
		}
	}
	return true
}

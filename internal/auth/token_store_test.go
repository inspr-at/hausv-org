package auth

import (
	"strings"
	"testing"
	"time"
)

func TestTokenStoreInvalidateRemovesUnusedMagicLink(t *testing.T) {
	store := NewTokenStore([]byte(strings.Repeat("s", 32)))
	store.Put("undelivered-token", "owner@example.com", "demo", 15*time.Minute)

	if _, _, _, ok := store.Peek("undelivered-token"); !ok {
		t.Fatal("new token is not readable")
	}
	store.Invalidate("undelivered-token")
	if _, _, _, ok := store.Peek("undelivered-token"); ok {
		t.Fatal("invalidated token remained readable")
	}

	// Delivery failure and shutdown can race to invalidate the same token.
	store.Invalidate("undelivered-token")
}

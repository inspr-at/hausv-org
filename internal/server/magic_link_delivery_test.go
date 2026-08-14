package server

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

type gatedMagicLinkMailer struct {
	started chan string
	release chan struct{}

	mu   sync.Mutex
	sent []sentMagicLink
}

func newGatedMagicLinkMailer() *gatedMagicLinkMailer {
	return &gatedMagicLinkMailer{
		started: make(chan string, 8),
		release: make(chan struct{}),
	}
}

func (m *gatedMagicLinkMailer) SendMagicLink(to string, link string, _ string) error {
	m.started <- to
	<-m.release
	m.mu.Lock()
	m.sent = append(m.sent, sentMagicLink{To: to, Link: link})
	m.mu.Unlock()
	return nil
}

func (m *gatedMagicLinkMailer) SendInvite(string, string, string) error {
	return nil
}

func (m *gatedMagicLinkMailer) SendNotification(string, string, string) error {
	return nil
}

func (m *gatedMagicLinkMailer) Configured() bool {
	return true
}

func (m *gatedMagicLinkMailer) sentLinks() []sentMagicLink {
	m.mu.Lock()
	defer m.mu.Unlock()
	return append([]sentMagicLink(nil), m.sent...)
}

type failingMagicLinkMailer struct {
	err error
}

func (m failingMagicLinkMailer) SendMagicLink(string, string, string) error {
	return m.err
}

func (failingMagicLinkMailer) SendInvite(string, string, string) error {
	return nil
}

func (failingMagicLinkMailer) SendNotification(string, string, string) error {
	return nil
}

func (failingMagicLinkMailer) Configured() bool {
	return true
}

type contextBlockedMagicLinkMailer struct {
	started  chan struct{}
	canceled chan struct{}
	once     sync.Once
}

func newContextBlockedMagicLinkMailer() *contextBlockedMagicLinkMailer {
	return &contextBlockedMagicLinkMailer{
		started:  make(chan struct{}),
		canceled: make(chan struct{}),
	}
}

func (*contextBlockedMagicLinkMailer) SendMagicLink(string, string, string) error {
	return errors.New("context-aware path was not used")
}

func (m *contextBlockedMagicLinkMailer) SendMagicLinkContext(ctx context.Context, _, _, _ string) error {
	m.once.Do(func() { close(m.started) })
	<-ctx.Done()
	close(m.canceled)
	return ctx.Err()
}

func (*contextBlockedMagicLinkMailer) SendInvite(string, string, string) error {
	return nil
}

func (*contextBlockedMagicLinkMailer) SendNotification(string, string, string) error {
	return nil
}

func (*contextBlockedMagicLinkMailer) Configured() bool {
	return true
}

func TestMagicLinkRequestDoesNotWaitForSlowMailer(t *testing.T) {
	a := newTestPortalApp(t, userProfile{
		Email:       "owner@example.com",
		Role:        roleOwner,
		Tenants:     []string{"demo"},
		AuthMethods: defaultAuthMethods(),
	})
	mailer := newGatedMagicLinkMailer()
	a.mailer = mailer
	a.magicLinkDelivery = newMagicLinkDeliveryQueue(2, 1)

	type result struct {
		responseCode int
		location     string
		body         string
	}
	knownResult := make(chan result, 1)
	go func() {
		response := requestMagicLink(t, a, "owner@example.com", "203.0.113.80:1234")
		knownResult <- result{
			responseCode: response.Code,
			location:     response.Header().Get("Location"),
			body:         response.Body.String(),
		}
	}()

	var known result
	select {
	case known = <-knownResult:
	case <-time.After(250 * time.Millisecond):
		close(mailer.release)
		t.Fatal("known-account request waited for the deliberately blocked mail transport")
	}
	select {
	case <-mailer.started:
	case <-time.After(time.Second):
		close(mailer.release)
		t.Fatal("queued magic-link delivery did not reach the worker")
	}

	unknown := requestMagicLink(t, a, "unknown@example.com", "203.0.113.81:1234")
	if known.responseCode != http.StatusSeeOther || unknown.Code != known.responseCode {
		t.Fatalf("known/unknown statuses = %d/%d", known.responseCode, unknown.Code)
	}
	if known.location != "/demo/?sent=1" || unknown.Header().Get("Location") != known.location {
		t.Fatalf("known/unknown locations = %q/%q", known.location, unknown.Header().Get("Location"))
	}
	if known.body != unknown.Body.String() {
		t.Fatalf("known/unknown response bodies differ: %q / %q", known.body, unknown.Body.String())
	}

	close(mailer.release)
	a.closeMagicLinkDelivery()
	if links := mailer.sentLinks(); len(links) != 1 || links[0].To != "owner@example.com" {
		t.Fatalf("delivered magic links = %+v", links)
	}
}

func TestMagicLinkDeliveryQueueIsBoundedAndDrainsOnClose(t *testing.T) {
	a := newTestPortalApp(t, userProfile{
		Email:       "first@example.com",
		Role:        roleOwner,
		Tenants:     []string{"demo"},
		AuthMethods: defaultAuthMethods(),
	})
	for _, email := range []string{"second@example.com", "third@example.com"} {
		a.profiles[email] = userProfile{
			Email:       email,
			Role:        roleOwner,
			Tenants:     []string{"demo"},
			AuthMethods: defaultAuthMethods(),
		}
	}
	mailer := newGatedMagicLinkMailer()
	a.mailer = mailer
	a.magicLinkDelivery = newMagicLinkDeliveryQueue(1, 1)

	first := requestMagicLink(t, a, "first@example.com", "203.0.113.90:1234")
	if first.Code != http.StatusSeeOther {
		t.Fatalf("first request status = %d", first.Code)
	}
	select {
	case <-mailer.started:
	case <-time.After(time.Second):
		close(mailer.release)
		t.Fatal("first delivery did not occupy the worker")
	}
	second := requestMagicLink(t, a, "second@example.com", "203.0.113.91:1234")
	third := requestMagicLink(t, a, "third@example.com", "203.0.113.92:1234")
	for name, response := range map[string]*httptest.ResponseRecorder{
		"second": second,
		"third":  third,
	} {
		if response.Code != http.StatusSeeOther || response.Header().Get("Location") != "/demo/?sent=1" {
			t.Fatalf("%s request = status %d location %q", name, response.Code, response.Header().Get("Location"))
		}
	}

	closeFinished := make(chan struct{})
	go func() {
		a.closeMagicLinkDelivery()
		close(closeFinished)
	}()
	select {
	case <-closeFinished:
		t.Fatal("shutdown returned before accepted deliveries were drained")
	case <-time.After(25 * time.Millisecond):
	}
	close(mailer.release)
	select {
	case <-closeFinished:
	case <-time.After(time.Second):
		t.Fatal("shutdown did not finish after the mail transport was released")
	}

	links := mailer.sentLinks()
	if len(links) != 2 {
		t.Fatalf("delivered %d magic links, want exactly worker + bounded buffer: %+v", len(links), links)
	}
	if links[0].To != "first@example.com" || links[1].To != "second@example.com" {
		t.Fatalf("delivery order = %+v", links)
	}
	if a.enqueueMagicLinkDelivery(magicLinkDeliveryJob{
		mailer:  mailer,
		to:      "after-close@example.com",
		link:    "secret-after-close",
		address: "after close",
	}) {
		t.Fatal("closed application accepted another delivery")
	}
}

func TestMagicLinkDeliveryShutdownIsBoundedAndInvalidatesUndeliveredTokens(t *testing.T) {
	a := newTestPortalApp(t, userProfile{
		Email:       "owner@example.com",
		Role:        roleOwner,
		Tenants:     []string{"demo"},
		AuthMethods: defaultAuthMethods(),
	})
	mailer := newContextBlockedMagicLinkMailer()
	queue := newMagicLinkDeliveryQueue(2, 1)
	a.magicLinkDelivery = queue

	for _, token := range []string{"active-token", "pending-token"} {
		a.tokens.Put(token, "owner@example.com", "demo", 15*time.Minute)
		token := token
		if !queue.enqueue(magicLinkDeliveryJob{
			mailer:  mailer,
			to:      "owner@example.com",
			link:    "https://example.test/auth/verify?token=" + token,
			address: "Private Address 7",
			invalidate: func() {
				a.tokens.Invalidate(token)
			},
		}) {
			t.Fatalf("delivery for %q was not queued", token)
		}
	}

	select {
	case <-mailer.started:
	case <-time.After(time.Second):
		t.Fatal("hanging delivery did not reach the fixed worker")
	}

	started := time.Now()
	drained := queue.close(40*time.Millisecond, 250*time.Millisecond)
	elapsed := time.Since(started)
	if drained {
		t.Fatal("queue reported a graceful drain despite the hanging transport")
	}
	if elapsed > 500*time.Millisecond {
		t.Fatalf("bounded queue shutdown took %s, want below 500ms", elapsed)
	}
	select {
	case <-mailer.canceled:
	case <-time.After(500 * time.Millisecond):
		t.Fatal("queue shutdown did not cancel the active transport")
	}
	select {
	case <-queue.workersDone:
	case <-time.After(500 * time.Millisecond):
		t.Fatal("fixed worker set did not exit after cancellation")
	}
	for _, token := range []string{"active-token", "pending-token"} {
		if _, _, _, ok := a.tokens.Peek(token); ok {
			t.Fatalf("undelivered token %q remained usable", token)
		}
	}
	if queue.enqueue(magicLinkDeliveryJob{
		mailer:  mailer,
		to:      "after-close@example.com",
		link:    "secret-after-close",
		address: "after close",
	}) {
		t.Fatal("closed queue accepted another delivery")
	}
	// The queue was closed directly with test-sized limits. Keep the app's
	// normal cleanup from closing the same deliberately non-graceful queue a
	// second time and emitting the production deadline warning.
	a.magicLinkDeliveryMu.Lock()
	a.magicLinkDelivery = nil
	a.magicLinkDeliveryMu.Unlock()
}

func TestMagicLinkDeliveryLogsDoNotContainMessageOrRecipientData(t *testing.T) {
	previousLogger := slog.Default()
	var output bytes.Buffer
	slog.SetDefault(slog.New(slog.NewJSONHandler(&output, nil)))
	t.Cleanup(func() {
		slog.SetDefault(previousLogger)
	})

	deliverMagicLink(context.Background(), magicLinkDeliveryJob{
		mailer: failingMagicLinkMailer{
			err: errors.New("relay rejected owner@example.com token-secret"),
		},
		to:      "owner@example.com",
		link:    "https://example.com/auth/verify?token=token-secret",
		address: "Private Address 7",
	})

	logged := output.String()
	for _, forbidden := range []string{
		"owner@example.com",
		"token-secret",
		"Private Address 7",
		"relay rejected",
	} {
		if strings.Contains(logged, forbidden) {
			t.Fatalf("delivery log leaked %q: %s", forbidden, logged)
		}
	}
	if !strings.Contains(logged, "magic link delivery failed") {
		t.Fatalf("delivery failure was not logged generically: %s", logged)
	}
}

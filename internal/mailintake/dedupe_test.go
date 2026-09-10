package mailintake

import (
	"strings"
	"testing"
	"time"
)

func TestDedupeKeyUsesMessageIDOrStableContent(t *testing.T) {
	base := Message{UID: 1, FromEmail: "Alina@Example.com", Subject: "Beleg", Text: "Text\r\nZeile 2", Date: time.Date(2026, 9, 9, 8, 0, 0, 0, time.UTC)}
	key := base.DedupeKey()
	if !strings.HasPrefix(key, "sha256:") || len(key) != len("sha256:")+64 {
		t.Fatalf("fallback is not a SHA-256 key: %q", key)
	}
	for _, test := range []struct {
		name   string
		change func(*Message)
		same   bool
	}{
		{"UID changes", func(m *Message) { m.UID = 900 }, true},
		{"sender casing", func(m *Message) { m.FromEmail = " alina@example.com " }, true},
		{"date timezone", func(m *Message) { m.Date = m.Date.In(time.FixedZone("CEST", 7200)) }, true},
		{"line endings", func(m *Message) { m.Text = "Text\nZeile 2" }, true},
		{"sender differs", func(m *Message) { m.FromEmail = "other@example.com" }, false},
		{"subject differs", func(m *Message) { m.Subject += " 2" }, false},
		{"date differs", func(m *Message) { m.Date = m.Date.Add(time.Second) }, false},
		{"body differs", func(m *Message) { m.Text += " 2" }, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			message := base
			test.change(&message)
			if got := message.DedupeKey(); (got == key) != test.same {
				t.Fatalf("same key=%v, want %v", got == key, test.same)
			}
		})
	}
	base.MessageID = " \t<ABC@example.com>\r\n"
	if got := base.DedupeKey(); got != "ABC@example.com" {
		t.Fatalf("Message-ID must retain its existing ledger format: %q", got)
	}
	base = Message{} // Missing Date must never be replaced by the fetch time.
	base.UID = 1
	key = base.DedupeKey()
	base.UID = 2
	if base.DedupeKey() != key {
		t.Fatal("undated mail changed identity")
	}
}

func TestDedupeKeyKeepsThePreviouslyStoredParsedMessageID(t *testing.T) {
	raw := "From: alina@example.com\r\nSubject: Beleg\r\nMessage-ID: <ABC-123@example.com>\r\n\r\nText"
	message, err := Parse([]byte(raw), testLimits)
	if err != nil {
		t.Fatal(err)
	}
	// Before HAUSV-724 the poller passed Parse's MessageID to Record, whose
	// normalizeMessageID already stripped surrounding whitespace/brackets.
	const previouslyStored = "ABC-123@example.com"
	if message.MessageID != previouslyStored || message.DedupeKey() != previouslyStored {
		t.Fatalf("existing Message-ID changed: parsed=%q key=%q", message.MessageID, message.DedupeKey())
	}
}

func TestDedupeKeyCoversBodyBeyondIntakeLimits(t *testing.T) {
	header := "From: alina@example.com\r\nSubject: Beleg\r\nContent-Type: text/plain\r\n\r\n"
	first, err := Parse([]byte(header+strings.Repeat("x", 5000)+"first"), testLimits)
	if err != nil {
		t.Fatal(err)
	}
	second, err := Parse([]byte(header+strings.Repeat("x", 5000)+"second"), testLimits)
	if err != nil {
		t.Fatal(err)
	}
	if first.Text != second.Text || first.DedupeKey() == second.DedupeKey() {
		t.Fatal("messages with the same displayed prefix must retain distinct full-body keys")
	}
	// Upload limits and CRLF/LF transport changes must not change identity.
	refetched, err := Parse([]byte(strings.ReplaceAll(header+strings.Repeat("x", 5000)+"first", "\r\n", "\n")), Limits{MaxTextBytes: 10})
	if err != nil || refetched.DedupeKey() != first.DedupeKey() {
		t.Fatalf("identity changed with limits/line endings: err=%v", err)
	}
	for _, data := range []string{"attachment-one", "attachment-two"} {
		raw := "From: alina@example.com\r\nSubject: Beleg\r\nContent-Type: application/pdf\r\nContent-Disposition: attachment; filename=beleg.pdf\r\n\r\n" + data
		message, err := Parse([]byte(raw), Limits{})
		if err != nil || message.Dropped != 1 {
			t.Fatalf("dropped attachment: err=%v count=%d", err, message.Dropped)
		}
		if data == "attachment-one" {
			first = message
		} else if first.DedupeKey() == message.DedupeKey() {
			t.Fatal("different dropped attachments collapsed into one message")
		}
	}
}

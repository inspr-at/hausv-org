package mail

import (
	"bytes"
	"context"
	"encoding/base64"
	"io"
	"mime"
	"mime/multipart"
	"net/mail"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"
)

func TestDocumentMessageMIME(t *testing.T) {
	input := bytes.Repeat([]byte{0, 1, 2, 255, 13, 10}, 100)
	subject := "Jahresabrechnung 2025 · Schönbrunn · Büro 2"
	before := time.Now().UTC().Truncate(time.Second)
	raw, err := documentMessage("Verwaltung Schönbrunn <verwaltung@example.com>", "partei@example.com", subject, "Guten Tag Groß,\nEntwurf zur Prüfung — keine Rechtsauskunft nach WEG/MRG", Attachment{Filename: "Abrechnung für Büro.pdf", ContentType: "application/pdf", Data: input})
	if err != nil {
		t.Fatal(err)
	}
	msg, err := mail.ReadMessage(bytes.NewReader(raw))
	if err != nil {
		t.Fatal(err)
	}
	sender, err := msg.Header.AddressList("From")
	if err != nil || len(sender) != 1 || sender[0].Name != "Verwaltung Schönbrunn" || sender[0].Address != "verwaltung@example.com" {
		t.Fatal("sender did not round-trip", sender, err)
	}
	if strings.Contains(msg.Header.Get("From"), "Schönbrunn") {
		t.Fatal("non-ASCII sender name was not encoded")
	}
	date, err := time.Parse(time.RFC1123Z, msg.Header.Get("Date"))
	if err != nil || date.Before(before) || date.After(time.Now().UTC()) {
		t.Fatal("invalid message date", date, err)
	}
	messageID := msg.Header.Get("Message-ID")
	if !regexp.MustCompile(`^<[0-9a-f]{32}@example\.com>$`).MatchString(messageID) {
		t.Fatal("invalid message ID", messageID)
	}
	another, err := documentMessage("verwaltung@example.com", "partei@example.com", subject, "", Attachment{Filename: "a.pdf", ContentType: "application/pdf"})
	if err != nil {
		t.Fatal(err)
	}
	second, err := mail.ReadMessage(bytes.NewReader(another))
	if err != nil || second.Header.Get("Message-ID") == messageID {
		t.Fatal("message ID reused", err)
	}
	decoded, err := new(mime.WordDecoder).DecodeHeader(msg.Header.Get("Subject"))
	if err != nil || decoded != subject {
		t.Fatal(decoded, err)
	}
	kind, params, err := mime.ParseMediaType(msg.Header.Get("Content-Type"))
	if err != nil || kind != "multipart/mixed" {
		t.Fatal(kind, err)
	}
	reader := multipart.NewReader(msg.Body, params["boundary"])
	body, err := reader.NextPart()
	if err != nil {
		t.Fatal(err)
	}
	text, err := io.ReadAll(body)
	if err != nil || !bytes.Contains(text, []byte("Entwurf zur Prüfung")) {
		t.Fatal(string(text), err)
	}
	part, err := reader.NextPart()
	if err != nil {
		t.Fatal(err)
	}
	disposition, params, err := mime.ParseMediaType(part.Header.Get("Content-Disposition"))
	if err != nil || disposition != "attachment" || params["filename"] != "Abrechnung für Büro.pdf" {
		t.Fatal(params, err)
	}
	if part.Header.Get("Content-Type") != "application/pdf" || part.Header.Get("Content-Transfer-Encoding") != "base64" {
		t.Fatal(part.Header)
	}
	encoded, err := io.ReadAll(part)
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSuffix(string(encoded), "\r\n"), "\r\n")
	encodedLength := base64.StdEncoding.EncodedLen(len(input))
	if len(lines) != (encodedLength+75)/76 {
		t.Fatal("wrong attachment line count", len(lines))
	}
	for i, line := range lines {
		if want := min(76, encodedLength-i*76); len(line) != want {
			t.Fatalf("attachment line %d: length %d, want %d", i, len(line), want)
		}
	}
	got, err := io.ReadAll(base64.NewDecoder(base64.StdEncoding, bytes.NewReader(encoded)))
	if err != nil || !bytes.Equal(got, input) {
		t.Fatal("attachment differs", err)
	}
	if _, err := reader.NextPart(); err != io.EOF {
		t.Fatal("extra part", err)
	}
	for _, line := range strings.Split(string(raw), "\r\n") {
		if len(line) > 998 {
			t.Fatal("overlong MIME line")
		}
	}
	for _, bad := range []string{"x\r\nBcc: victim@example.com", "x\ny"} {
		if _, err := documentMessage("a@example.com", "b@example.com", bad, "", Attachment{Filename: "a.pdf", ContentType: "application/pdf"}); err == nil {
			t.Fatal("header injection accepted")
		}
	}
}

func TestOutboxAllMessages(t *testing.T) {
	directory := filepath.Join(t.TempDir(), "outbox")
	m := NewSMTP("invalid.example", "25", "", "", "").WithOutbox(directory)
	if !m.Configured() || m.Validate() != nil || m.Mode() != "Postausgang als Datei — Testmodus" {
		t.Fatal("outbox not configured")
	}
	calls := []func() error{
		func() error { return m.SendMagicLink("a@example.com", "https://example.com/login", "Haus") },
		func() error {
			return m.SendMagicLinkContext(t.Context(), "a@example.com", "https://example.com/login", "Haus")
		},
		func() error { return m.SendHomeConfirmation("a@example.com", "https://example.com/confirm", "Haus") },
		func() error {
			return m.SendHomeConfirmationContext(t.Context(), "a@example.com", "https://example.com/confirm", "Haus")
		},
		func() error { return m.SendInvite("a@example.com", "https://example.com/login", "Haus") },
		func() error { return m.SendNotification("a@example.com", "Nachricht", "Inhalt") },
		func() error {
			return m.SendDocument(t.Context(), "a@example.com", "Abrechnung", "Inhalt", Attachment{Filename: "a.pdf", ContentType: "application/pdf", Data: []byte("%PDF")})
		},
	}
	for _, call := range calls {
		if err := call(); err != nil {
			t.Fatal(err)
		}
	}
	entries, err := os.ReadDir(directory)
	if err != nil || len(entries) != len(calls) {
		t.Fatal(len(entries), err)
	}
	info, _ := os.Stat(directory)
	if info.Mode().Perm() != 0700 {
		t.Fatal(info.Mode())
	}
	for _, entry := range entries {
		if !strings.HasSuffix(entry.Name(), ".eml") {
			t.Fatal(entry.Name())
		}
		info, _ := entry.Info()
		if info.Mode().Perm() != 0600 {
			t.Fatal(info.Mode())
		}
		raw, err := os.ReadFile(filepath.Join(directory, entry.Name()))
		if err != nil {
			t.Fatal(err)
		}
		if _, err := mail.ReadMessage(bytes.NewReader(raw)); err != nil {
			t.Fatal(err)
		}
	}
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	if err := m.SendMagicLinkContext(ctx, "a@example.com", "link", "Haus"); err == nil {
		t.Fatal("canceled mail sent")
	}
	after, _ := os.ReadDir(directory)
	if len(after) != len(entries) {
		t.Fatal("canceled mail wrote a file")
	}
	blocked := filepath.Join(t.TempDir(), "file")
	if err := os.WriteFile(blocked, []byte("x"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := m.WithOutbox(blocked).SendInvite("a@example.com", "link", "Haus"); err == nil {
		t.Fatal("outbox failure ignored")
	}
}

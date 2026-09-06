package mailintake

import (
	"bytes"
	"context"
	"log"
	"net"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/imapserver"
	"github.com/emersion/go-imap/v2/imapserver/imapmemserver"
)

type literal struct {
	*bytes.Reader
}

func (l literal) Size() int64 { return int64(l.Len()) }

// startMailbox runs an in-memory IMAP server with one user and the given raw
// messages in INBOX. It is a real TCP server, so the fetcher is exercised the
// way production would, minus TLS.
func startMailbox(t *testing.T, messages ...string) (Config, string) {
	t.Helper()
	memory := imapmemserver.New()
	user := imapmemserver.NewUser("post@musterstadt.example", "geheim")
	if err := user.Create("INBOX", nil); err != nil && !strings.Contains(err.Error(), "already exists") {
		t.Fatalf("create INBOX: %v", err)
	}
	for _, raw := range messages {
		if _, err := user.Append("INBOX", literal{bytes.NewReader([]byte(raw))}, &imap.AppendOptions{Time: time.Now()}); err != nil {
			t.Fatalf("append: %v", err)
		}
	}
	memory.AddUser(user)

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	options := &imapserver.Options{
		NewSession: func(*imapserver.Conn) (imapserver.Session, *imapserver.GreetingData, error) {
			return memory.NewSession(), nil, nil
		},
		InsecureAuth: true,
	}
	if os.Getenv("MAILINTAKE_DEBUG") != "" {
		options.Logger = log.New(os.Stderr, "imapserver: ", 0)
		options.DebugWriter = os.Stderr
	}
	server := imapserver.New(options)
	go func() { _ = server.Serve(listener) }()
	t.Cleanup(func() { _ = server.Close() })

	host, port, _ := net.SplitHostPort(listener.Addr().String())
	var portNumber int
	for _, digit := range port {
		portNumber = portNumber*10 + int(digit-'0')
	}
	return Config{Organisation: "musterstadt", Host: host, Port: portNumber, Username: "post@musterstadt.example", Folder: "INBOX", Insecure: true}, "geheim"
}

func rawMail(id, from, subject, body string) string {
	return strings.Join([]string{
		"From: " + from,
		"To: post@musterstadt.example",
		"Subject: " + subject,
		"Message-ID: <" + id + ">",
		"Date: Fri, 05 Sep 2026 09:15:00 +0200",
		"Content-Type: text/plain; charset=utf-8",
		"",
		body,
		"",
	}, "\r\n")
}

func TestFetcherReadsUnreadAndMarksSeenOnlyWhenAsked(t *testing.T) {
	config, password := startMailbox(t,
		rawMail("one@example.com", "Alina Auer <alina@example.com>", "Erste", "Hallo eins"),
		rawMail("two@example.com", "Matthias Dorn <matthias@example.com>", "Zweite", "Hallo zwei"),
	)
	fetcher := Fetcher{Config: config, Password: password, Limits: testLimits, DialTimeout: 5 * time.Second}
	ctx := context.Background()

	messages, err := fetcher.Unread(ctx, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(messages) != 2 {
		t.Fatalf("got %d messages, want 2", len(messages))
	}
	if messages[0].MessageID != "one@example.com" || messages[1].FromEmail != "matthias@example.com" {
		t.Fatalf("messages = %+v", messages)
	}
	if messages[0].UID == 0 {
		t.Fatal("UID missing")
	}

	// A read with Peek must not mark anything: the same two come back.
	again, err := fetcher.Unread(ctx, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(again) != 2 {
		t.Fatalf("peek changed flags: %d unread left", len(again))
	}

	if err := fetcher.MarkSeen(ctx, []uint32{messages[0].UID}); err != nil {
		t.Fatal(err)
	}
	rest, err := fetcher.Unread(ctx, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(rest) != 1 || rest[0].MessageID != "two@example.com" {
		t.Fatalf("after marking one seen: %+v", rest)
	}
}

func TestFetcherRefusesWrongPasswordWithoutEchoingIt(t *testing.T) {
	config, _ := startMailbox(t)
	fetcher := Fetcher{Config: config, Password: "falsch-und-geheim", Limits: testLimits, DialTimeout: 5 * time.Second}
	_, err := fetcher.Unread(context.Background(), 10)
	if err == nil {
		t.Fatal("wrong password was accepted")
	}
	if strings.Contains(err.Error(), "falsch-und-geheim") || strings.Contains(err.Error(), config.Username) {
		t.Fatalf("error leaks credentials: %v", err)
	}
}

func TestFetcherHonoursTheBatchLimit(t *testing.T) {
	config, password := startMailbox(t,
		rawMail("a@example.com", "a@example.com", "A", "a"),
		rawMail("b@example.com", "b@example.com", "B", "b"),
		rawMail("c@example.com", "c@example.com", "C", "c"),
	)
	fetcher := Fetcher{Config: config, Password: password, Limits: testLimits, DialTimeout: 5 * time.Second}
	messages, err := fetcher.Unread(context.Background(), 2)
	if err != nil {
		t.Fatal(err)
	}
	if len(messages) != 2 {
		t.Fatalf("batch limit ignored: %d", len(messages))
	}
}

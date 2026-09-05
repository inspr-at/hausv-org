package main

import (
	"bytes"
	"context"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/inspr-at/hausv-org/internal/mailintake"
)

func writeEML(t *testing.T, dir, name, id, subject string) {
	t.Helper()
	raw := strings.Join([]string{
		"From: Alina Auer <alina.eigentuemer@musterstadt.example>",
		"To: post@musterstadt.example",
		"Subject: " + subject,
		"Message-ID: <" + id + ">",
		"Date: Fri, 05 Sep 2026 09:15:00 +0200",
		"Content-Type: text/plain; charset=utf-8",
		"",
		"Text " + subject,
		"",
	}, "\r\n")
	if err := os.WriteFile(filepath.Join(dir, name), []byte(raw), 0o600); err != nil {
		t.Fatal(err)
	}
}

func freePort(t *testing.T) string {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	addr := listener.Addr().String()
	_ = listener.Close()
	return addr
}

func demoMailboxEnv(user, password string) func(string) string {
	return func(key string) string {
		switch key {
		case "DEMO_MAILBOX_USER":
			return user
		case "DEMO_MAILBOX_PASSWORD":
			return password
		}
		return ""
	}
}

func TestDemoMailboxServesSeedMailsAndPicksUpNewFiles(t *testing.T) {
	dir := t.TempDir()
	writeEML(t, dir, "01-beleg.eml", "one@demo", "Beleg")
	writeEML(t, dir, "02-heizung.eml", "two@demo", "Heizung")
	addr := freePort(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var stdout, stderr bytes.Buffer
	done := make(chan error, 1)
	go func() {
		done <- runDemoMailbox(ctx, []string{"-dir", dir, "-addr", addr, "-rescan", "150ms"}, &stdout, &stderr, demoMailboxEnv("post@musterstadt.example", "demo-postfach"))
	}()
	host, port, _ := net.SplitHostPort(addr)
	portNumber := 0
	for _, digit := range port {
		portNumber = portNumber*10 + int(digit-'0')
	}
	config := mailintake.Config{Organisation: "musterstadt", Host: host, Port: portNumber, Username: "post@musterstadt.example", Folder: "INBOX", Insecure: true}
	fetcher := mailintake.Fetcher{Config: config, Password: "demo-postfach", Limits: mailintake.Limits{MaxAttachmentBytes: 1 << 20, MaxAttachments: 5, MaxTextBytes: 1 << 16}, DialTimeout: 3 * time.Second}

	// The server needs a moment to listen; poll instead of sleeping blindly.
	var messages []mailintake.Message
	var err error
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		messages, err = fetcher.Unread(ctx, 10)
		if err == nil {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	if err != nil {
		t.Fatalf("mailbox not reachable: %v\nstdout: %s\nstderr: %s", err, stdout.String(), stderr.String())
	}
	if len(messages) != 2 || messages[0].Subject != "Beleg" || messages[1].Subject != "Heizung" {
		t.Fatalf("seed mails = %+v", messages)
	}

	// A file dropped in later is a new unread mail within a rescan.
	writeEML(t, dir, "03-neu.eml", "three@demo", "Neu")
	deadline = time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		messages, err = fetcher.Unread(ctx, 10)
		if err == nil && len(messages) == 3 {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	if err != nil || len(messages) != 3 || messages[2].Subject != "Neu" {
		t.Fatalf("dropped file not picked up: err=%v messages=%+v", err, messages)
	}

	// Marking read on the server sticks, and the file is not appended twice.
	if err := fetcher.MarkSeen(ctx, []uint32{messages[0].UID}); err != nil {
		t.Fatal(err)
	}
	time.Sleep(400 * time.Millisecond)
	rest, err := fetcher.Unread(ctx, 10)
	if err != nil || len(rest) != 2 {
		t.Fatalf("after MarkSeen: err=%v unread=%d (want 2)", err, len(rest))
	}

	if _, err := (mailintake.Fetcher{Config: config, Password: "falsch", Limits: fetcher.Limits, DialTimeout: 3 * time.Second}).Unread(ctx, 10); err == nil {
		t.Fatal("wrong password accepted")
	}
	cancel()
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("mailbox exited with %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("mailbox did not stop on context cancel")
	}
}

func TestDemoMailboxRequiresAccountAndDirectory(t *testing.T) {
	ctx := context.Background()
	var out bytes.Buffer
	if err := runDemoMailbox(ctx, []string{"-dir", t.TempDir()}, &out, &out, demoMailboxEnv("", "")); err == nil {
		t.Fatal("started without an account")
	}
	if err := runDemoMailbox(ctx, nil, &out, &out, demoMailboxEnv("u", "p")); err == nil {
		t.Fatal("started without a directory")
	}
}

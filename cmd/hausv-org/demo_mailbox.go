package main

import (
	"bytes"
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/imapserver"
	"github.com/emersion/go-imap/v2/imapserver/imapmemserver"
)

// demo-mailbox is the demo's own post office: a plain IMAP server that holds
// the seed mails unread and takes any `.eml` dropped into its directory as a
// new, unread message. It exists so the demo can show mail arriving without a
// real mailbox; it speaks plain IMAP and must only ever listen inside the
// compose network.
//
//	hausv-org demo-mailbox -dir /seed/mail -addr :143 [-rescan 15s]
//
// DEMO_MAILBOX_USER and DEMO_MAILBOX_PASSWORD name the single account. They
// are fixture values, not credentials for anything outside the demo network.

type demoMailbox struct {
	user     *imapmemserver.User
	dir      string
	mu       sync.Mutex
	loaded   map[string]struct{}
	appended int
	stdout   io.Writer
}

type emlLiteral struct{ *bytes.Reader }

func (l emlLiteral) Size() int64 { return int64(l.Len()) }

func runDemoMailbox(ctx context.Context, args []string, stdout, stderr io.Writer, getenv func(string) string) error {
	flags := flag.NewFlagSet("demo-mailbox", flag.ContinueOnError)
	flags.SetOutput(stderr)
	dir := flags.String("dir", "", "directory of .eml files to hold unread in INBOX")
	addr := flags.String("addr", ":143", "listen address (compose network only, plain IMAP)")
	rescan := flags.Duration("rescan", 15*time.Second, "how often to pick up new .eml files")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() != 0 || strings.TrimSpace(*dir) == "" {
		return errors.New("usage: hausv-org demo-mailbox -dir PATH [-addr :143] [-rescan 15s]")
	}
	username := strings.TrimSpace(getenv("DEMO_MAILBOX_USER"))
	password := strings.TrimSpace(getenv("DEMO_MAILBOX_PASSWORD"))
	if username == "" || password == "" {
		return errors.New("DEMO_MAILBOX_USER and DEMO_MAILBOX_PASSWORD are required")
	}

	memory := imapmemserver.New()
	user := imapmemserver.NewUser(username, password)
	if err := user.Create("INBOX", nil); err != nil && !strings.Contains(err.Error(), "already exists") {
		return fmt.Errorf("create INBOX: %w", err)
	}
	memory.AddUser(user)
	box := &demoMailbox{user: user, dir: *dir, loaded: map[string]struct{}{}, stdout: stdout}
	added, err := box.load()
	if err != nil {
		return err
	}
	fmt.Fprintf(stdout, "demo-mailbox: %d Mails aus %s in INBOX, Konto %s\n", added, *dir, username)

	listener, err := net.Listen("tcp", *addr)
	if err != nil {
		return fmt.Errorf("listen %s: %w", *addr, err)
	}
	server := imapserver.New(&imapserver.Options{
		NewSession: func(*imapserver.Conn) (imapserver.Session, *imapserver.GreetingData, error) {
			return memory.NewSession(), nil, nil
		},
		InsecureAuth: true,
	})
	go func() {
		ticker := time.NewTicker(*rescan)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if n, err := box.load(); err != nil {
					fmt.Fprintf(stderr, "demo-mailbox: rescan failed: %v\n", err)
				} else if n > 0 {
					fmt.Fprintf(stdout, "demo-mailbox: %d neue Mails aufgenommen\n", n)
				}
			}
		}
	}()
	go func() {
		<-ctx.Done()
		_ = server.Close()
	}()
	fmt.Fprintf(stdout, "demo-mailbox: IMAP auf %s\n", listener.Addr())
	if err := server.Serve(listener); err != nil && ctx.Err() == nil {
		return err
	}
	return nil
}

// load appends every .eml not yet seen, in name order, as an unread message.
// A file is remembered by name, so editing a mail in place does not duplicate
// it; a new mail is a new file.
func (b *demoMailbox) load() (int, error) {
	entries, err := os.ReadDir(b.dir)
	if err != nil {
		return 0, fmt.Errorf("read %s: %w", b.dir, err)
	}
	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(strings.ToLower(entry.Name()), ".eml") || strings.HasPrefix(entry.Name(), ".") {
			continue
		}
		names = append(names, entry.Name())
	}
	sort.Strings(names)
	b.mu.Lock()
	defer b.mu.Unlock()
	added := 0
	for _, name := range names {
		if _, done := b.loaded[name]; done {
			continue
		}
		raw, err := os.ReadFile(filepath.Join(b.dir, name))
		if err != nil {
			return added, err
		}
		if _, err := b.user.Append("INBOX", emlLiteral{bytes.NewReader(raw)}, &imap.AppendOptions{Time: time.Now()}); err != nil {
			return added, fmt.Errorf("append %s: %w", name, err)
		}
		b.loaded[name] = struct{}{}
		b.appended++
		added++
	}
	return added, nil
}

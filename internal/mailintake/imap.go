package mailintake

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"time"

	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/imapclient"
)

// Fetcher reads unread messages from one mailbox. Every call opens its own
// connection: a Verwaltung polls every few minutes, and a connection held
// across that gap is the kind of thing a mail server closes without asking.
type Fetcher struct {
	Config   Config
	Password string
	Limits   Limits
	// DialTimeout bounds connect and login; a mailbox that does not answer
	// must not stall the poller.
	DialTimeout time.Duration
}

// Unread returns the unread messages in the configured folder, oldest first,
// without changing their flags. Marking happens only after the caller has
// safely stored a message, so a crash between the two never loses mail.
func (f Fetcher) Unread(ctx context.Context, max int) ([]Message, error) {
	client, err := f.connect(ctx)
	if err != nil {
		return nil, err
	}
	// The command must be created inside the closure: `defer x.Logout().Wait()`
	// would issue LOGOUT right here, before the first real command.
	defer func() { _ = client.Logout().Wait() }()

	if _, err := client.Select(f.Config.Folder, &imap.SelectOptions{ReadOnly: true}).Wait(); err != nil {
		return nil, fmt.Errorf("mail intake: select %q: %w", f.Config.Folder, err)
	}
	search, err := client.UIDSearch(&imap.SearchCriteria{NotFlag: []imap.Flag{imap.FlagSeen}}, nil).Wait()
	if err != nil {
		return nil, fmt.Errorf("mail intake: search unread: %w", err)
	}
	uids := search.AllUIDs()
	if len(uids) == 0 {
		return nil, nil
	}
	if max > 0 && len(uids) > max {
		uids = uids[:max]
	}

	section := &imap.FetchItemBodySection{Peek: true}
	fetch := client.Fetch(imap.UIDSetNum(uids...), &imap.FetchOptions{UID: true, BodySection: []*imap.FetchItemBodySection{section}})
	buffers, err := fetch.Collect()
	if err != nil {
		return nil, fmt.Errorf("mail intake: fetch: %w", err)
	}
	messages := make([]Message, 0, len(buffers))
	for _, buffer := range buffers {
		var raw []byte
		for _, body := range buffer.BodySection {
			raw = body.Bytes
		}
		if len(raw) == 0 {
			continue
		}
		message, err := Parse(raw, f.Limits)
		if err != nil {
			// One unreadable mail must not block the rest of the mailbox. It
			// stays unread and is reported by UID, never by content.
			return nil, fmt.Errorf("mail intake: uid %d: %w", buffer.UID, err)
		}
		message.UID = uint32(buffer.UID)
		messages = append(messages, message)
	}
	return messages, nil
}

// MarkSeen flags the given messages as read on the server.
func (f Fetcher) MarkSeen(ctx context.Context, uids []uint32) error {
	if len(uids) == 0 {
		return nil
	}
	client, err := f.connect(ctx)
	if err != nil {
		return err
	}
	// The command must be created inside the closure: `defer x.Logout().Wait()`
	// would issue LOGOUT right here, before the first real command.
	defer func() { _ = client.Logout().Wait() }()
	if _, err := client.Select(f.Config.Folder, nil).Wait(); err != nil {
		return fmt.Errorf("mail intake: select %q: %w", f.Config.Folder, err)
	}
	set := imap.UIDSet{}
	for _, uid := range uids {
		set.AddNum(imap.UID(uid))
	}
	store := client.Store(set, &imap.StoreFlags{Op: imap.StoreFlagsAdd, Flags: []imap.Flag{imap.FlagSeen}, Silent: true}, nil)
	if err := store.Close(); err != nil {
		return fmt.Errorf("mail intake: mark seen: %w", err)
	}
	return nil
}

func (f Fetcher) connect(ctx context.Context) (*imapclient.Client, error) {
	if f.Password == "" {
		return nil, errors.New("mail intake: password missing")
	}
	timeout := f.DialTimeout
	if timeout <= 0 {
		timeout = 20 * time.Second
	}
	dialer := &net.Dialer{Timeout: timeout}
	dialCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	conn, err := dialer.DialContext(dialCtx, "tcp", f.Config.Address())
	if err != nil {
		return nil, fmt.Errorf("mail intake: connect %s: %w", f.Config.Host, err)
	}
	var client *imapclient.Client
	if f.Config.Insecure {
		client = imapclient.New(conn, nil)
	} else {
		tlsConn := tls.Client(conn, &tls.Config{ServerName: f.Config.Host, MinVersion: tls.VersionTLS12})
		if err := tlsConn.HandshakeContext(dialCtx); err != nil {
			conn.Close()
			return nil, fmt.Errorf("mail intake: tls %s: %w", f.Config.Host, err)
		}
		client = imapclient.New(tlsConn, nil)
	}
	if err := client.Login(f.Config.Username, f.Password).Wait(); err != nil {
		client.Close()
		// The server's wording could echo the username; the operator only
		// needs to know that the login was refused.
		return nil, errors.New("mail intake: login refused")
	}
	return client, nil
}

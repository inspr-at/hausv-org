package mail

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"net"
	"strings"
	"testing"
	"time"
)

type deadlineRecordingConn struct {
	net.Conn
	readDeadline  time.Time
	writeDeadline time.Time
}

func (c *deadlineRecordingConn) SetReadDeadline(deadline time.Time) error {
	c.readDeadline = deadline
	return c.Conn.SetReadDeadline(deadline)
}

func (c *deadlineRecordingConn) SetWriteDeadline(deadline time.Time) error {
	c.writeDeadline = deadline
	return c.Conn.SetWriteDeadline(deadline)
}

func TestSMTPMailerAllowsInternalRelayWithoutAuth(t *testing.T) {
	m := NewSMTP("smtp", "25", "", "", "WEG Portal <noreply@hausv.org>")

	if !m.Configured() {
		t.Fatal("mailer should be configured with host, port, and from")
	}
	if err := m.Validate(); err != nil {
		t.Fatalf("relay mailer should validate without auth: %v", err)
	}
	if auth := m.auth(); auth != nil {
		t.Fatal("relay mailer without user/pass should not create smtp auth")
	}
}

func TestSMTPMailerRequiresPairedCredentials(t *testing.T) {
	m := NewSMTP("smtp", "25", "user", "", "WEG Portal <noreply@hausv.org>")

	if err := m.Validate(); err == nil {
		t.Fatal("mailer should reject partial smtp credentials")
	}
}

func TestPrivacyURLUsesSamePortalOrigin(t *testing.T) {
	got := privacyURL("https://hausv.org/demo/auth/verify?token=secret")
	if got != "https://hausv.org/demo/datenschutz" {
		t.Fatalf("privacy URL = %q", got)
	}
}

func TestMagicLinkMessageUsesHouseLanguageWithoutProviderJargon(t *testing.T) {
	msg := magicLinkMessage(
		"hausv.org <noreply@hausv.org>",
		"max@example.com",
		"https://hausv.org/demo/auth/verify?token=secret",
		"Musterweg 1, 1010 Wien",
	)
	for _, want := range []string{
		"Subject: Ihr Anmeldelink für Musterweg 1, 1010 Wien",
		"im Hausportal für Musterweg 1, 1010 Wien",
		"Der Link ist 15 Minuten gültig",
		"https://hausv.org/demo/datenschutz",
		"Hausportal Musterweg 1, 1010 Wien",
	} {
		if !strings.Contains(msg, want) {
			t.Fatalf("magic-link message missing %q:\n%s", want, msg)
		}
	}
	for _, forbidden := range []string{"WEG Portal", "Zitadel", "SSO"} {
		if strings.Contains(msg, forbidden) {
			t.Fatalf("magic-link message exposes %q:\n%s", forbidden, msg)
		}
	}
}

func TestSMTPMailerConnectDeadlineIsBoundedAndSanitized(t *testing.T) {
	m := NewSMTP("smtp.invalid", "587", "", "", "hausv.org <noreply@hausv.org>")
	m.connectTimeout = 40 * time.Millisecond
	m.dialContext = func(ctx context.Context, _, _ string) (net.Conn, error) {
		<-ctx.Done()
		return nil, errors.New("dial failed for owner@example.com with token-secret")
	}

	started := time.Now()
	err := m.SendNotification(
		"owner@example.com",
		"private subject",
		"token-secret",
	)
	elapsed := time.Since(started)
	if err == nil {
		t.Fatal("hanging dial unexpectedly succeeded")
	}
	if elapsed > 500*time.Millisecond {
		t.Fatalf("connect deadline took %s, want below 500ms", elapsed)
	}
	if got := err.Error(); got != "smtp transport failed: connect" {
		t.Fatalf("connect error = %q", got)
	}
}

func TestSMTPMailerReadDeadlineClosesHangingEndpoint(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	t.Cleanup(func() { _ = listener.Close() })

	accepted := make(chan struct{})
	peerClosed := make(chan struct{})
	go func() {
		connection, acceptErr := listener.Accept()
		if acceptErr != nil {
			close(accepted)
			close(peerClosed)
			return
		}
		close(accepted)
		defer connection.Close()
		var oneByte [1]byte
		_, _ = connection.Read(oneByte[:])
		close(peerClosed)
	}()

	host, port, err := net.SplitHostPort(listener.Addr().String())
	if err != nil {
		t.Fatalf("split listener address: %v", err)
	}
	m := NewSMTP(host, port, "", "", "hausv.org <noreply@hausv.org>")
	m.connectTimeout = 250 * time.Millisecond
	m.transactionTimeout = 60 * time.Millisecond

	started := time.Now()
	err = m.SendMagicLink(
		"owner@example.com",
		"https://example.test/auth/verify?token=token-secret",
		"Private Address 7",
	)
	elapsed := time.Since(started)
	if err == nil {
		t.Fatal("SMTP endpoint without greeting unexpectedly succeeded")
	}
	if elapsed > 500*time.Millisecond {
		t.Fatalf("SMTP read deadline took %s, want below 500ms", elapsed)
	}
	if got := err.Error(); got != "smtp transport failed: greeting" {
		t.Fatalf("greeting error = %q", got)
	}
	select {
	case <-accepted:
	case <-time.After(500 * time.Millisecond):
		t.Fatal("hanging SMTP endpoint was not reached")
	}
	select {
	case <-peerClosed:
	case <-time.After(500 * time.Millisecond):
		t.Fatal("timed-out SMTP connection was not closed")
	}
}

func TestSMTPMailerDoesNotReturnRelayResponseData(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	t.Cleanup(func() { _ = listener.Close() })

	serverDone := make(chan error, 1)
	go func() {
		connection, acceptErr := listener.Accept()
		if acceptErr != nil {
			serverDone <- acceptErr
			return
		}
		defer connection.Close()
		reader := bufio.NewReader(connection)
		if _, writeErr := fmt.Fprint(connection, "220 test relay ready\r\n"); writeErr != nil {
			serverDone <- writeErr
			return
		}
		if _, readErr := reader.ReadString('\n'); readErr != nil {
			serverDone <- readErr
			return
		}
		if _, writeErr := fmt.Fprint(connection, "250 test relay\r\n"); writeErr != nil {
			serverDone <- writeErr
			return
		}
		if _, readErr := reader.ReadString('\n'); readErr != nil {
			serverDone <- readErr
			return
		}
		_, writeErr := fmt.Fprint(
			connection,
			"550 owner@example.com token-secret relay-response\r\n",
		)
		serverDone <- writeErr
	}()

	host, port, err := net.SplitHostPort(listener.Addr().String())
	if err != nil {
		t.Fatalf("split listener address: %v", err)
	}
	m := NewSMTP(host, port, "", "", "hausv.org <noreply@hausv.org>")
	m.connectTimeout = time.Second
	m.transactionTimeout = time.Second
	var recordedConnection *deadlineRecordingConn
	m.dialContext = func(ctx context.Context, network string, address string) (net.Conn, error) {
		connection, dialErr := (&net.Dialer{}).DialContext(ctx, network, address)
		if dialErr != nil {
			return nil, dialErr
		}
		recordedConnection = &deadlineRecordingConn{Conn: connection}
		return recordedConnection, nil
	}

	err = m.SendMagicLink(
		"owner@example.com",
		"https://example.test/auth/verify?token=token-secret",
		"Private Address 7",
	)
	if err == nil {
		t.Fatal("rejecting SMTP relay unexpectedly succeeded")
	}
	for _, forbidden := range []string{
		"owner@example.com",
		"token-secret",
		"Private Address 7",
		"relay-response",
	} {
		if strings.Contains(err.Error(), forbidden) {
			t.Fatalf("SMTP error leaked %q: %q", forbidden, err)
		}
	}
	if got := err.Error(); got != "smtp transport failed: sender" {
		t.Fatalf("sender error = %q", got)
	}
	if recordedConnection == nil ||
		recordedConnection.readDeadline.IsZero() ||
		recordedConnection.writeDeadline.IsZero() {
		t.Fatal("SMTP transaction did not set both read and write deadlines")
	}
	select {
	case serverErr := <-serverDone:
		if serverErr != nil {
			t.Fatalf("SMTP test endpoint: %v", serverErr)
		}
	case <-time.After(time.Second):
		t.Fatal("SMTP test endpoint did not finish")
	}
}

func TestSMTPMailerTreatsAcceptedDataAsDeliveryCommit(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	t.Cleanup(func() { _ = listener.Close() })

	serverDone := make(chan error, 1)
	go func() {
		connection, acceptErr := listener.Accept()
		if acceptErr != nil {
			serverDone <- acceptErr
			return
		}
		defer connection.Close()
		reader := bufio.NewReader(connection)
		writeResponse := func(response string) bool {
			if _, writeErr := fmt.Fprint(connection, response); writeErr != nil {
				serverDone <- writeErr
				return false
			}
			return true
		}
		readCommand := func() (string, bool) {
			line, readErr := reader.ReadString('\n')
			if readErr != nil {
				serverDone <- readErr
				return "", false
			}
			return line, true
		}

		if !writeResponse("220 test relay ready\r\n") {
			return
		}
		if _, ok := readCommand(); !ok || !writeResponse("250 test relay\r\n") {
			return
		}
		if _, ok := readCommand(); !ok || !writeResponse("250 sender accepted\r\n") {
			return
		}
		if _, ok := readCommand(); !ok || !writeResponse("250 recipient accepted\r\n") {
			return
		}
		if _, ok := readCommand(); !ok || !writeResponse("354 send data\r\n") {
			return
		}
		for {
			line, ok := readCommand()
			if !ok {
				return
			}
			if line == ".\r\n" {
				break
			}
		}
		if !writeResponse("250 message accepted\r\n") {
			return
		}
		// Close immediately after the SMTP delivery commit point. A client
		// that treats a later QUIT failure as delivery failure would return an
		// error and invalidate an already delivered Magic Link.
		serverDone <- nil
	}()

	host, port, err := net.SplitHostPort(listener.Addr().String())
	if err != nil {
		t.Fatalf("split listener address: %v", err)
	}
	m := NewSMTP(host, port, "", "", "hausv.org <noreply@hausv.org>")
	m.connectTimeout = time.Second
	m.transactionTimeout = time.Second

	if err := m.SendMagicLink(
		"owner@example.com",
		"https://example.test/auth/verify?token=token-secret",
		"Private Address 7",
	); err != nil {
		t.Fatalf("accepted SMTP delivery returned an error: %v", err)
	}
	select {
	case serverErr := <-serverDone:
		if serverErr != nil {
			t.Fatalf("SMTP test endpoint: %v", serverErr)
		}
	case <-time.After(time.Second):
		t.Fatal("SMTP test endpoint did not finish")
	}
}

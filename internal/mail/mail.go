// Package mail sends the portal's transactional email: magic links, invites and
// event notifications.
//
// It does NOT read stores or decide who should be notified — recipients arrive
// as arguments. That decision belongs one layer up.
package mail

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"net/mail"
	"net/smtp"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/inspr-at/hausv-org/internal/config"
)

const (
	defaultSMTPConnectTimeout     = 5 * time.Second
	defaultSMTPTransactionTimeout = 15 * time.Second
)

type Mailer interface {
	SendMagicLink(to string, link string, address string) error
	SendInvite(to string, loginURL string, address string) error
	SendNotification(to string, subject string, body string) error
	Configured() bool
}

// NewSMTP builds the SMTP transport. The fields stay unexported so the mailer
// cannot be half-constructed; main used to fill them directly, which only
// worked while they shared a package.
func NewSMTP(host, port, user, pass, from string) SmtpMailer {
	return SmtpMailer{
		host:               host,
		port:               port,
		user:               user,
		pass:               pass,
		from:               from,
		connectTimeout:     defaultSMTPConnectTimeout,
		transactionTimeout: defaultSMTPTransactionTimeout,
	}
}

type SmtpMailer struct {
	host               string
	port               string
	user               string
	pass               string
	from               string
	connectTimeout     time.Duration
	transactionTimeout time.Duration
	dialContext        func(context.Context, string, string) (net.Conn, error)
}

// smtpTransportError deliberately carries no wrapped network or relay error.
// SMTP replies can echo recipients or message content, so callers and logs get
// only a stable stage owned by this package.
type smtpTransportError struct {
	stage string
}

func (e smtpTransportError) Error() string {
	return "smtp transport failed: " + e.stage
}

type PortalNotification struct {
	Event      string
	Tenant     config.TenantConfig
	Recipients []string
	ActorEmail string
	Subject    string
	Lines      []string
	ActionURL  string
	ActionText string
}

func (event PortalNotification) Body() string {
	lines := append([]string(nil), event.Lines...)
	if event.ActionURL != "" {
		actionText := strings.TrimSpace(event.ActionText)
		if actionText == "" {
			actionText = "Öffnen"
		}
		lines = append(lines, "", actionText+": "+event.ActionURL)
	}
	return strings.Join(lines, "\n")
}

func (m SmtpMailer) Configured() bool {
	return m.host != "" && m.port != "" && m.from != ""
}

func (m SmtpMailer) Validate() error {
	if !m.Configured() {
		return nil
	}
	if _, err := mail.ParseAddress(m.from); err != nil {
		return fmt.Errorf("invalid MAIL_FROM")
	}
	if (m.user == "") != (m.pass == "") {
		return fmt.Errorf("SMTP_USER and SMTP_PASS must be set together")
	}
	if _, err := strconv.Atoi(m.port); err != nil {
		return fmt.Errorf("SMTP_PORT must be numeric")
	}
	return nil
}

func (m SmtpMailer) auth() smtp.Auth {
	if m.user == "" && m.pass == "" {
		return nil
	}
	return smtp.PlainAuth("", m.user, m.pass, m.host)
}

func (m SmtpMailer) SendMagicLink(to string, link string, address string) error {
	return m.SendMagicLinkContext(context.Background(), to, link, address)
}

// SendMagicLinkContext lets the bounded login queue cancel an in-flight SMTP
// exchange during process shutdown. The transport also has its own absolute
// deadlines, so callers that use SendMagicLink remain bounded.
func (m SmtpMailer) SendMagicLinkContext(ctx context.Context, to string, link string, address string) error {
	if !m.Configured() {
		return errors.New("smtp not configured")
	}

	address = strings.TrimSpace(address)
	if address == "" {
		address = "Ihrem Haus"
	}
	msg := magicLinkMessage(m.from, to, link, address)

	return m.send(ctx, to, []byte(msg))
}

func magicLinkMessage(from string, to string, link string, address string) string {
	address = strings.TrimSpace(address)
	if address == "" {
		address = "Ihrem Haus"
	}
	return strings.Join([]string{
		"From: " + from,
		"To: " + to,
		"Subject: Ihr Anmeldelink für " + address,
		"MIME-Version: 1.0",
		"Content-Type: text/plain; charset=UTF-8",
		"",
		"Hallo,",
		"",
		"mit diesem Link melden Sie sich im Hausportal für " + address + " an:",
		link,
		"",
		"Der Link ist 15 Minuten gültig und kann nur einmal verwendet werden.",
		"Datenschutzinformationen: " + privacyURL(link),
		"",
		"Freundliche Grüße",
		"Hausportal " + address,
	}, "\r\n")
}

func (m SmtpMailer) SendInvite(to string, loginURL string, address string) error {
	if !m.Configured() {
		return errors.New("smtp not configured")
	}
	msg := strings.Join([]string{
		"From: " + m.from,
		"To: " + to,
		"Subject: Einladung zum Hausportal " + address,
		"MIME-Version: 1.0",
		"Content-Type: text/plain; charset=UTF-8",
		"",
		"Hallo,",
		"",
		"Sie wurden zum privaten Hausportal für " + address + " eingeladen.",
		"Melden Sie sich mit dieser E-Mail-Adresse an:",
		loginURL,
		"",
		"Beim Anmelden erhalten Sie einen einmaligen Link per E-Mail",
		"oder verwenden die normale Anmeldung auf der Startseite.",
		"Datenschutzinformationen: " + privacyURL(loginURL),
		"",
		"Freundliche Grüße",
		"Hausportal " + address,
	}, "\r\n")
	return m.send(context.Background(), to, []byte(msg))
}

func privacyURL(raw string) string {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || u.Scheme == "" || u.Host == "" {
		return "/datenschutz"
	}
	parts := strings.Split(strings.Trim(u.Path, "/"), "/")
	if len(parts) > 0 && parts[0] != "" && parts[0] != "auth" {
		u.Path = "/" + parts[0] + "/datenschutz"
	} else {
		u.Path = "/datenschutz"
	}
	u.RawPath = ""
	u.RawQuery = ""
	u.Fragment = ""
	return u.String()
}

func (m SmtpMailer) SendNotification(to string, subject string, body string) error {
	if !m.Configured() {
		return errors.New("smtp not configured")
	}
	subject = strings.TrimSpace(subject)
	if subject == "" {
		subject = "Neue Nachricht aus Ihrem Hausportal"
	}
	body = strings.TrimSpace(body)
	if body == "" {
		body = "Es gibt eine neue Aktualisierung in Ihrem Hausportal."
	}
	msg := strings.Join([]string{
		"From: " + m.from,
		"To: " + to,
		"Subject: " + subject,
		"MIME-Version: 1.0",
		"Content-Type: text/plain; charset=UTF-8",
		"",
		body,
		"",
		"Freundliche Grüße",
		"hausv.org",
	}, "\r\n")
	return m.send(context.Background(), to, []byte(msg))
}

func (m SmtpMailer) send(ctx context.Context, to string, message []byte) error {
	if !m.Configured() {
		return errors.New("smtp not configured")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	fromAddr, err := mail.ParseAddress(m.from)
	if err != nil {
		return fmt.Errorf("invalid MAIL_FROM")
	}

	connectTimeout := m.connectTimeout
	if connectTimeout <= 0 {
		connectTimeout = defaultSMTPConnectTimeout
	}
	transactionTimeout := m.transactionTimeout
	if transactionTimeout <= 0 {
		transactionTimeout = defaultSMTPTransactionTimeout
	}
	dialContext := m.dialContext
	if dialContext == nil {
		dialer := &net.Dialer{}
		dialContext = dialer.DialContext
	}

	connectCtx, cancelConnect := context.WithTimeout(ctx, connectTimeout)
	connection, err := dialContext(connectCtx, "tcp", net.JoinHostPort(m.host, m.port))
	cancelConnect()
	if err != nil {
		return smtpTransportError{stage: "connect"}
	}
	defer connection.Close()

	deadline := time.Now().Add(transactionTimeout)
	if contextDeadline, ok := ctx.Deadline(); ok && contextDeadline.Before(deadline) {
		deadline = contextDeadline
	}
	if err := connection.SetReadDeadline(deadline); err != nil {
		return smtpTransportError{stage: "read deadline"}
	}
	if err := connection.SetWriteDeadline(deadline); err != nil {
		return smtpTransportError{stage: "write deadline"}
	}
	stopCancellation := context.AfterFunc(ctx, func() {
		_ = connection.Close()
	})
	defer stopCancellation()

	client, err := smtp.NewClient(connection, m.host)
	if err != nil {
		return smtpTransportError{stage: "greeting"}
	}
	defer client.Close()

	if ok, _ := client.Extension("STARTTLS"); ok {
		if err := client.StartTLS(&tls.Config{
			MinVersion: tls.VersionTLS12,
			ServerName: m.host,
		}); err != nil {
			return smtpTransportError{stage: "starttls"}
		}
	}
	if auth := m.auth(); auth != nil {
		if err := client.Auth(auth); err != nil {
			return smtpTransportError{stage: "auth"}
		}
	}
	if err := client.Mail(fromAddr.Address); err != nil {
		return smtpTransportError{stage: "sender"}
	}
	if err := client.Rcpt(to); err != nil {
		return smtpTransportError{stage: "recipient"}
	}
	writer, err := client.Data()
	if err != nil {
		return smtpTransportError{stage: "data"}
	}
	if _, err := writer.Write(message); err != nil {
		_ = writer.Close()
		return smtpTransportError{stage: "write"}
	}
	if err := writer.Close(); err != nil {
		return smtpTransportError{stage: "accept"}
	}
	// The relay's successful response to the final DATA terminator is the
	// delivery commit point. A later QUIT/connection-teardown failure must not
	// turn an already accepted Magic Link into a reported failure whose token
	// is then invalidated. This transport is one-shot; the deferred close ends
	// the SMTP session without waiting for a second server acknowledgement.
	return nil
}

func RedactedEmail(email string) string {
	parts := strings.Split(email, "@")
	if len(parts) != 2 {
		return "<redacted>"
	}
	name := parts[0]
	if len(name) > 1 {
		name = name[:1] + "***"
	} else {
		name = "***"
	}
	return name + "@" + parts[1]
}

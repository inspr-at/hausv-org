// Package mail sends the portal's transactional email: magic links, invites and
// event notifications.
//
// It does NOT read stores or decide who should be notified — recipients arrive
// as arguments. That decision belongs one layer up.
package mail

import (
	"errors"
	"fmt"
	"net"
	"net/mail"
	"net/smtp"
	"net/url"
	"strconv"
	"strings"

	"github.com/markus-barta/hausv-org/internal/config"
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
	return SmtpMailer{host: host, port: port, user: user, pass: pass, from: from}
}

type SmtpMailer struct {
	host string
	port string
	user string
	pass string
	from string
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
	if !m.Configured() {
		return errors.New("smtp not configured")
	}

	addr := net.JoinHostPort(m.host, m.port)
	fromAddr, err := mail.ParseAddress(m.from)
	if err != nil {
		return fmt.Errorf("invalid MAIL_FROM")
	}

	address = strings.TrimSpace(address)
	if address == "" {
		address = "Ihrem Haus"
	}
	msg := magicLinkMessage(m.from, to, link, address)

	return smtp.SendMail(addr, m.auth(), fromAddr.Address, []string{to}, []byte(msg))
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
	addr := net.JoinHostPort(m.host, m.port)
	fromAddr, err := mail.ParseAddress(m.from)
	if err != nil {
		return fmt.Errorf("invalid MAIL_FROM")
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
	return smtp.SendMail(addr, m.auth(), fromAddr.Address, []string{to}, []byte(msg))
}

func privacyURL(raw string) string {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || u.Scheme == "" || u.Host == "" {
		return "/datenschutz"
	}
	u.Path = "/datenschutz"
	u.RawPath = ""
	u.RawQuery = ""
	u.Fragment = ""
	return u.String()
}

func (m SmtpMailer) SendNotification(to string, subject string, body string) error {
	if !m.Configured() {
		return errors.New("smtp not configured")
	}
	addr := net.JoinHostPort(m.host, m.port)
	fromAddr, err := mail.ParseAddress(m.from)
	if err != nil {
		return fmt.Errorf("invalid MAIL_FROM")
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
	return smtp.SendMail(addr, m.auth(), fromAddr.Address, []string{to}, []byte(msg))
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

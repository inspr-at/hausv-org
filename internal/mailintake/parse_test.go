package mailintake

import (
	"encoding/base64"
	"strings"
	"testing"
)

var testLimits = Limits{MaxAttachmentBytes: 1024, MaxAttachments: 2, MaxTextBytes: 4096}

func TestParseReadsPlainMailWithEncodedHeaders(t *testing.T) {
	raw := strings.Join([]string{
		"From: =?UTF-8?Q?Alina_Auer?= <Alina.Auer@Example.com>",
		"To: post@musterstadt.example",
		"Subject: =?UTF-8?Q?Wasserschaden_K=C3=BCche_Top_1?=",
		"Message-ID: <abc-123@example.com>",
		"Date: Fri, 05 Sep 2026 09:15:00 +0200",
		"Content-Type: text/plain; charset=utf-8",
		"",
		"Guten Tag,",
		"in der Küche tropft es seit gestern.",
		"",
	}, "\r\n")
	message, err := Parse([]byte(raw), testLimits)
	if err != nil {
		t.Fatal(err)
	}
	if message.FromEmail != "alina.auer@example.com" || message.FromName != "Alina Auer" {
		t.Fatalf("from = %q <%s>", message.FromName, message.FromEmail)
	}
	if message.Subject != "Wasserschaden Küche Top 1" {
		t.Fatalf("subject = %q", message.Subject)
	}
	if message.MessageID != "abc-123@example.com" {
		t.Fatalf("message id = %q", message.MessageID)
	}
	if !strings.Contains(message.Text, "tropft es seit gestern") {
		t.Fatalf("text = %q", message.Text)
	}
	if message.Date.IsZero() || message.Date.Hour() != 7 {
		t.Fatalf("date not normalised to UTC: %v", message.Date)
	}
}

func TestParsePrefersPlainOverHTMLAndCollectsAttachments(t *testing.T) {
	pdf := base64.StdEncoding.EncodeToString([]byte("%PDF-1.4 fake receipt"))
	raw := strings.Join([]string{
		"From: matthias.mieter@musterstadt.example",
		"Subject: Beleg Hausreinigung",
		"Message-ID: <multi@example.com>",
		"Content-Type: multipart/mixed; boundary=\"outer\"",
		"",
		"--outer",
		"Content-Type: multipart/alternative; boundary=\"inner\"",
		"",
		"--inner",
		"Content-Type: text/plain; charset=iso-8859-1",
		"Content-Transfer-Encoding: quoted-printable",
		"",
		"Anbei der Beleg f=FCr die Hausreinigung.",
		"--inner",
		"Content-Type: text/html; charset=utf-8",
		"",
		"<html><body><p>Anbei der <b>Beleg</b></p></body></html>",
		"--inner--",
		"--outer",
		"Content-Type: application/pdf; name=\"beleg.pdf\"",
		"Content-Disposition: attachment; filename=\"beleg.pdf\"",
		"Content-Transfer-Encoding: base64",
		"",
		pdf,
		"--outer--",
		"",
	}, "\r\n")
	message, err := Parse([]byte(raw), testLimits)
	if err != nil {
		t.Fatal(err)
	}
	if message.Text != "Anbei der Beleg für die Hausreinigung." {
		t.Fatalf("text = %q (plain part with latin1 umlaut expected)", message.Text)
	}
	if len(message.Attachments) != 1 || message.Attachments[0].Filename != "beleg.pdf" || message.Attachments[0].ContentType != "application/pdf" {
		t.Fatalf("attachments = %+v", message.Attachments)
	}
	if string(message.Attachments[0].Data) != "%PDF-1.4 fake receipt" {
		t.Fatalf("attachment bytes = %q", message.Attachments[0].Data)
	}
}

func TestParseFallsBackToHTMLText(t *testing.T) {
	raw := strings.Join([]string{
		"From: sophie.bewohner@musterstadt.example",
		"Subject: Lärm",
		"Content-Type: text/html; charset=utf-8",
		"",
		"<html><head><style>p{color:red}</style></head><body><p>Seit Tagen</p><p>nächtlicher L&auml;rm &amp; Musik.</p><script>alert(1)</script></body></html>",
	}, "\r\n")
	message, err := Parse([]byte(raw), testLimits)
	if err != nil {
		t.Fatal(err)
	}
	if message.Text != "Seit Tagen\nnächtlicher Lärm & Musik." {
		t.Fatalf("html text = %q", message.Text)
	}
}

func TestParseEnforcesAttachmentLimits(t *testing.T) {
	big := base64.StdEncoding.EncodeToString(make([]byte, 2048))
	small := base64.StdEncoding.EncodeToString([]byte("ok"))
	raw := strings.Join([]string{
		"From: x@example.com",
		"Subject: Viele Anhänge",
		"Content-Type: multipart/mixed; boundary=\"b\"",
		"",
		"--b",
		"Content-Type: text/plain",
		"",
		"Text",
		"--b",
		"Content-Type: application/octet-stream; name=\"gross.bin\"",
		"Content-Disposition: attachment; filename=\"gross.bin\"",
		"Content-Transfer-Encoding: base64",
		"",
		big,
		"--b",
		"Content-Type: application/octet-stream; name=\"a.bin\"",
		"Content-Disposition: attachment; filename=\"a.bin\"",
		"Content-Transfer-Encoding: base64",
		"",
		small,
		"--b",
		"Content-Type: application/octet-stream; name=\"b.bin\"",
		"Content-Disposition: attachment; filename=\"b.bin\"",
		"Content-Transfer-Encoding: base64",
		"",
		small,
		"--b",
		"Content-Type: application/octet-stream; name=\"c.bin\"",
		"Content-Disposition: attachment; filename=\"c.bin\"",
		"Content-Transfer-Encoding: base64",
		"",
		small,
		"--b--",
		"",
	}, "\r\n")
	message, err := Parse([]byte(raw), testLimits)
	if err != nil {
		t.Fatal(err)
	}
	if len(message.Attachments) != 2 {
		t.Fatalf("kept %d attachments, want the limit of 2", len(message.Attachments))
	}
	// gross.bin over the size limit, c.bin over the count limit
	if message.Dropped != 2 {
		t.Fatalf("dropped = %d, want 2", message.Dropped)
	}
	if message.Attachments[0].Filename != "a.bin" || message.Attachments[1].Filename != "b.bin" {
		t.Fatalf("kept the wrong files: %+v", message.Attachments)
	}
}

func TestParseTruncatesLongText(t *testing.T) {
	raw := "From: x@example.com\r\nSubject: lang\r\n\r\n" + strings.Repeat("ä", 5000)
	message, err := Parse([]byte(raw), testLimits)
	if err != nil {
		t.Fatal(err)
	}
	if len(message.Text) > testLimits.MaxTextBytes+4 || !strings.HasSuffix(message.Text, "…") {
		t.Fatalf("text not truncated cleanly: len=%d", len(message.Text))
	}
}

func TestParseConfigsValidatesShape(t *testing.T) {
	configs, err := ParseConfigs(`{"Musterstadt": {"host": "imap.example", "username": "post@musterstadt.example", "password_env": "INTAKE_MAIL_PW", "interval": "30s"}}`)
	if err != nil {
		t.Fatal(err)
	}
	config, ok := configs["musterstadt"]
	if !ok {
		t.Fatalf("organisation key not slugged: %v", configs)
	}
	if config.Port != 993 || config.Folder != "INBOX" || config.Interval.Seconds() != 30 {
		t.Fatalf("defaults not applied: %+v", config)
	}
	for _, broken := range []string{
		`{"m": {"host": "imap.example", "username": "u"}}`,                                             // no password source
		`{"m": {"host": "imap.example", "username": "u", "password_env": "A", "password_file": "/x"}}`, // both
		`{"m": {"host": "imap.example", "username": "u", "password_env": "A B"}}`,                      // bad env name
		`{"m": {"host": "imap.example", "username": "u", "password_env": "A", "interval": "1s"}}`,      // too eager
		`{"m": {"username": "u", "password_env": "A"}}`,                                                // no host
		`not json`,
	} {
		if _, err := ParseConfigs(broken); err == nil {
			t.Fatalf("accepted broken config %s", broken)
		}
	}
}

func TestPasswordResolvesFromEnvWithoutLeaking(t *testing.T) {
	config := Config{PasswordEnv: "TEST_MAIL_PW"}
	if _, err := config.Password(func(string) string { return "" }); err == nil || strings.Contains(err.Error(), "TEST") {
		t.Fatalf("unset env: err=%v", err)
	}
	password, err := config.Password(func(name string) string {
		if name == "TEST_MAIL_PW" {
			return " secret-value \n"
		}
		return ""
	})
	if err != nil || password != "secret-value" {
		t.Fatalf("password = %q err=%v", password, err)
	}
}

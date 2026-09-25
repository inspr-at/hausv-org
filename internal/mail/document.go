package mail

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"mime"
	"mime/multipart"
	"mime/quotedprintable"
	"net/mail"
	"net/textproto"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type Attachment struct {
	Filename    string
	ContentType string
	Data        []byte
}

// WithOutbox routes every message to a private file sink instead of SMTP.
func (m SmtpMailer) WithOutbox(directory string) SmtpMailer {
	m.outboxDir = strings.TrimSpace(directory)
	if m.outboxDir != "" && m.from == "" {
		m.from = "hausv.org <noreply@example.invalid>"
	}
	return m
}

func (m SmtpMailer) Mode() string {
	if m.outboxDir != "" {
		return "Postausgang als Datei — Testmodus"
	}
	return "Jede Partei erhält ihr archiviertes PDF je Einheit per E-Mail an die hinterlegte Adresse."
}

func (m SmtpMailer) SendDocument(ctx context.Context, to, subject, body string, attachment Attachment) error {
	message, err := documentMessage(m.from, to, subject, body, attachment)
	if err != nil {
		return err
	}
	return m.send(ctx, to, message)
}

func documentMessage(from, to, subject, body string, attachment Attachment) ([]byte, error) {
	for _, value := range []string{from, to, subject, attachment.Filename, attachment.ContentType} {
		if strings.ContainsAny(value, "\r\n") {
			return nil, fmt.Errorf("invalid mail header")
		}
	}
	sender, err := mail.ParseAddress(from)
	if err != nil {
		return nil, fmt.Errorf("invalid sender")
	}
	recipient, err := mail.ParseAddress(to)
	if err != nil || recipient.Address != to {
		return nil, fmt.Errorf("invalid recipient")
	}
	contentType, _, err := mime.ParseMediaType(attachment.ContentType)
	if err != nil || attachment.Filename == "" {
		return nil, fmt.Errorf("invalid attachment")
	}
	var id [16]byte
	if _, err := rand.Read(id[:]); err != nil {
		return nil, fmt.Errorf("mail message ID unavailable")
	}
	domain := sender.Address[strings.LastIndexByte(sender.Address, '@')+1:]
	var message bytes.Buffer
	writer := multipart.NewWriter(&message)
	fmt.Fprintf(&message, "From: %s\r\nTo: %s\r\nDate: %s\r\nMessage-ID: <%x@%s>\r\nSubject: %s\r\nMIME-Version: 1.0\r\nContent-Type: multipart/mixed; boundary=%q\r\n\r\n", sender.String(), to, time.Now().UTC().Format(time.RFC1123Z), id, domain, mime.QEncoding.Encode("UTF-8", subject), writer.Boundary())
	part, err := writer.CreatePart(textproto.MIMEHeader{"Content-Type": {"text/plain; charset=UTF-8"}, "Content-Transfer-Encoding": {"quoted-printable"}})
	if err != nil {
		return nil, err
	}
	plain := quotedprintable.NewWriter(part)
	if _, err = plain.Write([]byte(body)); err != nil {
		return nil, err
	}
	if err = plain.Close(); err != nil {
		return nil, err
	}
	part, err = writer.CreatePart(textproto.MIMEHeader{"Content-Type": {contentType}, "Content-Transfer-Encoding": {"base64"}, "Content-Disposition": {mime.FormatMediaType("attachment", map[string]string{"filename": attachment.Filename})}})
	if err != nil {
		return nil, err
	}
	encoded := base64.StdEncoding.EncodeToString(attachment.Data)
	for len(encoded) > 0 {
		n := min(76, len(encoded))
		if _, err = fmt.Fprint(part, encoded[:n], "\r\n"); err != nil {
			return nil, err
		}
		encoded = encoded[n:]
	}
	if err = writer.Close(); err != nil {
		return nil, err
	}
	return message.Bytes(), nil
}

func (m SmtpMailer) writeOutbox(ctx context.Context, message []byte) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := os.MkdirAll(m.outboxDir, 0700); err != nil {
		return fmt.Errorf("mail outbox unavailable")
	}
	// Timestamp plus a random message ID is sortable and collision-safe across processes.
	var id [16]byte
	if _, err := rand.Read(id[:]); err != nil {
		return fmt.Errorf("mail outbox ID unavailable")
	}
	name := fmt.Sprintf("%s-%x.eml", time.Now().UTC().Format("20060102T150405.000000000Z"), id)
	path := filepath.Join(m.outboxDir, name)
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return fmt.Errorf("mail outbox unavailable")
	}
	discard := func() error { file.Close(); _ = os.Remove(path); return fmt.Errorf("mail outbox write failed") }
	if ctx.Err() != nil {
		return discard()
	}
	if _, err := file.Write(message); err != nil {
		return discard()
	}
	if err := file.Sync(); err != nil {
		return discard()
	}
	if err := file.Close(); err != nil {
		return discard()
	}
	return nil
}

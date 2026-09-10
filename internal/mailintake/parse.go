package mailintake

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"mime/quotedprintable"
	"net/mail"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"
)

// Message is one mail as the Posteingang receives it: already decoded, text
// only, attachments bounded.
type Message struct {
	UID         uint32
	MessageID   string
	InReplyTo   string
	FromName    string
	FromEmail   string
	Subject     string
	Text        string
	Date        time.Time
	Attachments []Attachment
	// Dropped counts attachments left behind because of the limits, so the
	// case can say "2 Anhänge nicht übernommen" instead of hiding it.
	Dropped int
	// bodyDigest covers the complete MIME body before display/upload limits.
	// Otherwise two receipts with different attachments or truncated tails
	// could accidentally become the same message without a Message-ID.
	bodyDigest string
}

// DedupeKey is stable across IMAP UIDs, mailbox restarts and repeated fetches.
// The persistent ledger scopes it to an organisation. Keep Message-IDs in the
// existing format so previously processed mail remains recognisable.
func (m Message) DedupeKey() string {
	if id := strings.Trim(strings.TrimSpace(m.MessageID), "<> \t\r\n"); id != "" {
		return id
	}
	body := m.bodyDigest
	if body == "" {
		body = strings.ReplaceAll(m.Text, "\r\n", "\n")
	}
	hash := sha256.New()
	for _, field := range []string{strings.ToLower(strings.TrimSpace(m.FromEmail)), strings.TrimSpace(m.Subject), m.Date.UTC().Format(time.RFC3339Nano), body} {
		// Length-prefix fields so their boundaries cannot cause collisions.
		fmt.Fprintf(hash, "%d:%s", len(field), field)
	}
	return fmt.Sprintf("sha256:%x", hash.Sum(nil))
}

type Attachment struct {
	Filename    string
	ContentType string
	Data        []byte
}

// Limits mirror what the portal accepts as an upload. They are parameters so
// the server hands in the same constants the attachment store enforces.
type Limits struct {
	MaxAttachmentBytes int64
	MaxAttachments     int
	MaxTextBytes       int
}

var wordDecoder = mime.WordDecoder{CharsetReader: charsetReader}

// Parse turns raw RFC 5322 bytes into a Message. Content is never executed or
// interpreted beyond decoding: HTML becomes text, everything else is bytes.
func Parse(raw []byte, limits Limits) (Message, error) {
	parsed, err := mail.ReadMessage(bytes.NewReader(raw))
	if err != nil {
		return Message{}, fmt.Errorf("mail intake: unreadable message: %w", err)
	}
	message := Message{
		MessageID: strings.Trim(strings.TrimSpace(parsed.Header.Get("Message-ID")), "<>"),
		InReplyTo: strings.Trim(strings.TrimSpace(parsed.Header.Get("In-Reply-To")), "<>"),
		Subject:   decodeHeader(parsed.Header.Get("Subject")),
	}
	if from, err := mail.ParseAddress(parsed.Header.Get("From")); err == nil {
		message.FromEmail = strings.ToLower(strings.TrimSpace(from.Address))
		message.FromName = decodeHeader(from.Name)
	}
	if date, err := parsed.Header.Date(); err == nil {
		message.Date = date.UTC()
	}
	if strings.Trim(message.MessageID, "<> \t\r\n") == "" {
		body, err := io.ReadAll(parsed.Body)
		if err != nil {
			return Message{}, fmt.Errorf("mail intake: unreadable body: %w", err)
		}
		message.bodyDigest = fmt.Sprintf("%x", sha256.Sum256(bytes.ReplaceAll(body, []byte("\r\n"), []byte("\n"))))
		parsed.Body = bytes.NewReader(body)
	}
	var plain, html strings.Builder
	if err := walkPart(parsed.Header.Get("Content-Type"), parsed.Header.Get("Content-Transfer-Encoding"),
		parsed.Header.Get("Content-Disposition"), parsed.Body, &plain, &html, &message, limits, 0); err != nil {
		return Message{}, err
	}
	text := strings.TrimSpace(plain.String())
	if text == "" {
		text = strings.TrimSpace(htmlToText(html.String()))
	}
	if limits.MaxTextBytes > 0 && len(text) > limits.MaxTextBytes {
		text = truncateUTF8(text, limits.MaxTextBytes)
	}
	message.Text = text
	if message.Subject == "" {
		message.Subject = "(kein Betreff)"
	}
	return message, nil
}

type header interface {
	Get(string) string
}

func walkPart(contentType, encoding, disposition string, body io.Reader, plain, html *strings.Builder, message *Message, limits Limits, depth int) error {
	if depth > 8 {
		return errors.New("mail intake: message nested too deeply")
	}
	mediaType, params, err := mime.ParseMediaType(contentType)
	if err != nil || mediaType == "" {
		mediaType = "text/plain"
		params = map[string]string{}
	}
	if strings.HasPrefix(mediaType, "multipart/") {
		boundary := params["boundary"]
		if boundary == "" {
			return errors.New("mail intake: multipart without boundary")
		}
		reader := multipart.NewReader(body, boundary)
		for {
			part, err := reader.NextPart()
			if errors.Is(err, io.EOF) {
				return nil
			}
			if err != nil {
				return fmt.Errorf("mail intake: broken multipart: %w", err)
			}
			if err := walkPart(part.Header.Get("Content-Type"), part.Header.Get("Content-Transfer-Encoding"),
				part.Header.Get("Content-Disposition"), part, plain, html, message, limits, depth+1); err != nil {
				return err
			}
		}
	}

	dispositionType, dispositionParams, _ := mime.ParseMediaType(disposition)
	filename := decodeHeader(firstNonEmpty(dispositionParams["filename"], params["name"]))
	isAttachment := dispositionType == "attachment" || (filename != "" && !strings.HasPrefix(mediaType, "text/"))
	decoded := decodeBody(body, encoding)

	if isAttachment {
		// Read at most one byte past the limit: enough to know it is too big,
		// without holding a multi-megabyte body for a file nobody will keep.
		data, err := io.ReadAll(io.LimitReader(decoded, limits.MaxAttachmentBytes+1))
		if err != nil {
			return fmt.Errorf("mail intake: attachment unreadable: %w", err)
		}
		if int64(len(data)) > limits.MaxAttachmentBytes || len(message.Attachments) >= limits.MaxAttachments {
			message.Dropped++
			return nil
		}
		if filename == "" {
			filename = "anhang-" + fmt.Sprint(len(message.Attachments)+1)
		}
		message.Attachments = append(message.Attachments, Attachment{Filename: filename, ContentType: mediaType, Data: data})
		return nil
	}

	text, err := io.ReadAll(io.LimitReader(decoded, 1<<20))
	if err != nil {
		return fmt.Errorf("mail intake: body unreadable: %w", err)
	}
	converted := convertCharset(text, params["charset"])
	switch mediaType {
	case "text/plain":
		plain.Write(converted)
		plain.WriteString("\n")
	case "text/html":
		html.Write(converted)
	}
	return nil
}

func decodeBody(body io.Reader, encoding string) io.Reader {
	switch strings.ToLower(strings.TrimSpace(encoding)) {
	case "base64":
		return base64.NewDecoder(base64.StdEncoding, newWhitespaceStripper(body))
	case "quoted-printable":
		return quotedprintable.NewReader(body)
	default:
		return body
	}
}

// whitespaceStripper removes line breaks from base64 bodies, which mail agents
// wrap at 76 columns and the standard decoder rejects.
type whitespaceStripper struct{ r io.Reader }

func newWhitespaceStripper(r io.Reader) io.Reader { return &whitespaceStripper{r: r} }

func (w *whitespaceStripper) Read(p []byte) (int, error) {
	n, err := w.r.Read(p)
	kept := 0
	for i := 0; i < n; i++ {
		switch p[i] {
		case '\r', '\n', ' ', '\t':
			continue
		}
		p[kept] = p[i]
		kept++
	}
	if kept == 0 && n > 0 && err == nil {
		return w.Read(p)
	}
	return kept, err
}

func decodeHeader(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	decoded, err := wordDecoder.DecodeHeader(value)
	if err != nil {
		return value
	}
	return strings.TrimSpace(decoded)
}

// charsetReader covers the encodings that reach an Austrian Hausverwaltung in
// practice. Anything else is passed through unchanged rather than refused.
func charsetReader(charset string, input io.Reader) (io.Reader, error) {
	switch strings.ToLower(strings.TrimSpace(charset)) {
	case "", "utf-8", "utf8", "us-ascii", "ascii":
		return input, nil
	case "iso-8859-1", "latin1", "iso-8859-15", "windows-1252", "cp1252":
		raw, err := io.ReadAll(input)
		if err != nil {
			return nil, err
		}
		return bytes.NewReader(latin1ToUTF8(raw)), nil
	default:
		return input, nil
	}
}

func convertCharset(data []byte, charset string) []byte {
	reader, err := charsetReader(charset, bytes.NewReader(data))
	if err != nil {
		return data
	}
	out, err := io.ReadAll(reader)
	if err != nil {
		return data
	}
	if !utf8.Valid(out) {
		return latin1ToUTF8(out)
	}
	return out
}

func latin1ToUTF8(raw []byte) []byte {
	var out bytes.Buffer
	out.Grow(len(raw) + len(raw)/4)
	for _, b := range raw {
		out.WriteRune(rune(b))
	}
	return out.Bytes()
}

var (
	htmlBlockBreak = regexp.MustCompile(`(?i)<\s*(br|/p|/div|/li|/h[1-6]|/tr)\s*/?>`)
	htmlTag        = regexp.MustCompile(`(?s)<[^>]*>`)
	htmlScript     = regexp.MustCompile(`(?is)<(script|style)[^>]*>.*?</(script|style)>`)
	blankLines     = regexp.MustCompile(`\n{3,}`)
)

// htmlToText keeps the words and the line structure, nothing else.
func htmlToText(html string) string {
	text := htmlScript.ReplaceAllString(html, "")
	text = htmlBlockBreak.ReplaceAllString(text, "\n")
	text = htmlTag.ReplaceAllString(text, "")
	text = strings.NewReplacer("&nbsp;", " ", "&amp;", "&", "&lt;", "<", "&gt;", ">", "&quot;", "\"", "&#39;", "'", "&auml;", "ä", "&ouml;", "ö", "&uuml;", "ü", "&Auml;", "Ä", "&Ouml;", "Ö", "&Uuml;", "Ü", "&szlig;", "ß").Replace(text)
	lines := strings.Split(text, "\n")
	for i, line := range lines {
		lines[i] = strings.TrimSpace(line)
	}
	return blankLines.ReplaceAllString(strings.Join(lines, "\n"), "\n\n")
}

func truncateUTF8(text string, max int) string {
	if len(text) <= max {
		return text
	}
	cut := max
	for cut > 0 && !utf8.RuneStart(text[cut]) {
		cut--
	}
	return strings.TrimSpace(text[:cut]) + " …"
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

var _ header = mail.Header{}

package server

import (
	"context"
	appmail "github.com/inspr-at/hausv-org/internal/mail"
)

func (m *recordingMailer) SendDocument(_ context.Context, to, subject, body string, _ appmail.Attachment) error {
	return m.SendNotification(to, subject, body)
}
func (m *gatedMagicLinkMailer) SendDocument(_ context.Context, to, subject, body string, _ appmail.Attachment) error {
	return m.SendNotification(to, subject, body)
}
func (m failingMagicLinkMailer) SendDocument(_ context.Context, to, subject, body string, _ appmail.Attachment) error {
	return m.err
}
func (m *contextBlockedMagicLinkMailer) SendDocument(_ context.Context, to, subject, body string, _ appmail.Attachment) error {
	return m.SendNotification(to, subject, body)
}
func (m *intakeReplyMailer) SendDocument(_ context.Context, to, subject, body string, _ appmail.Attachment) error {
	return m.SendNotification(to, subject, body)
}

package mailer

import "embed"

//go:embed "templates"
var FS embed.FS

type MailTemplateFile string

const (
	TemplateSignatureRequestInvitation MailTemplateFile = "templates/signature_request_invitation.tmpl"
)

type SignatureRequestInvitationData struct {
	RecipientName           string
	InviterName             string
	CertificateProjectTitle string
	SigningURL              string
	APP_NAME                string
	APP_LOGO_URL            string
}

type Client interface {
	Send(templateFile MailTemplateFile, toEmail string, data any) (int, error)
}

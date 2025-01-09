package mailer

import "embed"

const (
	FROM_NAME        = "CS2"
	MAX_RETRY        = 3
	EXAMPLE_TEMPLATE = "example.tmpl"
)

//go:embed "templates"
var FS embed.FS

type Client interface {
	Send(templateFile, toUsername, toEmail string, data any) (int, error)
}

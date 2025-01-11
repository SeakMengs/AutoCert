package mailer

import (
	"bytes"
	"fmt"
	"html/template"
	"time"

	"github.com/SeakMengs/AutoCert/internal/util"
	"github.com/sendgrid/sendgrid-go"
	"github.com/sendgrid/sendgrid-go/helpers/mail"
	"go.uber.org/zap"
)

type SendGridMailer struct {
	fromEmail string
	client    *sendgrid.Client
	isSandBox bool
	logger    *zap.SugaredLogger
	enabled   bool
}

func NewSendgrid(apiKey string, fromEmail string, isProduction bool, logger *zap.SugaredLogger) *SendGridMailer {
	// For unit test
	if logger == nil {
		logger = util.NewLogger()
	}

	client := sendgrid.NewSendClient(apiKey)

	return &SendGridMailer{
		fromEmail: fromEmail,
		client:    client,
		// Sandbox mode is only used to validate your request. The email will never be delivered while this feature is enabled!
		isSandBox: !isProduction,
		logger:    logger,
	}
}

// Data is struct entity where it will be used in template.
//
//		Example: vars := struct {
//		Username      string
//		ActivationURL string
//	}{
//
//		Username:      user.Username,
//		ActivationURL: activationURL,
//	}
//
//	Check templates/example.tmpl to see how we use the entity
//
//	Example usage:
//	status, err := Send(mailer.EXAMPLE_TEMPLATE, user.Username, user.Email, vars)
func (m SendGridMailer) Send(templateFile, toUsername, toEmail string, data any) (int, error) {
	from := mail.NewEmail(FROM_NAME, m.fromEmail)
	to := mail.NewEmail(toUsername, toEmail)

	// template parsing and building
	tmpl, err := template.ParseFS(FS, "templates/"+templateFile)
	if err != nil {
		m.logger.Errorf("Error occurred during mail template parsing, error: %v", err)
		return -1, err
	}

	subject := new(bytes.Buffer)
	err = tmpl.ExecuteTemplate(subject, "subject", data)
	if err != nil {
		m.logger.Errorf("Error occurred during extracting subject from mail template, error: %v", err)
		return -1, err
	}

	body := new(bytes.Buffer)
	err = tmpl.ExecuteTemplate(body, "body", data)
	if err != nil {
		m.logger.Errorf("Error occurred during extracting body from mail template, error: %v", err)
		return -1, err
	}

	message := mail.NewSingleEmail(from, subject.String(), to, "", body.String())

	message.SetMailSettings(&mail.MailSettings{
		SandboxMode: &mail.Setting{
			Enable: &m.isSandBox,
		},
	})

	var retryErr error
	for i := 0; i < MAX_RETRY; i++ {
		response, retryErr := m.client.Send(message)
		if retryErr != nil {
			// exponential backoff
			time.Sleep(time.Second * time.Duration(i+1))
			continue
		}

		return response.StatusCode, nil
	}

	m.logger.Errorf("Failed to send email after %d attempt, error: %v", MAX_RETRY, retryErr)

	return -1, fmt.Errorf("Failed to send email after %d attempt, error: %v", MAX_RETRY, retryErr)
}

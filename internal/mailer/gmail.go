package mailer

import (
	"bytes"
	"fmt"
	"html/template"
	"net/http"

	"github.com/SeakMengs/AutoCert/internal/util"
	"go.uber.org/zap"
	"gopkg.in/gomail.v2"
)

type GmailMailer struct {
	fromEmail string
	fromName  string
	host      string
	port      int
	username  string
	password  string
	logger    *zap.SugaredLogger
}

func NewGmailMailer(username, password string, logger *zap.SugaredLogger) *GmailMailer {
	return &GmailMailer{
		fromEmail: username,
		fromName:  util.GetAppName(),
		host:      "smtp.gmail.com",
		port:      587,
		username:  username,
		password:  password,
		logger:    logger,
	}
}

func (gm *GmailMailer) Send(templateFile MailTemplateFile, toEmail string, data any) (int, error) {
	tmpl, err := template.ParseFS(FS, string(templateFile))
	if err != nil {
		gm.logger.Errorw("failed to parse email template", "error", err, "templateFile", templateFile)
		return http.StatusInternalServerError, err
	}

	subject := new(bytes.Buffer)
	if err := tmpl.ExecuteTemplate(subject, "subject", data); err != nil {
		gm.logger.Errorw("failed to execute subject template", "error", err, "templateFile", templateFile)
		return http.StatusInternalServerError, err
	}

	body := new(bytes.Buffer)
	if err := tmpl.ExecuteTemplate(body, "body", data); err != nil {
		gm.logger.Errorw("failed to execute body template", "error", err, "templateFile", templateFile)
		return http.StatusInternalServerError, err
	}

	message := gomail.NewMessage()
	message.SetHeader("From", fmt.Sprintf("%s <%s>", gm.fromName, gm.fromEmail))
	message.SetHeader("To", toEmail)
	message.SetHeader("Subject", subject.String())
	message.SetBody("text/html", body.String())

	dialer := gomail.NewDialer(gm.host, gm.port, gm.username, gm.password)

	if err := dialer.DialAndSend(message); err != nil {
		gm.logger.Errorw("failed to send email", "error", err, "toEmail", toEmail, "templateFile", templateFile)
		return http.StatusInternalServerError, fmt.Errorf("failed to send email: %w", err)
	}

	gm.logger.Infow("email sent successfully", "toEmail", toEmail, "templateFile", templateFile)

	return http.StatusOK, nil
}

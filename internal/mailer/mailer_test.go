package mailer

import (
	"log"
	"net/http"
	"testing"

	"github.com/SeakMengs/go-api-boilerplate/internal/config"
	"github.com/joho/godotenv"
)

func TestSendMail(t *testing.T) {
	if err := godotenv.Load("../../.env"); err != nil {
		log.Fatal(err)
	}

	cfg := config.GetConfig()
	// isProduction = false to ensure that the send mail test always run in sandbox mode which won't send actual email to the user
	mail := NewSendgrid(cfg.Mail.SEND_GRID.API_KEY, cfg.Mail.FROM_EMAIL, false, nil)

	vars := struct {
		Username      string
		ActivationURL string
	}{
		Username:      "Example Username inject",
		ActivationURL: "Example actiation url inject",
	}

	status, err := mail.Send(EXAMPLE_TEMPLATE, "toExampleUserName", "hseakmeng22@gmail.com", vars)

	switch status {
	case http.StatusUnauthorized:
		t.Errorf("Unauthorized to send mail, check mail api_key and from_email")
	case http.StatusForbidden:
		t.Errorf("Forbidden to send mail, check mail from_email is it the correct email authorized in send grid?")
	}

	// If status == 202, it mean successful
	if status != http.StatusAccepted && status != http.StatusOK {
		t.Errorf("We got status %d, error: %v", status, err)
	}
}

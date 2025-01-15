package mailer

import (
	"embed"
	"time"

	"github.com/go-mail/mail/v2"
)

// we declare a new variable with the type embed.FS
// (embedded file system). This has a comment directive in the
// format //go:embed <path> which indicates to GO that we want
// to store the contents of ./templates directory in the
// templatesFS embedded file system variable

//go:embed "templates"
var templatesFS embed.FS

// define a mailer instace which contains a mail.Dialer instance
// used to connect to an SMTP server  and the sender information
// for your emails (the names and address you want the emails to be from)

type Mailer struct {
	dialer *mail.Dialer
	sender string
}

func New(host string, port int, username, password, sender string) Mailer {
	// Initialize a new mail.Dialer instance with the given
	// SMTP server settings. we also configure this to use a 5-second timeout when sending emails
	dialer := mail.NewDialer(host, port, username, password)
	dialer.Timeout = 5 * time.Second

	// return mailer instance containing the dialer and sender information
	return Mailer{
		dialer: dialer,
		sender: sender,
	}
}

// define a send

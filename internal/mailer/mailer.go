package mailer

import (
	"bytes"
	"embed"
	"html/template"
	"time"

	"github.com/go-mail/mail/v2"
)

// we declare a new variable with the type embed.FS
// (embedded file system). This has a comment directive in the
// format //go:embed <path> which indicates to GO that we want
// to store the contents of ./templates directory in the
// templatesFS embedded file system variable

//go:embed "templates"
var templateFS embed.FS

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

// define a Send() method on the Mailer type. This takes the recipients email address as
// the first parameter, the name of the file containing the templates, and any dynamic data
// for the templates as an any parameter
func (m Mailer) Send(recipient, templateFile string, data any) error {
	// ParseFS() method to parse the required file from the embedded file system
	tmpl, err := template.New("email").ParseFS(templateFS, "templates/"+templateFile)
	if err != nil {
		return err
	}

	// execute the named template "subject", passing in the dynamic data and storing the result
	// in a bytes.Buffer variable
	subject := new(bytes.Buffer)
	err = tmpl.ExecuteTemplate(subject, "subject", data)
	if err != nil {
		return err
	}

	// following the same pattern we execute the plainBody template and store
	// the result in the plainBody variable
	plainBody := new(bytes.Buffer)
	err = tmpl.ExecuteTemplate(plainBody, "plainBody", data)
	if err != nil {
		return err
	}

	// and similarly the htmlBody template
	htmlBody := new(bytes.Buffer)
	err = tmpl.ExecuteTemplate(htmlBody, "htmlBody", data)
	if err != nil {
		return err
	}

	// use the mail.NewMessage() function to initialize a new mail.NewMessage instance
	// Then we use the SetHeader() method to set the email recipient, send and subject headers
	// and the setBody() method to set the plain-text body, and the AddAlternative() method to set the
	// HTML body. It's important to note that the AddAlternative() should always be called after SetBody()
	msg := mail.NewMessage()
	msg.SetHeader("To", recipient)
	msg.SetHeader("From", m.sender)
	msg.SetHeader("Subject", subject.String())
	msg.SetBody("text/plain", plainBody.String())
	msg.AddAlternative("text/plain", htmlBody.String())

	// call the DialAndSend() method on the dialer, passing in the message to send.
	// This opens a connection to SMTP server, sends the message, then closes the conncetion
	// If there is a timeout, it will return a "dial tcp: i/o timeout" error
	err = m.dialer.DialAndSend(msg)

	if err != nil {
		return err
	}

	return nil
}

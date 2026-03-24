package model

import (
	"encoding/gob"
	"errors"
	"fmt"
	"net/mail"

	"github.com/CCorderZugcat/zugoui/observable"
)

var ErrRequired = errors.New("required")
var ErrTooLong = errors.New("too long")

// ContactForm
type ContactForm struct {
	First   string `bind:"fname"`
	Last    string `bind:"lname"`
	Email   string `bind:"email"`
	Subject string `bind:"subject"`
	Message string `bind:"message"`
}

type ContactControls struct {
	Status         string `bind:"emailStatus>innerText"`
	SubmitDisabled bool   `bind:"submit>disabled"`
}

func checkString(errors observable.ValidationError, name, s string, maxLen int) {
	if len(s) < 1 {
		errors[name] = fmt.Errorf("%s is %w", name, ErrRequired)
	}
	if len(s) > maxLen {
		errors[name] = fmt.Errorf("%s is %w", name, ErrTooLong)
	}
}

func (c ContactForm) ValidateModel() error {
	var errors = make(observable.ValidationError)

	checkString(errors, "First", c.First, 128)
	checkString(errors, "Last", c.Last, 128)
	checkString(errors, "Email", c.Email, 128)
	checkString(errors, "Subject", c.Subject, 128)
	checkString(errors, "Message", c.Message, 1000)

	if _, err := mail.ParseAddress(c.Email); err != nil {
		errors["Email"] = fmt.Errorf("Email: %w", err)
	}

	if len(errors) == 0 {
		return nil
	}

	return errors
}

func init() {
	// It is critical the shared model types are registered with gob.
	// Use an init method in your shared pacakge to do this automatically.
	gob.Register(new(ContactForm))
	gob.Register(new(ContactControls))
}

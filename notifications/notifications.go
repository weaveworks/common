package notifications

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/pkg/errors"
)

// Sender contains the configuration information to send events to the notification service
type Sender struct {
	URL string
}

// EmailMessage contains the required fields for formatting email messages
type EmailMessage struct {
	Subject string `json:"subject"`
	Body    string `json:"body"`
}

// SlackMessage contains the required fields for formatting slack messages
type SlackMessage struct {
	Text string `json:"text"`
}

// BrowserMessage contains the required fields for formatting browser notifications
type BrowserMessage struct {
	Type      string    `json:"type"`
	Text      string    `json:"text"`
	Timestamp time.Time `json:"timestamp"`
}

// Message contains the mappings for formatting notification messages
type Message struct {
	Browser BrowserMessage `json:"browser"`
	Slack   SlackMessage   `json:"slack"`
	Email   EmailMessage   `json:"email"`
}

type event struct {
	Type       string    `json:"type"`
	InstanceID string    `json:"instance_id"`
	Timestamp  time.Time `json:"timestamp"`
	Messages   *Message  `json:"messages"`
}

// CreateSender creates a Sender
func CreateSender(url string) Sender {
	return Sender{
		URL: url,
	}
}

// SendEvent sends an event to the notification service.
func (s *Sender) SendEvent(et string, instance string, t time.Time, msg *Message) error {
	e := event{
		Type:       et,
		InstanceID: instance,
		Timestamp:  t,
		Messages:   msg,
	}

	eventBytes, err := json.Marshal(e)
	if err != nil {
		return errors.Wrapf(err, "Cannot marshal event to []byte %s", err)
	}

	postEventURL := fmt.Sprintf("%s/api/notification/events", s.URL)

	req, err := http.NewRequest("POST", postEventURL, bytes.NewBuffer(eventBytes))
	if err != nil {
		return errors.Wrapf(err, "POST request error to URL %s", s.URL)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return errors.Wrap(err, "Failed to POST event")
	}

	defer resp.Body.Close()

	return nil
}

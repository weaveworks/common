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
type emailMessage struct {
	Subject string `json:"subject"`
	Body    string `json:"body"`
}

type attachment struct {
	Fallback   string   `json:"fallback,omitempty"`
	Text       string   `json:"text"`
	Color      string   `json:"color,omitempty"`
	MarkdownIn []string `json:"mrkdwn_in,omitempty"`
}

// SlackMessage contains the required fields for formatting slack messages
type slackMessage struct {
	Text        string       `json:"text"`
	Attachments []attachment `json:"attachments"`
}

// BrowserMessage contains the required fields for formatting browser notifications
type browserMessage struct {
	Type      string    `json:"type"`
	Text      string    `json:"text"`
	Timestamp time.Time `json:"timestamp"`
}

// Message contains the mappings for formatting notification messages
type message struct {
	Browser  browserMessage `json:"browser"`
	Slack    slackMessage   `json:"slack"`
	Email    emailMessage   `json:"email"`
	Fallback string         `json:"fallback"`
}

type event struct {
	Type       string    `json:"type"`
	InstanceID string    `json:"instance_id"`
	Timestamp  time.Time `json:"timestamp"`
	Messages   message   `json:"messages"`
}

// CreateSender creates a Sender
func CreateSender(url string) Sender {
	return Sender{
		URL: url,
	}
}

func createMessage(eventType string, timestamp time.Time, text string, attachments []string) message {
	var slackAttachments []attachment

	for i := range attachments {
		raw := attachments[i]
		slackAttachments = append(slackAttachments, attachment{
			Fallback:   text,
			Text:       raw,
			Color:      "#439FE0",
			MarkdownIn: []string{"text"},
		})
	}
	return message{
		Fallback: text,
		Browser: browserMessage{
			Type:      eventType,
			Text:      text,
			Timestamp: timestamp,
		},
		Email: emailMessage{
			Subject: eventType,
			Body:    text,
		},
		Slack: slackMessage{
			Text:        text,
			Attachments: slackAttachments,
		},
	}
}

// SendEvent sends an event to the notification service.
func (s *Sender) SendEvent(eventType string, instance string, t time.Time, msg string, attachments []string) error {
	e := event{
		Type:       eventType,
		InstanceID: instance,
		Timestamp:  t,
		Messages:   createMessage(eventType, t, msg, attachments),
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

## Notifications

The `notifications` package is a client library for the [weaveworks/notifications](https://github.com/weaveworks/notification) service.


```golang
package main

import (
	"fmt"
	"time"

	"github.com/weaveworks/common/notifications"
)

func main() {
	url := "http://eventmanager.notification.svc.cluster.local.:80"
	sender := notifications.CreateSender(url)

	instanceID := "1"
	instanceName := "super-cool-instance"

	// Format your message according to the different channels to which it will be sent
	text := fmt.Sprintf("Instance %s has exploded", instanceName)
	html := fmt.Sprintf("Instance <b>%s</b> has exploded", instanceName)
	markdown := fmt.Sprintf("Instance _%s_ has exploded", instanceName)

	timestamp := time.Now()

	message := notifications.Message{
		Browser: notifications.BrowserMessage{
			Type:      "critical",
			Text:      text,
			Timestamp: timestamp,
		},
		Email: notifications.EmailMessage{
			Subject: "Explosion!",
			Body:    html,
		},
		Slack: notifications.SlackMessage{
			Text: markdown,
		},
	}

	if err := sender.SendEvent("critical", instanceID, timestamp, &message); err != nil {
		fmt.Printf("Error: %v", err)
	}
}
```

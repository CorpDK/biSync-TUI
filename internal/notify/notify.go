package notify

import (
	"fmt"
	"os/exec"
	"time"

	"github.com/CorpDK/bisync-tui/internal/config"
)

// Notifier sends desktop notifications.
type Notifier struct {
	enabled   bool
	onSuccess bool
	onFailure bool
}

// NewNotifier creates a Notifier from config settings.
func NewNotifier(settings config.NotificationSettings) *Notifier {
	return &Notifier{
		enabled:   settings.Enabled,
		onSuccess: settings.OnSuccess,
		onFailure: settings.OnFailure,
	}
}

// Notify sends a desktop notification via notify-send.
func (n *Notifier) Notify(title, body string, isError bool) error {
	if !n.enabled {
		return nil
	}

	urgency := "normal"
	icon := "dialog-information"
	if isError {
		urgency = "critical"
		icon = "dialog-error"
	}

	cmd := exec.Command("notify-send",
		"--urgency", urgency,
		"--icon", icon,
		"--app-name", "syncctl",
		title, body,
	)
	return cmd.Run()
}

// NotifySyncResult sends a notification based on sync outcome.
func (n *Notifier) NotifySyncResult(mappingName string, success bool, duration time.Duration, errMsg string) {
	if !n.enabled {
		return
	}

	if success && n.onSuccess {
		n.Notify(
			fmt.Sprintf("Sync complete: %s", mappingName),
			fmt.Sprintf("Finished in %s", duration.Truncate(time.Second)),
			false,
		)
	} else if !success && n.onFailure {
		n.Notify(
			fmt.Sprintf("Sync failed: %s", mappingName),
			errMsg,
			true,
		)
	}
}

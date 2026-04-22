package dispatcher

import (
	"context"
	"errors"
	"log"
	"strings"

	"github.com/leave-app/notification-dispatcher/internal/client"
	"github.com/leave-app/notification-dispatcher/internal/mailer"
)

type Dispatcher struct {
	api    *client.Client
	mailer *mailer.Mailer
}

func New(api *client.Client, m *mailer.Mailer) *Dispatcher {
	return &Dispatcher{api: api, mailer: m}
}

// Run polls leave-api for pending notifications, attempts to deliver each one,
// and marks successfully delivered notifications as sent. Failures for a single
// notification are logged and do not abort the run.
func (d *Dispatcher) Run(ctx context.Context) error {
	log.Printf("dispatcher: polling leave-api for pending notifications")

	notifs, err := d.api.PendingNotifications(ctx)
	if err != nil {
		return err
	}

	log.Printf("dispatcher: fetched %d pending notification(s)", len(notifs))
	if len(notifs) == 0 {
		return nil
	}

	var sent, failed, skipped int
	for _, n := range notifs {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if n.ID == "" {
			log.Printf("dispatcher: skipping notification with empty id")
			skipped++
			continue
		}
		status, err := d.process(ctx, n)
		switch {
		case err != nil:
			log.Printf("dispatcher: notification id=%s type=%s failed: %v", n.ID, n.Type, err)
			failed++
		case status == statusSkipped:
			skipped++
		default:
			sent++
		}
	}
	log.Printf("dispatcher: run complete sent=%d failed=%d skipped=%d", sent, failed, skipped)
	return nil
}

type processStatus int

const (
	statusSent processStatus = iota
	statusSkipped
)

func (d *Dispatcher) process(ctx context.Context, n client.Notification) (processStatus, error) {
	addr := strings.TrimSpace(n.Address())
	subject := n.Subject
	if subject == "" {
		subject = defaultSubject(n.Type)
	}
	body := n.Text()
	if body == "" {
		body = subject
	}

	if addr == "" {
		// Nothing to send but still mark-sent so the leave-api does not
		// keep returning the same malformed row.
		log.Printf("dispatcher: notification id=%s has no recipient; marking sent to drain queue", n.ID)
		if err := d.api.MarkSent(ctx, n.ID); err != nil {
			return statusSkipped, err
		}
		return statusSkipped, nil
	}

	log.Printf("dispatcher: dispatching id=%s type=%s to=%s", n.ID, n.Type, addr)
	if err := d.mailer.Send(addr, subject, body); err != nil {
		return statusSkipped, errors.New("send: " + err.Error())
	}
	if err := d.api.MarkSent(ctx, n.ID); err != nil {
		return statusSkipped, errors.New("mark-sent: " + err.Error())
	}
	return statusSent, nil
}

func defaultSubject(t string) string {
	switch strings.ToLower(t) {
	case "submission":
		return "Leave request submitted"
	case "approval", "approved":
		return "Your leave request was approved"
	case "rejection", "rejected":
		return "Your leave request was rejected"
	case "cancellation", "cancelled", "canceled":
		return "Leave request cancelled"
	case "reminder":
		return "Upcoming leave reminder"
	default:
		return "Leave management notification"
	}
}

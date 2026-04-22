package handlers

import (
	"fmt"
	"net/http"
	"sort"
	"time"

	"github.com/asdlc/leave-api/internal/middleware"
	"github.com/asdlc/leave-api/internal/models"
	"github.com/asdlc/leave-api/internal/store"
)

type pendingNotificationDTO struct {
	ID        string    `json:"id"`
	Kind      string    `json:"kind"`
	ToUserID  string    `json:"toUserId"`
	ToEmail   string    `json:"toEmail"`
	Subject   string    `json:"subject"`
	Body      string    `json:"body"`
	RequestID string    `json:"requestId,omitempty"`
	CreatedAt time.Time `json:"createdAt"`
}

func (h *Handlers) PendingNotifications(w http.ResponseWriter, r *http.Request) {
	// Trigger reminder generation lazily on every poll so tests/deployments
	// see up-to-date reminders without waiting for the background ticker.
	h.GenerateReminders()
	list := h.Store.ListPendingEmailNotifications()
	out := make([]pendingNotificationDTO, 0, len(list))
	for _, n := range list {
		out = append(out, pendingNotificationDTO{
			ID: n.ID, Kind: string(n.Kind), ToUserID: n.ToUserID, ToEmail: n.ToEmail,
			Subject: n.Subject, Body: n.Body, RequestID: n.RequestID, CreatedAt: n.CreatedAt,
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt.Before(out[j].CreatedAt) })
	writeJSON(w, http.StatusOK, out)
}

func (h *Handlers) MarkNotificationSent(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if !h.Store.MarkNotificationSent(id) {
		writeErr(w, http.StatusNotFound, "notification not found")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handlers) InAppNotifications(w http.ResponseWriter, r *http.Request) {
	p, _ := middleware.GetPrincipal(r)
	list := h.Store.ListInAppForUser(p.UserID)
	sort.Slice(list, func(i, j int) bool { return list[i].CreatedAt.After(list[j].CreatedAt) })
	writeJSON(w, http.StatusOK, list)
}

// GenerateReminders creates idempotent 3-day-out email reminders for approved leaves.
func (h *Handlers) GenerateReminders() {
	today := time.Now().UTC().Truncate(24 * time.Hour)
	for _, lr := range h.Store.ListApprovedRequestsForReminder() {
		start, err := parseDate(lr.StartDate)
		if err != nil {
			continue
		}
		days := int(start.Sub(today).Hours() / 24)
		if days < 0 || days > 3 {
			continue
		}
		key := "reminder:" + lr.ID + ":" + lr.StartDate
		if h.Store.ReminderExists(key) {
			continue
		}
		u, ok := h.Store.GetUser(lr.UserID)
		if !ok {
			continue
		}
		h.Store.CreateNotification(&models.Notification{
			ID: store.NewID(), Kind: models.NotifReminder, Channel: "email",
			ToUserID: u.ID, ToEmail: u.Email,
			Subject:   fmt.Sprintf("Reminder: leave starts %s", lr.StartDate),
			Body:      fmt.Sprintf("Your %s leave begins on %s.", lr.LeaveType, lr.StartDate),
			RequestID: lr.ID, CreatedAt: time.Now().UTC(),
			ReminderKey: key,
		})
		h.Store.CreateNotification(&models.Notification{
			ID: store.NewID(), Kind: models.NotifReminder, Channel: "inapp",
			ToUserID: u.ID, Subject: "Upcoming leave",
			Body:      fmt.Sprintf("Your %s leave starts %s.", lr.LeaveType, lr.StartDate),
			RequestID: lr.ID, CreatedAt: time.Now().UTC(),
			ReminderKey: key + ":inapp",
		})
	}
}

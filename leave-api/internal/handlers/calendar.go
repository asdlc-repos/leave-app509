package handlers

import (
	"net/http"
	"sort"
	"time"

	"github.com/asdlc/leave-api/internal/middleware"
	"github.com/asdlc/leave-api/internal/models"
)

type calendarEvent struct {
	RequestID   string  `json:"requestId"`
	UserID      string  `json:"userId"`
	UserName    string  `json:"userName"`
	Department  string  `json:"department"`
	Team        string  `json:"team"`
	LeaveType   string  `json:"leaveType"`
	StartDate   string  `json:"startDate"`
	EndDate     string  `json:"endDate"`
	Days        float64 `json:"days"`
	Status      string  `json:"status"`
}

type capacityDay struct {
	Date        string  `json:"date"`
	Total       int     `json:"total"`
	OnLeave     int     `json:"onLeave"`
	CapacityPct float64 `json:"capacityPct"`
}

type calendarResponse struct {
	Events   []calendarEvent `json:"events"`
	Capacity []capacityDay   `json:"capacity"`
}

func (h *Handlers) Calendar(w http.ResponseWriter, r *http.Request) {
	_, ok := middleware.GetPrincipal(r)
	if !ok {
		writeErr(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	q := r.URL.Query()
	filterDept := q.Get("department")
	filterTeam := q.Get("team")
	filterUser := q.Get("userId")
	writeJSON(w, http.StatusOK, h.buildCalendar(filterDept, filterTeam, filterUser, q.Get("from"), q.Get("to")))
}

func (h *Handlers) buildCalendar(dept, team, userID, fromStr, toStr string) calendarResponse {
	now := time.Now().UTC().Truncate(24 * time.Hour)
	from := now.AddDate(0, 0, -30)
	to := now.AddDate(0, 0, 90)
	if fromStr != "" {
		if t, err := parseDate(fromStr); err == nil {
			from = t
		}
	}
	if toStr != "" {
		if t, err := parseDate(toStr); err == nil {
			to = t
		}
	}

	users := h.Store.ListUsers()
	userByID := map[string]*models.User{}
	for _, u := range users {
		userByID[u.ID] = u
	}

	events := []calendarEvent{}
	for _, lr := range h.Store.ListLeaveRequests() {
		if lr.Status != models.StatusApproved {
			continue
		}
		u := userByID[lr.UserID]
		if u == nil {
			continue
		}
		if dept != "" && u.Department != dept {
			continue
		}
		if team != "" && u.Team != team {
			continue
		}
		if userID != "" && u.ID != userID {
			continue
		}
		s, err1 := parseDate(lr.StartDate)
		e, err2 := parseDate(lr.EndDate)
		if err1 != nil || err2 != nil {
			continue
		}
		if !rangesOverlap(s, e, from, to) {
			continue
		}
		events = append(events, calendarEvent{
			RequestID: lr.ID, UserID: u.ID, UserName: u.Name,
			Department: u.Department, Team: u.Team,
			LeaveType: lr.LeaveType, StartDate: lr.StartDate, EndDate: lr.EndDate,
			Days: lr.Days, Status: string(lr.Status),
		})
	}
	sort.Slice(events, func(i, j int) bool { return events[i].StartDate < events[j].StartDate })

	// Capacity: count matching users in the filter scope per day vs on-leave.
	scopeUsers := []*models.User{}
	for _, u := range users {
		if dept != "" && u.Department != dept {
			continue
		}
		if team != "" && u.Team != team {
			continue
		}
		if userID != "" && u.ID != userID {
			continue
		}
		if u.Role == models.RoleHR {
			continue
		}
		scopeUsers = append(scopeUsers, u)
	}
	total := len(scopeUsers)

	capacity := []capacityDay{}
	for d := from; !d.After(to); d = d.AddDate(0, 0, 1) {
		if d.Weekday() == time.Saturday || d.Weekday() == time.Sunday {
			continue
		}
		onLeave := 0
		for _, u := range scopeUsers {
			for _, lr := range h.Store.ListLeaveRequests() {
				if lr.UserID != u.ID || lr.Status != models.StatusApproved {
					continue
				}
				s, _ := parseDate(lr.StartDate)
				e, _ := parseDate(lr.EndDate)
				if !d.Before(s) && !d.After(e) {
					onLeave++
					break
				}
			}
		}
		pct := 100.0
		if total > 0 {
			pct = float64(total-onLeave) / float64(total) * 100.0
		}
		capacity = append(capacity, capacityDay{
			Date:  d.Format("2006-01-02"),
			Total: total, OnLeave: onLeave, CapacityPct: pct,
		})
	}

	return calendarResponse{Events: events, Capacity: capacity}
}

package handlers

import (
	"net/http"
	"sort"

	"github.com/asdlc/leave-api/internal/middleware"
	"github.com/asdlc/leave-api/internal/models"
)

func (h *Handlers) ManagerQueue(w http.ResponseWriter, r *http.Request) {
	p, _ := middleware.GetPrincipal(r)
	if p.Role != string(models.RoleManager) && p.Role != string(models.RoleHR) {
		writeErr(w, http.StatusForbidden, "managers or HR only")
		return
	}
	out := []*models.LeaveRequest{}
	for _, lr := range h.Store.ListLeaveRequests() {
		if lr.Status != models.StatusPending {
			continue
		}
		if p.Role == string(models.RoleManager) && lr.ManagerID != p.UserID {
			continue
		}
		out = append(out, lr)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt.Before(out[j].CreatedAt) })
	writeJSON(w, http.StatusOK, out)
}

func (h *Handlers) ManagerTeamCalendar(w http.ResponseWriter, r *http.Request) {
	p, _ := middleware.GetPrincipal(r)
	if p.Role != string(models.RoleManager) && p.Role != string(models.RoleHR) {
		writeErr(w, http.StatusForbidden, "managers or HR only")
		return
	}
	// Resolve the manager's team members (or all users for HR).
	var team, dept string
	if p.Role == string(models.RoleManager) {
		if mgr, ok := h.Store.GetUser(p.UserID); ok {
			team = mgr.Team
			dept = mgr.Department
		}
	} else {
		dept = r.URL.Query().Get("department")
		team = r.URL.Query().Get("team")
	}
	writeJSON(w, http.StatusOK, h.buildCalendar(dept, team, "", r.URL.Query().Get("from"), r.URL.Query().Get("to")))
}

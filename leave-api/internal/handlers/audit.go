package handlers

import (
	"net/http"
	"sort"

	"github.com/asdlc/leave-api/internal/middleware"
	"github.com/asdlc/leave-api/internal/models"
)

func (h *Handlers) AuditForEmployee(w http.ResponseWriter, r *http.Request) {
	p, _ := middleware.GetPrincipal(r)
	employeeID := r.PathValue("employeeId")
	// Employees can only view their own audit; HR and the user's manager can view others.
	if p.Role == string(models.RoleEmployee) && p.UserID != employeeID {
		writeErr(w, http.StatusForbidden, "cannot view another user's audit")
		return
	}
	if p.Role == string(models.RoleManager) {
		if u, ok := h.Store.GetUser(employeeID); !ok || (u.ManagerID != p.UserID && u.ID != p.UserID) {
			writeErr(w, http.StatusForbidden, "not your team member")
			return
		}
	}
	entries := h.Store.ListAuditForEmployee(employeeID)
	sort.Slice(entries, func(i, j int) bool { return entries[i].Timestamp.After(entries[j].Timestamp) })
	writeJSON(w, http.StatusOK, entries)
}

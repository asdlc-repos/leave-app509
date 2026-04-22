package handlers

import (
	"net/http"
	"sort"

	"github.com/asdlc/leave-api/internal/middleware"
	"github.com/asdlc/leave-api/internal/models"
	"github.com/asdlc/leave-api/internal/store"
)

type leaveTypeInput struct {
	Name           string `json:"name"`
	DefaultDays    int    `json:"defaultDays"`
	NoticeDays     int    `json:"noticeDays"`
	RequiresReason bool   `json:"requiresReason"`
	Active         *bool  `json:"active,omitempty"`
}

func (h *Handlers) ListLeaveTypes(w http.ResponseWriter, r *http.Request) {
	types := h.Store.ListLeaveTypes()
	sort.Slice(types, func(i, j int) bool { return types[i].Name < types[j].Name })
	writeJSON(w, http.StatusOK, types)
}

func (h *Handlers) CreateLeaveType(w http.ResponseWriter, r *http.Request) {
	p, _ := middleware.GetPrincipal(r)
	if p.Role != string(models.RoleHR) {
		writeErr(w, http.StatusForbidden, "only HR may create leave types")
		return
	}
	var in leaveTypeInput
	if err := decodeJSON(r, &in); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid json body")
		return
	}
	if in.Name == "" {
		writeErr(w, http.StatusBadRequest, "name is required")
		return
	}
	active := true
	if in.Active != nil {
		active = *in.Active
	}
	lt := &models.LeaveType{
		ID:             "lt-" + store.NewID(),
		Name:           in.Name,
		DefaultDays:    in.DefaultDays,
		NoticeDays:     in.NoticeDays,
		RequiresReason: in.RequiresReason,
		Active:         active,
	}
	h.Store.CreateLeaveType(lt)
	writeJSON(w, http.StatusCreated, lt)
}

func (h *Handlers) UpdateLeaveType(w http.ResponseWriter, r *http.Request) {
	p, _ := middleware.GetPrincipal(r)
	if p.Role != string(models.RoleHR) {
		writeErr(w, http.StatusForbidden, "only HR may update leave types")
		return
	}
	id := r.PathValue("id")
	existing, ok := h.Store.GetLeaveType(id)
	if !ok {
		writeErr(w, http.StatusNotFound, "leave type not found")
		return
	}
	var in leaveTypeInput
	if err := decodeJSON(r, &in); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid json body")
		return
	}
	updated := *existing
	if in.Name != "" {
		updated.Name = in.Name
	}
	if in.DefaultDays != 0 {
		updated.DefaultDays = in.DefaultDays
	}
	updated.NoticeDays = in.NoticeDays
	updated.RequiresReason = in.RequiresReason
	if in.Active != nil {
		updated.Active = *in.Active
	}
	h.Store.UpdateLeaveType(&updated)
	writeJSON(w, http.StatusOK, updated)
}

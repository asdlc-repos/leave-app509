package handlers

import (
	"net/http"
	"sort"
	"time"

	"github.com/asdlc/leave-api/internal/middleware"
	"github.com/asdlc/leave-api/internal/models"
	"github.com/asdlc/leave-api/internal/store"
)

type blackoutInput struct {
	Name        string `json:"name"`
	StartDate   string `json:"startDate"`
	EndDate     string `json:"endDate"`
	Department  string `json:"department"`
	Description string `json:"description"`
}

func (h *Handlers) ListBlackouts(w http.ResponseWriter, r *http.Request) {
	bs := h.Store.ListBlackouts()
	sort.Slice(bs, func(i, j int) bool { return bs[i].StartDate < bs[j].StartDate })
	writeJSON(w, http.StatusOK, bs)
}

func (h *Handlers) CreateBlackout(w http.ResponseWriter, r *http.Request) {
	p, _ := middleware.GetPrincipal(r)
	if p.Role != string(models.RoleHR) {
		writeErr(w, http.StatusForbidden, "HR only")
		return
	}
	var in blackoutInput
	if err := decodeJSON(r, &in); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid json body")
		return
	}
	if in.Name == "" || in.StartDate == "" || in.EndDate == "" {
		writeErr(w, http.StatusBadRequest, "name, startDate and endDate are required")
		return
	}
	if _, err := parseDate(in.StartDate); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid startDate")
		return
	}
	if _, err := parseDate(in.EndDate); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid endDate")
		return
	}
	b := &models.Blackout{
		ID:          "bo-" + store.NewID(),
		Name:        in.Name,
		StartDate:   in.StartDate,
		EndDate:     in.EndDate,
		Department:  in.Department,
		Description: in.Description,
	}
	h.Store.CreateBlackout(b)
	h.Store.AppendAudit(&models.AuditEntry{
		ID: store.NewID(), EntityType: "blackout", EntityID: b.ID,
		ActorID: p.UserID, ActorName: p.Name, Action: "created",
		Details: map[string]interface{}{"name": b.Name, "department": b.Department},
		Timestamp: time.Now().UTC(),
	})
	writeJSON(w, http.StatusCreated, b)
}

package handlers

import (
	"net/http"
	"sort"
	"time"

	"github.com/asdlc/leave-api/internal/middleware"
	"github.com/asdlc/leave-api/internal/models"
	"github.com/asdlc/leave-api/internal/store"
)

func (h *Handlers) ListUsers(w http.ResponseWriter, r *http.Request) {
	users := h.Store.ListUsers()
	out := make([]userDTO, 0, len(users))
	for _, u := range users {
		out = append(out, userDTO{
			ID: u.ID, Email: u.Email, Name: u.Name, Role: string(u.Role),
			Department: u.Department, Team: u.Team, ManagerID: u.ManagerID,
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	writeJSON(w, http.StatusOK, out)
}

func (h *Handlers) GetBalances(w http.ResponseWriter, r *http.Request) {
	userID := r.PathValue("id")
	p, _ := middleware.GetPrincipal(r)
	// Employees may only view their own balances; managers and HR can view others.
	if p.Role == string(models.RoleEmployee) && p.UserID != userID {
		writeErr(w, http.StatusForbidden, "cannot view other users' balances")
		return
	}
	if _, ok := h.Store.GetUser(userID); !ok {
		writeErr(w, http.StatusNotFound, "user not found")
		return
	}
	balances := h.Store.GetBalances(userID)
	// Decorate with leave type name for convenience.
	type balanceDTO struct {
		UserID      string  `json:"userId"`
		LeaveTypeID string  `json:"leaveTypeId"`
		LeaveType   string  `json:"leaveType"`
		Total       float64 `json:"total"`
		Used        float64 `json:"used"`
		Reserved    float64 `json:"reserved"`
		Available   float64 `json:"available"`
	}
	out := make([]balanceDTO, 0, len(balances))
	for _, b := range balances {
		name := ""
		if lt, ok := h.Store.GetLeaveType(b.LeaveTypeID); ok {
			name = lt.Name
		}
		out = append(out, balanceDTO{
			UserID: b.UserID, LeaveTypeID: b.LeaveTypeID, LeaveType: name,
			Total: b.Total, Used: b.Used, Reserved: b.Reserved, Available: b.Available,
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].LeaveType < out[j].LeaveType })
	writeJSON(w, http.StatusOK, out)
}

type adjustRequest struct {
	LeaveTypeID string  `json:"leaveTypeId"`
	Delta       float64 `json:"delta"`
	Note        string  `json:"note"`
}

func (h *Handlers) AdjustBalance(w http.ResponseWriter, r *http.Request) {
	userID := r.PathValue("id")
	p, _ := middleware.GetPrincipal(r)
	if p.Role != string(models.RoleHR) {
		writeErr(w, http.StatusForbidden, "only HR may adjust balances")
		return
	}
	var req adjustRequest
	if err := decodeJSON(r, &req); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid json body")
		return
	}
	if req.Note == "" {
		writeErr(w, http.StatusBadRequest, "note is required for audit")
		return
	}
	if _, ok := h.Store.GetUser(userID); !ok {
		writeErr(w, http.StatusNotFound, "user not found")
		return
	}
	if _, ok := h.Store.GetLeaveType(req.LeaveTypeID); !ok {
		writeErr(w, http.StatusBadRequest, "unknown leaveTypeId")
		return
	}
	balance, _ := h.Store.AdjustBalance(userID, req.LeaveTypeID, req.Delta)
	h.Store.AppendAudit(&models.AuditEntry{
		ID:         store.NewID(),
		EntityType: "user",
		EntityID:   userID,
		ActorID:    p.UserID,
		ActorName:  p.Name,
		Action:     "balance_adjusted",
		Note:       req.Note,
		Details: map[string]interface{}{
			"leaveTypeId": req.LeaveTypeID,
			"delta":       req.Delta,
		},
		Timestamp: time.Now().UTC(),
	})
	// In-app notification for the affected user.
	if u, ok := h.Store.GetUser(userID); ok {
		h.Store.CreateNotification(&models.Notification{
			ID:        store.NewID(),
			Kind:      models.NotifAdjustment,
			Channel:   "inapp",
			ToUserID:  u.ID,
			ToEmail:   u.Email,
			Subject:   "Leave balance adjusted",
			Body:      req.Note,
			CreatedAt: time.Now().UTC(),
		})
	}
	writeJSON(w, http.StatusOK, balance)
}

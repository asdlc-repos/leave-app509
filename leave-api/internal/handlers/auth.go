package handlers

import (
	"net/http"

	"github.com/asdlc/leave-api/internal/middleware"
)

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type loginResponse struct {
	Token     string `json:"token"`
	ExpiresIn int    `json:"expiresIn"`
	User      userDTO `json:"user"`
}

type userDTO struct {
	ID         string `json:"id"`
	Email      string `json:"email"`
	Name       string `json:"name"`
	Role       string `json:"role"`
	Department string `json:"department"`
	Team       string `json:"team"`
	ManagerID  string `json:"managerId,omitempty"`
}

func (h *Handlers) Login(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := decodeJSON(r, &req); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid json body")
		return
	}
	u, ok := h.Store.GetUserByEmail(req.Email)
	if !ok || u.Password != req.Password {
		writeErr(w, http.StatusUnauthorized, "invalid credentials")
		return
	}
	token := h.Signer.Issue(u.ID, string(u.Role), u.Name)
	writeJSON(w, http.StatusOK, loginResponse{
		Token:     token,
		ExpiresIn: int(h.Signer.TTL().Seconds()),
		User: userDTO{
			ID: u.ID, Email: u.Email, Name: u.Name, Role: string(u.Role),
			Department: u.Department, Team: u.Team, ManagerID: u.ManagerID,
		},
	})
}

func (h *Handlers) Logout(w http.ResponseWriter, r *http.Request) {
	// Stateless JWT — client discards token. Return 204.
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handlers) Me(w http.ResponseWriter, r *http.Request) {
	p, ok := middleware.GetPrincipal(r)
	if !ok {
		writeErr(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	u, ok := h.Store.GetUser(p.UserID)
	if !ok {
		writeErr(w, http.StatusNotFound, "user not found")
		return
	}
	writeJSON(w, http.StatusOK, userDTO{
		ID: u.ID, Email: u.Email, Name: u.Name, Role: string(u.Role),
		Department: u.Department, Team: u.Team, ManagerID: u.ManagerID,
	})
}

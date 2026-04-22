package handlers

import (
	"encoding/base64"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/asdlc/leave-api/internal/middleware"
	"github.com/asdlc/leave-api/internal/models"
	"github.com/asdlc/leave-api/internal/store"
)

type createLeaveRequestInput struct {
	LeaveTypeID string `json:"leaveTypeId"`
	StartDate   string `json:"startDate"`
	EndDate     string `json:"endDate"`
	Reason      string `json:"reason"`
	UserID      string `json:"userId,omitempty"`
}

func (h *Handlers) ListLeaveRequests(w http.ResponseWriter, r *http.Request) {
	p, _ := middleware.GetPrincipal(r)
	q := r.URL.Query()
	filterUser := q.Get("userId")
	filterStatus := q.Get("status")

	all := h.Store.ListLeaveRequests()
	var filtered []*models.LeaveRequest
	for _, lr := range all {
		switch p.Role {
		case string(models.RoleEmployee):
			if lr.UserID != p.UserID {
				continue
			}
		case string(models.RoleManager):
			// Managers see requests they manage, plus their own.
			if lr.ManagerID != p.UserID && lr.UserID != p.UserID {
				continue
			}
		}
		if filterUser != "" && lr.UserID != filterUser {
			continue
		}
		if filterStatus != "" && string(lr.Status) != filterStatus {
			continue
		}
		filtered = append(filtered, lr)
	}
	sort.Slice(filtered, func(i, j int) bool { return filtered[i].CreatedAt.After(filtered[j].CreatedAt) })
	writeJSON(w, http.StatusOK, filtered)
}

func (h *Handlers) CreateLeaveRequest(w http.ResponseWriter, r *http.Request) {
	p, _ := middleware.GetPrincipal(r)
	var in createLeaveRequestInput
	if err := decodeJSON(r, &in); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid json body")
		return
	}
	targetUserID := p.UserID
	if in.UserID != "" {
		// HR can create on behalf of a user.
		if p.Role != string(models.RoleHR) && in.UserID != p.UserID {
			writeErr(w, http.StatusForbidden, "cannot create request for another user")
			return
		}
		targetUserID = in.UserID
	}
	user, ok := h.Store.GetUser(targetUserID)
	if !ok {
		writeErr(w, http.StatusBadRequest, "unknown user")
		return
	}
	lt, ok := h.Store.GetLeaveType(in.LeaveTypeID)
	if !ok || !lt.Active {
		writeErr(w, http.StatusBadRequest, "unknown or inactive leave type")
		return
	}
	start, err := parseDate(in.StartDate)
	if err != nil {
		writeErr(w, http.StatusBadRequest, "invalid startDate (YYYY-MM-DD required)")
		return
	}
	end, err := parseDate(in.EndDate)
	if err != nil {
		writeErr(w, http.StatusBadRequest, "invalid endDate (YYYY-MM-DD required)")
		return
	}
	if end.Before(start) {
		writeErr(w, http.StatusBadRequest, "endDate is before startDate")
		return
	}
	if lt.RequiresReason && strings.TrimSpace(in.Reason) == "" {
		writeErr(w, http.StatusBadRequest, "reason is required for this leave type")
		return
	}
	days := workingDays(start, end)
	if days <= 0 {
		writeErr(w, http.StatusBadRequest, "leave must include at least one business day")
		return
	}
	// Notice-period check (skip for sick leave where notice=0).
	if lt.NoticeDays > 0 {
		today := time.Now().UTC().Truncate(24 * time.Hour)
		if start.Sub(today) < time.Duration(lt.NoticeDays)*24*time.Hour {
			writeErr(w, http.StatusBadRequest, fmt.Sprintf("this leave type requires at least %d days notice", lt.NoticeDays))
			return
		}
	}
	// Blackout check.
	for _, b := range h.Store.ListBlackouts() {
		bs, err1 := parseDate(b.StartDate)
		be, err2 := parseDate(b.EndDate)
		if err1 != nil || err2 != nil {
			continue
		}
		if b.Department != "" && b.Department != user.Department {
			continue
		}
		if rangesOverlap(start, end, bs, be) {
			writeErr(w, http.StatusBadRequest, "dates fall within a blackout period: "+b.Name)
			return
		}
	}
	// Overlap check against the same user's approved or pending requests.
	for _, other := range h.Store.ListLeaveRequests() {
		if other.UserID != targetUserID {
			continue
		}
		if other.Status != models.StatusApproved && other.Status != models.StatusPending {
			continue
		}
		os, _ := parseDate(other.StartDate)
		oe, _ := parseDate(other.EndDate)
		if rangesOverlap(start, end, os, oe) {
			writeErr(w, http.StatusBadRequest, "overlaps with an existing request")
			return
		}
	}
	// Reserve balance.
	if !h.Store.ReserveBalance(targetUserID, lt.ID, days) {
		writeErr(w, http.StatusBadRequest, "insufficient balance")
		return
	}
	now := time.Now().UTC()
	lr := &models.LeaveRequest{
		ID:          "lr-" + store.NewID(),
		UserID:      targetUserID,
		UserName:    user.Name,
		LeaveTypeID: lt.ID,
		LeaveType:   lt.Name,
		StartDate:   in.StartDate,
		EndDate:     in.EndDate,
		Days:        days,
		Reason:      in.Reason,
		Status:      models.StatusPending,
		ManagerID:   user.ManagerID,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	h.Store.CreateLeaveRequest(lr)
	// Audit
	h.Store.AppendAudit(&models.AuditEntry{
		ID: store.NewID(), EntityType: "leave_request", EntityID: lr.ID,
		ActorID: p.UserID, ActorName: p.Name, Action: "created",
		Details: map[string]interface{}{"days": days, "leaveType": lt.Name},
		Timestamp: now,
	})
	// Notifications: email to manager, inapp to user.
	if user.ManagerID != "" {
		if mgr, ok := h.Store.GetUser(user.ManagerID); ok {
			h.Store.CreateNotification(&models.Notification{
				ID: store.NewID(), Kind: models.NotifSubmitted, Channel: "email",
				ToUserID: mgr.ID, ToEmail: mgr.Email,
				Subject:   fmt.Sprintf("Leave request from %s", user.Name),
				Body:      fmt.Sprintf("%s requested %s from %s to %s (%g days).", user.Name, lt.Name, lr.StartDate, lr.EndDate, days),
				RequestID: lr.ID,
				CreatedAt: now,
			})
		}
	}
	h.Store.CreateNotification(&models.Notification{
		ID: store.NewID(), Kind: models.NotifSubmitted, Channel: "inapp",
		ToUserID: user.ID, ToEmail: user.Email,
		Subject:   "Leave request submitted",
		Body:      fmt.Sprintf("Your %s request (%s to %s) is pending approval.", lt.Name, lr.StartDate, lr.EndDate),
		RequestID: lr.ID,
		CreatedAt: now,
	})
	writeJSON(w, http.StatusCreated, lr)
}

func (h *Handlers) GetLeaveRequest(w http.ResponseWriter, r *http.Request) {
	p, _ := middleware.GetPrincipal(r)
	id := r.PathValue("id")
	lr, ok := h.Store.GetLeaveRequest(id)
	if !ok {
		writeErr(w, http.StatusNotFound, "leave request not found")
		return
	}
	if !h.canViewRequest(p, lr) {
		writeErr(w, http.StatusForbidden, "not permitted")
		return
	}
	writeJSON(w, http.StatusOK, lr)
}

func (h *Handlers) canViewRequest(p middleware.Principal, lr *models.LeaveRequest) bool {
	switch p.Role {
	case string(models.RoleHR):
		return true
	case string(models.RoleManager):
		return lr.ManagerID == p.UserID || lr.UserID == p.UserID
	case string(models.RoleEmployee):
		return lr.UserID == p.UserID
	}
	return false
}

func (h *Handlers) CancelLeaveRequest(w http.ResponseWriter, r *http.Request) {
	p, _ := middleware.GetPrincipal(r)
	id := r.PathValue("id")
	lr, ok := h.Store.GetLeaveRequest(id)
	if !ok {
		writeErr(w, http.StatusNotFound, "leave request not found")
		return
	}
	if lr.UserID != p.UserID && p.Role != string(models.RoleHR) {
		writeErr(w, http.StatusForbidden, "only the requester or HR may cancel")
		return
	}
	if lr.Status != models.StatusPending {
		writeErr(w, http.StatusBadRequest, "only pending requests can be cancelled")
		return
	}
	lr.Status = models.StatusCancelled
	h.Store.UpdateLeaveRequest(lr)
	h.Store.ReleaseReserved(lr.UserID, lr.LeaveTypeID, lr.Days)
	now := time.Now().UTC()
	h.Store.AppendAudit(&models.AuditEntry{
		ID: store.NewID(), EntityType: "leave_request", EntityID: lr.ID,
		ActorID: p.UserID, ActorName: p.Name, Action: "cancelled",
		Timestamp: now,
	})
	h.Store.CreateNotification(&models.Notification{
		ID: store.NewID(), Kind: models.NotifCancelled, Channel: "inapp",
		ToUserID: lr.UserID, Subject: "Leave request cancelled",
		Body:      fmt.Sprintf("Your %s request (%s to %s) was cancelled.", lr.LeaveType, lr.StartDate, lr.EndDate),
		RequestID: lr.ID, CreatedAt: now,
	})
	if lr.ManagerID != "" {
		if mgr, ok := h.Store.GetUser(lr.ManagerID); ok {
			h.Store.CreateNotification(&models.Notification{
				ID: store.NewID(), Kind: models.NotifCancelled, Channel: "email",
				ToUserID: mgr.ID, ToEmail: mgr.Email,
				Subject:   "Leave request cancelled",
				Body:      fmt.Sprintf("%s cancelled their %s request (%s to %s).", lr.UserName, lr.LeaveType, lr.StartDate, lr.EndDate),
				RequestID: lr.ID, CreatedAt: now,
			})
		}
	}
	writeJSON(w, http.StatusOK, lr)
}

type reviewInput struct {
	Comment string `json:"comment"`
}

func (h *Handlers) ApproveLeaveRequest(w http.ResponseWriter, r *http.Request) {
	p, _ := middleware.GetPrincipal(r)
	id := r.PathValue("id")
	lr, ok := h.Store.GetLeaveRequest(id)
	if !ok {
		writeErr(w, http.StatusNotFound, "leave request not found")
		return
	}
	if p.Role != string(models.RoleManager) && p.Role != string(models.RoleHR) {
		writeErr(w, http.StatusForbidden, "only managers or HR may approve")
		return
	}
	if p.Role == string(models.RoleManager) && lr.ManagerID != p.UserID {
		writeErr(w, http.StatusForbidden, "not your team member")
		return
	}
	if lr.Status != models.StatusPending {
		writeErr(w, http.StatusBadRequest, "only pending requests can be approved")
		return
	}
	var in reviewInput
	_ = decodeJSON(r, &in)
	now := time.Now().UTC()
	lr.Status = models.StatusApproved
	lr.ReviewedBy = p.UserID
	lr.ReviewedAt = &now
	lr.Comment = in.Comment
	h.Store.UpdateLeaveRequest(lr)
	h.Store.CommitReserved(lr.UserID, lr.LeaveTypeID, lr.Days)
	h.Store.AppendAudit(&models.AuditEntry{
		ID: store.NewID(), EntityType: "leave_request", EntityID: lr.ID,
		ActorID: p.UserID, ActorName: p.Name, Action: "approved",
		Note: in.Comment, Timestamp: now,
	})
	if u, ok := h.Store.GetUser(lr.UserID); ok {
		h.Store.CreateNotification(&models.Notification{
			ID: store.NewID(), Kind: models.NotifApproved, Channel: "email",
			ToUserID: u.ID, ToEmail: u.Email,
			Subject:   "Leave request approved",
			Body:      fmt.Sprintf("Your %s request (%s to %s) has been approved.", lr.LeaveType, lr.StartDate, lr.EndDate),
			RequestID: lr.ID, CreatedAt: now,
		})
		h.Store.CreateNotification(&models.Notification{
			ID: store.NewID(), Kind: models.NotifApproved, Channel: "inapp",
			ToUserID: u.ID, Subject: "Leave approved",
			Body:      fmt.Sprintf("Your %s request (%s to %s) was approved.", lr.LeaveType, lr.StartDate, lr.EndDate),
			RequestID: lr.ID, CreatedAt: now,
		})
	}
	writeJSON(w, http.StatusOK, lr)
}

func (h *Handlers) RejectLeaveRequest(w http.ResponseWriter, r *http.Request) {
	p, _ := middleware.GetPrincipal(r)
	id := r.PathValue("id")
	lr, ok := h.Store.GetLeaveRequest(id)
	if !ok {
		writeErr(w, http.StatusNotFound, "leave request not found")
		return
	}
	if p.Role != string(models.RoleManager) && p.Role != string(models.RoleHR) {
		writeErr(w, http.StatusForbidden, "only managers or HR may reject")
		return
	}
	if p.Role == string(models.RoleManager) && lr.ManagerID != p.UserID {
		writeErr(w, http.StatusForbidden, "not your team member")
		return
	}
	if lr.Status != models.StatusPending {
		writeErr(w, http.StatusBadRequest, "only pending requests can be rejected")
		return
	}
	var in reviewInput
	if err := decodeJSON(r, &in); err != nil || strings.TrimSpace(in.Comment) == "" {
		writeErr(w, http.StatusBadRequest, "comment is required for rejection")
		return
	}
	now := time.Now().UTC()
	lr.Status = models.StatusRejected
	lr.ReviewedBy = p.UserID
	lr.ReviewedAt = &now
	lr.Comment = in.Comment
	h.Store.UpdateLeaveRequest(lr)
	h.Store.ReleaseReserved(lr.UserID, lr.LeaveTypeID, lr.Days)
	h.Store.AppendAudit(&models.AuditEntry{
		ID: store.NewID(), EntityType: "leave_request", EntityID: lr.ID,
		ActorID: p.UserID, ActorName: p.Name, Action: "rejected",
		Note: in.Comment, Timestamp: now,
	})
	if u, ok := h.Store.GetUser(lr.UserID); ok {
		h.Store.CreateNotification(&models.Notification{
			ID: store.NewID(), Kind: models.NotifRejected, Channel: "email",
			ToUserID: u.ID, ToEmail: u.Email,
			Subject:   "Leave request rejected",
			Body:      fmt.Sprintf("Your %s request (%s to %s) was rejected: %s", lr.LeaveType, lr.StartDate, lr.EndDate, in.Comment),
			RequestID: lr.ID, CreatedAt: now,
		})
		h.Store.CreateNotification(&models.Notification{
			ID: store.NewID(), Kind: models.NotifRejected, Channel: "inapp",
			ToUserID: u.ID, Subject: "Leave rejected",
			Body:      fmt.Sprintf("Your %s request (%s to %s) was rejected.", lr.LeaveType, lr.StartDate, lr.EndDate),
			RequestID: lr.ID, CreatedAt: now,
		})
	}
	writeJSON(w, http.StatusOK, lr)
}

type attachmentInput struct {
	Filename string `json:"filename"`
	MimeType string `json:"mimeType"`
	Data     string `json:"data"`
}

const maxAttachmentBytes = 5 * 1024 * 1024

var allowedMimes = map[string]bool{
	"application/pdf": true,
	"image/jpeg":      true,
	"image/jpg":       true,
	"image/png":       true,
}

func (h *Handlers) UploadAttachment(w http.ResponseWriter, r *http.Request) {
	p, _ := middleware.GetPrincipal(r)
	id := r.PathValue("id")
	lr, ok := h.Store.GetLeaveRequest(id)
	if !ok {
		writeErr(w, http.StatusNotFound, "leave request not found")
		return
	}
	if lr.UserID != p.UserID && p.Role != string(models.RoleHR) {
		writeErr(w, http.StatusForbidden, "cannot attach to another user's request")
		return
	}
	var in attachmentInput
	if err := decodeJSON(r, &in); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid json body")
		return
	}
	if in.Filename == "" || in.Data == "" {
		writeErr(w, http.StatusBadRequest, "filename and data are required")
		return
	}
	if !allowedMimes[strings.ToLower(in.MimeType)] {
		writeErr(w, http.StatusBadRequest, "mime type must be PDF, JPG, or PNG")
		return
	}
	// Strip data-URI prefix if present.
	data := in.Data
	if i := strings.Index(data, ","); strings.HasPrefix(data, "data:") && i > 0 {
		data = data[i+1:]
	}
	decoded, err := base64.StdEncoding.DecodeString(data)
	if err != nil {
		writeErr(w, http.StatusBadRequest, "data must be valid base64")
		return
	}
	if len(decoded) > maxAttachmentBytes {
		writeErr(w, http.StatusBadRequest, "attachment exceeds 5MB limit")
		return
	}
	att := models.Attachment{
		ID:       "att-" + store.NewID(),
		Filename: in.Filename,
		MimeType: strings.ToLower(in.MimeType),
		Size:     len(decoded),
		Data:     data,
	}
	lr.Attachments = append(lr.Attachments, att)
	h.Store.UpdateLeaveRequest(lr)
	h.Store.AppendAudit(&models.AuditEntry{
		ID: store.NewID(), EntityType: "leave_request", EntityID: lr.ID,
		ActorID: p.UserID, ActorName: p.Name, Action: "attachment_uploaded",
		Details:   map[string]interface{}{"filename": att.Filename, "size": att.Size},
		Timestamp: time.Now().UTC(),
	})
	// Omit raw data in response for brevity.
	respAtt := att
	respAtt.Data = ""
	writeJSON(w, http.StatusCreated, respAtt)
}

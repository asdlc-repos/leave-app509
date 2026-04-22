package models

import "time"

type Role string

const (
	RoleEmployee Role = "employee"
	RoleManager  Role = "manager"
	RoleHR       Role = "hr"
)

type User struct {
	ID         string `json:"id"`
	Email      string `json:"email"`
	Password   string `json:"-"`
	Name       string `json:"name"`
	Role       Role   `json:"role"`
	Department string `json:"department"`
	Team       string `json:"team"`
	ManagerID  string `json:"managerId,omitempty"`
}

type LeaveType struct {
	ID             string `json:"id"`
	Name           string `json:"name"`
	DefaultDays    int    `json:"defaultDays"`
	NoticeDays     int    `json:"noticeDays"`
	RequiresReason bool   `json:"requiresReason"`
	Active         bool   `json:"active"`
}

type Balance struct {
	UserID      string  `json:"userId"`
	LeaveTypeID string  `json:"leaveTypeId"`
	Total       float64 `json:"total"`
	Used        float64 `json:"used"`
	Reserved    float64 `json:"reserved"`
	Available   float64 `json:"available"`
}

type LeaveStatus string

const (
	StatusPending   LeaveStatus = "pending"
	StatusApproved  LeaveStatus = "approved"
	StatusRejected  LeaveStatus = "rejected"
	StatusCancelled LeaveStatus = "cancelled"
)

type Attachment struct {
	ID       string `json:"id"`
	Filename string `json:"filename"`
	MimeType string `json:"mimeType"`
	Size     int    `json:"size"`
	Data     string `json:"data,omitempty"`
}

type LeaveRequest struct {
	ID          string       `json:"id"`
	UserID      string       `json:"userId"`
	UserName    string       `json:"userName"`
	LeaveTypeID string       `json:"leaveTypeId"`
	LeaveType   string       `json:"leaveType"`
	StartDate   string       `json:"startDate"`
	EndDate     string       `json:"endDate"`
	Days        float64      `json:"days"`
	Reason      string       `json:"reason"`
	Status      LeaveStatus  `json:"status"`
	ManagerID   string       `json:"managerId,omitempty"`
	Comment     string       `json:"comment,omitempty"`
	CreatedAt   time.Time    `json:"createdAt"`
	UpdatedAt   time.Time    `json:"updatedAt"`
	ReviewedBy  string       `json:"reviewedBy,omitempty"`
	ReviewedAt  *time.Time   `json:"reviewedAt,omitempty"`
	Attachments []Attachment `json:"attachments,omitempty"`
}

type Blackout struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	StartDate   string `json:"startDate"`
	EndDate     string `json:"endDate"`
	Department  string `json:"department,omitempty"`
	Description string `json:"description,omitempty"`
}

type AuditEntry struct {
	ID         string                 `json:"id"`
	EntityType string                 `json:"entityType"`
	EntityID   string                 `json:"entityId"`
	ActorID    string                 `json:"actorId"`
	ActorName  string                 `json:"actorName"`
	Action     string                 `json:"action"`
	Details    map[string]interface{} `json:"details,omitempty"`
	Note       string                 `json:"note,omitempty"`
	Timestamp  time.Time              `json:"timestamp"`
}

type NotificationKind string

const (
	NotifSubmitted  NotificationKind = "submitted"
	NotifApproved   NotificationKind = "approved"
	NotifRejected   NotificationKind = "rejected"
	NotifCancelled  NotificationKind = "cancelled"
	NotifReminder   NotificationKind = "reminder"
	NotifAdjustment NotificationKind = "adjustment"
)

type Notification struct {
	ID         string           `json:"id"`
	Kind       NotificationKind `json:"kind"`
	Channel    string           `json:"channel"`
	ToUserID   string           `json:"toUserId"`
	ToEmail    string           `json:"toEmail"`
	Subject    string           `json:"subject"`
	Body       string           `json:"body"`
	RequestID  string           `json:"requestId,omitempty"`
	Sent       bool             `json:"sent"`
	Read       bool             `json:"read"`
	CreatedAt  time.Time        `json:"createdAt"`
	SentAt     *time.Time       `json:"sentAt,omitempty"`
	ReminderKey string          `json:"-"`
}

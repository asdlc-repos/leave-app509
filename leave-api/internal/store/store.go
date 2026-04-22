package store

import (
	"crypto/rand"
	"encoding/hex"
	"sync"
	"time"

	"github.com/asdlc/leave-api/internal/models"
)

// Store is a thread-safe in-memory store for all domain state.
type Store struct {
	mu sync.RWMutex

	users         map[string]*models.User
	leaveTypes    map[string]*models.LeaveType
	balances      map[string]map[string]*models.Balance // userID -> leaveTypeID
	leaveRequests map[string]*models.LeaveRequest
	blackouts     map[string]*models.Blackout
	audits        []*models.AuditEntry
	notifications map[string]*models.Notification
	reminderKeys  map[string]bool
}

func New() *Store {
	s := &Store{
		users:         make(map[string]*models.User),
		leaveTypes:    make(map[string]*models.LeaveType),
		balances:      make(map[string]map[string]*models.Balance),
		leaveRequests: make(map[string]*models.LeaveRequest),
		blackouts:     make(map[string]*models.Blackout),
		audits:        []*models.AuditEntry{},
		notifications: make(map[string]*models.Notification),
		reminderKeys:  make(map[string]bool),
	}
	s.seed()
	return s
}

func NewID() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// -------- Users --------

func (s *Store) GetUser(id string) (*models.User, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	u, ok := s.users[id]
	return u, ok
}

func (s *Store) GetUserByEmail(email string) (*models.User, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, u := range s.users {
		if u.Email == email {
			return u, true
		}
	}
	return nil, false
}

func (s *Store) ListUsers() []*models.User {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*models.User, 0, len(s.users))
	for _, u := range s.users {
		out = append(out, u)
	}
	return out
}

// -------- Leave types --------

func (s *Store) ListLeaveTypes() []*models.LeaveType {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*models.LeaveType, 0, len(s.leaveTypes))
	for _, lt := range s.leaveTypes {
		out = append(out, lt)
	}
	return out
}

func (s *Store) GetLeaveType(id string) (*models.LeaveType, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	lt, ok := s.leaveTypes[id]
	return lt, ok
}

func (s *Store) CreateLeaveType(lt *models.LeaveType) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.leaveTypes[lt.ID] = lt
	// ensure all users get a balance row for this new type
	for userID := range s.users {
		if _, ok := s.balances[userID]; !ok {
			s.balances[userID] = make(map[string]*models.Balance)
		}
		if _, ok := s.balances[userID][lt.ID]; !ok {
			s.balances[userID][lt.ID] = &models.Balance{
				UserID:      userID,
				LeaveTypeID: lt.ID,
				Total:       float64(lt.DefaultDays),
			}
		}
	}
}

func (s *Store) UpdateLeaveType(lt *models.LeaveType) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.leaveTypes[lt.ID]; !ok {
		return false
	}
	s.leaveTypes[lt.ID] = lt
	return true
}

// -------- Balances --------

func (s *Store) GetBalances(userID string) []*models.Balance {
	s.mu.RLock()
	defer s.mu.RUnlock()
	m := s.balances[userID]
	out := make([]*models.Balance, 0, len(m))
	for _, b := range m {
		copy := *b
		copy.Available = copy.Total - copy.Used - copy.Reserved
		out = append(out, &copy)
	}
	return out
}

func (s *Store) GetBalance(userID, leaveTypeID string) (*models.Balance, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	m := s.balances[userID]
	if m == nil {
		return nil, false
	}
	b, ok := m[leaveTypeID]
	if !ok {
		return nil, false
	}
	copy := *b
	copy.Available = copy.Total - copy.Used - copy.Reserved
	return &copy, true
}

func (s *Store) AdjustBalance(userID, leaveTypeID string, delta float64) (*models.Balance, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	m := s.balances[userID]
	if m == nil {
		s.balances[userID] = make(map[string]*models.Balance)
		m = s.balances[userID]
	}
	b, ok := m[leaveTypeID]
	if !ok {
		b = &models.Balance{UserID: userID, LeaveTypeID: leaveTypeID}
		m[leaveTypeID] = b
	}
	b.Total += delta
	copy := *b
	copy.Available = copy.Total - copy.Used - copy.Reserved
	return &copy, true
}

// -------- Leave requests --------

func (s *Store) CreateLeaveRequest(lr *models.LeaveRequest) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.leaveRequests[lr.ID] = lr
}

func (s *Store) GetLeaveRequest(id string) (*models.LeaveRequest, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	lr, ok := s.leaveRequests[id]
	return lr, ok
}

func (s *Store) ListLeaveRequests() []*models.LeaveRequest {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*models.LeaveRequest, 0, len(s.leaveRequests))
	for _, lr := range s.leaveRequests {
		out = append(out, lr)
	}
	return out
}

func (s *Store) UpdateLeaveRequest(lr *models.LeaveRequest) {
	s.mu.Lock()
	defer s.mu.Unlock()
	lr.UpdatedAt = time.Now().UTC()
	s.leaveRequests[lr.ID] = lr
}

// ReserveBalance adds to reserved; returns false if insufficient.
func (s *Store) ReserveBalance(userID, leaveTypeID string, days float64) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	m := s.balances[userID]
	if m == nil {
		return false
	}
	b, ok := m[leaveTypeID]
	if !ok {
		return false
	}
	if b.Total-b.Used-b.Reserved < days {
		return false
	}
	b.Reserved += days
	return true
}

// ReleaseReserved returns reserved balance (on cancel/reject).
func (s *Store) ReleaseReserved(userID, leaveTypeID string, days float64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	m := s.balances[userID]
	if m == nil {
		return
	}
	b, ok := m[leaveTypeID]
	if !ok {
		return
	}
	b.Reserved -= days
	if b.Reserved < 0 {
		b.Reserved = 0
	}
}

// CommitReserved moves reserved -> used (on approve).
func (s *Store) CommitReserved(userID, leaveTypeID string, days float64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	m := s.balances[userID]
	if m == nil {
		return
	}
	b, ok := m[leaveTypeID]
	if !ok {
		return
	}
	b.Reserved -= days
	if b.Reserved < 0 {
		b.Reserved = 0
	}
	b.Used += days
}

// -------- Blackouts --------

func (s *Store) ListBlackouts() []*models.Blackout {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*models.Blackout, 0, len(s.blackouts))
	for _, b := range s.blackouts {
		out = append(out, b)
	}
	return out
}

func (s *Store) CreateBlackout(b *models.Blackout) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.blackouts[b.ID] = b
}

// -------- Audit --------

func (s *Store) AppendAudit(e *models.AuditEntry) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.audits = append(s.audits, e)
}

func (s *Store) ListAuditForEntity(entityID string) []*models.AuditEntry {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := []*models.AuditEntry{}
	for _, a := range s.audits {
		if a.EntityID == entityID {
			out = append(out, a)
		}
	}
	return out
}

func (s *Store) ListAuditForEmployee(employeeID string) []*models.AuditEntry {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := []*models.AuditEntry{}
	// include direct employee-entity audits and any request-entity audits owned by the employee
	for _, a := range s.audits {
		if a.EntityType == "user" && a.EntityID == employeeID {
			out = append(out, a)
			continue
		}
		if a.EntityType == "leave_request" {
			if lr, ok := s.leaveRequests[a.EntityID]; ok && lr.UserID == employeeID {
				out = append(out, a)
			}
		}
	}
	return out
}

// -------- Notifications --------

func (s *Store) CreateNotification(n *models.Notification) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.notifications[n.ID] = n
	if n.ReminderKey != "" {
		s.reminderKeys[n.ReminderKey] = true
	}
}

func (s *Store) ReminderExists(key string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.reminderKeys[key]
}

func (s *Store) ListPendingEmailNotifications() []*models.Notification {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := []*models.Notification{}
	for _, n := range s.notifications {
		if n.Channel == "email" && !n.Sent {
			out = append(out, n)
		}
	}
	return out
}

func (s *Store) ListInAppForUser(userID string) []*models.Notification {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := []*models.Notification{}
	for _, n := range s.notifications {
		if n.Channel == "inapp" && n.ToUserID == userID {
			out = append(out, n)
		}
	}
	return out
}

func (s *Store) MarkNotificationSent(id string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	n, ok := s.notifications[id]
	if !ok {
		return false
	}
	n.Sent = true
	t := time.Now().UTC()
	n.SentAt = &t
	return true
}

func (s *Store) ListApprovedRequestsForReminder() []*models.LeaveRequest {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := []*models.LeaveRequest{}
	for _, lr := range s.leaveRequests {
		if lr.Status == models.StatusApproved {
			out = append(out, lr)
		}
	}
	return out
}

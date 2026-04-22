package store

import (
	"time"

	"github.com/asdlc/leave-api/internal/models"
)

func (s *Store) seed() {
	// Users
	users := []*models.User{
		{ID: "u-hr-1", Email: "hr@example.com", Password: "hr123", Name: "Hannah Reed", Role: models.RoleHR, Department: "People", Team: "HR"},
		{ID: "u-mgr-1", Email: "manager@example.com", Password: "manager123", Name: "Marcus Glover", Role: models.RoleManager, Department: "Engineering", Team: "Platform"},
		{ID: "u-mgr-2", Email: "manager2@example.com", Password: "manager123", Name: "Diana Price", Role: models.RoleManager, Department: "Engineering", Team: "Product"},
		{ID: "u-emp-1", Email: "employee@example.com", Password: "employee123", Name: "Ethan Park", Role: models.RoleEmployee, Department: "Engineering", Team: "Platform", ManagerID: "u-mgr-1"},
		{ID: "u-emp-2", Email: "alice@example.com", Password: "alice123", Name: "Alice Chen", Role: models.RoleEmployee, Department: "Engineering", Team: "Platform", ManagerID: "u-mgr-1"},
		{ID: "u-emp-3", Email: "bob@example.com", Password: "bob123", Name: "Bob Singh", Role: models.RoleEmployee, Department: "Engineering", Team: "Product", ManagerID: "u-mgr-2"},
		{ID: "u-emp-4", Email: "carol@example.com", Password: "carol123", Name: "Carol Nguyen", Role: models.RoleEmployee, Department: "Engineering", Team: "Product", ManagerID: "u-mgr-2"},
	}
	for _, u := range users {
		s.users[u.ID] = u
	}

	// Leave types
	types := []*models.LeaveType{
		{ID: "lt-annual", Name: "Annual", DefaultDays: 20, NoticeDays: 7, RequiresReason: false, Active: true},
		{ID: "lt-sick", Name: "Sick", DefaultDays: 10, NoticeDays: 0, RequiresReason: false, Active: true},
		{ID: "lt-personal", Name: "Personal", DefaultDays: 5, NoticeDays: 2, RequiresReason: true, Active: true},
	}
	for _, lt := range types {
		s.leaveTypes[lt.ID] = lt
	}

	// Balances
	for _, u := range users {
		s.balances[u.ID] = make(map[string]*models.Balance)
		for _, lt := range types {
			s.balances[u.ID][lt.ID] = &models.Balance{
				UserID:      u.ID,
				LeaveTypeID: lt.ID,
				Total:       float64(lt.DefaultDays),
			}
		}
	}

	// Sample blackouts
	yr := time.Now().UTC().Year()
	s.blackouts["bo-endofyear"] = &models.Blackout{
		ID:          "bo-endofyear",
		Name:        "End of Year Freeze",
		StartDate:   fmtDate(yr, 12, 20),
		EndDate:     fmtDate(yr, 12, 31),
		Description: "Year-end release freeze",
	}

	// Sample leave requests
	now := time.Now().UTC()
	sampleStart := now.AddDate(0, 0, 14)
	sampleEnd := now.AddDate(0, 0, 16)
	sr := &models.LeaveRequest{
		ID:          "lr-sample-1",
		UserID:      "u-emp-1",
		UserName:    "Ethan Park",
		LeaveTypeID: "lt-annual",
		LeaveType:   "Annual",
		StartDate:   sampleStart.Format("2006-01-02"),
		EndDate:     sampleEnd.Format("2006-01-02"),
		Days:        3,
		Reason:      "Family trip",
		Status:      models.StatusPending,
		ManagerID:   "u-mgr-1",
		CreatedAt:   now.Add(-24 * time.Hour),
		UpdatedAt:   now.Add(-24 * time.Hour),
	}
	s.leaveRequests[sr.ID] = sr
	s.balances["u-emp-1"]["lt-annual"].Reserved = 3

	sr2Start := now.AddDate(0, 0, 30)
	sr2End := now.AddDate(0, 0, 34)
	sr2 := &models.LeaveRequest{
		ID:          "lr-sample-2",
		UserID:      "u-emp-2",
		UserName:    "Alice Chen",
		LeaveTypeID: "lt-annual",
		LeaveType:   "Annual",
		StartDate:   sr2Start.Format("2006-01-02"),
		EndDate:     sr2End.Format("2006-01-02"),
		Days:        5,
		Reason:      "Vacation",
		Status:      models.StatusApproved,
		ManagerID:   "u-mgr-1",
		ReviewedBy:  "u-mgr-1",
		CreatedAt:   now.Add(-72 * time.Hour),
		UpdatedAt:   now.Add(-48 * time.Hour),
	}
	reviewed := now.Add(-48 * time.Hour)
	sr2.ReviewedAt = &reviewed
	s.leaveRequests[sr2.ID] = sr2
	s.balances["u-emp-2"]["lt-annual"].Used = 5

	sr3Start := now.AddDate(0, 0, -5)
	sr3End := now.AddDate(0, 0, -3)
	sr3 := &models.LeaveRequest{
		ID:          "lr-sample-3",
		UserID:      "u-emp-3",
		UserName:    "Bob Singh",
		LeaveTypeID: "lt-sick",
		LeaveType:   "Sick",
		StartDate:   sr3Start.Format("2006-01-02"),
		EndDate:     sr3End.Format("2006-01-02"),
		Days:        3,
		Reason:      "Flu",
		Status:      models.StatusApproved,
		ManagerID:   "u-mgr-2",
		ReviewedBy:  "u-mgr-2",
		CreatedAt:   now.Add(-120 * time.Hour),
		UpdatedAt:   now.Add(-96 * time.Hour),
	}
	rv3 := now.Add(-96 * time.Hour)
	sr3.ReviewedAt = &rv3
	s.leaveRequests[sr3.ID] = sr3
	s.balances["u-emp-3"]["lt-sick"].Used = 3

	// Seed audits for the sample events
	s.audits = append(s.audits, &models.AuditEntry{
		ID: NewID(), EntityType: "leave_request", EntityID: sr.ID,
		ActorID: "u-emp-1", ActorName: "Ethan Park", Action: "created",
		Timestamp: sr.CreatedAt,
	})
	s.audits = append(s.audits, &models.AuditEntry{
		ID: NewID(), EntityType: "leave_request", EntityID: sr2.ID,
		ActorID: "u-emp-2", ActorName: "Alice Chen", Action: "created",
		Timestamp: sr2.CreatedAt,
	})
	s.audits = append(s.audits, &models.AuditEntry{
		ID: NewID(), EntityType: "leave_request", EntityID: sr2.ID,
		ActorID: "u-mgr-1", ActorName: "Marcus Glover", Action: "approved",
		Timestamp: *sr2.ReviewedAt,
	})
	s.audits = append(s.audits, &models.AuditEntry{
		ID: NewID(), EntityType: "leave_request", EntityID: sr3.ID,
		ActorID: "u-emp-3", ActorName: "Bob Singh", Action: "created",
		Timestamp: sr3.CreatedAt,
	})
	s.audits = append(s.audits, &models.AuditEntry{
		ID: NewID(), EntityType: "leave_request", EntityID: sr3.ID,
		ActorID: "u-mgr-2", ActorName: "Diana Price", Action: "approved",
		Timestamp: *sr3.ReviewedAt,
	})
}

func fmtDate(y int, m time.Month, d int) string {
	return time.Date(y, m, d, 0, 0, 0, 0, time.UTC).Format("2006-01-02")
}

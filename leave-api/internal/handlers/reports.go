package handlers

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"time"

	"github.com/asdlc/leave-api/internal/middleware"
	"github.com/asdlc/leave-api/internal/models"
)

type utilizationRow struct {
	UserID     string  `json:"userId"`
	UserName   string  `json:"userName"`
	Department string  `json:"department"`
	Team       string  `json:"team"`
	LeaveType  string  `json:"leaveType"`
	Total      float64 `json:"total"`
	Used       float64 `json:"used"`
	Reserved   float64 `json:"reserved"`
	Available  float64 `json:"available"`
	UsagePct   float64 `json:"usagePct"`
}

func (h *Handlers) UtilizationReport(w http.ResponseWriter, r *http.Request) {
	p, _ := middleware.GetPrincipal(r)
	if p.Role != string(models.RoleHR) && p.Role != string(models.RoleManager) {
		writeErr(w, http.StatusForbidden, "HR or manager only")
		return
	}
	format := r.URL.Query().Get("format")
	if format == "" {
		format = "json"
	}

	rows := h.buildUtilization(p)

	switch format {
	case "json":
		writeJSON(w, http.StatusOK, rows)
	case "csv":
		w.Header().Set("Content-Type", "text/csv; charset=utf-8")
		w.Header().Set("Content-Disposition", "attachment; filename=\"utilization.csv\"")
		cw := csv.NewWriter(w)
		_ = cw.Write([]string{"user_id", "user_name", "department", "team", "leave_type", "total", "used", "reserved", "available", "usage_pct"})
		for _, row := range rows {
			_ = cw.Write([]string{
				row.UserID, row.UserName, row.Department, row.Team, row.LeaveType,
				strconv.FormatFloat(row.Total, 'f', -1, 64),
				strconv.FormatFloat(row.Used, 'f', -1, 64),
				strconv.FormatFloat(row.Reserved, 'f', -1, 64),
				strconv.FormatFloat(row.Available, 'f', -1, 64),
				strconv.FormatFloat(row.UsagePct, 'f', 2, 64),
			})
		}
		cw.Flush()
	case "pdf":
		pdf := buildUtilizationPDF(rows)
		w.Header().Set("Content-Type", "application/pdf")
		w.Header().Set("Content-Disposition", "attachment; filename=\"utilization.pdf\"")
		_, _ = w.Write(pdf)
	default:
		writeErr(w, http.StatusBadRequest, "format must be json, csv, or pdf")
	}
}

func (h *Handlers) buildUtilization(p middleware.Principal) []utilizationRow {
	users := h.Store.ListUsers()
	types := map[string]*models.LeaveType{}
	for _, lt := range h.Store.ListLeaveTypes() {
		types[lt.ID] = lt
	}
	rows := []utilizationRow{}
	for _, u := range users {
		if p.Role == string(models.RoleManager) && u.ManagerID != p.UserID && u.ID != p.UserID {
			continue
		}
		for _, b := range h.Store.GetBalances(u.ID) {
			lt := types[b.LeaveTypeID]
			name := ""
			if lt != nil {
				name = lt.Name
			}
			usage := 0.0
			if b.Total > 0 {
				usage = (b.Used / b.Total) * 100.0
			}
			rows = append(rows, utilizationRow{
				UserID: u.ID, UserName: u.Name,
				Department: u.Department, Team: u.Team,
				LeaveType: name, Total: b.Total, Used: b.Used,
				Reserved: b.Reserved, Available: b.Available,
				UsagePct: usage,
			})
		}
	}
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].UserName == rows[j].UserName {
			return rows[i].LeaveType < rows[j].LeaveType
		}
		return rows[i].UserName < rows[j].UserName
	})
	return rows
}

// buildUtilizationPDF builds a minimal but valid PDF-1.4 file with the report content.
// Uses only stdlib — no external fpdf dependency.
func buildUtilizationPDF(rows []utilizationRow) []byte {
	var content bytes.Buffer
	content.WriteString("BT\n/F1 12 Tf\n72 770 Td\n")
	title := fmt.Sprintf("Leave Utilization Report — %s", time.Now().UTC().Format("2006-01-02 15:04 UTC"))
	content.WriteString("(" + pdfEscape(title) + ") Tj\n")
	content.WriteString("0 -20 Td\n")
	content.WriteString("/F1 10 Tf\n")
	content.WriteString("(User | Dept | Team | Type | Total | Used | Reserved | Avail | Usage%) Tj\n")
	content.WriteString("0 -14 Td\n")
	for _, row := range rows {
		line := fmt.Sprintf("%s | %s | %s | %s | %.1f | %.1f | %.1f | %.1f | %.1f%%",
			row.UserName, row.Department, row.Team, row.LeaveType,
			row.Total, row.Used, row.Reserved, row.Available, row.UsagePct)
		content.WriteString("(" + pdfEscape(line) + ") Tj\n")
		content.WriteString("0 -14 Td\n")
	}
	content.WriteString("ET\n")
	stream := content.Bytes()

	// Objects
	objects := [][]byte{
		[]byte("<< /Type /Catalog /Pages 2 0 R >>"),
		[]byte("<< /Type /Pages /Count 1 /Kids [3 0 R] >>"),
		[]byte("<< /Type /Page /Parent 2 0 R /MediaBox [0 0 612 792] /Resources << /Font << /F1 5 0 R >> >> /Contents 4 0 R >>"),
		[]byte(fmt.Sprintf("<< /Length %d >>\nstream\n%s\nendstream", len(stream), stream)),
		[]byte("<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica >>"),
	}

	var buf bytes.Buffer
	buf.WriteString("%PDF-1.4\n")
	offsets := make([]int, len(objects)+1)
	for i, obj := range objects {
		offsets[i+1] = buf.Len()
		fmt.Fprintf(&buf, "%d 0 obj\n%s\nendobj\n", i+1, obj)
	}
	xrefStart := buf.Len()
	fmt.Fprintf(&buf, "xref\n0 %d\n", len(objects)+1)
	buf.WriteString("0000000000 65535 f \n")
	for i := 1; i <= len(objects); i++ {
		fmt.Fprintf(&buf, "%010d 00000 n \n", offsets[i])
	}
	fmt.Fprintf(&buf, "trailer\n<< /Size %d /Root 1 0 R >>\nstartxref\n%d\n%%%%EOF\n", len(objects)+1, xrefStart)
	return buf.Bytes()
}

func pdfEscape(s string) string {
	out := make([]byte, 0, len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch c {
		case '\\', '(', ')':
			out = append(out, '\\', c)
		default:
			if c < 32 || c > 126 {
				continue
			}
			out = append(out, c)
		}
	}
	return string(out)
}
